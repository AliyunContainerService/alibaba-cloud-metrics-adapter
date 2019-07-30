package sls

import (
	"testing"

	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
)

func BenchmarkSls(b *testing.B) {
	var sls SLSMetricSource
	r := make([]labels.Requirement, 0)
	re1, e := labels.NewRequirement("sls.project", "=", []string{"k8s-log-c550367cdf1e84dfabab013b277cc6bc2"})
	re2, e := labels.NewRequirement("sls.logstore", "=", []string{"nginx-ingress"})
	if e != nil {
		b.Errorf("Failed to new requirement: %v", e)
	}
	r = append(r, *re1, *re2)
	info := p.ExternalMetricInfo{"sls_ingress_qps"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, e := sls.GetExternalMetric(info, "", r)
		if e != nil {
			b.Fatalf("Failed to get metric: %v", e)
		}
	}
}
