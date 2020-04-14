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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chubaofs/chubaofs/util/config"
	"github.com/chubaofs/chubaofs/util/log"
)

const (
	JCMBackendName        = "jcm"
	JCMCollectPeriod      = 15 // second
	JCMCollectSize        = 32 * 1024
	JCMPendingSize        = 32 * 1024 //
	DataPointsPostMaxSize = 50        //一次最多发送datapoint个数

	JCMConfigKey           = JCMBackendName
	JCMConfigPushPeriodKey = "push_period"
	JCMDPHostIPKey         = "hostip"
	JCMDPClusterKey        = "cluster"
	JCMDPRoleKey           = "role"
)

type DataPoint struct {
	Metric    string            `json:"metric"`
	Timestamp int64             `json:"timestamp"`
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"` //
}

func (d *DataPoint) Key() string {
	return fmt.Sprintf("{%s: %s}", d.Metric, stringMapToString(d.Tags))
}

func NewDataPoint(name string, tags map[string]string) *DataPoint {
	return &DataPoint{
		Metric: name,
		Tags:   tags,
	}
}

func (dp *DataPoint) SetValue(val float64) {
	dp.Value = val
}

func (dp *DataPoint) SetTS(interval int64) {
	ts := time.Now().Unix()
	dp.Timestamp = ts - (ts % interval)
}

type JCMReq struct {
	AppCode     string      `json:"appCode"`
	ServiceCode string      `json:"serviceCode"`
	DataCenter  string      `json:"dataCenter,omitempty"`
	Region      string      `json:"region,omitempty"`
	ResourceId  string      `json:"resourceId"`
	DataPoints  []DataPoint `json:"dataPoints"`
}

type JCMResp struct {
	Failed    int64  `json:"failed"`
	Success   int64  `json:"success"`
	RequestId string `json:"requestId"`
}

type JCMConfig struct {
	AppCode     string            `json:"appCode"`
	ServiceCode string            `json:"serviceCode"`
	DataCenter  string            `json:"dataCenter"`
	ResourceId  string            `json:"resourceId"`
	Url         string            `json:"url"`
	PushPeriod  int64             `json:"push_peroid,omitempty"`
	Metrics     map[string]string `json:"metrics,omitempty"` //
}

type JCMBackend struct {
	stopC    chan bool       //stop
	pendingC chan *DataPoint //metrics which would be push out
	config   *JCMConfig
}

func NewJCMBackend(config *JCMConfig) *JCMBackend {
	b := new(JCMBackend)
	b.stopC = make(chan bool)
	b.pendingC = make(chan *DataPoint, JCMCollectSize)
	b.config = config

	return b
}

//parse jcm config
func ParseJCMConfig(cfg *config.Config) (jcmCfg *JCMConfig) {
	if data := cfg.GetKeyRaw(JCMConfigKey); data != nil {
		jcmCfg = new(JCMConfig)
		if err := json.Unmarshal(data, jcmCfg); err != nil {
			jcmCfg = nil
			log.LogErrorf("parse jcm config error: %v", cfg)
		}
		if jcmCfg.PushPeriod == 0 {
			jcmCfg.PushPeriod = JCMCollectPeriod
		}
	}

	return
}

func (dp *DataPoint) AddCommonTags() {
	if dp.Tags == nil {
		dp.Tags = make(map[string]string)
	}
	dp.Tags[JCMDPHostIPKey] = HostIP
	dp.Tags[JCMDPRoleKey] = Role
	dp.Tags[JCMDPClusterKey] = clustername
}

// push metric to backend
func (b *JCMBackend) Push(m Metric) {
	name := m.Name()
	if len(b.config.Metrics) > 0 {
		if newName, ok := b.config.Metrics[name]; ok {
			name = newName
		} else {
			return
		}
	}
	dp := NewDataPoint(name, m.Labels())
	dp.AddCommonTags()
	dp.SetValue(m.Val())
	dp.SetTS(b.config.PushPeriod)

	//
	select {
	case b.pendingC <- dp:
	default:
	}
}

// start jcm backend
func (b *JCMBackend) Start() {
	ticker := time.NewTicker(time.Duration(b.config.PushPeriod) * time.Second)
	defer func() {
		if err := recover(); err != nil {
			ticker.Stop()
			log.LogErrorf("start jcm collector error,err[%v]", err)
		}
	}()
	client := &http.Client{}
	defer client.CloseIdleConnections()

	dps := make(map[string]*DataPoint)
	go func() {
		for {
			select {
			case <-b.stopC:
				log.LogInfo("jcm collector stopped.")
				return
			case <-ticker.C:
				pending_empty := false
				for !pending_empty {
					for len(dps) < DataPointsPostMaxSize {
						select {
						case d := <-b.pendingC:
							dps[d.Key()] = d
						default:
							pending_empty = true
							goto BatchSend
						}
					}
				BatchSend:
					if len(dps) > 0 {
						req := b.makeReq(dps)
						if resp, e := client.Do(req); e != nil {
							log.LogErrorf("exporter: jcm_backend sent error %v %v", resp, e)
						}
						log.LogDebugf("exporter: batch send to jcm size %v, %v", len(dps), req)
					}
				}
			}
		}
	}()

	log.LogDebugf("exporter: jcm backend start")
}

// stop jcm backend
func (b *JCMBackend) Stop() {
	log.LogInfo("jcm collector stopping.")
	b.stopC <- true
}

func (b *JCMBackend) makeReq(dps map[string]*DataPoint) (req *http.Request) {
	monitorReq := JCMReq{
		AppCode:     b.config.AppCode,
		ServiceCode: b.config.ServiceCode,
		DataCenter:  b.config.DataCenter,
		ResourceId:  b.config.ResourceId,
	}
	for _, dp := range dps {
		monitorReq.DataPoints = append(monitorReq.DataPoints, *dp)
	}
	reqBytes, err := json.Marshal(&monitorReq)
	if err != nil {
		log.LogErrorf("marshal error, %v", err.Error())
		return nil
	}
	req, err = http.NewRequest(http.MethodPut, b.config.Url, bytes.NewBuffer(reqBytes))
	if err != nil {
		log.LogErrorf("new request error, %v", err.Error())
		return nil
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	return
}
