package prometheus_provider

import (
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/prometheus"
	naming2 "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/prometheus/naming"
	"sync"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"

	pmodel "github.com/prometheus/common/model"
	"k8s.io/klog"
)

// NB: container metrics sourced from cAdvisor don't consistently follow naming conventions,
// so we need to whitelist them and handle them on a case-by-case basis.  Metrics ending in `_total`
// *should* be counters, but may actually be guages in this case.

// SeriesType represents the kind of series backing a metric.
type SeriesType int

// SeriesRegistry provides conversions between Prometheus series and MetricInfo
type SeriesRegistry interface {
	// SetSeries replaces the known series in this registry.
	// Each slice in series should correspond to a MetricNamer in namers.
	SetSeries(series [][]prometheus.Series, namers []naming2.MetricNamer) error
	// ListAllMetrics lists all metrics known to this registry
	ListAllMetrics() []provider.CustomMetricInfo
	// QueryForMetric SeriesForMetric looks up the minimum required series information to make a query for the given metric
	// against the given resource (namespace may be empty for non-namespaced resources)
	QueryForMetric(info provider.CustomMetricInfo, namespace string, metricSelector labels.Selector, resourceNames ...string) (query prometheus.Selector, found bool)
	// MatchValuesToNames matches result values to resource names for the given metric and value set
	MatchValuesToNames(metricInfo provider.CustomMetricInfo, values pmodel.Vector) (matchedValues map[string]pmodel.SampleValue, found bool)
}

type seriesInfo struct {
	// seriesName is the name of the corresponding Prometheus series
	seriesName string

	// namer is the MetricNamer used to name this series
	namer naming2.MetricNamer
}

// overridableSeriesRegistry is a basic SeriesRegistry
type basicSeriesRegistry struct {
	mu sync.RWMutex

	// info maps metric info to information about the corresponding series
	info map[provider.CustomMetricInfo]seriesInfo
	// metrics is the list of all known metrics
	metrics []provider.CustomMetricInfo

	mapper apimeta.RESTMapper
}

func (r *basicSeriesRegistry) SetSeries(newSeriesSlices [][]prometheus.Series, namers []naming2.MetricNamer) error {
	if len(newSeriesSlices) != len(namers) {
		return fmt.Errorf("need one set of series per namer")
	}
	newInfo := make(map[provider.CustomMetricInfo]seriesInfo)
	for i, newSeries := range newSeriesSlices {
		namer := namers[i]
		for _, series := range newSeries {
			// TODO: warn if it doesn't match any resources
			resources, namespaced := namer.ResourcesForSeries(series)
			name, err := namer.MetricNameForSeries(series)
			if err != nil {
				klog.Errorf("unable to name series %q, skipping: %v", series.String(), err)
				continue
			}
			for _, resource := range resources {
				info := provider.CustomMetricInfo{
					GroupResource: resource,
					Namespaced:    namespaced,
					Metric:        name,
				}

				// some metrics aren't counted as namespaced
				if resource == naming2.NsGroupResource || resource == naming2.NodeGroupResource || resource == naming2.PVGroupResource {
					info.Namespaced = false
				}

				// we don't need to re-normalize, because the metric namer should have already normalized for us
				newInfo[info] = seriesInfo{
					seriesName: series.Name,
					namer:      namer,
				}
			}
		}
	}

	// regenerate metrics
	newMetrics := make([]provider.CustomMetricInfo, 0, len(newInfo))
	for info := range newInfo {
		newMetrics = append(newMetrics, info)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.info = newInfo
	r.metrics = newMetrics

	return nil
}

func (r *basicSeriesRegistry) ListAllMetrics() []provider.CustomMetricInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.metrics
}

func (r *basicSeriesRegistry) QueryForMetric(metricInfo provider.CustomMetricInfo, namespace string, metricSelector labels.Selector, resourceNames ...string) (prometheus.Selector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(resourceNames) == 0 {
		klog.Errorf("no resource names requested while producing a query for metric %s", metricInfo.String())
		return "", false
	}

	metricInfo, _, err := metricInfo.Normalized(r.mapper)
	if err != nil {
		klog.Errorf("unable to normalize group resource while producing a query: %v", err)
		return "", false
	}

	info, infoFound := r.info[metricInfo]
	if !infoFound {
		klog.V(10).Infof("metric %v not registered", metricInfo)
		return "", false
	}

	query, err := info.namer.QueryForSeries(info.seriesName, metricInfo.GroupResource, namespace, metricSelector, resourceNames...)
	if err != nil {
		klog.Errorf("unable to construct query for metric %s: %v", metricInfo.String(), err)
		return "", false
	}

	return query, true
}

func (r *basicSeriesRegistry) MatchValuesToNames(metricInfo provider.CustomMetricInfo, values pmodel.Vector) (matchedValues map[string]pmodel.SampleValue, found bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metricInfo, _, err := metricInfo.Normalized(r.mapper)
	if err != nil {
		klog.Errorf("unable to normalize group resource while matching values to names: %v", err)
		return nil, false
	}

	info, infoFound := r.info[metricInfo]
	if !infoFound {
		return nil, false
	}

	resourceLbl, err := info.namer.LabelForResource(metricInfo.GroupResource)
	if err != nil {
		klog.Errorf("unable to construct resource label for metric %s: %v", metricInfo.String(), err)
		return nil, false
	}

	res := make(map[string]pmodel.SampleValue, len(values))
	for _, val := range values {
		if val == nil {
			// skip empty values
			continue
		}
		res[string(val.Metric[resourceLbl])] = val.Value
	}
	return res, true
}
