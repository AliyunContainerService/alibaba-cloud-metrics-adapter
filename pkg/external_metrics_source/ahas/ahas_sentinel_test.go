package ahas

import (
	"testing"

	"k8s.io/apimachinery/pkg/labels"
)

func TestInvalidGetAhasSentinelParams(t *testing.T) {
	r := make([]labels.Requirement, 0)
	_, e := getAhasSentinelParams(r, "")
	if e != nil {
		t.Log("pass TestInvalidGetAhasSentinelParams")
		return
	}
	t.Fatalf("Failed to pass TestInvalidGetAhasSentinelParams")
}

func TestValidGetAhasSentinelParams(t *testing.T) {
	r := make([]labels.Requirement, 0)
	requirement, e := labels.NewRequirement(SentinelAppNameKey, "=", []string{"sentinel-console"})
	if e != nil {
		t.Fatalf("new requirement err: %v", e)
	}
	r = append(r, *requirement)
	params, e := getAhasSentinelParams(r, "")
	if e == nil && params.AppName == "sentinel-console" {
		t.Logf("Pass TestValidGetAhasSentinelParams")
	} else {
		t.Fatalf("Failed to pass TestValidGetAhasSentinelParams")
	}
}

func TestGetExternalMetricInfoList(t *testing.T) {
	var ahas AHASSentinelMetricSource
	list := ahas.GetExternalMetricInfoList()
	for _, info := range list {
		t.Logf("AHAS Sentinel External Metric-Info-List include: %v", info)
	}
}
