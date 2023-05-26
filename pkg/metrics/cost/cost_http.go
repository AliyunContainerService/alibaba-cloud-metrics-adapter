package cost

import (
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type CostOptions struct {
	Summary        bool
	DimensionType  string
	Dimension      string
	LabelSelector  string
	TimeUnit       string
	StartTime      string
	EndTime        string
	Step           int
	externalClient externalclient.ExternalMetricsClient
}

var CostTotal v1beta1.ExternalMetricValue
var RangeParam RangeParams

type RangeParams struct {
	StartTime time.Time
	EndTime   time.Time
	Step      time.Duration
	Range     bool
}

type PodMetrics struct {
	Metadata `json:"metadata"`
	Request  `json:"request"`
	Usage    `json:"usage"`
	Limit    `json:"limit"`

	PerCorePricing float64 `json:"perCorePricing"`
	CostRatio      float64 `json:"costRatio"`
	Cost           float64 `json:"cost"`
	//StartTime      int64 `json:"startTime"`
	//EndTime        int   `json:"endTime"`
}

type Metadata struct {
	Timestamp     string `json:"timestamp"`
	TimeUnit      string `json:"timeUnit"`
	DimensionType string `json:"DimensionType"`
	Dimension     string `json:"Dimension"`
	PodName       string `json:"PodName"`
}

type Request struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Gpu    float64 `json:"gpu"`
	GpuMem float64 `json:"gpuMem"`
}

type Usage struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Gpu    float64 `json:"gpu"`
	GpuMem float64 `json:"gpuMem"`
}

type Limit struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Gpu    float64 `json:"gpu"`
	GpuMem float64 `json:"gpuMem"`
}

func (co *CostOptions) getClient() (err error) {
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}

	externalClient, err := externalclient.NewForConfig(config)
	if err != nil {
		return err
	}

	co.externalClient = externalClient
	return nil
}

// retain three decimal places
func parseMetricValue(metricValue float64) float64 {
	value, err := strconv.ParseFloat(fmt.Sprintf("%.3f", metricValue), 64)
	if err != nil {
		klog.Error("")
		return metricValue
	}
	return value

}

func (co *CostOptions) convertPodCostMap(metricsName string, podMetricsMap map[string]*PodMetrics, metricsList []v1beta1.ExternalMetricValue) map[string]*PodMetrics {
	for _, value := range metricsList {
		pod := value.MetricLabels["pod"]
		if podMetricsMap[pod] == nil && pod != "" {
			podMetricsMap[pod] = &PodMetrics{}
			podMetricsMap[pod].Metadata.PodName = value.MetricLabels["pod"]
			if RangeParam.Range {
				podMetricsMap[pod].Metadata.Timestamp = fmt.Sprintf("%s-%s", RangeParam.StartTime, RangeParam.EndTime)
			} else {
				podMetricsMap[pod].Metadata.Timestamp = value.Timestamp.String()
			}
			podMetricsMap[pod].Metadata.TimeUnit = co.TimeUnit
			podMetricsMap[pod].Metadata.DimensionType = co.DimensionType
			podMetricsMap[pod].Metadata.Dimension = co.Dimension
		}
		singlePodMetrics := podMetricsMap[pod]
		switch value.MetricName {
		case COST_MEMORY_REQUEST:
			singlePodMetrics.Request.Memory = parseMetricValue(float64(value.Value.MilliValue()) / 1000 / 1024)
		case COST_MEMORY_LIMIT:
			singlePodMetrics.Limit.Memory = parseMetricValue(float64(value.Value.MilliValue()) / 1000 / 1024)
		case COST_MEMORY_USAGE:
			singlePodMetrics.Usage.Memory = parseMetricValue(float64(value.Value.MilliValue()) / 1000 / 1024)
		case COST_PERCOREPRICING:
			singlePodMetrics.PerCorePricing = parseMetricValue(float64(value.Value.MilliValue()) / 1000)
		case COST_CPU_REQUEST:
			singlePodMetrics.Request.CPU = parseMetricValue(float64(value.Value.MilliValue()) / 1000)
		case COST_CPU_LIMIT:
			singlePodMetrics.Limit.Memory = parseMetricValue(float64(value.Value.MilliValue()) / 1000)
		case COST_CPU_USAGE:
			singlePodMetrics.Usage.CPU = parseMetricValue(float64(value.Value.MilliValue()) / 1000)
		case COST, COST_HOUR, COST_DAY, COST_WEEK, COST_MONTH:
			singlePodMetrics.Cost = parseMetricValue(float64(value.Value.MilliValue()) / 1000)
			singlePodMetrics.CostRatio = parseMetricValue(float64(value.Value.MilliValue()) / float64(CostTotal.Value.MilliValue()))
		}
	}
	return podMetricsMap
}

func (co *CostOptions) convertCostSummaryMap(metricName string, podMetric PodMetrics, metrics *v1beta1.ExternalMetricValueList) PodMetrics {
	metric := co.buildMetricSummary(metricName, metrics)

	switch metricName {
	case COST_CPU_REQUEST:
		podMetric.Request.CPU = parseMetricValue(float64(metric.Value.MilliValue()) / 1000)
	case COST_CPU_LIMIT:
		podMetric.Limit.Memory = parseMetricValue(float64(metric.Value.MilliValue()) / 1000)
	case COST_CPU_USAGE:
		podMetric.Usage.CPU = parseMetricValue(float64(metric.Value.MilliValue()) / 1000)
	case COST_MEMORY_REQUEST:
		podMetric.Request.Memory = parseMetricValue(float64(metric.Value.MilliValue()) / 1000 / 1024)
	case COST_MEMORY_LIMIT:
		podMetric.Limit.Memory = parseMetricValue(float64(metric.Value.MilliValue()) / 1000 / 1024)
	case COST_MEMORY_USAGE:
		podMetric.Usage.Memory = parseMetricValue(float64(metric.Value.MilliValue()) / 1000 / 1024)
	case COST_PERCOREPRICING:
		podMetric.PerCorePricing = parseMetricValue(float64(metric.Value.MilliValue()) / 1000)
	case COST, COST_HOUR, COST_DAY, COST_WEEK, COST_MONTH:
		podMetric.Cost = parseMetricValue(float64(metric.Value.MilliValue()) / 1000)
		podMetric.CostRatio = parseMetricValue(float64(metric.Value.MilliValue()) / float64(CostTotal.Value.MilliValue()))
		if RangeParam.Range {
			podMetric.Metadata.Timestamp = fmt.Sprintf("%s - %s", RangeParam.StartTime, RangeParam.EndTime)
		} else {
			podMetric.Metadata.Timestamp = metric.Timestamp.String()
		}
		podMetric.Metadata.TimeUnit = co.TimeUnit
		podMetric.Metadata.DimensionType = co.DimensionType
		podMetric.Metadata.Dimension = co.Dimension
	}
	return podMetric
}

func (co *CostOptions) buildMetricSummary(metricName string, metricList *v1beta1.ExternalMetricValueList) (metric v1beta1.ExternalMetricValue) {
	var summaryValue int64
	for _, value := range metricList.Items {
		summaryValue += value.Value.MilliValue()
	}
	return v1beta1.ExternalMetricValue{
		Timestamp:  v1.Time{time.Now()},
		MetricName: metricName,
		Value:      *resource.NewMilliQuantity(summaryValue, resource.DecimalSI),
	}
}

func (co *CostOptions) DescribeCostSummary(namespace string, labelMatch labels.Selector, metricList []string) (podMetricsList []PodMetrics, err error) {
	var podMetric PodMetrics
	for _, metricName := range metricList {
		metrics, err := co.externalClient.NamespacedMetrics(namespace).List(metricName, labelMatch)
		if err != nil {
			klog.Errorf("unable to fetch metrics from apiServer: %v", err)
			return nil, err
		}
		podMetric = co.convertCostSummaryMap(metricName, podMetric, metrics)
	}
	podMetricsList = []PodMetrics{podMetric}
	return podMetricsList, nil
}

// DescribePodCostDetail convergence all pod detail cost metric
func (co *CostOptions) DescribePodCostDetail(namespace string, labelMatch labels.Selector, metricList []string) (podMetricsList []PodMetrics, err error) {
	var podMetricsMap map[string]*PodMetrics
	podMetricsMap = make(map[string]*PodMetrics)
	for _, metricName := range metricList {
		metrics, err := co.externalClient.NamespacedMetrics(namespace).List(metricName, labelMatch)
		if err != nil {
			klog.Errorf("unable to fetch metrics %s from apiServer: %v", metricName, err)
			return nil, err
		}
		if strings.HasPrefix(metricName, COST_TOTAL) {
			CostTotal = metrics.Items[0]
		} else {
			podMetricsMap = co.convertPodCostMap(metricName, podMetricsMap, metrics.Items)
		}
	}
	for _, value := range podMetricsMap {
		podMetricsList = append(podMetricsList, *value)
	}

	return podMetricsList, nil
}

func (co *CostOptions) buildParams(params map[string]string) (namespace, labelSelector, podLabel string, err error) {
	for key, value := range params {
		switch key {
		case "DimensionType":
			co.DimensionType = value
		case "Dimension":
			co.Dimension = value
		case "LabelSelector":
			co.LabelSelector = value
		case "TimeUnit":
			co.TimeUnit = value
		case "StartTime":
			co.StartTime = value
		case "EndTime":
			co.EndTime = value
		case "Step":
			co.Step, err = strconv.Atoi(value)
			if err != nil {
				return
			}
		case "Summary":
			if value == "true" {
				co.Summary = true
			}
		}
	}

	if co.Step == 0 {
		co.Step = 60
	}

	if co.DimensionType == "Namespace" && co.Dimension != "" {
		namespace = co.Dimension
	} else {
		namespace = "*"
	}

	if co.DimensionType == "Pod" && co.Dimension != "" {
		podLabel = co.Dimension
	} else {
		podLabel = ""
	}

	if co.TimeUnit == "" {
		co.TimeUnit = "hour"
	}

	if co.TimeUnit == "range" {
		err = co.parseRangeParams(co.StartTime, co.EndTime, co.Step)
		if err != nil {
			return
		}
	}

	labelSelector = co.LabelSelector
	klog.Infof("cost http recieve params: namespace %s, labelSelector %s, podLabel %s, startTime %s, endTime %s, step %d", namespace, labelSelector, podLabel, co.StartTime, co.EndTime, co.Step)
	return namespace, labelSelector, podLabel, nil
}

func (co *CostOptions) parseRangeParams(startTime, endTime string, step int) (err error) {
	RangeParam.StartTime, err = time.Parse("2006-01-02T15:04:05Z", startTime)
	if err != nil {
		return err
	}

	RangeParam.EndTime, err = time.Parse("2006-01-02T15:04:05Z", endTime)
	if err != nil {
		return err
	}

	RangeParam.Step = time.Duration(step) * time.Second
	RangeParam.Range = true
	return nil
}

// Combine labelSelector and podLabel
func (co *CostOptions) buildLabelMatches(label, podLabel string) (labelMatches labels.Selector, err error) {
	if label == "" {
		labelMatches, err = labels.Parse(podLabel)
		return labelMatches, err
	}
	labelSelector := fmt.Sprintf("label_%s", label)
	if podLabel == "" {
		labelMatches, err = labels.Parse(labelSelector)
		return labelMatches, err
	}

	labelList := []string{labelSelector, podLabel}

	labelMatches, err = labels.Parse(strings.Join(labelList, ","))
	return labelMatches, err
}

func (co *CostOptions) getCostMetrics(params map[string]string) (podMetricsList []PodMetrics, err error) {
	namespace, labelSelector, podLabel, err := co.buildParams(params)
	if err != nil {
		klog.Errorf("failed parse params: %v", err)
		return nil, err
	}

	err = co.getClient()
	if err != nil {
		klog.Errorf("unable to construct  externalclient: %v", err)
		return nil, err
	}

	labelMatch, err := co.buildLabelMatches(labelSelector, podLabel)
	if err != nil {
		klog.Errorf("failed parse labelMatches: %v", err)
		return nil, err
	}

	metricList := []string{"cost_cpu_request", "cost_cpu_limit", "cost_memory_request", "cost_memory_limit", "cost_memory_usage", "cost_cpu_usage", "cost_percorepricing"}
	if co.TimeUnit == "range" {
		metricList = append(metricList, "cost")
		metricList = append(metricList, "cost_total")
	} else {
		metricList = append(metricList, fmt.Sprintf("cost_total_%s", co.TimeUnit))
		metricList = append(metricList, fmt.Sprintf("cost_%s", co.TimeUnit))
	}

	if co.Summary == true {
		podMetricsList, err = co.DescribeCostSummary(namespace, labelMatch, metricList)
	} else {
		podMetricsList, err = co.DescribePodCostDetail(namespace, labelMatch, metricList)
	}
	if err != nil {
		return nil, err
	}
	return podMetricsList, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.Query()
	paramsMap := make(map[string]string)
	for k, v := range res {
		paramsMap[k] = v[0]
	}
	w.Header().Set("content-type", "application/json")
	var costOptions = CostOptions{}
	podMetricsList, err := costOptions.getCostMetrics(paramsMap)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("cost data fetch error. err: %v", err))
		return
	}
	p, _ := json.Marshal(podMetricsList)
	io.WriteString(w, string(p))
}
