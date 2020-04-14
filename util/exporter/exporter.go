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
	"github.com/chubaofs/chubaofs/util/config"
	"github.com/chubaofs/chubaofs/util/log"
	"github.com/gorilla/mux"
)

const (
	AppName                 = "cfs"            //app name
	ConfigKeyExporterEnable = "exporterEnable" //exporter enable
	ConfigKeyExporterPort   = "exporterPort"   //exporter port
	ConfigKeyConsulAddr     = "consulAddr"     //consul addr
)

var (
	namespace         string
	clustername       string
	modulename        string
	Role              string
	HostIP            string
	exporterPort      int64
	enabledPrometheus = false
	collector         *Collector
)

func init() {
	collector = NewCollector()
	HostIP, _ = GetLocalIpAddr()
}

// Init initializes the exporter.
func Init(role string, cfg *config.Config) {
	Role = role

	//register backend
	if promCfg := ParsePromConfig(role, cfg); promCfg != nil && promCfg.enabled {
		b := NewPrometheusBackend(promCfg)
		collector.RegisterBackend(PromBackendName, b)
	}

	if conf := ParseJCMConfig(cfg); conf != nil {
		b := NewJCMBackend(conf)
		collector.RegisterBackend(JCMConfigKey, b)
	}

	collector.Start()
}

// Init initializes the exporter.
func InitWithRouter(role string, cfg *config.Config, router *mux.Router, exPort string) {
	Role = role

	//register backend
	if promCfg := ParsePromConfigWithRouter(role, cfg, router, exPort); promCfg != nil && promCfg.enabled {
		b := NewPrometheusBackend(promCfg)
		collector.RegisterBackend(PromBackendName, b)
	}

	if conf := ParseJCMConfig(cfg); conf != nil {
		b := NewJCMBackend(conf)
		collector.RegisterBackend(JCMConfigKey, b)
	}

	collector.Start()
}

func Stop() {
	collector.Stop()
	log.LogInfo("exporter stopped")
}
