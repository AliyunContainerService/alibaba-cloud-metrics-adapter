package cost

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"
	"net/http"
	"strings"
	"time"
)

type CostOptions struct {
	Summary        bool
	DimensionType  string
	Dimension      string
	LabelSelector  string
	TimeUnit       string
	externalClient externalclient.ExternalMetricsClient
}

//
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
	Timestamp     v1.Time `json:"timestamp"`
	TimeUnit      string  `json:"timeUnit"`
	DimensionType string  `json:"DimensionType"`
	Dimension     string  `json:"Dimension"`
	PodName       string  `json:"PodName"`
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

func (co *CostOptions) convertPodCostMap(metricsName string, podMetricsMap map[string]*PodMetrics, metricsList []v1beta1.ExternalMetricValue) map[string]*PodMetrics {
	klog.Infof("metricsList :%+v", metricsList)
	for _, value := range metricsList {
		pod := value.MetricLabels["pod"]
		if podMetricsMap[pod] == nil {
			podMetricsMap[pod] = &PodMetrics{}
			podMetricsMap[pod].Metadata.PodName = value.MetricLabels["pod"]
			podMetricsMap[pod].Metadata.TimeUnit = co.TimeUnit
			podMetricsMap[pod].Metadata.DimensionType = co.DimensionType
			podMetricsMap[pod].Metadata.Dimension = co.Dimension
			podMetricsMap[pod].Metadata.Timestamp = value.Timestamp
		}
		singlePodMetrics := podMetricsMap[pod]
		switch value.MetricName {
		case COST_MEMORY_REQUEST:
			singlePodMetrics.Request.Memory = float64(value.Value.MilliValue()) / 1000 / 1024
		case COST_MEMORY_LIMIT:
			singlePodMetrics.Limit.Memory = float64(value.Value.MilliValue()) / 1000 / 1024
		case COST_MEMORY_USAGE:
			singlePodMetrics.Usage.Memory = float64(value.Value.MilliValue()) / 1000 / 1024
		case COST_PERCOREPRICING:
			singlePodMetrics.PerCorePricing = float64(value.Value.MilliValue()) / 1000
		case COST_CPU_REQUEST:
			singlePodMetrics.Request.CPU = float64(value.Value.MilliValue()) / 1000
		case COST_CPU_LIMIT:
			singlePodMetrics.Limit.Memory = float64(value.Value.MilliValue()) / 1000
		case COST_CPU_USAGE:
			singlePodMetrics.Usage.CPU = float64(value.Value.MilliValue()) / 1000
		case COST_HOUR, COST_DAY, COST_WEEK, COST_MONTH:
			singlePodMetrics.Cost = float64(value.Value.MilliValue()) / 1000
		}
	}
	return podMetricsMap
}

func (co *CostOptions) convertCostSummaryMap(metricName string, podMetric PodMetrics, metrics *v1beta1.ExternalMetricValueList) PodMetrics {
	metric := co.buildMetricSummary(metricName, metrics)

	switch metricName {
	case COST_CPU_REQUEST:
		podMetric.Request.CPU = float64(metric.Value.MilliValue()) / 1000
	case COST_CPU_LIMIT:
		podMetric.Limit.Memory = float64(metric.Value.MilliValue()) / 1000
	case COST_CPU_USAGE:
		podMetric.Usage.CPU = float64(metric.Value.MilliValue()) / 1000
	case COST_MEMORY_REQUEST:
		podMetric.Request.Memory = float64(metric.Value.MilliValue()) / 1000 / 1024
	case COST_MEMORY_LIMIT:
		podMetric.Limit.Memory = float64(metric.Value.MilliValue()) / 1000 / 1024
	case COST_MEMORY_USAGE:
		podMetric.Usage.Memory = float64(metric.Value.MilliValue()) / 1000 / 1024
	case COST_PERCOREPRICING:
		podMetric.PerCorePricing = float64(metric.Value.MilliValue()) / 1000
	case COST_HOUR, COST_DAY, COST_WEEK, COST_MONTH:
		podMetric.Cost = float64(metric.Value.MilliValue()) / 1000
		podMetric.Metadata.Timestamp = metric.Timestamp
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

func (co *CostOptions) DescribeCostSummary(namespace string, labelMatch labels.Selector, metricList []string) (podMetricsList []PodMetrics) {
	var podMetric PodMetrics
	for _, metricName := range metricList {
		metrics, err := co.externalClient.NamespacedMetrics(namespace).List(metricName, labelMatch)
		if err != nil {
			klog.Errorf("unable to fetch metrics from apiServer: %v", err)
		}
		podMetric = co.convertCostSummaryMap(metricName, podMetric, metrics)
	}
	podMetricsList = []PodMetrics{podMetric}
	return podMetricsList
}

// DescribePodCostDetail convergence all pod detail cost metric
func (co *CostOptions) DescribePodCostDetail(namespace string, labelMatch labels.Selector, metricList []string) (podMetricsList []PodMetrics) {
	var podMetricsMap map[string]*PodMetrics
	podMetricsMap = make(map[string]*PodMetrics)
	for _, metricName := range metricList {
		metrics, err := co.externalClient.NamespacedMetrics(namespace).List(metricName, labelMatch)
		if err != nil {
			klog.Errorf("unable to fetch metrics %s from apiServer: %v", metricName, err)
		}
		podMetricsMap = co.convertPodCostMap(metricName, podMetricsMap, metrics.Items)
	}
	for _, value := range podMetricsMap {
		podMetricsList = append(podMetricsList, *value)
	}

	return podMetricsList
}

func (co *CostOptions) buildParams(params map[string]string) (namespace, labelSelector, podLabel string) {
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
		case "Summary":
			if value == "true" {
				co.Summary = true
			}
		}
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

	labelSelector = co.LabelSelector
	klog.Infof("cost http recieve params: namespace %s, labelSelector %s, podLabel %s", namespace, labelSelector, podLabel)
	return namespace, labelSelector, podLabel
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

func (co *CostOptions) getCostMetrics(params map[string]string) (podMetricsList []PodMetrics) {
	namespace, labelSelector, podLabel := co.buildParams(params)
	err := co.getClient()
	if err != nil {
		fmt.Errorf("unable to construct  externalclient: %v", err)
	}

	labelMatch, err := co.buildLabelMatches(labelSelector, podLabel)
	if err != nil {
		klog.Errorf("failed parse labelMatches: %v", err)
	}

	metricList := []string{"cost_cpu_request", "cost_cpu_limit", "cost_memory_request", "cost_memory_limit", "cost_memory_usage", "cost_percorepricing", "cost_cpu_usage"}
	metricList = append(metricList, fmt.Sprintf("cost_%s", co.TimeUnit))
	if co.Summary == true {
		podMetricsList = co.DescribeCostSummary(namespace, labelMatch, metricList)
	} else {
		podMetricsList = co.DescribePodCostDetail(namespace, labelMatch, metricList)
	}
	return podMetricsList
}

func Handler(w http.ResponseWriter, r *http.Request) {
	res, _ := ioutil.ReadAll(r.Body)
	paramsMap := make(map[string]string)
	err := json.Unmarshal(res, &paramsMap)
	if err != nil {
		klog.Errorf("parse params failed,because of %s", err)
	}
	var costOptions = CostOptions{}
	podMetricsList := costOptions.getCostMetrics(paramsMap)

	w.Header().Set("content-type", "application/json")
	res, _ = json.Marshal(podMetricsList)
	io.WriteString(w, string(res))
}
