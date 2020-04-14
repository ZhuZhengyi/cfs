package exporter

import (
	"sync"
	"time"

	"github.com/chubaofs/chubaofs/util/log"
)

const (
	CollectChSize = 1024 * 32 //collect chan size
)

//
type Backend interface {
	Start()        //start with backend
	Stop()         //
	Push(m Metric) //push metric with backend
}

type Collector struct {
	collectCh chan Metric //metrics collect channel
	stopCh    chan bool   //stop signal channel
	backends  map[string]Backend
	mu        *sync.Mutex
}

func NewCollector() *Collector {
	c := new(Collector)
	c.collectCh = make(chan Metric, CollectChSize)
	c.stopCh = make(chan bool, 0)
	c.backends = make(map[string]Backend)
	c.mu = new(sync.Mutex)

	return c
}

// register backend for metric collector
func (c *Collector) RegisterBackend(name string, b Backend) {
	c.mu.Lock()
	c.backends[name] = b
	c.mu.Unlock()

	log.LogInfof("exporter: metric collector register backend: %v", name)
}

func (c *Collector) Start() {
	for _, b := range c.backends {
		b.Start()
	}

	go c.doCollect()

	m := NewGauge("start_time")
	m.Set(time.Now().Unix() * 1000)
}

// stop exporter
func (c *Collector) Stop() {
	c.stopCh <- true

	for _, b := range c.backends {
		b.Stop()
	}
}

func (c *Collector) doCollect() {
	for {
		select {
		case <-c.stopCh:
			log.LogInfof("stop exporter collect")
			return
		case m := <-c.collectCh:
			for _, b := range c.backends {
				b.Push(m)
			}
		}
		//time.Sleep(10 * tim)
	}
}
