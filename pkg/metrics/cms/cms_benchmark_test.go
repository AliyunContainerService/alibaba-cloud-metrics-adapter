package cms

import (
	"testing"

	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
)

func BenchmarkCms(b *testing.B) {
	var cms CMSMetricSource
	r := make([]labels.Requirement, 0)
	re1, e := labels.NewRequirement("k8s.cluster.id", "=", []string{"c550367cdf1e84dfabab013b277cc6bc2"})
	re2, e := labels.NewRequirement("k8s.workload.type", "=", []string{"Deployment"})
	re3, e := labels.NewRequirement("k8s.workload.name", "=", []string{"coredns"})
	if e != nil {
		b.Errorf("Failed to new requirement: %v", e)
	}
	r = append(r, *re1, *re2, *re3)
	info := p.ExternalMetricInfo{"k8s_workload_cpu_util"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, e := cms.GetExternalMetric(info, "acs_kubernetes", r)
		if e != nil {
			b.Fatalf("Failed to get metric: %v", e)
		}
	}
}
