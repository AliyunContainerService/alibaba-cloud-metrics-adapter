package costv2

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"testing"
)

func TestParseRequirements(t *testing.T) {
	selector := labels.NewSelector()
	req, _ := labels.NewRequirement("testKey", selection.Equals, []string{"testValue"})
	selector = selector.Add(*req)
	expected := map[string][]string{"testKey": {"testValue"}}
	requirements, _ := selector.Requirements()

	result := parseRequirements(requirements)

	assert.Equal(t, expected, result)
}

func TestBuildExternalQuery(t *testing.T) {
	// 测试 buildExternalQuery
	fakeRequirementMap := map[string][]string{
		"window_start":    {"20210101000000"},
		"window_end":      {"20210101235959"},
		"window_layout":   {"20060102150405"},
		"label_app":       {"myapp"},
		"namespace":       {"default"},
		"created_by_kind": {"Deployment"},
		"created_by_name": {"myapp-deployment"},
		"pod":             {"myapp-pod"},
	}

	testCases := []struct {
		name           string
		metricName     string
		expectedString string
	}{
		{
			name:           "CPUCoreUsageAverage query",
			metricName:     CPUCoreUsageAverage,
			expectedString: `sum(avg_over_time(rate(container_cpu_usage_seconds_total[1m])[86399s:1h])) by(namespace, pod) * max_over_time((max(kube_pod_labels{label_app=~"myapp"}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{namespace=~"default",pod=~"myapp-pod",created_by_kind=~"Deployment",created_by_name=~"myapp-deployment"})[86399s:1h])`,
		},
		{
			name:           "CPUCoreUsageAverage query",
			metricName:     CPUCoreUsageAverage,
			expectedString: `sum(avg_over_time(rate(container_cpu_usage_seconds_total[1m])[86399s:1h])) by(namespace, pod) * max_over_time((max(kube_pod_labels{label_app=~"myapp"}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{namespace=~"default",pod=~"myapp-pod",created_by_kind=~"Deployment",created_by_name=~"myapp-deployment"})[86399s:1h])`,
		},
		{
			name:           "MemoryRequestAverage query",
			metricName:     MemoryRequestAverage,
			expectedString: `sum(avg_over_time((max(kube_pod_container_resource_requests{job="_kube-state-metrics", resource="memory"}) by (pod,namespace,container))[86399s:1h])) by (namespace, pod) * max_over_time((max(kube_pod_labels{label_app=~"myapp"}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{namespace=~"default",pod=~"myapp-pod",created_by_kind=~"Deployment",created_by_name=~"myapp-deployment"})[86399s:1h])`,
		},
		{
			name:           "MemoryUsageAverage query",
			metricName:     MemoryUsageAverage,
			expectedString: `sum(avg_over_time(container_memory_working_set_bytes[86399s:1h])) by(namespace, pod) * max_over_time((max(kube_pod_labels{label_app=~"myapp"}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{namespace=~"default",pod=~"myapp-pod",created_by_kind=~"Deployment",created_by_name=~"myapp-deployment"})[86399s:1h])`,
		},
		{
			name:           "CostPodCPURequest query",
			metricName:     CostPodCPURequest,
			expectedString: `sum(sum_over_time((max(node_current_price) by (node) / on (node)  group_left kube_node_status_capacity{job="_kube-state-metrics",resource="cpu"} * on(node) group_right kube_pod_container_resource_requests{job="_kube-state-metrics",resource="cpu"})[86399s:1h])) by (namespace, pod) * 3600 * max_over_time((max(kube_pod_labels{label_app=~"myapp"}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{namespace=~"default",pod=~"myapp-pod",created_by_kind=~"Deployment",created_by_name=~"myapp-deployment"})[86399s:1h])`,
		},
		{
			name:           "CostPodMemoryRequest query",
			metricName:     CostPodMemoryRequest,
			expectedString: `sum(sum_over_time((max(node_current_price) by (node) / on (node)  group_left kube_node_status_capacity{job="_kube-state-metrics",resource="memory"} * on(node) group_right kube_pod_container_resource_requests{job="_kube-state-metrics",resource="memory"})[86399s:1h])) by (namespace, pod) * 3600 * max_over_time((max(kube_pod_labels{label_app=~"myapp"}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{namespace=~"default",pod=~"myapp-pod",created_by_kind=~"Deployment",created_by_name=~"myapp-deployment"})[86399s:1h])`,
		},
		{
			name:           "CostCustom query",
			metricName:     CostCustom,
			expectedString: `sum_over_time((max(label_replace(label_replace(pod_custom_price, "namespace", "$1", "exported_namespace", "(.*)"), "pod", "$1", "exported_pod", "(.*)")) by (namespace,pod))[86399s:1h]) * 3600 * max_over_time((max(kube_pod_labels{label_app=~"myapp"}) by (pod,namespace) * on(pod, namespace) group_right kube_pod_info{namespace=~"default",pod=~"myapp-pod",created_by_kind=~"Deployment",created_by_name=~"myapp-deployment"})[86399s:1h])`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := buildExternalQuery(tc.metricName, fakeRequirementMap)
			t.Logf("query: %s", query)
			assert.Equal(t, tc.expectedString, string(query))
		})
	}
}
