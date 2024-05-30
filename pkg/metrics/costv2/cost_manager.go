package costv2

import (
	"encoding/json"
	"fmt"
	types "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/types"
	util "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/util"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider"
	"io"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	externalv1beta1 "k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

type APIType string

const (
	TypeCost       APIType = "cost"
	TypeAllocation APIType = "allocation"
)

type CostManager struct {
	externalClient externalclient.ExternalMetricsClient
	client         kubernetes.Interface
}

func NewCostManager() *CostManager {
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Fatalf("failed to get client config: %s", err)
	}

	externalClient, err := externalclient.NewForConfig(config)
	if err != nil {
		klog.Fatalf("failed to create external metrics client: %s", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create clientSet: %s", err)
	}

	return &CostManager{
		externalClient: externalClient,
		client:         client,
	}
}

func (cm *CostManager) getExternalMetrics(namespace, metricName string, metricSelector labels.Selector) *externalv1beta1.ExternalMetricValueList {
	metrics, err := cm.externalClient.NamespacedMetrics(namespace).List(metricName, metricSelector)
	if err != nil {
		klog.Errorf("unable to fetch metrics %s from apiServer: %v", metricName, err)
	}
	return metrics
}

func (cm *CostManager) ComputeAllocation(apiType APIType, start, end time.Time, resolution time.Duration, filter *types.Filter, costType types.CostType) (*types.AllocationSet, error) {
	klog.V(4).Infof("compute allocation params: apiType: %v, start: %v, end: %v, resolution: %v, filter: %v, costTpe: %v", apiType, start, end, resolution, filter, costType)

	window := types.NewWindow(&start, &end)
	allocSet := types.NewAllocationSet()
	podMap := map[types.PodMeta]*types.Pod{}

	selectorStr := make([]string, 0)
	if window.GetLabelSelectorStr() != "" {
		selectorStr = append(selectorStr, window.GetLabelSelectorStr())
	}
	if filter.GetLabelSelectorStr() != "" {
		selectorStr = append(selectorStr, filter.GetLabelSelectorStr())
	}

	metricSelector, err := labels.Parse(strings.Join(selectorStr, ","))
	if err != nil {
		klog.Errorf("failed to parse metricSelector, error: %v", err)
		return nil, err
	}

	cm.applyMetricToPodMap(window, CPUCoreRequestAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CPUCoreUsageAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, MemoryRequestAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, MemoryUsageAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostPodCPURequest, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostPodMemoryRequest, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostCustom, metricSelector, podMap)

	weightCPU, weightRAM := getCostWeights()
	totalCost := cm.getSingleValueMetric(CostTotal, metricSelector)

	totalBilling := 0.0
	switch costType {
	case types.AllocationPretaxAmount:
		totalBilling = cm.getSingleValueMetric(BillingPretaxAmountTotal, metricSelector)
	case types.AllocationPretaxGrossAmount:
		totalBilling = cm.getSingleValueMetric(BillingPretaxGrossAmountTotal, metricSelector)
	}
	klog.Infof("compute allocation for %v API. totalCost: %v, totalBilling: %v", apiType, totalCost, totalBilling)

	for _, pod := range podMap {
		pod.Allocations.Cost = pod.CostMeta.CostCPURequest*weightCPU + pod.CostMeta.CostRAMRequest*weightRAM

		if totalCost != 0 {
			pod.Allocations.CostRatio = pod.Allocations.Cost / totalCost

			if apiType == TypeAllocation {
				pod.Allocations.Cost = pod.Allocations.CostRatio * totalBilling
			}
		}

		pod.Allocations.Cost = math.Round(pod.Allocations.Cost*1000) / 1000

		allocSet.Set(pod.Allocations)
	}

	return allocSet, nil
}

func (cm *CostManager) applyMetricToPodMap(window types.Window, metricName string, metricSelector labels.Selector, podMap map[types.PodMeta]*types.Pod) {
	valueList := cm.getExternalMetrics("*", metricName, metricSelector)
	if valueList == nil || valueList.Items == nil {
		klog.Errorf("external metric %s value is empty", metricName)
		return
	}
	for _, value := range valueList.Items {
		pod, ok := value.MetricLabels["pod"]
		if !ok {
			klog.Errorf("failed to get pod name from external metric %s value", metricName)
			return
		}

		namespace, ok := value.MetricLabels["namespace"]
		if !ok {
			klog.Errorf("failed to get pod namespace from external metric %s value", metricName)
			return
		}

		key := types.PodMeta{Namespace: namespace, Pod: pod}

		// init podMap metadata
		if _, ok := podMap[key]; !ok {
			podMap[key] = &types.Pod{
				Key: key,
				Allocations: &types.Allocation{
					Name:  fmt.Sprintf("%s/%s", namespace, pod),
					Start: *window.Start(),
					End:   *window.End(),
					Properties: &types.AllocationProperties{
						Controller:     value.MetricLabels["created_by_name"],
						ControllerKind: value.MetricLabels["created_by_kind"],
						Pod:            pod,
						Namespace:      namespace,
					},
				},
				Window: window,
			}
		}

		switch metricName {
		case CPUCoreRequestAverage:
			podMap[key].Allocations.CPUCoreRequestAverage = float64(value.Value.MilliValue()) / 1000
		case CPUCoreUsageAverage:
			podMap[key].Allocations.CPUCoreUsageAverage = float64(value.Value.MilliValue()) / 1000
		case MemoryRequestAverage:
			podMap[key].Allocations.RAMBytesRequestAverage = float64(value.Value.MilliValue()) / 1000
		case MemoryUsageAverage:
			podMap[key].Allocations.RAMBytesUsageAverage = float64(value.Value.MilliValue()) / 1000
		case CostPodCPURequest:
			podMap[key].CostMeta.CostCPURequest = float64(value.Value.MilliValue()) / 1000
		case CostPodMemoryRequest:
			podMap[key].CostMeta.CostRAMRequest = float64(value.Value.MilliValue()) / 1000
		case CostCustom:
			podMap[key].Allocations.CustomCost = float64(value.Value.MilliValue()) / 1000
		}
	}
}

type CostWeights struct {
	CPU    float64 `json:"cpu,string"`
	Memory float64 `json:"memory,string"`
	GPU    float64 `json:"gpu,string,omitempty"`
}

func getCostWeights() (cpu, memory float64) {
	costWeightsStr := prometheusProvider.GlobalConfig.CostWeights
	costWeights := CostWeights{}
	err := json.Unmarshal([]byte(costWeightsStr), &costWeights)
	if err != nil {
		klog.Errorf("error parsing cost weights from %s, fallback to cpu weight 100%. error: %v", costWeightsStr, err)
		return 1, 0
	}
	klog.Infof("parsed cost weights: cpu: %f, memory: %f, gpu: %f", costWeights.CPU, costWeights.Memory, costWeights.GPU)
	return costWeights.CPU, costWeights.Memory
}

func (cm *CostManager) GetRangeAllocation(apiType APIType, window types.Window, resolution, step time.Duration, aggregate []string, filter *types.Filter, format string, accumulateBy AccumulateOption, costType types.CostType) (*types.AllocationSetRange, error) {
	klog.Infof("get range allocation params: apiType: %s, window: %s, resolution: %s, step: %s, aggregate: %s, filter: %s, format: %s, accumulateBy: %s, costType: %s", apiType, window, resolution, step, aggregate, filter, format, accumulateBy, costType)

	// Validate window is legal
	if window.IsOpen() || window.IsNegative() {
		return nil, fmt.Errorf("bad request - illegal window: %s", window)
	}

	// Begin with empty response
	asr := types.NewAllocationSetRange()

	// Query for AllocationSets in increments of the given step duration,
	// appending each to the response.
	stepStart := *window.Start()
	stepEnd := stepStart.Add(step)
	for window.End().After(stepStart) {
		allocSet, err := cm.ComputeAllocation(apiType, stepStart, stepEnd, resolution, filter, costType)
		if err != nil {
			return nil, fmt.Errorf("error computing allocations for %s: %w", types.NewClosedWindow(stepStart, stepEnd), err)
		}

		asr.Append(allocSet)

		stepStart = stepEnd
		stepEnd = stepStart.Add(step)
		if stepEnd.After(*window.End()) {
			stepEnd = *window.End()
		}
	}

	// todo Aggregate
	err := asr.AggregateBy(aggregate)
	if err != nil {
		return nil, fmt.Errorf("error aggregating for %s: %w", window, err)
	}

	// Accumulate, if requested
	//if accumulateBy != AccumulateOptionNone {
	//
	//}

	return asr, nil
}

//func (cm *CostManager) ComputeEstimatedCost(start, end time.Time, resolution time.Duration) (*types.AllocationSet, error) {
//	return nil, nil
//}
//
//func (cm *CostManager) GetRangeEstimatedCost(window types.Window, resolution, step time.Duration, aggregate []string, filter string) (*types.AllocationSetRange, error) {
//	return nil, nil
//}

func (cm *CostManager) getSingleValueMetric(metricName string, metricSelector labels.Selector) float64 {
	valueList := cm.getExternalMetrics("*", metricName, metricSelector)
	if valueList == nil || len(valueList.Items) == 0 {
		klog.Errorf("external metric %s value is empty", metricName)
		return 0
	}
	return float64(valueList.Items[0].Value.MilliValue() / 1000)
}

func ComputeAllocationHandler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.Query()
	paramsMap := make(map[string]string)
	for k, v := range res {
		paramsMap[k] = v[0]
	}
	klog.Infof("compute allocation params: %v", paramsMap)

	window, err := types.ParseWindow(paramsMap["window"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid 'window' parameter: %s", err), http.StatusBadRequest)
		return
	}
	if window.Duration() < time.Hour*24 {
		http.Error(w, fmt.Sprintf("Invalid 'window' parameter: %s", fmt.Errorf("window duration should be at least 1 day")), http.StatusBadRequest)
		return
	}

	filter := &types.Filter{}
	if filterStr, ok := paramsMap["filter"]; ok {
		filter, err = types.ParseFilter(filterStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'filter' parameter: %s", err), http.StatusBadRequest)
			return
		}
	}

	step := window.Duration()
	if stepStr, ok := paramsMap["step"]; ok {
		step, err = util.ParseDuration(stepStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'step' parameter: %s", err), http.StatusBadRequest)
			return
		}
	}

	// todo: this param need to follow finops focus, now default is PretaxAmount
	if costType, ok := paramsMap["costType"]; ok {
		klog.Infof("compute allocation params: costType: %s", costType)
	}

	// todo: parse other params
	aggregate := make([]string, 0)
	resolution := time.Duration(0)

	cm := NewCostManager()
	asr, err := cm.GetRangeAllocation(TypeAllocation, window, resolution, step, aggregate, filter, "", AccumulateOptionNone, types.AllocationPretaxAmount)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "bad request") {
			WriteError(w, BadRequest(err.Error()))
		} else {
			WriteError(w, InternalServerError(err.Error()))
		}

		return
	}

	w.Header().Set("content-type", "application/json")
	p, _ := json.Marshal(asr)
	io.WriteString(w, string(p))
}

func ComputeEstimatedCostHandler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.Query()
	paramsMap := make(map[string]string)
	for k, v := range res {
		paramsMap[k] = v[0]
	}
	klog.Infof("compute estimated cost params: %v", paramsMap)

	window, err := types.ParseWindow(paramsMap["window"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid 'window' parameter: %s", err), http.StatusBadRequest)
		return
	}

	filter := &types.Filter{}
	if filterStr, ok := paramsMap["filter"]; ok {
		filter, err = types.ParseFilter(filterStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'filter' parameter: %s", err), http.StatusBadRequest)
			return
		}
	}

	step := window.Duration()
	if stepStr, ok := paramsMap["step"]; ok {
		step, err = util.ParseDuration(stepStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'step' parameter: %s", err), http.StatusBadRequest)
			return
		}
	}

	// todo: parse other params
	aggregate := make([]string, 0)
	resolution := time.Duration(0)

	cm := NewCostManager()
	asr, err := cm.GetRangeAllocation(TypeCost, window, resolution, step, aggregate, filter, "", AccumulateOptionNone, types.CostEstimated)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "bad request") {
			WriteError(w, BadRequest(err.Error()))
		} else {
			WriteError(w, InternalServerError(err.Error()))
		}

		return
	}

	w.Header().Set("content-type", "application/json")
	p, _ := json.Marshal(asr)
	io.WriteString(w, string(p))
}

type Error struct {
	StatusCode int
	Body       string
}

type Response struct {
	Code    int         `json:"code"`
	Status  string      `json:"status"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
	Warning string      `json:"warning,omitempty"`
}

func BadRequest(message string) Error {
	return Error{
		StatusCode: http.StatusBadRequest,
		Body:       message,
	}
}

func InternalServerError(message string) Error {
	if message == "" {
		message = "Internal Server Error"
	}
	return Error{
		StatusCode: http.StatusInternalServerError,
		Body:       message,
	}
}

func WriteError(w http.ResponseWriter, err Error) {
	status := err.StatusCode
	if status == 0 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)

	resp, _ := json.Marshal(&Response{
		Code:    status,
		Message: fmt.Sprintf("Error: %s", err.Body),
	})
	w.Write(resp)
}
