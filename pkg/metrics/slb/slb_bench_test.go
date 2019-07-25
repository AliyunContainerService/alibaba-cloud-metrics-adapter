package slb

import (
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
	"testing"
)

func BenchmarkGetExternalMetric(b *testing.B) {
	var slb SLBMetricSource
	r := make([]labels.Requirement, 0)
	requirement1, e := labels.NewRequirement("slb.instanceId", "=", []string{"lb-2zedu6pk8bryv2z4hnrig"})
	requirement2, e := labels.NewRequirement("slb.region", "=", []string{"cn-beijing"})
	requirement3, e := labels.NewRequirement("slb.port", "=", []string{"80"})
	if e != nil {
		b.Errorf("Failed to new requirement: %v", e)
	}
	r = append(r, *requirement1, *requirement2, *requirement3)
	info := p.ExternalMetricInfo{"slb_l4_trafficrx"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, e := slb.GetExternalMetric(info, "acs_slb_dashboard", r)
		if e != nil {
			b.Fatalf("Failed to get metric: %v", e)
		}
	}
}
