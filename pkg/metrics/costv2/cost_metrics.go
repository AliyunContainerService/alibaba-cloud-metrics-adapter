package costv2

import (
	"context"
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider"
	"github.com/prometheus/common/model"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	p "sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	prom "sigs.k8s.io/prometheus-adapter/pkg/client"
	"strings"
)

const (
	FilteredPodInfo       = "filtered_pod_info"
	CPUCoreRequestAverage = "cpu_core_request_average"
	CPUCoreUsageAverage   = "cpu_core_usage_average"
	MemoryRequestAverage  = "memory_request_average"
	MemoryUsageAverage    = "memory_usage_average"

	QueryFilteredPodInfo       = `max(kube_pod_labels{%s}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{%s}`
	QueryCPUCoreRequestAverage = `sum(avg_over_time(kube_pod_container_resource_requests{job="_kube-state-metrics", resource="cpu"}[%s])) by (namespace, pod)`
	QueryCPUCoreUsageAverage   = `sum(avg_over_time(rate(container_cpu_usage_seconds_total[1m])[%s])) by(namespace, pod)`
	QueryMemoryRequestAverage  = `sum(avg_over_time(kube_pod_container_resource_requests{job="_kube-state-metrics", resource="memory"}[%s])) by (namespace, pod)`
	QueryMemoryUsageAverage    = `sum(avg_over_time(container_memory_working_set_bytes[%s])) by(namespace, pod)`
)

type COSTV2MetricSource struct {
	*prometheusProvider.AlibabaMetricsAdapterOptions
}

// list all external metric
func (cs *COSTV2MetricSource) GetExternalMetricInfoList() []p.ExternalMetricInfo {
	metricInfoList := make([]p.ExternalMetricInfo, 0)
	var MetricArray = []string{
		CPUCoreRequestAverage,
		CPUCoreUsageAverage,
		MemoryRequestAverage,
		MemoryUsageAverage,
	}
	for _, metric := range MetricArray {
		metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
			Metric: metric,
		})
	}
	return metricInfoList
}

// according to the incoming label, get the metric..
func (cs *COSTV2MetricSource) GetExternalMetric(info p.ExternalMetricInfo, namespace string, requirements labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {
	promQL := getPrometheusSql(info.Metric)
	query := buildExternalQuery(promQL, requirements)
	values, err = cs.getCOSTMetrics(namespace, info.Metric, query)
	if err != nil {
		klog.Warningf("Failed to GetExternalMetric %s,because of %v", info.Metric, err)
	}
	return values, err
}

func (cs *COSTV2MetricSource) getCOSTMetrics(namespace, metricName string, query prom.Selector) ([]external_metrics.ExternalMetricValue, error) {
	client, err := prometheusProvider.GlobalConfig.MakePromClient()
	if err != nil {
		klog.Errorf("Failed to create prometheus client,because of %v", err)
		return nil, err
	}
	klog.V(4).Infof("external query :%+v", query)
	queryResult, err := client.Query(context.TODO(), model.Now(), query)
	if err != nil {
		klog.Errorf("unable to fetch metrics from prometheus: %v", err)
		return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics"))
	}
	klog.V(4).Infof("queryResult for %s: %v", metricName, queryResult)
	return cs.convertVector(metricName, queryResult)
}

func (cs *COSTV2MetricSource) convertVector(metricName string, queryResult prom.QueryResult) (value []external_metrics.ExternalMetricValue, err error) {
	if queryResult.Type != model.ValVector {
		return nil, fmt.Errorf("incorrect query result type")
	}

	toConvert := *queryResult.Vector
	if toConvert == nil || toConvert.Len() == 0 {
		return nil, fmt.Errorf("the provided input did not contain vector query results")
	}

	items := make([]external_metrics.ExternalMetricValue, 0)
	for _, val := range toConvert {
		singleMetric, err := convertSample(metricName, val)
		if err != nil {
			return nil, fmt.Errorf("unable to convert vector: %v", err)
		}
		items = append(items, *singleMetric)
	}
	return items, nil
}

func convertSample(metricName string, sample *model.Sample) (*external_metrics.ExternalMetricValue, error) {
	labels := convertLabels(sample.Metric)
	singleMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Timestamp: metav1.Time{
			sample.Timestamp.Time(),
		},
		Value:        *resource.NewMilliQuantity(int64(sample.Value*1000.0), resource.DecimalSI),
		MetricLabels: labels,
	}
	return &singleMetric, nil
}

func convertLabels(metric model.Metric) map[string]string {
	labels := make(map[string]string)
	for k, v := range metric {
		labels[string(k)] = string(v)
	}
	return labels
}

func buildExternalQuery(promQL string, requirements labels.Requirements) (externalQuery prom.Selector) {
	requirementMap := make(map[string]string)
	kubePodLabelStr := ""
	for _, value := range requirements {
		if strings.HasPrefix(value.Key(), "label_") {
			kubePodLabelStr = fmt.Sprintf(`%s=~"%s"`, value.Key(), value.Values().List()[0])
		} else {
			klog.Infof("requirement key: %s, value: %s", value.Key(), value.Values().List()[0])
			if value.Values().List()[0] == "" {
				requirementMap[value.Key()] = ".*"
			} else {
				requirementMap[value.Key()] = value.Values().List()[0]
			}
		}
	}
	klog.Infof("requirementMap: %v", requirementMap)
	kubePodInfoStr := fmt.Sprintf(`namespace=~"%s",created_by_kind=~"%s",created_by_name=~"%s",pod=~"%s"`,
		requirementMap["namespace"], requirementMap["created_by_kind"], requirementMap["created_by_name"], requirementMap["pod"])
	externalQuery = prom.Selector(fmt.Sprintf(promQL, "1h:30m", kubePodLabelStr, kubePodInfoStr))
	return externalQuery
}

func getPrometheusSql(metricName string) (item string) {
	switch metricName {
	case CPUCoreRequestAverage:
		//item = `avg(avg_over_time(kube_pod_container_resource_requests{resource="cpu", unit="core", container!="", container!="POD", node!="", %s}[%s])) by (container, pod, namespace, node, %s)`
		//item = `sum(kube_pod_container_resource_requests_cpu_cores{job="_kube-state-metrics"}) by(pod) * on(pod) group_right sum(kube_pod_labels{%s}) by(pod)`
		item = fmt.Sprintf("%s * %s", QueryCPUCoreRequestAverage, QueryFilteredPodInfo)
	case CPUCoreUsageAverage:
		item = fmt.Sprintf("%s * %s", QueryCPUCoreUsageAverage, QueryFilteredPodInfo)
	case MemoryRequestAverage:
		item = fmt.Sprintf("%s * %s", QueryMemoryRequestAverage, QueryFilteredPodInfo)
	case MemoryUsageAverage:
		item = fmt.Sprintf("%s * %s", QueryMemoryUsageAverage, QueryFilteredPodInfo)
	}
	return item
}

func NewCOSTV2MetricSource() *COSTV2MetricSource {
	return &COSTV2MetricSource{
		prometheusProvider.GlobalConfig,
	}
}
