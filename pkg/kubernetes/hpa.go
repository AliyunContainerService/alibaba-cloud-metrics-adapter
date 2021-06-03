package kubernetes

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"time"
)

const (
	PROMETHEUS_QUERY       = "prometheus.query"
	PROMETHEUS_METRIC_NAME = "prometheus.metric.name"
)

func HasSpecificAnnotation(hpa *autoscalingv2.HorizontalPodAutoscaler) bool {
	var (
		switch1 bool
		switch2 bool
	)

	for key := range hpa.Annotations {
		if key == PROMETHEUS_QUERY {
			switch2 = true
		}
		if key == PROMETHEUS_METRIC_NAME {
			switch1 = true
		}
	}

	return switch1 && switch2
}

func GetPrometheusValue(hpa *autoscalingv2.HorizontalPodAutoscaler, prometheusServer string) (value external_metrics.ExternalMetricValue, err error) {
	var (
		prometheusQuery string
		metricName      string
	)

	for key, value := range hpa.Annotations {
		if key == PROMETHEUS_QUERY {
			prometheusQuery = value
		}
		if key == PROMETHEUS_METRIC_NAME {
			metricName = value
		}
	}

	result, warnings, err := Query(prometheusServer, prometheusQuery)
	if err != nil || result == nil {
		return value, errors.New("prometheus response is err or empty")
	}

	if len(warnings) > 0 {
		return value, errors.New("prometheus response has warning")
	}

	switch result.Type() {
	case model.ValVector:
		samples := result.(model.Vector)
		if len(samples) == 0 {
			return value, errors.New("vector value is empty")
		}
		sampleValue := samples[0].Value
		value = external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(sampleValue), resource.DecimalSI),
			Timestamp:  metav1.Now(),
		}
	case model.ValScalar:
		scalar := result.(*model.Scalar)
		sampleValue := scalar.Value
		value = external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(sampleValue), resource.DecimalSI),
			Timestamp:  metav1.Now(),
		}
	}

	return value, nil
}

func Query(prometheusServer, prometheusQuery string) (model.Value, v1.Warnings, error) {

	promClient, err := api.NewClient(api.Config{
		Address: prometheusServer,
	})
	if err != nil {
		klog.Errorf("new prometheus client err: %v", err)
		return nil, nil, err
	}

	v1api := v1.NewAPI(promClient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := v1api.Query(ctx, prometheusQuery, time.Now())
	if err != nil {
		klog.Errorf("query prometheus %v err: %v", prometheusServer, err)
		return nil, nil, err
	}

	return result, warnings, nil
}
