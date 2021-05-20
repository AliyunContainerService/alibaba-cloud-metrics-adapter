package prom

import (
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"testing"
)

func TestPrometheusSource_AddExternalMetric(t *testing.T) {
	source := &PrometheusSource{
		PrometheusUrl: "http://localhost:9090",
		MetricList:    make(map[string]*ExternalMetric),
	}

	testLabels := make(map[string]string)
	testLabels["foo"] = "bar"

	t.Run("Register external metric for the prometheus", func(t *testing.T) {
		testMetric := &ExternalMetric{
			Labels: testLabels,
		}

		source.AddExternalMetric("test-metric", testMetric)
		if _, ok := source.MetricList["test-metric"]; ok {
			t.Log("Verify passed")
			return
		}
		t.Errorf("It shoule be include testMetric")
	})
}

func TestPrometheusSource_DeleteExternalMetric(t *testing.T) {
	source := &PrometheusSource{
		PrometheusUrl: "http://localhost:9090",
		MetricList:    make(map[string]*ExternalMetric),
	}
	testLabels := make(map[string]string)
	testLabels["foo"] = "bar"

	t.Run("Delete external metric for the prometheus", func(t *testing.T) {
		testMetric := &ExternalMetric{
			Labels: testLabels,
		}

		source.AddExternalMetric("test-metric", testMetric)
		source.DeleteExternalMetric("test-metric")
		t.Log(source.GetExternalMetricInfoList())
		if _, ok := source.MetricList["test-metric"]; !ok {
			t.Log("Verify passed")
			return
		}
		t.Errorf("It shoule be not include testMetric")
	})
}

func TestPrometheusSource_GetExternalMetricInfoList(t *testing.T) {
	source := &PrometheusSource{
		PrometheusUrl: "http://localhost:9090",
		MetricList:    make(map[string]*ExternalMetric),
	}
	testLabels := make(map[string]string)
	testLabels["foo"] = "bar"

	t.Run("Get external metric list", func(t *testing.T) {
		testMetric := &ExternalMetric{
			Labels: testLabels,
		}

		want := p.ExternalMetricInfo{
			Metric: "test-metric",
		}

		source.AddExternalMetric("test-metric", testMetric)
		got := source.GetExternalMetricInfoList()[0]
		if got != want {
			t.Error("test metric has been registered external metric.")
		}
		t.Log(got)
	})
}

func TestPrometheusSource_GetExternalMetric(t *testing.T) {
	source := &PrometheusSource{
		PrometheusUrl: "http://localhost:9090",
		MetricList:    make(map[string]*ExternalMetric),
	}

	t.Run("Get external metric", func(t *testing.T) {
		labelRequirements := labels.Requirements{}

		testMetric := &ExternalMetric{
			Value: external_metrics.ExternalMetricValue{
				MetricName: "test-metrics",
				MetricLabels: map[string]string{
					"foo": "bar",
				},
				Value: *resource.NewQuantity(42, resource.DecimalSI),
			},
		}
		metricInfo := p.ExternalMetricInfo{Metric: "test-metric"}

		source.AddExternalMetric("test-metric", testMetric)
		value, err := source.GetExternalMetric(metricInfo, "default", labelRequirements)
		if err != nil {
			t.Error(err)
		}
		t.Log(value)
	})
}
