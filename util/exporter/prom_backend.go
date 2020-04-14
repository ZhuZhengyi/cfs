// Copyright 2018 The Chubao Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package exporter

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/chubaofs/chubaofs/util/config"
	"github.com/chubaofs/chubaofs/util/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	PromBackendName    = "prometheus"
	PromHandlerPattern = "/metrics" // prometheus handler
)

type PrometheusBackend struct {
	clustername string
	config      *PromConfig
}

type PromConfig struct {
	enabled    bool
	port       int64
	role       string
	modulename string
	namespace  string
	router     *mux.Router
}

func ParsePromConfigWithRouter(role string, cfg *config.Config, router *mux.Router, exPort string) *PromConfig {
	promCfg := ParsePromConfig(role, cfg)
	promCfg.router = router
	promCfg.port, _ = strconv.ParseInt(exPort, 10, 64)
	if promCfg.port == 0 {
		promCfg.enabled = false
		enabledPrometheus = false
	} else {
		promCfg.enabled = true
		exporterPort = promCfg.port
		enabledPrometheus = true
	}

	return promCfg
}

func ParsePromConfig(role string, cfg *config.Config) *PromConfig {
	promCfg := new(PromConfig)
	promCfg.modulename = role
	promCfg.namespace = AppName + "_" + role
	namespace = promCfg.namespace
	modulename = promCfg.modulename

	if !cfg.GetBoolWithDefault(ConfigKeyExporterEnable, true) {
		log.LogInfof("%v metrics exporter disabled", cfg)
		promCfg.enabled = false
		return promCfg
	}

	exporterPort = cfg.GetInt64(ConfigKeyExporterPort)
	promCfg.port = exporterPort
	promCfg.role = role
	promCfg.enabled = true
	enabledPrometheus = true

	return promCfg
}

func NewPrometheusBackend(cfg *PromConfig) (b *PrometheusBackend) {
	b = new(PrometheusBackend)
	b.config = cfg
	return
}

func (b *PrometheusBackend) Start() {
	if b.config.router == nil {
		http.Handle(PromHandlerPattern, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
			Timeout: 10 * time.Second,
		}))
		addr := fmt.Sprintf(":%d", b.config.port)
		go func() {
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				log.LogError("exporter http serve error: ", err)
			}
		}()
	} else {
		b.config.router.NewRoute().Name(PromHandlerPattern).
			Methods(http.MethodGet).
			Path(PromHandlerPattern).
			Handler(promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
				Timeout: 10 * time.Second,
			}))

		log.LogDebugf("exporter: prom_backend start router %v", PromHandlerPattern)
	}

	log.LogDebugf("exporter: prometheus backend start")
}

func (b *PrometheusBackend) Stop() {
}

func (b *PrometheusBackend) Push(m Metric) {
	switch mt := m.(type) {
	case *Gauge:
		metric := mt.Metric()
		metric.Set(float64(mt.Val()))
	case *Counter:
		metric := mt.Metric()
		metric.Add(float64(mt.Val()))
	case *TimePoint:
		metric := mt.Metric()
		metric.Set(float64(mt.Val()))
	case *Alarm:
		metric := mt.Metric()
		metric.Add(float64(mt.Val()))
	default:
	}
}
