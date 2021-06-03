/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alibaba_cloud_provider

import (
	"errors"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/external_metrics_source"
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
	log "k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type AlibabaCloudMetricsProvider struct {
	// eManager is used by manage all metric source
	eManager *external_metrics_source.ExternalMetricsManager
}

// NewAlibabaCloudProvider return a alibabaCloudProvider
func NewAlibabaCloudProvider(prometheusUrl string) p.ExternalMetricsProvider {
	em := external_metrics_source.NewExternalMetricsManager(prometheusUrl)
	em.RegisterMetricsSource()

	return &AlibabaCloudMetricsProvider{
		eManager: em,
	}
}

// GetExternalMetric return metrics with specific labels
func (ep *AlibabaCloudMetricsProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info p.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	log.V(4).Infof("Received request for namespace: %s, metric name: %s, metric selectors: %s", namespace, info.Metric, metricSelector.String())

	r, selectable := metricSelector.Requirements()
	if !selectable {
		err := errors.New("External metrics need at least one label provided. ")
		log.Errorf("Failed to GetExternalMetric %s, because of %v", info.Metric, err)
		return nil, err
	}

	metricValues, err := ep.eManager.GetExternalMetrics(namespace, r, info)
	if err != nil {
		log.Errorf("Failed to GetExternalMetrics, because of %v ", err)
		return nil, err
	}

	matchingMetrics := make([]external_metrics.ExternalMetricValue, 0)
	matchingMetrics = append(matchingMetrics, metricValues...)

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, nil
}

// ListAllExternalMetrics return registered metrics
func (ep *AlibabaCloudMetricsProvider) ListAllExternalMetrics() []p.ExternalMetricInfo {
	externalMetricsInfo := make([]p.ExternalMetricInfo, 0)

	metrics := ep.eManager.GetMetricsInfoList()
	for _, metric := range metrics {
		externalMetricsInfo = append(externalMetricsInfo, p.ExternalMetricInfo{
			Metric: metric.Metric,
		})
	}

	return externalMetricsInfo
}
