package metrics

import (
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/cms"

	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/slb"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/sls"
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
	log "k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

var externalMetricsManager *ExternalMetricsManager

func init() {
	externalMetricsManager = &ExternalMetricsManager{
		metricsSource: make(map[p.ExternalMetricInfo]MetricSource),
	}

	// add metrics source
	register(sls.NewSLSMetricSource())
	register(slb.NewSLBMetricSource())
	register(cms.NewCMSMetricSource())
}

func GetExternalMetricsManager() *ExternalMetricsManager {
	return externalMetricsManager
}

func register(m MetricSource) {
	externalMetricsManager.AddMetricsSource(m)
}

type MetricSource interface {
	GetExternalMetricInfoList() []p.ExternalMetricInfo
	GetExternalMetric(info p.ExternalMetricInfo, namespace string, requirements labels.Requirements) ([]external_metrics.ExternalMetricValue, error)
}

type ExternalMetricsManager struct {
	metricsSource map[p.ExternalMetricInfo]MetricSource
}

func (em *ExternalMetricsManager) AddMetricsSource(m MetricSource) {
	metricInfoList := m.GetExternalMetricInfoList()
	for _, p := range metricInfoList {
		log.Infof("Register metric: %v to external metrics manager\n", p)
		em.metricsSource[p] = m
	}
}

func (em *ExternalMetricsManager) GetMetricsInfoList() []p.ExternalMetricInfo {
	metricsInfoList := make([]p.ExternalMetricInfo, 0)
	for source, _ := range em.metricsSource {
		metricsInfoList = append(metricsInfoList, source)
	}
	return metricsInfoList
}

func (em *ExternalMetricsManager) GetExternalMetrics(namespace string, requirements labels.Requirements, info p.ExternalMetricInfo) ([]external_metrics.ExternalMetricValue, error) {
	if source, ok := em.metricsSource[info]; ok {
		return source.GetExternalMetric(info, namespace, requirements)
	}

	return nil, fmt.Errorf("The specific metric source %s is not found.\n", info.Metric)
}
