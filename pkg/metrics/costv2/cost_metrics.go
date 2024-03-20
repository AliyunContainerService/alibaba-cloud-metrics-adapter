package costv2

import (
	"context"
	"fmt"
	util "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/util"
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
	"time"
)

const (
	FilteredPodInfo       = "filtered_pod_info"
	CPUCoreRequestAverage = "cpu_core_request_average"
	CPUCoreUsageAverage   = "cpu_core_usage_average"
	MemoryRequestAverage  = "memory_request_average"
	MemoryUsageAverage    = "memory_usage_average"
	CostCPURequest        = "cost_cpu_request"
	CostMemoryRequest     = "cost_memory_request"
	CostTotal             = "cost_total"

	QueryFilteredPodInfo       = `max(kube_pod_labels{%s}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{%s}`
	QueryCPUCoreRequestAverage = `sum(avg_over_time(kube_pod_container_resource_requests{job="_kube-state-metrics", resource="cpu"}[%s])) by (namespace, pod)`
	QueryCPUCoreUsageAverage   = `sum(avg_over_time(rate(container_cpu_usage_seconds_total[1m])[%s])) by(namespace, pod)`
	QueryMemoryRequestAverage  = `sum(avg_over_time(kube_pod_container_resource_requests{job="_kube-state-metrics", resource="memory"}[%s])) by (namespace, pod)`
	QueryMemoryUsageAverage    = `sum(avg_over_time(container_memory_working_set_bytes[%s])) by(namespace, pod)`
	QueryCostCPURequest        = `sum(sum_over_time((max(node_current_price) by (node) / on (node)  group_left kube_node_status_capacity{job="_kube-state-metrics",resource="cpu"} * on(node) group_right kube_pod_container_resource_requests{job="_kube-state-metrics",resource="cpu"})[%s])) by (namespace, pod) * 3600`
	QueryCostMemoryRequest     = `sum(sum_over_time((max(node_current_price) by (node) / on (node)  group_left kube_node_status_capacity{job="_kube-state-metrics",resource="memory"} * on(node) group_right kube_pod_container_resource_requests{job="_kube-state-metrics",resource="memory"})[%s])) by (namespace, pod) * 3600`
	QueryCostTotal             = `sum(sum_over_time((max(node_current_price) by (node))[%s])) * 3600`
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
		CostCPURequest,
		CostMemoryRequest,
		CostTotal,
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
	requirementMap := parseRequirements(requirements)
	query := buildExternalQuery(info.Metric, requirementMap)
	end, err := time.Parse(requirementMap["window_layout"][0], requirementMap["window_end"][0])
	if err != nil {
		fmt.Println("Error parsing end time:", err)
		return
	}
	values, err = cs.getCOSTMetricsAtTime(namespace, info.Metric, query, end)
	if err != nil {
		klog.Warningf("Failed to GetExternalMetric %s,because of %v", info.Metric, err)
	}
	return values, err
}

func (cs *COSTV2MetricSource) getCOSTMetricsAtTime(namespace, metricName string, query prom.Selector, end time.Time) ([]external_metrics.ExternalMetricValue, error) {
	client, err := prometheusProvider.GlobalConfig.MakePromClient()
	if err != nil {
		klog.Errorf("Failed to create prometheus client,because of %v", err)
		return nil, err
	}

	endUTC := util.GetUTCTime(end)
	endTime := model.TimeFromUnixNano(endUTC.UnixNano())
	klog.V(4).Infof("external query at UTC time %v: %v", endUTC, query)

	queryResult, err := client.Query(context.TODO(), endTime, query)
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

func parseRequirements(requirements labels.Requirements) (requirementMap map[string][]string) {
	requirementMap = make(map[string][]string)

	for _, value := range requirements {
		klog.Infof("requirement key: %s, value: %s", value.Key(), value.Values().List()[0])
		requirementMap[value.Key()] = value.Values().List()
	}

	klog.Infof("requirementMap: %v", requirementMap)
	return requirementMap
}

func parsePromLabel(item []string) string {
	// todo use array for multi ns, pod etc.
	if item[0] == "" {
		return ".*"
	}
	return item[0]
}

func buildExternalQuery(metricName string, requirementMap map[string][]string) (externalQuery prom.Selector) {
	// build str for kube_pod_labels
	kubePodLabelStr := ""
	for key, value := range requirementMap {
		if strings.HasPrefix(key, "label_") {
			kubePodLabelStr = fmt.Sprintf(`%s=~"%s"`, key, value[0])
		}
	}

	// build str for kube_pod_info
	kubePodInfoStr := fmt.Sprintf(`namespace=~"%s",created_by_kind=~"%s",created_by_name=~"%s",pod=~"%s"`,
		parsePromLabel(requirementMap["namespace"]), parsePromLabel(requirementMap["created_by_kind"]), parsePromLabel(requirementMap["created_by_name"]), parsePromLabel(requirementMap["pod"]))

	// build str for prom duration
	layout := requirementMap["window_layout"][0]
	start, err := time.Parse(layout, requirementMap["window_start"][0])
	if err != nil {
		fmt.Println("Error parsing start time:", err)
		return
	}
	end, err := time.Parse(layout, requirementMap["window_end"][0])
	if err != nil {
		fmt.Println("Error parsing end time:", err)
		return
	}
	durStr := fmt.Sprintf("%s:%s", util.DurationString(end.Sub(start)), "1h")

	switch metricName {
	case CPUCoreRequestAverage:
		item := fmt.Sprintf("%s * %s", QueryCPUCoreRequestAverage, QueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr))
	case CPUCoreUsageAverage:
		item := fmt.Sprintf("%s * %s", QueryCPUCoreUsageAverage, QueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr))
	case MemoryRequestAverage:
		item := fmt.Sprintf("%s * %s", QueryMemoryRequestAverage, QueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr))
	case MemoryUsageAverage:
		item := fmt.Sprintf("%s * %s", QueryMemoryUsageAverage, QueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr))
	case CostCPURequest:
		item := fmt.Sprintf("%s * %s", QueryCostCPURequest, QueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr))
	case CostMemoryRequest:
		item := fmt.Sprintf("%s * %s", QueryCostMemoryRequest, QueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr))
	case CostTotal:
		item := fmt.Sprintf("%s", QueryCostTotal)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr))
	}

	return externalQuery
}

func getPrometheusSql(metricName string) (item string) {
	switch metricName {
	case CPUCoreRequestAverage:
		item = fmt.Sprintf("%s * %s", QueryCPUCoreRequestAverage, QueryFilteredPodInfo)
	case CPUCoreUsageAverage:
		item = fmt.Sprintf("%s * %s", QueryCPUCoreUsageAverage, QueryFilteredPodInfo)
	case MemoryRequestAverage:
		item = fmt.Sprintf("%s * %s", QueryMemoryRequestAverage, QueryFilteredPodInfo)
	case MemoryUsageAverage:
		item = fmt.Sprintf("%s * %s", QueryMemoryUsageAverage, QueryFilteredPodInfo)
	case CostCPURequest:
		item = fmt.Sprintf("%s * %s", QueryCostCPURequest, QueryFilteredPodInfo)
	case CostMemoryRequest:
		item = fmt.Sprintf("%s * %s", QueryCostMemoryRequest, QueryFilteredPodInfo)
	case CostTotal:
		item = fmt.Sprintf("%s", QueryCostTotal)
	}
	return item
}

func NewCOSTV2MetricSource() *COSTV2MetricSource {
	return &COSTV2MetricSource{
		prometheusProvider.GlobalConfig,
	}
}
