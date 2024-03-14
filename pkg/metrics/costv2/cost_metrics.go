package costv2

//
//const (
//	COST_CPU_REQUEST = "cost_cpu_request"
//)
//
//type COSTV2MetricSource struct {
//	*prometheusProvider.AlibabaMetricsAdapterOptions
//}
//
//// list all external metric
//func (cs *COSTV2MetricSource) GetExternalMetricInfoList() []p.ExternalMetricInfo {
//	metricInfoList := make([]p.ExternalMetricInfo, 0)
//	var MetricArray = []string{
//		COST_CPU_REQUEST,
//	}
//	for _, metric := range MetricArray {
//		metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
//			Metric: metric,
//		})
//	}
//	return metricInfoList
//}
//
//// according to the incoming label, get the metric..
//func (cs *COSTV2MetricSource) GetExternalMetric(info p.ExternalMetricInfo, namespace string, requirements labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {
//	promSql := getPrometheusSql(info.Metric)
//	query := buildExternalQuery(namespace, promSql, requirements)
//	if info.Metric == COST_TOTAL_HOUR || info.Metric == COST_TOTAL_MIN || info.Metric == COST_TOTAL_DAY || info.Metric == COST_TOTAL_WEEK || info.Metric == COST_TOTAL_MONTH {
//		values, err = cs.getCOSTMetrics(namespace, info.Metric, prom.Selector(promSql))
//	} else {
//		values, err = cs.getCOSTMetrics(namespace, info.Metric, query)
//	}
//	if err != nil {
//		log.Warningf("Failed to GetExternalMetric %s,because of %v", info.Metric, err)
//	}
//	return values, err
//}
//
//func getPrometheusSql(metricName string) (item string) {
//	// ksm custom set
//	switch metricName {
//	case COST_CPU_REQUEST:
//		item = `avg(avg_over_time(kube_pod_container_resource_requests{resource="cpu", unit="core", container!="", container!="POD", node!="", %s}[%s])) by (container, pod, namespace, node, %s)`
//		item = `sum(kube_pod_container_resource_requests_cpu_cores{job="_kube-state-metrics"}) by(pod) * on(pod) group_right sum(kube_pod_labels{%s}) by(pod)`
//
//	}
//}
//
//// add custom param to promql
//func buildPromqlQuery(namespace, promql string, requirements labels.Requirements) (externalQuery prom.Selector) {
//}
//
//func NewCOSTV2MetricSource() *COSTV2MetricSource {
//	return &COSTV2MetricSource{
//		prometheusProvider.GlobalConfig,
//	}
//}
