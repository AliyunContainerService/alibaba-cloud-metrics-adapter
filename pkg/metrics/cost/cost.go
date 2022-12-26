package cost

import (
	"context"
	"errors"
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider"
	"github.com/prometheus/common/model"
	pmodel "github.com/prometheus/common/model"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	log "k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	p "sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	prom "sigs.k8s.io/prometheus-adapter/pkg/client"
	"strings"
)

const (
	COST_CPU_REQUEST        = "cost_cpu_request"
	COST_CPU_LIMIT          = "cost_cpu_limit"
	COST_CPU_USAGE          = "cost_cpu_usage"
	COST_CPU_UTILIZATION    = "cost_cpu_utilization"
	COST_MEMORY_REQUEST     = "cost_memory_request"
	COST_MEMORY_LIMIT       = "cost_memory_limit"
	COST_MEMORY_USAGE       = "cost_memory_usage"
	COST_MEMORY_UTILIZATION = "cost_memory_utilization"
	COST_HOUR               = "cost_hour"
	COST_DAY                = "cost_day"
	COST_WEEK               = "cost_week"
	COST_MONTH              = "cost_month"
	COST_MIN                = "cost_min"
	COST_RATIO              = "cost_ratio"
	COST_PERCOREPRICING     = "cost_percorepricing"
)

type COSTParams struct {
	DimensionType string
	Dimension     string
	TimeUnit      string
	StartTime     string
	EndTime       string
	Label         string
}

type COSTMetricSource struct {
	*prometheusProvider.AlibabaMetricsAdapterOptions
}

//list all external metric
func (cs *COSTMetricSource) GetExternalMetricInfoList() []p.ExternalMetricInfo {
	metricInfoList := make([]p.ExternalMetricInfo, 0)
	var MetricArray = []string{
		COST_CPU_REQUEST,
		COST_CPU_LIMIT,
		COST_CPU_USAGE,
		COST_CPU_UTILIZATION,
		COST_MEMORY_REQUEST,
		COST_MEMORY_LIMIT,
		COST_MEMORY_USAGE,
		COST_MEMORY_UTILIZATION,
		COST_HOUR,
		COST_DAY,
		COST_WEEK,
		COST_MONTH,
		COST_MIN,
		COST_RATIO,
		COST_PERCOREPRICING,
	}
	for _, metric := range MetricArray {
		metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
			Metric: metric,
		})
	}
	return metricInfoList
}

//according to the incoming label, get the metric..
func (cs *COSTMetricSource) GetExternalMetric(info p.ExternalMetricInfo, namespace string, requirements labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {

	promSql := getPrometheusSql(info.Metric)
	query := buildExternalQuery(namespace, promSql, requirements)
	switch info.Metric {
	case COST_CPU_REQUEST:
		values, err = cs.getCOSTMetrics(namespace, COST_CPU_REQUEST, query)
	case COST_CPU_LIMIT:
		values, err = cs.getCOSTMetrics(namespace, COST_CPU_LIMIT, query)
	case COST_CPU_USAGE:
		values, err = cs.getCOSTMetrics(namespace, COST_CPU_USAGE, query)
	case COST_CPU_UTILIZATION:
		values, err = cs.getCOSTMetrics(namespace, COST_CPU_UTILIZATION, query)
	case COST_MEMORY_REQUEST:
		values, err = cs.getCOSTMetrics(namespace, COST_MEMORY_REQUEST, query)
	case COST_MEMORY_LIMIT:
		values, err = cs.getCOSTMetrics(namespace, COST_MEMORY_LIMIT, query)
	case COST_MEMORY_USAGE:
		values, err = cs.getCOSTMetrics(namespace, COST_MEMORY_USAGE, query)
	case COST_MEMORY_UTILIZATION:
		values, err = cs.getCOSTMetrics(namespace, COST_MEMORY_UTILIZATION, query)
	case COST_HOUR:
		values, err = cs.getCOSTMetrics(namespace, COST_HOUR, query)
	case COST_DAY:
		values, err = cs.getCOSTMetrics(namespace, COST_DAY, query)
	case COST_WEEK:
		values, err = cs.getCOSTMetrics(namespace, COST_WEEK, query)
	case COST_MONTH:
		values, err = cs.getCOSTMetrics(namespace, COST_MONTH, query)
	case COST_MIN:
		values, err = cs.getCOSTMetrics(namespace, COST_MIN, query)
	case COST_RATIO:
		values, err = cs.getCOSTMetrics(namespace, COST_RATIO, query)
	case COST_PERCOREPRICING:
		values, err = cs.getCOSTMetrics(namespace, COST_PERCOREPRICING, query)
	}
	if err != nil {
		log.Warningf("Failed to GetExternalMetric %s,because of %v", info.Metric, err)
	}
	return values, err
}

func getPrometheusSql(metricName string) (item string) {
	switch metricName {
	case COST_CPU_REQUEST:
		item = "sum(kube_pod_container_resource_requests_cpu_cores{job=\"_kube-state-metrics\"}) by(pod) * on(pod) group_right sum(kube_pod_labels{%s}) by(pod)"
	case COST_CPU_LIMIT:
		item = "sum(kube_pod_container_resource_limits_cpu_cores{job=\"_kube-state-metrics\"}) by(pod) * on(pod) group_left sum(kube_pod_labels{%s}) by(pod)"
	case COST_CPU_USAGE:
		item = "sum(rate (container_cpu_usage_seconds_total[1m])) by(pod) * on(pod) group_right sum(kube_pod_labels{%s}) by(pod)"
	case COST_CPU_UTILIZATION:
		item = "sum(rate (container_cpu_usage_seconds_total{}[1m])) by (pod) * on(pod) group_right sum(kube_pod_labels) by (pod) / sum(kube_pod_container_resource_requests_cpu_cores{job=\"_kube-state-metrics\"} ) by (pod) * on(pod) group_right sum(kube_pod_labels{%s}) by (pod)"
	case COST_MEMORY_REQUEST:
		item = "sum(kube_pod_container_resource_requests_memory_bytes{job=\"_kube-state-metrics\"}) by(pod)  * on(pod) group_right sum(kube_pod_labels{%s}) by(pod)"
	case COST_MEMORY_LIMIT:
		item = "sum(kube_pod_container_resource_limits_memory_bytes{job=\"_kube-state-metrics\"}) by(pod) * on(pod) group_left sum(kube_pod_labels{%s}) by(pod)"
	case COST_MEMORY_USAGE:
		item = "sum(container_memory_working_set_bytes) by (pod)  * on(pod) group_right sum(kube_pod_labels{%s}) by(pod)"
	case COST_MEMORY_UTILIZATION:
		item = "sum(container_memory_working_set_bytes) by (pod) * on(pod) group_right sum(kube_pod_labels{}) by (pod) / sum(kube_pod_container_resource_requests_memory_bytes{job=\"_kube-state-metrics\"} ) by (pod) * on(pod) group_right sum(kube_pod_labels{%s}) by (pod)"
	case COST_HOUR:
		item = "abs(sum(node_current_price / on (node)  group_left kube_node_status_capacity_cpu_cores{job=\"_kube-state-metrics\"} * on(node) group_right kube_pod_container_resource_requests_cpu_cores{job=\"_kube-state-metrics\"}) by (pod) * on(pod) group_right sum(kube_pod_labels{%s}) by (pod) * 3600)"
	case COST_MIN:
		item = "abs(sum(node_current_price / on (node)  group_left kube_node_status_capacity_cpu_cores{job=\"_kube-state-metrics\"} * on(node) group_right kube_pod_container_resource_requests_cpu_cores{job=\"_kube-state-metrics\"}) by (pod) * on(pod) group_right sum(kube_pod_labels{%s}) by (pod) * 60)"
	case COST_DAY:
		item = "sum(sum_over_time((sum(node_current_price / on (node)  group_left kube_node_status_capacity_cpu_cores{job=\"_kube-state-metrics\"}) by (node) * on(node) group_right kube_pod_container_resource_requests_cpu_cores{job=\"_kube-state-metrics\"})[24h:1m])) by(pod) * on(pod) group_right sum(kube_pod_labels{%s}) by (pod) * 60"
	case COST_WEEK:
		item = "sum(sum_over_time((sum(node_current_price / on (node)  group_left kube_node_status_capacity_cpu_cores{job=\"_kube-state-metrics\"}) by (node) * on(node) group_right kube_pod_container_resource_requests_cpu_cores{job=\"_kube-state-metrics\"})[168h:1h])) by(pod) * on(pod) group_right sum(kube_pod_labels{%s}) by (pod) * 3600"
	case COST_MONTH:
		item = "sum(sum_over_time((sum(node_current_price / on (node)  group_left kube_node_status_capacity_cpu_cores{job=\"_kube-state-metrics\"}) by (node) * on(node) group_right kube_pod_container_resource_requests_cpu_cores{job=\"_kube-state-metrics\"})[720h:1h])) by(pod) * on(pod) group_right sum(kube_pod_labels{%s}) by (pod) * 3600"
	case COST_RATIO:
		item = ""
	case COST_PERCOREPRICING:
		item = "sum(max(node_current_price) by (node) / on (node)  group_left kube_node_status_capacity_cpu_cores{job=\"_kube-state-metrics\"} * on(node) group_right kube_pod_container_resource_requests_cpu_cores{job=\"_kube-state-metrics\"}) by (pod) * on(pod) group_right sum(kube_pod_labels{%s}) by (pod) / on(pod) sum(kube_pod_container_resource_requests_cpu_cores) by (pod) * 3600"
	}
	return item
}

//get the slb specific metric values
func (cs *COSTMetricSource) getCOSTMetrics(namespace, metricName string, query prom.Selector) (values []external_metrics.ExternalMetricValue, err error) {
	client, err := cs.MakePromClient()
	if err != nil {
		log.Errorf("Failed to create prometheus client,because of %v", err)
		return values, err
	}
	queryResult, err := client.Query(context.TODO(), pmodel.Now(), query)
	if err != nil {
		klog.Errorf("unable to fetch metrics from prometheus: %v", err)
		return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics"))
	}

	return cs.convertVector(metricName, queryResult)
}

func (cs *COSTMetricSource) convertVector(metricName string, queryResult prom.QueryResult) (value []external_metrics.ExternalMetricValue, err error) {
	if queryResult.Type != model.ValVector {
		return nil, errors.New("incorrect query result type")
	}

	toConvert := *queryResult.Vector

	if toConvert == nil {
		return nil, errors.New("the provided input did not contain vector query results")
	}

	items := []external_metrics.ExternalMetricValue{}

	numSamples := toConvert.Len()
	if numSamples == 0 {
		return items, nil
	}

	for _, val := range toConvert {
		singleMetric, err := cs.convertSample(metricName, val)
		if err != nil {
			return nil, fmt.Errorf("unable to convert vector: %v", err)
		}
		items = append(items, *singleMetric)
	}
	return items, nil
}

// add namespace and pod for sql filter metric
func buildExternalQuery(namespace, promSql string, requirements labels.Requirements) (externalQuery prom.Selector) {
	podLabel := buildPodLabel(requirements)
	namespaceLabel := buildNamespaceLabel(namespace)

	if namespaceLabel == "" {
		return prom.Selector(fmt.Sprintf(promSql, podLabel))
	}
	if podLabel == "" {
		return prom.Selector(fmt.Sprintf(promSql, namespaceLabel))
	}

	labelList := []string{podLabel, namespaceLabel}
	labelMatches := strings.Join(labelList, ",")
	externalQuery = prom.Selector(fmt.Sprintf(promSql, labelMatches))
	return externalQuery
}

func buildNamespaceLabel(namespace string) (namespaceLabel string) {
	if namespace != "*" {
		namespaceLabel = fmt.Sprintf("namespace=\"%s\"", namespace)
	}
	return namespaceLabel
}

func buildPodLabel(requirements labels.Requirements) string {
	if len(requirements) == 0 {
		return ""
	}
	var labelMap map[string][]string
	labelMap = make(map[string][]string)

	for _, value := range requirements {
		if value.Values().List()[0] != "" {
			labelMap[value.Key()] = append(labelMap[value.Key()], value.Values().List()[0])
		}
	}
	if len(labelMap) == 0 {
		return ""
	}
	return convertPodLabels(labelMap)
}
func convertPodLabels(labelMap map[string][]string) (podLabel string) {
	var podLabelList []string
	for key, value := range labelMap {
		label := fmt.Sprintf("%s=~\"%s\"", key, strings.Join(value, "|"))
		podLabelList = append(podLabelList, label)
	}
	podLabel = strings.Join(podLabelList, ",")
	return podLabel
}

func (cs *COSTMetricSource) convertSample(metricName string, sample *model.Sample) (*external_metrics.ExternalMetricValue, error) {
	label := cs.convertLabels(sample.Metric)
	singleMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Timestamp: metav1.Time{
			sample.Timestamp.Time(),
		},
		Value:        *resource.NewMilliQuantity(int64(sample.Value*1000.0), resource.DecimalSI),
		MetricLabels: label,
	}
	return &singleMetric, nil
}

func (cs *COSTMetricSource) convertLabels(inLabels model.Metric) map[string]string {
	numLabels := len(inLabels)
	outLabels := make(map[string]string, numLabels)
	for labelName, labelVal := range inLabels {
		outLabels[string(labelName)] = string(labelVal)
	}
	return outLabels
}

func NewCOSTMetricSource() *COSTMetricSource {
	return &COSTMetricSource{
		prometheusProvider.GlobalConfig,
	}
}
