package external_metrics_source

import (
	"fmt"
	"time"

	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/external_metrics_source/ahas"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/external_metrics_source/cms"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/external_metrics_source/prom"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/external_metrics_source/slb"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/external_metrics_source/sls"
	"github.com/emirpasic/gods/sets/hashset"
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const DefaultInterval = 30 * time.Second

// RegisterMetricsSource add external metrics source
func (em *ExternalMetricsManager) RegisterMetricsSource() {
	em.register(sls.NewSLSMetricSource())
	em.register(slb.NewSLBMetricSource())
	em.register(cms.NewCMSMetricSource())
	em.register(ahas.NewAHASSentinelMetricSource())
	prometheusSource := prom.NewPrometheusSource(em.prometheusUrl)

	go func() {
		for {
			em.register(prometheusSource)
			time.Sleep(DefaultInterval)
		}
	}()
}

func NewExternalMetricsManager(prometheusUrl string) *ExternalMetricsManager {
	recorder := make(map[string]*hashset.Set)
	recorder["prometheus"] = hashset.New()

	return &ExternalMetricsManager{
		prometheusUrl: prometheusUrl,
		recorder:      recorder,
		metrics:       make(map[p.ExternalMetricInfo]MetricSource),
	}
}

func (em *ExternalMetricsManager) register(m MetricSource) {
	em.AddMetricsSource(m)
}

type MetricSource interface {
	Name() string
	GetExternalMetricInfoList() []p.ExternalMetricInfo
	GetExternalMetric(info p.ExternalMetricInfo, namespace string, requirements labels.Requirements) ([]external_metrics.ExternalMetricValue, error)
}

type ExternalMetricsManager struct {
	prometheusUrl string
	recorder      map[string]*hashset.Set
	metrics       map[p.ExternalMetricInfo]MetricSource
}

func (em *ExternalMetricsManager) AddMetricsSource(m MetricSource) {
	metricInfoList := m.GetExternalMetricInfoList()
	if em.recorder[m.Name()].Size() != len(metricInfoList) {
		if em.recorder[m.Name()].Size() == 0 {
			// nothing
		} else if len(metricInfoList) == 0 {
			for _, oldMetric := range em.recorder[m.Name()].Values() {
				delete(em.metrics, oldMetric.(p.ExternalMetricInfo))
			}
		} else {
			for _, oldMetric := range em.recorder[m.Name()].Values() {
				for index, newMetric := range metricInfoList {
					if oldMetric == newMetric {
						continue
					} else if index == len(metricInfoList)-1 {
						delete(em.metrics, oldMetric.(p.ExternalMetricInfo))
					}
				}
			}
		}
	}

	em.recorder[m.Name()].Clear()
	for _, metricInfo := range metricInfoList {
		em.metrics[metricInfo] = m
		em.recorder[m.Name()].Add(metricInfo)
	}
}

func (em *ExternalMetricsManager) GetMetricsInfoList() []p.ExternalMetricInfo {
	metricsInfoList := make([]p.ExternalMetricInfo, 0)

	for metricInfo := range em.metrics {
		metricsInfoList = append(metricsInfoList, metricInfo)
	}
	return metricsInfoList
}

func (em *ExternalMetricsManager) GetExternalMetrics(namespace string, requirements labels.Requirements, info p.ExternalMetricInfo) ([]external_metrics.ExternalMetricValue, error) {
	if source, ok := em.metrics[info]; ok {
		return source.GetExternalMetric(info, namespace, requirements)
	}

	return nil, fmt.Errorf("metric %s is not found", info.Metric)
}
