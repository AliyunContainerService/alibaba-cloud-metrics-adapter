package costv2

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/cost"
	types "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/types"
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

func (cm *CostManager) ComputeAllocation(start, end time.Time, resolution time.Duration, filter string) (*types.AllocationSet, error) {
	window := types.NewWindow(&start, &end)
	allocSet := types.NewAllocationSet(start, end)
	podMap := map[types.PodMeta]*types.Pod{}

	// parse from filter
	namespaces, err := cm.client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	metricSelector, err := labels.Parse("")
	if err != nil {
	}

	for _, namespace := range namespaces.Items {
		// init podMap metadata
		cm.buildPodMap(window, podMap, namespace.Name, metricSelector)
		cm.applyMetricToPodMap(namespace.Name, cost.COST_CPU_REQUEST, metricSelector, podMap)
	}

	for _, pod := range podMap {
		allocSet.Set(pod.Allocations)
	}

	return allocSet, nil
}

func (cm *CostManager) applyMetricToPodMap(namespace, metricName string, metricSelector labels.Selector, podMap map[types.PodMeta]*types.Pod) {
	valueList := cm.getExternalMetrics(namespace, metricName, metricSelector)
	for _, value := range valueList.Items {
		pod, ok := value.MetricLabels["pod"]
		if !ok {
			return
		}

		key := types.PodMeta{Namespace: namespace, Pod: pod}
		if _, ok := podMap[key]; ok {
			switch metricName {
			case cost.COST_CPU_REQUEST:
				podMap[key].Allocations.CPUCoreRequestAverage = float64(value.Value.MilliValue()) / 1000 / 1024
			}
		}
	}
}

func (cm *CostManager) GetRangeAllocation(window types.Window, resolution, step time.Duration, aggregate []string, filter string, format string, accumulateBy AccumulateOption) (*types.AllocationSetRange, error) {
	// Validate window is legal
	if window.IsOpen() || window.IsNegative() {
		return nil, fmt.Errorf("illegal window: %s", window)
	}

	// Begin with empty response
	asr := types.NewAllocationSetRange()

	// Query for AllocationSets in increments of the given step duration,
	// appending each to the response.
	stepStart := *window.Start()
	//stepEnd := stepStart.Add(step)
	stepEnd := *window.End()
	for window.End().After(stepStart) {
		allocSet, err := cm.ComputeAllocation(stepStart, stepEnd, resolution, filter)
		if err != nil {
			return nil, fmt.Errorf("error computing allocations for %s: %w", types.NewClosedWindow(stepStart, stepEnd), err)
		}

		asr.Append(allocSet)

		stepStart = stepEnd
		stepEnd = stepStart.Add(step)
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

func (cm *CostManager) ComputeEstimatedCost(start, end time.Time, resolution time.Duration) (*types.AllocationSet, error) {
	return nil, nil
}

func (cm *CostManager) GetRangeEstimatedCost(window types.Window, resolution, step time.Duration, aggregate []string, filter string) (*types.AllocationSetRange, error) {
	return nil, nil
}

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

func ComputeAllocationHandler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.Query()
	paramsMap := make(map[string]string)
	for k, v := range res {
		paramsMap[k] = v[0]
	}

	window, err := types.ParseWindowWithOffset(paramsMap["window"], time.Duration(0))
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid 'window' parameter: %s", err), http.StatusBadRequest)
	}

	// todo: parse other params
	resolution := time.Duration(0)
	step := time.Duration(0)
	aggregate := make([]string, 0)

	cm := NewCostManager()
	asr, err := cm.GetRangeAllocation(window, resolution, step, aggregate, "", "", AccumulateOptionNone)
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
