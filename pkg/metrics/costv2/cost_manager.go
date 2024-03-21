package costv2

import (
	"context"
	"encoding/json"
	"fmt"
	types "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/types"
	util "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/util"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	externalv1beta1 "k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"
	"log"
	"net/http"
	"strings"
	"time"
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

func (cm *CostManager) ComputeAllocation(costType types.CostType, start, end time.Time, resolution time.Duration, filter *types.Filter) (*types.AllocationSet, error) {
	klog.V(4).Infof("ComputeAllocation: start: %v, end: %v, resolution: %v, filter: %v", start, end, resolution, filter)

	window := types.NewWindow(&start, &end)
	allocSet := types.NewAllocationSet(costType, start, end)
	podMap := map[types.PodMeta]*types.Pod{}

	// parse from filter
	//if err := cm.buildPodMapV2(window, podMap, "", filter); err != nil {
	//	return nil, err
	//}

	selectorStr := []string{
		window.GetLabelSelectorStr(),
		filter.GetLabelSelectorStr(),
	}

	metricSelector, err := labels.Parse(strings.Join(selectorStr, ","))
	if err != nil {
		return nil, err
	}

	// buildPodMap can use FilteredPodInfo
	cm.applyMetricToPodMap(window, CPUCoreRequestAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CPUCoreUsageAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, MemoryRequestAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, MemoryUsageAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostCPURequest, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostMemoryRequest, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostCustom, metricSelector, podMap)

	weightCPU, weightRAM := getCostWeights()
	totalCost := cm.getSingleValueMetric(CostTotal, metricSelector)

	for _, pod := range podMap {
		pod.Allocations.Cost = pod.Allocations.CostCPURequest*weightCPU + pod.Allocations.CostRAMRequest*weightRAM

		if totalCost != 0 {
			pod.Allocations.CostRatio = pod.Allocations.Cost / totalCost
		}
	}

	//for _, namespace := range filter.Namespace {
	//	// init podMap metadata
	//	cm.buildPodMap(window, podMap, namespace, metricSelector)
	//	cm.applyMetricToPodMap(window, namespace, cost.COST_CPU_REQUEST, metricSelector, podMap)
	//}

	//namespaces, err := cm.client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	//metricSelector, err := labels.Parse("")
	//if err != nil {
	//}
	//
	//for _, namespace := range namespaces.Items {
	//	// init podMap metadata
	//	cm.buildPodMap(window, podMap, namespace.Name, metricSelector)
	//	cm.applyMetricToPodMap(namespace.Name, cost.COST_CPU_REQUEST, metricSelector, podMap)
	//}

	for _, pod := range podMap {
		allocSet.Set(pod.Allocations)
	}
	//klog.Infof("ComputeAllocation: window: %v", allocSet)
	//allocSet.Window = window

	return allocSet, nil
}

func (cm *CostManager) applyMetricToPodMap(window types.Window, metricName string, metricSelector labels.Selector, podMap map[types.PodMeta]*types.Pod) {
	valueList := cm.getExternalMetrics("*", metricName, metricSelector)
	if valueList == nil || valueList.Items == nil {
		return
	}
	for _, value := range valueList.Items {
		pod, ok := value.MetricLabels["pod"]
		if !ok {
			return
		}

		namespace, ok := value.MetricLabels["namespace"]
		if !ok {
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
		case CostCPURequest:
			podMap[key].Allocations.CostCPURequest = float64(value.Value.MilliValue()) / 1000
		case CostMemoryRequest:
			podMap[key].Allocations.CostRAMRequest = float64(value.Value.MilliValue()) / 1000
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
		fmt.Println("Error parsing cost weights:", err)
		return 1, 0
	}
	klog.Infof("cost weights: cpu: %f, memory: %f, gpu: %f", costWeights.CPU, costWeights.Memory, costWeights.GPU)
	return costWeights.CPU, costWeights.Memory
}

func (cm *CostManager) GetRangeAllocation(costType types.CostType, window types.Window, resolution, step time.Duration, aggregate []string, filter *types.Filter, format string, accumulateBy AccumulateOption) (*types.AllocationSetRange, error) {
	klog.Infof("get range allocation params: window: %s, resolution: %s, step: %s, aggregate: %s, filter: %s, format: %s, accumulateBy: %s", window, resolution, step, aggregate, filter, format, accumulateBy)

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
		allocSet, err := cm.ComputeAllocation(costType, stepStart, stepEnd, resolution, filter)
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

	// Aggregate
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

func (cm *CostManager) buildPodMap(window types.Window, podMap map[types.PodMeta]*types.Pod, namespace string, labelSelector labels.Selector) {
	pods, err := cm.client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("unable to fetch pods from apiServer: %v", err)
	}
	klog.Infof("buildPodMap pods: %v", pods.Items)
	for _, pod := range pods.Items {
		podMeta := types.PodMeta{
			Namespace: pod.Namespace,
			Pod:       pod.Name,
		}
		podMap[podMeta] = &types.Pod{
			Node:        pod.Spec.NodeName,
			Allocations: &types.Allocation{Name: fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)},
			Key:         podMeta,
			Window:      window,
		}
	}
	klog.Infof("buildPodMap podMap: %v", podMap)

}

//func (cm *CostManager) buildPodMapV2(window types.Window, podMap map[types.PodMeta]*types.Pod, namespace string, labelSelector labels.Selector, filter *types.Filter) error {
//	client, err := prometheusProvider.GlobalConfig.MakePromClient()
//	if err != nil {
//		return fmt.Errorf("failed to create prometheus client, because of %v", err)
//	}
//
//	queryFilteredPodInfo := prom.Selector(fmt.Sprintf(QueryFilteredPodInfo, filter.GetKubePodLabelStr(), filter.GetKubePodInfoStr()))
//	klog.V(4).Infof("external query，query filtered pod info: %v", queryFilteredPodInfo)
//
//	queryResult, err := client.Query(context.TODO(), pmodel.Now(), queryFilteredPodInfo)
//	if err != nil {
//		return fmt.Errorf("unable to query from prometheus: %v", err)
//	}
//	klog.Infof("external query result: %v", queryResult)
//	return nil
//}

func (cm *CostManager) getSingleValueMetric(metricName string, metricSelector labels.Selector) float64 {
	valueList := cm.getExternalMetrics("*", metricName, metricSelector)
	if len(valueList.Items) == 0 {
		return 0
	}
	return float64(valueList.Items[0].Value.MilliValue())
}

func ComputeAllocationHandler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.Query()
	paramsMap := make(map[string]string)
	for k, v := range res {
		paramsMap[k] = v[0]
	}
	klog.Infof("compute allocation params: %v", paramsMap)

	klog.Infof("compute allocation params: window: %s", paramsMap["window"])
	window, err := types.ParseWindow(paramsMap["window"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid 'window' parameter: %s", err), http.StatusBadRequest)
	}

	filter := &types.Filter{}
	if filterStr, ok := paramsMap["filter"]; ok {
		klog.Infof("compute allocation params: filter: %s", filterStr)
		filter, err = types.ParseFilter(filterStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'filter' parameter: %s", err), http.StatusBadRequest)
			return
		}
	}

	step := window.Duration()
	if stepStr, ok := paramsMap["step"]; ok {
		klog.Infof("compute allocation params: step: %s", stepStr)
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
	asr, err := cm.GetRangeAllocation(types.TypeAllocation, window, resolution, step, aggregate, filter, "", AccumulateOptionNone)
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
	klog.Infof("compute allocation params: %v", paramsMap)

	klog.Infof("compute allocation params: window: %s", paramsMap["window"])
	window, err := types.ParseWindow(paramsMap["window"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid 'window' parameter: %s", err), http.StatusBadRequest)
	}

	filter := &types.Filter{}
	if filterStr, ok := paramsMap["filter"]; ok {
		klog.Infof("compute allocation params: filter: %s", filterStr)
		filter, err = types.ParseFilter(filterStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'filter' parameter: %s", err), http.StatusBadRequest)
			return
		}
	}

	step := window.Duration()
	if stepStr, ok := paramsMap["step"]; ok {
		klog.Infof("compute allocation params: step: %s", stepStr)
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
	asr, err := cm.GetRangeAllocation(types.TypeCost, window, resolution, step, aggregate, filter, "", AccumulateOptionNone)
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
