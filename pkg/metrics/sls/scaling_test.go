package sls

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

func TestGetSLSPredResult(t *testing.T) {
	ss := &SLSMetricSource{}
	param := &SLSScalingParams{
		SLSGlobalParams{
			Project:  "k8s-log-c0ae5df15fbf34b47ba3a9684e6ee2bee",
			LogStore: "internal-ml-log",
			Internal: false,
			MaxRetry: 3,
			Interval: 24 * 60 * 60,

			JobName: "etl-1648450179377-917195",
			Entity:  "service.ali.com-default-new-nginx-80",
			Metric:  "metric",
		},
	}
	client := sls.CreateNormalInterface(
		"cn-beijing.log.aliyuncs.com",
		os.Getenv("shiji_test_sub_ak_id"),
		os.Getenv("shiji_test_sub_ak_key"),
		"",
	)

	begin, end, query, err := ss.getSLSScalingQuery(param, SLS_SCALING)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	fmt.Printf("prediction query:\n%s\n", query)

	resp, err := client.GetLogs(param.Project, param.LogStore, "", begin, end, query, 10000, 0, false)
	if err != nil || len(resp.Logs) == 0 {
		t.Errorf("failed to get sls response: err info %v", err)
		return
	}
	if !resp.IsComplete() {
		t.Errorf("sls response is not complete")
		return
	}

	var values []external_metrics.ExternalMetricValue
	for _, log := range resp.Logs {
		ts, err := strconv.ParseInt(log["time"], 10, 64)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		val, err := strconv.ParseFloat(log["value"], 64)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		values = append(values, external_metrics.ExternalMetricValue{
			MetricName: SLS_SCALING,
			// TODO: values format need to be decided
			Value:     *resource.NewScaledQuantity(int64(val), resource.Scale(-6)),
			Timestamp: metav1.Unix(ts, 0),
		})
	}
	if len(values) == 0 {
		t.Errorf("sls prediction response has empty result")
		return
	}
	fmt.Printf("sls prediction response has %d values\n", len(values))
}
