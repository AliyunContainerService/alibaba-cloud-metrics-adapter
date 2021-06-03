package external_metrics_source

import (
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/external_metrics_source/prom"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"testing"
)

func TestExternalMetricsManager_GetMetricsInfoList(t *testing.T) {
	promUrl := "http://localhost:9090"
	manager := NewExternalMetricsManager(promUrl)

	t.Run("Test add && delete external metric for prometheus source", func(t *testing.T) {
		prometheusSource := &prom.PrometheusSource{
			PrometheusUrl: promUrl,
			MetricList:    make(map[string]*prom.ExternalMetric),
		}
		externalMetric := &prom.ExternalMetric{
			Value: external_metrics.ExternalMetricValue{
				MetricName: "test-metrics",
				MetricLabels: map[string]string{
					"foo": "bar",
				},
				Value: *resource.NewQuantity(42, resource.DecimalSI),
			},
		}

		prometheusSource.AddExternalMetric("test-metric", externalMetric)
		manager.register(prometheusSource)
		if len(manager.GetMetricsInfoList()) != 1 {
			t.Errorf("add external metric failed")
		}

		prometheusSource.DeleteExternalMetric("test-metric")
		manager.register(prometheusSource)
		if len(manager.GetMetricsInfoList()) != 0 {
			t.Log(manager.GetMetricsInfoList())
			t.Errorf("external metric list should be empty")
		}
	})

}
