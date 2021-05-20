package kubernetes

import (
	"github.com/agiledragon/gomonkey"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"testing"
)

func TestBelongPrometheusSource(t *testing.T) {
	var HPA1, HPA2 autoscalingv2.HorizontalPodAutoscaler
	HPA1.Annotations = make(map[string]string)
	HPA1.Annotations[PROMETHEUS_QUERY] = "up"
	HPA1.Annotations[PROMETHEUS_METRIC_NAME] = "test-metric"

	HPA2.Annotations = make(map[string]string)
	HPA2.Annotations[PROMETHEUS_QUERY] = "up"

	tests := []struct {
		name string
		hpa  autoscalingv2.HorizontalPodAutoscaler
		want bool
	}{
		{name: "hpa1 is completed",
			hpa:  HPA1,
			want: true},
		{name: "hpa2 lack prometheus.metric.name",
			hpa:  HPA2,
			want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasSpecificAnnotation(&tt.hpa); got != tt.want {
				t.Errorf("HasSpecificAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPrometheusMetricValue(t *testing.T) {
	var HPA1 autoscalingv2.HorizontalPodAutoscaler
	HPA1.Annotations = make(map[string]string)
	HPA1.Annotations[PROMETHEUS_QUERY] = "http_requests_total"
	HPA1.Annotations[PROMETHEUS_METRIC_NAME] = "test-metric"

	t.Run("response is empty", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(Query, func(_ string, _ string) (model.Value, v1.Warnings, error) {
			sample := model.Sample{
				Value: 1,
			}
			v := model.Vector{}
			v = append(v, &sample)
			warnings := v1.Warnings{}
			return v, warnings, nil
		})

		defer patches.Reset()

		value, err := GetPrometheusValue(&HPA1, "http://localhost:9090")
		if err != nil {
			t.Error(err)
		}
		t.Log(value)
	})

	t.Run("response has warnings", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(Query, func(_ string, _ string) (model.Value, v1.Warnings, error) {
			sample := model.Sample{
				Value: 1,
			}
			v := model.Vector{}
			v = append(v, &sample)
			warnings := v1.Warnings{}
			warnings = append(warnings, "this is a warning")
			return v, warnings, nil
		})

		defer patches.Reset()

		_, err := GetPrometheusValue(&HPA1, "http://localhost:9090")
		if err == nil {
			t.Errorf("response has warning,err should not be empty")
		}
	})

}
