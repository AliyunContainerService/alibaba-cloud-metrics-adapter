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
	"strconv"
	"strings"
	"time"
)

const (
	// custom metric name
	CPUCoreRequestAverage         = "cpu_core_request_average"
	CPUCoreUsageAverage           = "cpu_core_usage_average"
	MemoryRequestAverage          = "memory_request_average"
	MemoryUsageAverage            = "memory_usage_average"
	CostPodCPURequest             = "cost_pod_cpu_request"
	CostPodMemoryRequest          = "cost_pod_memory_request"
	CostTotal                     = "cost_total"
	CostNode                      = "cost_node"
	CostCustom                    = "cost_custom"
	BillingPretaxAmountTotal      = "billing_pretax_amount_total"
	BillingPretaxGrossAmountTotal = "billing_pretax_gross_amount_total"
	BillingPretaxAmountNode       = "billing_pretax_amount_node"

	KubePodInfo   = "metrics_kube_pod_info"
	KubePodLabels = "metrics_kube_pod_labels"
	KubeNodeInfo  = "metrics_kube_node_info"

	// PromQL
	QueryCPUCoreRequestAverage         = `sum(avg_over_time((max(kube_pod_container_resource_requests{resource="cpu"}) by (pod,namespace,container))[%s])) by (namespace, pod)`
	QueryCPUCoreUsageAverage           = `sum(avg_over_time(rate(container_cpu_usage_seconds_total[1m])[%s])) by(namespace, pod)`
	QueryMemoryRequestAverage          = `sum(avg_over_time((max(kube_pod_container_resource_requests{resource="memory"}) by (pod,namespace,container))[%s])) by (namespace, pod)`
	QueryMemoryUsageAverage            = `sum(avg_over_time(container_memory_working_set_bytes[%s])) by(namespace, pod)`
	QueryCostPodCPURequest             = `sum(sum_over_time((max(node_current_price) by (node) / on (node)  group_left max(kube_node_status_capacity{resource="cpu"}) by(node) * on(node) group_right max(kube_pod_container_resource_requests{resource="cpu"}) by (node,pod,namespace,container) * on(pod, namespace) group_left max(kube_pod_status_phase{phase=~"Running"}) by (pod,namespace))[%s])) by (namespace, pod) * %s`
	QueryCostPodMemoryRequest          = `sum(sum_over_time((max(node_current_price) by (node) / on (node)  group_left max(kube_node_status_capacity{resource="memory"}) by(node) * on(node) group_right max(kube_pod_container_resource_requests{resource="memory"}) by (node,pod,namespace,container) * on(pod, namespace) group_left max(kube_pod_status_phase{phase=~"Running"}) by (pod,namespace))[%s])) by (namespace, pod) * %s`
	QueryCostTotal                     = `sum(sum_over_time((max(node_current_price{%s}) by (node))[%s])) * %s`
	QueryCostNode                      = `sum_over_time((max(node_current_price{%s}) by (node))[%s]) * %s`
	QueryCostCustom                    = `sum_over_time((max(label_replace(label_replace(pod_custom_price, "namespace", "$1", "exported_namespace", "(.*)"), "pod", "$1", "exported_pod", "(.*)")) by (namespace,pod))[%s]) * %s`
	QueryBillingPretaxAmountTotal      = `sum(sum_over_time(max(pretax_amount{%s}) by (product_code, instance_id)[%s]))`
	QueryBillingPretaxGrossAmountTotal = `sum(sum_over_time(max(pretax_gross_amount{%s}) by (product_code, instance_id)[%s]))`
	QueryBillingPretaxAmountNode       = `sum(sum_over_time(max(pretax_amount{product_code="ecs"%s}) by (product_code, instance_id)[%s]))`

	// QueryFilteredPodInfo is the Pod Filter
	// `max(kube_pod_labels{%s}) by (pod,namespace)`, value is 1, used to filter pods with specified labels.
	// `kube_pod_info{%s}`, value is 1, used to filter pods with specified name, controller or controller name.
	QueryFilteredPodInfo   = `max_over_time((max(kube_pod_labels{%s}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{%s})[%s])`
	QueryFilteredPodLabels = `max_over_time((max(kube_pod_info{%s}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_labels{%s})[%s])`
	QueryNodeInfo          = `max_over_time(kube_node_info{%s}[%s])`
	//先初始化meta，后面只做计算逻辑
	//后面如果遇到不一致，就只判断有meta就填入，，没有就算了】
)

type COSTV2MetricSource struct {
	*prometheusProvider.AlibabaMetricsAdapterOptions
}

// list all external metric
func (cs *COSTV2MetricSource) GetExternalMetricInfoList() []p.ExternalMetricInfo {
	metricInfoList := make([]p.ExternalMetricInfo, 0)
	var MetricArray = []string{
		KubePodInfo,
		KubePodLabels,
		KubeNodeInfo,
		CPUCoreRequestAverage,
		CPUCoreUsageAverage,
		MemoryRequestAverage,
		MemoryUsageAverage,
		CostPodCPURequest,
		CostPodMemoryRequest,
		CostTotal,
		CostNode,
		CostCustom,
		BillingPretaxAmountTotal,
		BillingPretaxGrossAmountTotal,
		BillingPretaxAmountNode,
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
	values, err = cs.getCostMetricsAtTime(namespace, info.Metric, query, end)
	if err != nil {
		klog.Warningf("Failed to GetExternalMetric %s,because of %v", info.Metric, err)
	}
	return values, err
}

func (cs *COSTV2MetricSource) getCostMetricsAtTime(namespace, metricName string, query prom.Selector, end time.Time) ([]external_metrics.ExternalMetricValue, error) {
	client, err := prometheusProvider.GlobalConfig.MakePromClient()
	if err != nil {
		klog.Errorf("Failed to create prometheus client,because of %v", err)
		return nil, err
	}

	// billing metrics are always 00:00:00, add -1 second to avoid data duplication
	if metricName == BillingPretaxGrossAmountTotal || metricName == BillingPretaxAmountTotal || metricName == BillingPretaxAmountNode {
		if end.Hour() == 0 && end.Minute() == 0 && end.Second() == 0 {
			end = end.Add(-time.Second)
		}
	}
	endUTC := util.GetUTCTime(end)
	endTime := model.TimeFromUnixNano(endUTC.UnixNano())
	klog.V(4).Infof("external query at UTC time %v: %v", endUTC, query)

	queryResult, err := client.Query(context.TODO(), endTime, query)
	if err != nil {
		klog.Errorf("unable to fetch metrics from prometheus: %v", err)
		return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics"))
	}
	klog.V(4).Infof("fetch metrics successfully, queryResult for %s: %v", metricName, queryResult)

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
		requirementMap[value.Key()] = value.Values().List()
	}

	klog.Infof("parse requirements to requirementMap: %v", requirementMap)
	return requirementMap
}

func buildExternalQuery(metricName string, requirementMap map[string][]string) (externalQuery prom.Selector) {
	// build str for common prometheus label, such as cluster
	commonPromLabelStr := ""
	commonPromLabelStrList := make([]string, 0)
	if list, ok := requirementMap["cluster"]; ok {
		commonPromLabelStrList = append(commonPromLabelStrList, fmt.Sprintf(`cluster=~"%s"`, strings.Join(list, "|")))
	}
	if len(commonPromLabelStrList) > 0 {
		commonPromLabelStr = fmt.Sprintf("%s", strings.Join(commonPromLabelStrList, ","))
	}

	// build str for kube_pod_labels
	kubePodLabelStr := ""
	for key, value := range requirementMap {
		// only support single label currently
		// todo: check promql special symbol conversion, eg. "label_a/b" -> "label_a_b"
		if strings.HasPrefix(key, "label_") {
			kubePodLabelStr = fmt.Sprintf(`%s=~"%s"`, key, value[0])
		}
	}

	// build str for kube_pod_info
	//kubePodInfoStr := fmt.Sprintf(`namespace=~"%s",created_by_kind=~"%s",created_by_name=~"%s",pod=~"%s"`,
	//	parsePromLabel(requirementMap["namespace"]), parsePromLabel(requirementMap["created_by_kind"]), parsePromLabel(requirementMap["created_by_name"]), parsePromLabel(requirementMap["pod"]))
	groupedQueryFilteredPodInfo := fmt.Sprintf("on(pod, namespace) group_right %s", QueryFilteredPodInfo)
	kubePodInfoStr := ""
	kubePodInfoStrList := make([]string, 0)
	if list, ok := requirementMap["namespace"]; ok {
		kubePodInfoStrList = append(kubePodInfoStrList, fmt.Sprintf(`namespace=~"%s"`, strings.Join(list, "|")))
	}
	if list, ok := requirementMap["pod"]; ok {
		kubePodInfoStrList = append(kubePodInfoStrList, fmt.Sprintf(`pod=~"%s"`, strings.Join(list, "|")))
	}
	if list, ok := requirementMap["created_by_kind"]; ok {
		kubePodInfoStrList = append(kubePodInfoStrList, fmt.Sprintf(`created_by_kind=~"%s"`, strings.Join(list, "|")))
	}
	if list, ok := requirementMap["created_by_name"]; ok {
		kubePodInfoStrList = append(kubePodInfoStrList, fmt.Sprintf(`created_by_name=~"%s"`, strings.Join(list, "|")))
	}
	if len(kubePodInfoStrList) > 0 {
		kubePodInfoStrList = append(kubePodInfoStrList, commonPromLabelStrList...)
		kubePodInfoStr = strings.Join(kubePodInfoStrList, ",")
	}

	// build str for prom duration
	layout := requirementMap["window_layout"][0]
	start, err := time.Parse(layout, requirementMap["window_start"][0])
	if err != nil {
		klog.Errorf("Error parsing start time: %v", err)
		return
	}
	end, err := time.Parse(layout, requirementMap["window_end"][0])
	if err != nil {
		klog.Errorf("Error parsing end time: %v", err)
		return
	}
	duration := end.Sub(start)
	resolutionStr, resolutionSecs := util.ResolutionStringAndSeconds(duration)
	if res, ok := requirementMap["resolution"]; ok {
		resolutionDur, err := util.ParseDuration(res[0])
		if err != nil {
			klog.Errorf("Error parsing resolution to duration, resolution: %s, error: %v", resolutionStr, err)
		} else {
			resolutionStr = res[0]
			resolutionSecs = strconv.FormatFloat(resolutionDur.Seconds(), 'f', -1, 64)
		}
	}
	durStr := fmt.Sprintf("%s:%s", util.DurationString(duration), resolutionStr)

	switch metricName {
	case KubePodInfo:
		item := fmt.Sprintf("%s", QueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, kubePodLabelStr, kubePodInfoStr, durStr))
	case KubePodLabels:
		item := fmt.Sprintf("%s", QueryFilteredPodLabels)
		externalQuery = prom.Selector(fmt.Sprintf(item, kubePodInfoStr, kubePodLabelStr, durStr))
	case KubeNodeInfo:
		item := fmt.Sprintf("%s", QueryNodeInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, commonPromLabelStr, durStr))
	case CPUCoreRequestAverage:
		item := fmt.Sprintf("%s * %s", QueryCPUCoreRequestAverage, groupedQueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr, durStr))
	case CPUCoreUsageAverage:
		item := fmt.Sprintf("%s * %s", QueryCPUCoreUsageAverage, groupedQueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr, durStr))
	case MemoryRequestAverage:
		item := fmt.Sprintf("%s * %s", QueryMemoryRequestAverage, groupedQueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr, durStr))
	case MemoryUsageAverage:
		item := fmt.Sprintf("%s * %s", QueryMemoryUsageAverage, groupedQueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, kubePodLabelStr, kubePodInfoStr, durStr))
	case CostPodCPURequest:
		item := fmt.Sprintf("%s * %s", QueryCostPodCPURequest, groupedQueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, resolutionSecs, kubePodLabelStr, kubePodInfoStr, durStr))
	case CostPodMemoryRequest:
		item := fmt.Sprintf("%s * %s", QueryCostPodMemoryRequest, groupedQueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, resolutionSecs, kubePodLabelStr, kubePodInfoStr, durStr))
	case CostTotal:
		item := fmt.Sprintf("%s", QueryCostTotal)
		externalQuery = prom.Selector(fmt.Sprintf(item, commonPromLabelStr, durStr, resolutionSecs))
	case CostNode:
		item := fmt.Sprintf("%s", QueryCostNode)
		externalQuery = prom.Selector(fmt.Sprintf(item, commonPromLabelStr, durStr, resolutionSecs))
	case CostCustom:
		item := fmt.Sprintf("%s * %s", QueryCostCustom, groupedQueryFilteredPodInfo)
		externalQuery = prom.Selector(fmt.Sprintf(item, durStr, resolutionSecs, kubePodLabelStr, kubePodInfoStr, durStr))
	case BillingPretaxAmountTotal:
		item := fmt.Sprintf("%s", QueryBillingPretaxAmountTotal)
		externalQuery = prom.Selector(fmt.Sprintf(item, commonPromLabelStr, durStr))
	case BillingPretaxGrossAmountTotal:
		item := fmt.Sprintf("%s", QueryBillingPretaxGrossAmountTotal)
		externalQuery = prom.Selector(fmt.Sprintf(item, commonPromLabelStr, durStr))
	case BillingPretaxAmountNode:
		item := fmt.Sprintf("%s", QueryBillingPretaxAmountNode)
		externalQuery = prom.Selector(fmt.Sprintf(item, ","+commonPromLabelStr, durStr))
	}

	return externalQuery
}

func NewCOSTV2MetricSource() *COSTV2MetricSource {
	return &COSTV2MetricSource{
		prometheusProvider.GlobalConfig,
	}
}
