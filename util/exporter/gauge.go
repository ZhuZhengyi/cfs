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
	"sync"

	"github.com/chubaofs/chubaofs/util/log"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	GaugeGroup sync.Map
)

type Gauge struct {
	name   string
	labels map[string]string
	val    float64
}

func NewGauge(name string) (g *Gauge) {
	if !enabledPrometheus {
		return
	}
	g = new(Gauge)
	g.name = MetricsName(name)
	return
}

func (g *Gauge) Key() string {
	return fmt.Sprintf("{%s: %s}", g.name, stringMapToString(g.labels))
}

func (g *Gauge) Name() string {
	return g.name
}

func (g *Gauge) Labels() map[string]string {
	return g.labels
}

func (g *Gauge) Val() float64 {
	return g.val
}

func (g *Gauge) String() string {
	return fmt.Sprintf("{name: %s, labels: %s, val: %v}", g.name, stringMapToString(g.labels), g.val)
}

func (c *Gauge) Metric() prometheus.Gauge {
	metric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        c.name,
			ConstLabels: c.labels,
		})
	key := c.Key()
	actualMetric, load := GaugeGroup.LoadOrStore(key, metric)
	if !load {
		err := prometheus.Register(actualMetric.(prometheus.Collector))
		if err == nil {
			log.LogInfof("register metric %v", c.Name())
		} else {
			log.LogErrorf("register metric %v, %v", c.Name(), err)
		}
	}

	return actualMetric.(prometheus.Gauge)
}

func (g *Gauge) Set(val int64) {
	if !enabledPrometheus {
		return
	}
	g.val = float64(val)
	g.Publish()
}

func (c *Gauge) Publish() {
	select {
	case collector.collectCh <- c:
	default:
	}
}

func (g *Gauge) SetWithLabels(val int64, labels map[string]string) {
	if !enabledPrometheus {
		return
	}
	g.labels = labels
	g.Set(val)
}
