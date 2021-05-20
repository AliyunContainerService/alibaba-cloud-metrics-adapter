package cms

import (
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
	log "k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	// metrics
	K8S_WORKLOAD_CPUUTIL          = "k8s_workload_cpu_util"
	K8S_WORKLOAD_CPULIMIT         = "k8s_workload_cpu_limit"
	K8S_WORKLOAD_CPUREQUEST       = "k8s_workload_cpu_request"
	K8S_WORKLOAD_MEMORYUSAGE      = "k8s_workload_memory_usage"
	K8S_WORKLOAD_MEMORYREQUEST    = "k8s_workload_memory_request"
	K8S_WORKLOAD_MEMORYLIMIT      = "k8s_workload_memory_limit"
	K8S_WORKLOAD_MEMORYWORKINGSET = "k8s_workload_memory_working_set"
	K8S_WORKLOAD_MEMORYRSS        = "k8s_workload_memory_rss"
	K8S_WORKLOAD_MEMORYCACHE      = "k8s_workload_memory_cache"
	K8S_WORKLOAD_NETWORKTXRATE    = "k8s_workload_network_tx_rate"
	K8S_WORKLOAD_NETWORKRXRATE    = "k8s_workload_network_rx_rate"
	K8S_WORKLOAD_NETWORKTXERRORS  = "k8s_workload_network_tx_errors"
	K8S_WORKLOAD_NETWORKRXERRORS  = "k8s_workload_network_rx_errors"
)

type CMSMetricSource struct{}

func (cs *CMSMetricSource) GetExternalMetricInfoList() []p.ExternalMetricInfo {
	metricInfoList := make([]p.ExternalMetricInfo, 0)
	var metricInfo = []string{
		K8S_WORKLOAD_CPUUTIL,
		K8S_WORKLOAD_CPULIMIT,
		K8S_WORKLOAD_CPUREQUEST,
		K8S_WORKLOAD_MEMORYUSAGE,
		K8S_WORKLOAD_MEMORYREQUEST,
		K8S_WORKLOAD_MEMORYLIMIT,
		K8S_WORKLOAD_MEMORYWORKINGSET,
		K8S_WORKLOAD_MEMORYRSS,
		K8S_WORKLOAD_MEMORYCACHE,
		K8S_WORKLOAD_NETWORKTXRATE,
		K8S_WORKLOAD_NETWORKRXRATE,
		K8S_WORKLOAD_NETWORKTXERRORS,
		K8S_WORKLOAD_NETWORKRXERRORS}
	for _, m := range metricInfo {
		metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
			Metric: m,
		})
	}
	return metricInfoList
}

func (cs *CMSMetricSource) Name() string {
	return "cms"
}

func (cs *CMSMetricSource) GetExternalMetric(info p.ExternalMetricInfo, namespace string, requirements labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {
	switch info.Metric {
	case K8S_WORKLOAD_CPUUTIL:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.cpu.usage_rate",
		})
	case K8S_WORKLOAD_CPULIMIT:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.cpu.limit",
		})
	case K8S_WORKLOAD_CPUREQUEST:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.cpu.request",
		})
	case K8S_WORKLOAD_MEMORYUSAGE:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.memory.usage",
		})
	case K8S_WORKLOAD_MEMORYREQUEST:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.memory.request",
		})
	case K8S_WORKLOAD_MEMORYLIMIT:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.memory.limit",
		})
	case K8S_WORKLOAD_MEMORYWORKINGSET:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.memory.working_set",
		})
	case K8S_WORKLOAD_MEMORYRSS:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.memory.rss",
		})
	case K8S_WORKLOAD_MEMORYCACHE:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.memory.cache",
		})
	case K8S_WORKLOAD_NETWORKTXRATE:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.network.tx_rate",
		})
	case K8S_WORKLOAD_NETWORKRXRATE:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.network.rx_rate",
		})
	case K8S_WORKLOAD_NETWORKTXERRORS:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.network.tx_errors",
		})
	case K8S_WORKLOAD_NETWORKRXERRORS:
		values, err = cs.getCMSWorkLoadMetrics(namespace, requirements, p.ExternalMetricInfo{
			Metric: "group.network.rx_errors",
		})
	}

	if err != nil {
		log.Warningf("Failed to GetMetricBySelector %s,because of %v", info.Metric, err)
	}

	return values, err
}

// register cms metric source to provider
func NewCMSMetricSource() *CMSMetricSource {
	return &CMSMetricSource{}
}
