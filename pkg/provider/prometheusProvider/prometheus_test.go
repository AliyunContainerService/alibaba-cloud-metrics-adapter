package prometheusProvider

import (
	"context"
	pmodel "github.com/prometheus/common/model"
	prom "sigs.k8s.io/prometheus-adapter/pkg/client"
	"testing"
)

func TestMakePromClient(t *testing.T) {

	promUrl := "http://ack-prometheus-operator-prometheus.monitoring.svc:9090"
	armsPromAuthToken := "testAuthToken"

	// input param
	cmdOpts := NewAlibabaMetricsAdapterOptions()
	cmdOpts.PrometheusURL = promUrl
	cmdOpts.PrometheusHeaders = []string{
		"Authorization=" + armsPromAuthToken,
	}

	//
	promClient, createClientErr := cmdOpts.MakePromClient()
	if createClientErr != nil {
		t.Fatalf("failed create prom client.")
	}

	queryPromQL := prom.Selector("up")

	queryResult, queryErr := promClient.Query(context.TODO(), pmodel.Now(), queryPromQL)
	if queryErr != nil {
		t.Fatalf("failed to query prom")
	}

	t.Logf("prom query resutl := %v", queryResult)

}
