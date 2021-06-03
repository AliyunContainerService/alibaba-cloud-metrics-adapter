package prometheus_provider

import (
	"context"
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/prometheus"
	naming2 "github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/prometheus/naming"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider/helpers"
	pmodel "github.com/prometheus/common/model"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"time"
)

// Runnable represents something that can be run until told to stop.
type Runnable interface {
	// Run runs the runnable forever.
	Run()
	// RunUntil runs the runnable until the given channel is closed.
	RunUntil(stopChan <-chan struct{})
}

type PrometheusProvider struct {
	mapper     apimeta.RESTMapper
	kubeClient dynamic.Interface
	promClient prometheus.Client
	SeriesRegistry
}

func NewPrometheusProvider(mapper apimeta.RESTMapper, kubeClient dynamic.Interface, promClient prometheus.Client, namers []naming2.MetricNamer, updateInterval time.Duration, maxAge time.Duration) (provider.CustomMetricsProvider, Runnable) {
	lister := &cachingMetricsLister{
		updateInterval: updateInterval,
		maxAge:         maxAge,
		promClient:     promClient,
		namers:         namers,

		SeriesRegistry: &basicSeriesRegistry{
			mapper: mapper,
		},
	}

	return &PrometheusProvider{
		mapper:     mapper,
		kubeClient: kubeClient,
		promClient: promClient,

		SeriesRegistry: lister,
	}, lister
}

func (p *PrometheusProvider) metricFor(value pmodel.SampleValue, name types.NamespacedName, info provider.CustomMetricInfo) (*custom_metrics.MetricValue, error) {
	ref, err := helpers.ReferenceFor(p.mapper, name, info)
	if err != nil {
		return nil, err
	}

	return &custom_metrics.MetricValue{
		DescribedObject: ref,
		Metric: custom_metrics.MetricIdentifier{
			Name: info.Metric,
		},
		Timestamp: metav1.Time{Time: time.Now()},
		Value:     *resource.NewMilliQuantity(int64(value*1000.0), resource.DecimalSI),
	}, nil
}

func (p *PrometheusProvider) metricsFor(valueSet pmodel.Vector, info provider.CustomMetricInfo, namespace string, names []string) (*custom_metrics.MetricValueList, error) {
	values, found := p.MatchValuesToNames(info, valueSet)
	if !found {
		return nil, provider.NewMetricNotFoundError(info.GroupResource, info.Metric)
	}
	res := []custom_metrics.MetricValue{}

	for _, name := range names {
		if _, found := values[name]; !found {
			continue
		}

		value, err := p.metricFor(values[name], types.NamespacedName{Namespace: namespace, Name: name}, info)
		if err != nil {
			return nil, err
		}
		res = append(res, *value)
	}

	return &custom_metrics.MetricValueList{
		Items: res,
	}, nil
}

func (p *PrometheusProvider) buildQuery(info provider.CustomMetricInfo, namespace string, metricSelector labels.Selector, names ...string) (pmodel.Vector, error) {
	query, found := p.QueryForMetric(info, namespace, metricSelector, names...)
	if !found {
		return nil, provider.NewMetricNotFoundError(info.GroupResource, info.Metric)
	}

	// TODO: use an actual context
	queryResults, err := p.promClient.Query(context.TODO(), pmodel.Now(), query)
	if err != nil {
		klog.Errorf("unable to fetch metrics from prom: %v", err)
		// don't leak implementation details to the user
		return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics"))
	}

	if queryResults.Type != pmodel.ValVector {
		klog.Errorf("unexpected results from prom: expected %s, got %s on results %v", pmodel.ValVector, queryResults.Type, queryResults)
		return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics"))
	}

	return *queryResults.Vector, nil
}

func (p *PrometheusProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (value *custom_metrics.MetricValue, err error) {
	// construct a query
	queryResults, err := p.buildQuery(info, name.Namespace, metricSelector, name.Name)
	if err != nil {
		return nil, err
	}
	// associate the metrics
	if len(queryResults) < 1 {
		return nil, provider.NewMetricNotFoundForError(info.GroupResource, info.Metric, name.Name)
	}

	namedValues, found := p.MatchValuesToNames(info, queryResults)
	if !found {
		return nil, provider.NewMetricNotFoundError(info.GroupResource, info.Metric)
	}

	if len(namedValues) > 1 {
		klog.V(2).Infof("Got more than one result (%v results) when fetching metric %s for %q, using the first one with a matching name...", len(queryResults), info.String(), name)
	}

	resultValue, nameFound := namedValues[name.Name]
	if !nameFound {
		klog.Errorf("None of the results returned by when fetching metric %s for %q matched the resource name", info.String(), name)
		return nil, provider.NewMetricNotFoundForError(info.GroupResource, info.Metric, name.Name)
	}

	// return the resulting metric
	return p.metricFor(resultValue, name, info)
}

func (p *PrometheusProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	// fetch a list of relevant resource names
	resourceNames, err := helpers.ListObjectNames(p.mapper, p.kubeClient, namespace, selector, info)
	if err != nil {
		klog.Errorf("unable to list matching resource names: %v", err)
		// don't leak implementation details to the user
		return nil, apierr.NewInternalError(fmt.Errorf("unable to list matching resources"))
	}

	// construct the actual query
	queryResults, err := p.buildQuery(info, namespace, metricSelector, resourceNames...)
	if err != nil {
		return nil, err
	}

	// return the resulting metrics
	return p.metricsFor(queryResults, info, namespace, resourceNames)
}

type cachingMetricsLister struct {
	SeriesRegistry

	promClient     prometheus.Client
	updateInterval time.Duration
	maxAge         time.Duration
	namers         []naming2.MetricNamer
}

func (l *cachingMetricsLister) Run() {
	l.RunUntil(wait.NeverStop)
}

func (l *cachingMetricsLister) RunUntil(stopChan <-chan struct{}) {
	go wait.Until(func() {
		if err := l.updateMetrics(); err != nil {
			utilruntime.HandleError(err)
		}
	}, l.updateInterval, stopChan)
}

type selectorSeries struct {
	selector prometheus.Selector
	series   []prometheus.Series
}

func (l *cachingMetricsLister) updateMetrics() error {
	startTime := pmodel.Now().Add(-1 * l.maxAge)

	// don't do duplicate queries when it's just the matchers that change
	seriesCacheByQuery := make(map[prometheus.Selector][]prometheus.Series)

	// these can take a while on large clusters, so launch in parallel
	// and don't duplicate
	selectors := make(map[prometheus.Selector]struct{})
	selectorSeriesChan := make(chan selectorSeries, len(l.namers))
	errs := make(chan error, len(l.namers))
	for _, namer := range l.namers {
		sel := namer.Selector()
		if _, ok := selectors[sel]; ok {
			errs <- nil
			selectorSeriesChan <- selectorSeries{}
			continue
		}
		selectors[sel] = struct{}{}
		go func() {

			series, err := l.promClient.Series(context.TODO(), pmodel.Interval{Start: startTime, End: 0}, sel)
			if err != nil {
				errs <- fmt.Errorf("unable to fetch metrics for query %q: %v", sel, err)
				return
			}
			errs <- nil
			selectorSeriesChan <- selectorSeries{
				selector: sel,
				series:   series,
			}
		}()
	}

	// iterate through, blocking until we've got all results
	for range l.namers {
		if err := <-errs; err != nil {
			return fmt.Errorf("unable to update list of all metrics: %v", err)
		}
		if ss := <-selectorSeriesChan; ss.series != nil {
			seriesCacheByQuery[ss.selector] = ss.series
		}
	}
	close(errs)

	newSeries := make([][]prometheus.Series, len(l.namers))
	for i, namer := range l.namers {
		//TODO
		series, cached := seriesCacheByQuery[namer.Selector()]
		if !cached {
			return fmt.Errorf("unable to update list of all metrics: no metrics retrieved for query %q", namer.Selector())
		}
		newSeries[i] = namer.FilterSeries(series)
	}

	klog.V(6).Infof("Set available metric list from Prometheus to: %v", newSeries)

	return l.SetSeries(newSeries, l.namers)
}
