package costv2

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	types "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/types"
	util "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics/costv2/util"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	externalv1beta1 "k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type APIType string

const (
	TypeCost       APIType = "cost"
	TypeAllocation APIType = "allocation"

	ShareSplitWeighted = "weighted"
	ShareSplitEven     = "even"
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

type AllocationParams struct {
	apiType      APIType
	window       types.Window
	resolution   string
	step         time.Duration
	aggregate    string
	filter       *types.Filter
	accumulateBy AccumulateOption
	costType     types.CostType
	idle         bool
	shareIdle    bool
	shareSplit   string
	idleByNode   bool
	targetType   string
}

func (cm *CostManager) ComputeAllocation(start, end time.Time, params AllocationParams) (*types.AllocationSet, error) {
	klog.V(4).Infof("compute allocation params from %v to %v: %+v", start, end, params)

	window := types.NewWindow(&start, &end)
	allocSet := types.NewAllocationSet()
	podMap := map[types.PodMeta]*types.Pod{}

	selectorStr := make([]string, 0)
	if window.GetLabelSelectorStr() != "" {
		selectorStr = append(selectorStr, window.GetLabelSelectorStr())
	}
	params.filter = cm.preprocessFilter(params.filter)
	if params.filter.GetLabelSelectorStr() != "" {
		selectorStr = append(selectorStr, params.filter.GetLabelSelectorStr())
	}
	if params.resolution != "" {
		selectorStr = append(selectorStr, fmt.Sprintf("resolution=%s", params.resolution))
	}

	metricSelector, err := labels.Parse(strings.Join(selectorStr, ","))
	if err != nil {
		klog.Errorf("failed to parse metricSelector, error: %v", err)
		return nil, err
	}

	cm.initPodMap(window, metricSelector, podMap)

	cm.applyMetricToPodMap(window, CPUCoreRequestAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CPUCoreUsageAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, MemoryRequestAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, MemoryUsageAverage, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostPodCPURequest, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostPodMemoryRequest, metricSelector, podMap)
	cm.applyMetricToPodMap(window, CostCustom, metricSelector, podMap)

	weightCPU, weightRAM := getCostWeights()
	nodeCostList := cm.getExternalMetrics("*", CostNode, metricSelector)
	totalCost := 0.0
	for _, nodeCost := range nodeCostList.Items {
		totalCost += float64(nodeCost.Value.MilliValue() / 1000)
	}

	// compute pod estimated cost
	totalPodCost := 0.0
	totalPodCostRatio := 0.0
	for _, pod := range podMap {
		pod.Allocations.Cost = pod.CostMeta.CostCPURequest*weightCPU + pod.CostMeta.CostRAMRequest*weightRAM
		pod.Allocations.Cost = math.Round(pod.Allocations.Cost*1000) / 1000
		if totalCost != 0 {
			pod.Allocations.CostRatio = pod.Allocations.Cost / totalCost
		}

		totalPodCost += pod.Allocations.Cost
		totalPodCostRatio += pod.Allocations.CostRatio

		allocSet.Set(pod.Allocations)
	}

	// if allocation api, compute pod billing allocation
	if params.apiType == TypeAllocation {
		totalBilling := 0.0
		if params.targetType == "cluster" {
			switch params.costType {
			case types.AllocationPretaxAmount:
				totalBilling = cm.getSingleValueMetric(BillingPretaxAmountTotal, metricSelector)
			case types.AllocationPretaxGrossAmount:
				totalBilling = cm.getSingleValueMetric(BillingPretaxGrossAmountTotal, metricSelector)
			}
		} else if params.targetType == "node" {
			switch params.costType {
			case types.AllocationPretaxAmount:
				totalBilling = cm.getSingleValueMetric(BillingPretaxAmountNode, metricSelector)
			}
		} else {
			return nil, fmt.Errorf("invalid 'targetType' parameter: %s", params.targetType)
		}
		klog.Infof("compute allocation for %v API. totalCost: %v, totalBilling: %v", params.apiType, totalCost, totalBilling)

		totalCost = totalBilling
		totalPodCost = 0.0
		for _, pod := range podMap {
			pod.Allocations.Cost = pod.Allocations.CostRatio * totalCost
			totalPodCost += pod.Allocations.Cost
		}
	}

	// idle cost
	if params.idle && (params.filter == nil || params.filter.IsNonClusterEmpty()) {
		klog.Infof("compute idle cost for %s API. shareIdle: %v, shareSplit: %s, idleByNode: %v", params.apiType, params.shareIdle, params.shareSplit, params.idleByNode)
		totalIdleCost := totalCost - totalPodCost
		totalIdleCostRatio := 1 - totalPodCostRatio

		if params.shareIdle {
			// share idle cost to each pod
			for _, pod := range podMap {
				switch params.shareSplit {
				case ShareSplitWeighted:
					pod.Allocations.Cost += totalIdleCost * pod.Allocations.Cost / totalPodCost
					pod.Allocations.CostRatio = pod.Allocations.Cost / totalCost
				case ShareSplitEven:
					pod.Allocations.Cost += totalIdleCost / float64(len(podMap))
					pod.Allocations.CostRatio = pod.Allocations.Cost / totalCost
				default:
					return nil, fmt.Errorf("invalid 'shareSplit' parameter: %s", params.shareSplit)
				}
			}
		} else {
			// show idle cost separately
			if params.aggregate == "node" && params.idleByNode {
				// here only record node price. idleByNode cost will be computed while aggregating nodes.
				for _, nodeCost := range nodeCostList.Items {
					idleNodeAllocation := &types.Allocation{
						Name:      fmt.Sprintf("%s%s", types.SplitIdlePrefix, nodeCost.MetricLabels["node"]),
						Start:     *window.Start(),
						End:       *window.End(),
						Cost:      float64(nodeCost.Value.MilliValue() / 1000),
						CostRatio: float64(nodeCost.Value.MilliValue()/1000) / totalCost,
					}
					allocSet.Set(idleNodeAllocation)
				}
			} else {
				idleAllocation := &types.Allocation{
					Name:      types.IdleSuffix,
					Start:     *window.Start(),
					End:       *window.End(),
					Cost:      totalIdleCost,
					CostRatio: totalIdleCostRatio,
				}
				allocSet.Set(idleAllocation)
			}
		}
	}

	return allocSet, nil
}

func (cm *CostManager) initPodMap(window types.Window, metricSelector labels.Selector, podMap map[types.PodMeta]*types.Pod) {
	klog.Infof("init podMap with window: %v", window)

	// add pod properties from kube_pod_info
	kubePodInfoList := cm.getExternalMetrics("*", KubePodInfo, metricSelector)
	if kubePodInfoList == nil || kubePodInfoList.Items == nil {
		klog.Errorf("external metric %s value is empty", KubePodInfo)
	}
	for _, item := range kubePodInfoList.Items {
		pod, ok := item.MetricLabels["pod"]
		if !ok {
			klog.Errorf("failed to get pod name from external metric %s value for metric %+v", KubePodInfo, item)
			continue
		}

		namespace, ok := item.MetricLabels["namespace"]
		if !ok {
			klog.Errorf("failed to get pod namespace from external metric %s value for metric %+v", KubePodInfo, item)
			continue
		}

		key := types.PodMeta{Namespace: namespace, Pod: pod}

		// init podMap metadata
		if _, ok := podMap[key]; !ok {
			controllerKind := strings.ToLower(item.MetricLabels["created_by_kind"])
			controller := strings.ToLower(item.MetricLabels["created_by_name"])
			node := item.MetricLabels["node"]

			if controllerKind == "replicaset" {
				replicaSet, err := cm.client.AppsV1().ReplicaSets(namespace).Get(context.TODO(), controller, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("failed to get ReplicaSet meta: %s, error: %v", controller, err)
				}

				ownerRefs := replicaSet.OwnerReferences
				if len(ownerRefs) > 0 {
					controllerKind = strings.ToLower(ownerRefs[0].Kind)
					controller = ownerRefs[0].Name
				} else {
					klog.Errorf("No owner references found for ReplicaSet: %s", controller)
				}
			}

			podMap[key] = &types.Pod{
				Key: key,
				Allocations: &types.Allocation{
					Name:  fmt.Sprintf("%s/%s", namespace, pod),
					Start: *window.Start(),
					End:   *window.End(),
					Properties: &types.AllocationProperties{
						Controller:     controller,
						ControllerKind: controllerKind,
						Pod:            pod,
						Namespace:      namespace,
						Node:           node,
					},
				},
				Window: window,
			}

			// set when metric has "cluster" label, for self-build prometheus
			if cluster, ok := item.MetricLabels["cluster"]; ok {
				podMap[key].Allocations.Properties.Cluster = cluster
			}
		}
	}

	// add pod properties from kube_pod_labels
	kubePodLabelsList := cm.getExternalMetrics("*", KubePodLabels, metricSelector)
	if kubePodLabelsList == nil || kubePodLabelsList.Items == nil {
		klog.Errorf("external metric %s value is empty", KubePodLabels)
	}
	for _, item := range kubePodLabelsList.Items {
		pod, ok := item.MetricLabels["pod"]
		if !ok {
			klog.Errorf("failed to get pod name from external metric %s value for metric %+v", KubePodInfo, item)
			continue
		}

		namespace, ok := item.MetricLabels["namespace"]
		if !ok {
			klog.Errorf("failed to get pod namespace from external metric %s value for metric %+v", KubePodInfo, item)
			continue
		}

		key := types.PodMeta{Namespace: namespace, Pod: pod}
		if _, ok := podMap[key]; ok {
			labels := getLabelsFromMetricLabels(item.MetricLabels)
			podMap[key].Allocations.Properties.Labels = labels
		}
	}

	// add pod properties from kube_node_info
	nodeInfoList := cm.getExternalMetrics("*", KubeNodeInfo, metricSelector)
	if nodeInfoList == nil || nodeInfoList.Items == nil {
		klog.Errorf("external metric %s value is empty", KubeNodeInfo)
	}
	nodeProviderIdMap := make(map[string]string)
	for _, item := range nodeInfoList.Items {
		node, ok := item.MetricLabels["node"]
		if !ok {
			klog.Errorf("failed to get node name from external metric %s value for metric %+v", KubeNodeInfo, item)
			continue
		}

		providerId, ok := item.MetricLabels["provider_id"]
		if !ok {
			klog.Errorf("failed to get providerID from external metric %s value for metric %+v", KubeNodeInfo, item)
			continue
		}

		nodeProviderIdMap[node] = providerId
	}
	for _, pod := range podMap {
		if providerId, ok := nodeProviderIdMap[pod.Allocations.Properties.Node]; ok {
			pod.Allocations.Properties.ProviderID = providerId
		}
	}
}

func getLabelsFromMetricLabels(metricLabels map[string]string) map[string]string {
	result := make(map[string]string)

	// Find All keys with prefix label_, remove prefix, add to labels
	for k, v := range metricLabels {
		if !strings.HasPrefix(k, "label_") {
			continue
		}

		label := strings.TrimPrefix(k, "label_")
		result[label] = v
	}

	return result
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

func (cm *CostManager) GetRangeAllocation(params AllocationParams) (*types.AllocationSetRange, error) {
	klog.Infof("get range allocation params: +%v", params)

	// Validate window is legal
	if params.window.IsOpen() || params.window.IsNegative() {
		return nil, fmt.Errorf("bad request - illegal window: %v", params.window)
	}

	// Begin with empty response
	asr := types.NewAllocationSetRange()

	// Query for AllocationSets in increments of the given step duration,
	// appending each to the response.
	stepStart := *params.window.Start()
	stepEnd := stepStart.Add(params.step)
	for params.window.End().After(stepStart) {
		allocSet, err := cm.ComputeAllocation(stepStart, stepEnd, params)
		if err != nil {
			return nil, fmt.Errorf("error computing allocations for %v: %w", types.NewClosedWindow(stepStart, stepEnd), err)
		}

		asr.Append(allocSet)

		stepStart = stepEnd
		stepEnd = stepStart.Add(params.step)
		if stepEnd.After(*params.window.End()) {
			stepEnd = *params.window.End()
		}
	}

	if err := asr.AggregateBy(params.aggregate, params.idleByNode); err != nil {
		return nil, fmt.Errorf("error aggregating allocations: %w", err)
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
		http.Error(w, fmt.Sprintf("Invalid 'window' parameter %s: %s", paramsMap["window"], fmt.Errorf("window duration should be at least 1 day")), http.StatusBadRequest)
		return
	}

	filter := &types.Filter{}
	if filterStr, ok := paramsMap["filter"]; ok {
		filter, err = types.ParseFilter(filterStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'filter' parameter %s: %s", paramsMap["filter"], err), http.StatusBadRequest)
			return
		}
	}

	step := window.Duration()
	if stepStr, ok := paramsMap["step"]; ok {
		step, err = util.ParseDuration(stepStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'step' parameter %s: %s", paramsMap["step"], err), http.StatusBadRequest)
			return
		}
		if step < time.Hour*24 {
			http.Error(w, fmt.Sprintf("Invalid 'step' parameter %s: %s", stepStr, fmt.Errorf("step duration should be at least 1 day")), http.StatusBadRequest)
			return
		}
	}

	targetType := "cluster"
	if targetTypeStr, ok := paramsMap["targetType"]; ok {
		targetType = targetTypeStr
	}

	// todo: this param need to follow finops focus, now default is PretaxAmount
	if costType, ok := paramsMap["costType"]; ok {
		klog.Infof("compute allocation params: costType: %s", costType)
	}

	// todo: parse other params
	aggregate := ""
	if aggregateStr, ok := paramsMap["aggregate"]; ok {
		aggregate = aggregateStr
	}

	resolution := ""
	if resolutionStr, ok := paramsMap["resolution"]; ok {
		matched, err := util.IsValidDurationString(resolutionStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'resolution' parameter %s: %s", paramsMap["resolution"], err), http.StatusBadRequest)
			return
		}
		if !matched {
			http.Error(w, fmt.Sprintf("Invalid 'resolution' parameter %s: %s", paramsMap["resolution"], fmt.Errorf("resolution should be a valid duration string")), http.StatusBadRequest)
			return
		}
		resolution = resolutionStr
	}

	idle := true
	if idleStr, ok := paramsMap["idle"]; ok {
		idle, err = strconv.ParseBool(idleStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'idle' parameter %s: %s", paramsMap["idle"], err), http.StatusBadRequest)
			return
		}
	}

	shareIdle := false
	if shareIdleStr, ok := paramsMap["shareIdle"]; ok {
		shareIdle, err = strconv.ParseBool(shareIdleStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'shareIdle' parameter %s: %s", paramsMap["shareIdle"], err), http.StatusBadRequest)
			return
		}
	}

	shareSplit := ShareSplitWeighted
	if shareSplitStr, ok := paramsMap["shareSplit"]; ok {
		shareSplit = shareSplitStr
	}

	idleByNode := false
	if idleByNodeStr, ok := paramsMap["idleByNode"]; ok {
		idleByNode, err = strconv.ParseBool(idleByNodeStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'idleByNode' parameter %s: %s", paramsMap["idleByNode"], err), http.StatusBadRequest)
			return
		}
	}

	cm := NewCostManager()
	allocationParams := AllocationParams{
		window:       window,
		resolution:   resolution,
		step:         step,
		aggregate:    aggregate,
		filter:       filter,
		apiType:      TypeAllocation,
		accumulateBy: AccumulateOptionNone,
		costType:     types.AllocationPretaxAmount,
		idle:         idle,
		shareIdle:    shareIdle,
		shareSplit:   shareSplit,
		idleByNode:   idleByNode,
		targetType:   targetType,
	}
	asr, err := cm.GetRangeAllocation(allocationParams)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "bad request") {
			WriteError(w, BadRequest(err.Error()))
		} else {
			WriteError(w, InternalServerError(err.Error()))
		}

		return
	}

	format := ""
	if formatStr, ok := paramsMap["format"]; ok {
		format = formatStr
	}
	switch format {
	case "json", "":
		w.Header().Set("content-type", "application/json")
		p, _ := json.Marshal(asr)
		io.WriteString(w, string(p))
	case "csv":
		filename := "allocation.csv"

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		var dimension string
		if aggregate == "" {
			dimension = "Pod"
		} else {
			caser := cases.Title(language.English)
			dimension = caser.String(aggregate)
		}
		if err := csvWriter.Write([]string{dimension, "Start", "End", "Cost", "CostRatio"}); err != nil {
			http.Error(w, fmt.Sprintf("Failed to write csv: %s", err), http.StatusInternalServerError)
			return
		}

		for _, as := range asr.Allocations {
			for _, a := range *as {
				record := []string{
					a.Name,
					a.Start.Format(time.RFC3339),
					a.End.Format(time.RFC3339),
					fmt.Sprintf("%f", a.Cost),
					fmt.Sprintf("%f", a.CostRatio),
				}

				if err := csvWriter.Write(record); err != nil {
					http.Error(w, fmt.Sprintf("Failed to write csv %s: %s", record, err), http.StatusInternalServerError)
					return
				}
			}
		}

	}
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
	aggregate := ""
	if aggregateStr, ok := paramsMap["aggregate"]; ok {
		aggregate = aggregateStr
	}

	resolution := ""
	if resolutionStr, ok := paramsMap["resolution"]; ok {
		matched, err := util.IsValidDurationString(resolutionStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'resolution' parameter %s: %s", paramsMap["resolution"], err), http.StatusBadRequest)
			return
		}
		if !matched {
			http.Error(w, fmt.Sprintf("Invalid 'resolution' parameter %s: %s", paramsMap["resolution"], fmt.Errorf("resolution should be a valid duration string")), http.StatusBadRequest)
			return
		}
		resolution = resolutionStr
	}

	idle := true
	if idleStr, ok := paramsMap["idle"]; ok {
		idle, err = strconv.ParseBool(idleStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'idle' parameter %s: %s", paramsMap["idle"], err), http.StatusBadRequest)
			return
		}
	}

	shareIdle := false
	if shareIdleStr, ok := paramsMap["shareIdle"]; ok {
		shareIdle, err = strconv.ParseBool(shareIdleStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'shareIdle' parameter %s: %s", paramsMap["shareIdle"], err), http.StatusBadRequest)
			return
		}
	}

	shareSplit := ShareSplitWeighted
	if shareSplitStr, ok := paramsMap["shareSplit"]; ok {
		shareSplit = shareSplitStr
	}

	idleByNode := false
	if idleByNodeStr, ok := paramsMap["idleByNode"]; ok {
		idleByNode, err = strconv.ParseBool(idleByNodeStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'idleByNode' parameter %s: %s", paramsMap["idleByNode"], err), http.StatusBadRequest)
			return
		}
	}

	cm := NewCostManager()
	allocationParams := AllocationParams{
		window:       window,
		resolution:   resolution,
		step:         step,
		aggregate:    aggregate,
		filter:       filter,
		apiType:      TypeCost,
		accumulateBy: AccumulateOptionNone,
		costType:     types.CostEstimated,
		idle:         idle,
		shareIdle:    shareIdle,
		shareSplit:   shareSplit,
		idleByNode:   idleByNode,
	}
	asr, err := cm.GetRangeAllocation(allocationParams)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "bad request") {
			WriteError(w, BadRequest(err.Error()))
		} else {
			WriteError(w, InternalServerError(err.Error()))
		}

		return
	}

	format := ""
	if formatStr, ok := paramsMap["format"]; ok {
		format = formatStr
	}
	switch format {
	case "json", "":
		w.Header().Set("content-type", "application/json")
		p, _ := json.Marshal(asr)
		io.WriteString(w, string(p))
	case "csv":
		filename := "cost.csv"

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		var dimension string
		if aggregate == "" {
			dimension = "Pod"
		} else {
			caser := cases.Title(language.English)
			dimension = caser.String(aggregate)
		}
		if err := csvWriter.Write([]string{dimension, "Start", "End", "Cost", "CostRatio"}); err != nil {
			http.Error(w, fmt.Sprintf("Failed to write csv: %s", err), http.StatusInternalServerError)
			return
		}

		for _, as := range asr.Allocations {
			for _, a := range *as {
				record := []string{
					a.Name,
					a.Start.Format(time.RFC3339),
					a.End.Format(time.RFC3339),
					fmt.Sprintf("%f", a.Cost),
					fmt.Sprintf("%f", a.CostRatio),
				}

				if err := csvWriter.Write(record); err != nil {
					http.Error(w, fmt.Sprintf("Failed to write csv %s: %s", record, err), http.StatusInternalServerError)
					return
				}
			}
		}
	}
}

// preprocessFilter preprocess filter for deployment -> replicaSet
func (cm *CostManager) preprocessFilter(filter *types.Filter) *types.Filter {
	if filter == nil {
		return filter
	}

	if filter.ControllerName != nil {
		newControllerName := make([]string, 0)
		for _, controller := range filter.ControllerName {
			deployments, err := cm.client.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{
				FieldSelector: "metadata.name=" + controller,
			})
			if err != nil {
				klog.Errorf("Failed to list deployments for %s: %s", controller, err)
			}

			if len(deployments.Items) == 0 {
				newControllerName = append(newControllerName, controller)
				continue
			}

			for _, deployment := range deployments.Items {
				replicaSets, err := cm.client.AppsV1().ReplicaSets(deployment.Namespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
				})
				if err != nil {
					klog.Errorf("Failed to list replicaSets for %s: %s", controller, err)
				}

				for _, rs := range replicaSets.Items {
					for _, ownerRef := range rs.OwnerReferences {
						if *ownerRef.Controller && ownerRef.UID == deployment.UID {
							newControllerName = append(newControllerName, rs.Name)
						}
					}
				}
			}
		}
		filter.ControllerName = newControllerName
	}

	return filter
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
