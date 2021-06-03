package prom

import (
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/kubernetes"
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubewatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"time"
)

type ExternalMetric struct {
	Labels map[string]string
	Value  external_metrics.ExternalMetricValue
}

type PrometheusSource struct {
	PrometheusUrl string
	MetricList    map[string]*ExternalMetric
}

func NewPrometheusSource(url string) *PrometheusSource {
	ps := &PrometheusSource{
		PrometheusUrl: url,
		MetricList:    make(map[string]*ExternalMetric)}

	go ps.MonitorHPAs()

	return ps
}

func (prom *PrometheusSource) Name() string {
	return "prometheus"
}

func (prom *PrometheusSource) GetExternalMetricInfoList() []p.ExternalMetricInfo {
	metricInfoList := make([]p.ExternalMetricInfo, 0)

	for metric := range prom.MetricList {
		metricInfo := p.ExternalMetricInfo{
			Metric: metric,
		}
		metricInfoList = append(metricInfoList, metricInfo)
	}

	return metricInfoList
}

func (prom *PrometheusSource) AddExternalMetric(metricName string, metric *ExternalMetric) {
	if _, ok := prom.MetricList[metricName]; ok {
		klog.Warningf("metric %s has been registered as an external metric. ", metricName)
	}

	klog.Infof("metric %s is registered as an external metric. ", metricName)
	prom.MetricList[metricName] = metric
}

func (prom *PrometheusSource) DeleteExternalMetric(metricName string) {
	klog.Infof("metric %s is delete from external metric list. ", metricName)
	delete(prom.MetricList, metricName)
}

func (prom *PrometheusSource) GetExternalMetric(info p.ExternalMetricInfo, _ string, _ labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {
	for metric := range prom.MetricList {
		if metric == info.Metric {
			values = append(values, prom.MetricList[metric].Value)
			return values, nil
		}
	}

	return nil, fmt.Errorf("not found metric %s from metric list", info.Metric)
}

func (prom *PrometheusSource) MonitorHPAs() {
	client := kubernetes.NewKubernetesClient()

	for {
		watcher, err := client.AutoscalingV2beta2().HorizontalPodAutoscalers(metav1.NamespaceAll).Watch(metav1.ListOptions{
			Watch: true,
		})
		if err != nil {
			klog.Errorf("Failed to start watch for new HPAs: %v", err)
			time.Sleep(time.Second)
			continue
		}

		watchChannel := watcher.ResultChan()

	innerLoop:
		for {
			select {
			case watchUpdate, ok := <-watchChannel:
				klog.Infof("HPA watch channel update. watchChanObject: %v", watchUpdate)
				if !ok {
					klog.Errorf("Event watch channel closed")
					break innerLoop
				}

				if watchUpdate.Type == kubewatch.Error {
					if status, ok := watchUpdate.Object.(*metav1.Status); ok {
						klog.Errorf("Error during watch: %#v", status)
						break innerLoop
					}
					klog.Errorf("Received unexpected error: %#v", watchUpdate.Object)
					break innerLoop
				}

				if hpa, ok := watchUpdate.Object.(*autoscalingv2.HorizontalPodAutoscaler); ok {
					switch watchUpdate.Type {
					case kubewatch.Added, kubewatch.Modified:
						if kubernetes.HasSpecificAnnotation(hpa) {
							metric := &ExternalMetric{}
							metricValue, err := kubernetes.GetPrometheusValue(hpa, prom.PrometheusUrl)
							if err != nil {
								klog.Errorf("failed to get value from prometheus server: %v", err)
								continue
							}
							metric.Value = metricValue
							prom.AddExternalMetric(metricValue.MetricName, metric)
						}
					case kubewatch.Deleted:
						if kubernetes.HasSpecificAnnotation(hpa) {
							metric := hpa.Annotations[kubernetes.PROMETHEUS_METRIC_NAME]
							prom.DeleteExternalMetric(metric)
						}
					default:
						klog.Warningf("Unknown watchUpdate.Type: %#v", watchUpdate.Type)
					}
				} else {
					klog.Errorf("Wrong object received: %v", watchUpdate)
				}
			}
		}
	}
}
