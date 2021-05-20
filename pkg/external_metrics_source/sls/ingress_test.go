package sls

import (
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"testing"
)

func TestInvalidGetSLSParams(t *testing.T) {
	r := make([]labels.Requirement, 0)

	_, e := getSLSParams(r)
	if e != nil {
		t.Log("pass TestInvalidGetSLSParams")
		return
	}

	t.Fatalf("Failed to pass TestInvalidGetSLSParams")
}

func TestValidGetSLSParams(t *testing.T) {
	// todo
	t.Log("pass TestValidGetSLSParams")
}

func TestIngressQuery(t *testing.T) {
	var sms SLSMetricSource
	params := &SLSIngressParams{
		SLSGlobalParams: SLSGlobalParams{
			Interval:     60,
			DelaySeconds: 5,
			MaxRetry:     5,
		},
		Route: "default-svc-80",
	}
	var metricInfos = sms.GetExternalMetricInfoList()
	for _, metricInfo := range metricInfos {
		begin, end, query := sms.getSLSIngressQuery(params, metricInfo.Metric)
		fmt.Printf("M:%s, B:%d, E:%d, Q:%s \n", metricInfo.Metric, begin, end, query)
	}
}
