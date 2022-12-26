package provider

import (
	"context"
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/alibabaCloudProvider"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider"
	prometheusCustomMetricsProvider "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider/custom-provider"
	prometheusExternalMetricsProvider "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/provider/prometheusProvider/external-provider"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	p "sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"sigs.k8s.io/prometheus-adapter/pkg/naming"
)

// custom and external api manager
// todo
// convert to map would be better
// 2022/01/08
type providerManager struct {
	alibabaCloudProvider       *alibabaCloudProvider.AlibabaCloudMetricsProvider
	prometheusCustomProvider   p.CustomMetricsProvider
	prometheusExternalProvider p.ExternalMetricsProvider
}

func (pm *providerManager) GetMetricByName(ctx context.Context, name types.NamespacedName, info p.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	return pm.prometheusCustomProvider.GetMetricByName(ctx, name, info, metricSelector)
}

func (pm *providerManager) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info p.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	return pm.prometheusCustomProvider.GetMetricBySelector(ctx, namespace, selector, info, metricSelector)
}

// ListAllMetrics provides a list of all available metrics at
// the current time.  Note that this is not allowed to return
// an error, so it is reccomended that implementors cache and
// periodically update this list, instead of querying every time.
func (pm *providerManager) ListAllMetrics() []p.CustomMetricInfo {
	return pm.prometheusCustomProvider.ListAllMetrics()
}

func (pm *providerManager) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info p.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	alibabaCloudMetrics := pm.alibabaCloudProvider.ListAllExternalMetrics()

	for _, m := range alibabaCloudMetrics {
		if m.Metric == info.Metric {
			// found metric
			return pm.alibabaCloudProvider.GetExternalMetric(namespace, metricSelector, info)
		}
	}
	prometheusMetrics := pm.prometheusExternalProvider.ListAllExternalMetrics()

	for _, m := range prometheusMetrics {
		if m.Metric == info.Metric {
			// found metric
			return pm.prometheusExternalProvider.GetExternalMetric(ctx, namespace, metricSelector, info)
		}
	}
	return nil, fmt.Errorf("no any matched metrics from provider: %v", info)
}

func (pm *providerManager) ListAllExternalMetrics() []p.ExternalMetricInfo {
	metrics := make([]p.ExternalMetricInfo, 0)
	alibabaCloudMetrics := pm.alibabaCloudProvider.ListAllExternalMetrics()
	prometheusMetrics := pm.prometheusExternalProvider.ListAllExternalMetrics()
	metrics = append(metrics, alibabaCloudMetrics...)
	metrics = append(metrics, prometheusMetrics...)
	return metrics
}

func NewProviderManager(opts *prometheusProvider.AlibabaMetricsAdapterOptions, stopCh chan struct{}) (provider.MetricsProvider, error) {
	var prometheusCustomMetricsProviderInstance p.CustomMetricsProvider
	var prometheusExternalMetricsProviderInstance p.ExternalMetricsProvider
	var customRunner prometheusCustomMetricsProvider.Runnable
	var externalRunner prometheusExternalMetricsProvider.Runnable

	mapper, err := opts.RESTMapper()
	if err != nil {
		return nil, fmt.Errorf("unable to construct discovery REST mapper: %v", err)
	}

	dynamicClient, err := opts.DynamicClient()
	if err != nil {
		return nil, fmt.Errorf("unable to construct dynamic k8s client: %v", err)
	}

	alibabaCloudProviderInstance, err := alibabaCloudProvider.NewAlibabaCloudProvider(mapper, dynamicClient)
	if err != nil {
		return nil, fmt.Errorf("failed to setup alibaba-cloud-metircs-adapter provider: %v", err)
	}

	pm := &providerManager{
		alibabaCloudProvider: alibabaCloudProviderInstance,
	}

	if opts.MetricsMaxAge < opts.MetricsRelistInterval {
		return nil, fmt.Errorf("max age must not be less than relist interval")
	}

	err = opts.LoadConfig()
	if err != nil {
		klog.Warningf("failed to load prometheus rules from file: %s", opts.AdapterConfigFile)
	}

	// extract the namers
	namers, err := naming.NamersFromConfig(opts.MetricsConfig.Rules, mapper)
	if err != nil {
		return nil, fmt.Errorf("unable to construct naming scheme from metrics rules: %v", err)
	}

	// make the prometheus client
	promClient, err := opts.MakePromClient()
	if err != nil {
		klog.Fatalf("unable to construct Prometheus client: %v", err)
	}

	// construct the provider and start it
	prometheusCustomMetricsProviderInstance, customRunner = prometheusCustomMetricsProvider.NewPrometheusProvider(mapper, dynamicClient, promClient, namers, opts.MetricsRelistInterval, opts.MetricsMaxAge)
	customRunner.RunUntil(stopCh)

	prometheusExternalMetricsProviderInstance, externalRunner = prometheusExternalMetricsProvider.NewExternalPrometheusProvider(promClient, namers, opts.MetricsRelistInterval, opts.MetricsMaxAge)
	externalRunner.RunUntil(stopCh)
	pm.prometheusCustomProvider = prometheusCustomMetricsProviderInstance
	pm.prometheusExternalProvider = prometheusExternalMetricsProviderInstance

	return pm, nil
}
