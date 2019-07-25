package slb

import (
	"k8s.io/apimachinery/pkg/labels"
	"testing"
)

func TestInvalidGetSLBParams(t *testing.T) {
	r := make([]labels.Requirement, 0)
	_, e := getSLBParams(r)
	if e != nil {
		t.Log("pass TestInvalidGetSLBParams")
		return
	}
	t.Fatalf("Failed to pass TestInvalidGetSLBParams")
}

func TestValidGetSLBParams(t *testing.T) {
	r := make([]labels.Requirement, 0)
	requirement, e := labels.NewRequirement("slb.instanceId", "=", []string{"first01"})
	if e != nil {
		t.Fatalf("new requirement err: %v", e)
	}
	r = append(r, *requirement)
	_, e = getSLBParams(r)
	if e == nil {
		t.Logf("Pass TstValidGetSLBParams")
	}
}

func TestGetExternalMetricInfoList(t *testing.T) {
	var slb SLBMetricSource
	list := slb.GetExternalMetricInfoList()
	for _, info := range list {
		t.Logf("slb External Metric-Info-List include: %v", info)
	}
}
