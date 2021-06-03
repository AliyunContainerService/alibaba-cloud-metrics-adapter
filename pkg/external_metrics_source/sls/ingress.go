package sls

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"regexp"

	slssdk "github.com/aliyun/aliyun-log-go-sdk"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	log "k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type QPSResponse struct {
	Data []qps `json:"data"`
}
type qps struct {
	Count int64 `json:"count"`
}

type SLSIngressParams struct {
	SLSGlobalParams
	Route string
}

func (ss *SLSMetricSource) getSLSIngressQuery(params *SLSIngressParams, metricName string) (begin int64, end int64, query string) {
	now := time.Now().Unix()
	queryRealBegin := now - int64(params.DelaySeconds) - int64(params.Interval)
	end = now - int64(params.DelaySeconds)
	begin = now - 100
	if len(params.Route) == 0 {
		params.Route = "*"
	}
	var queryItem string
	switch metricName {
	case SLS_INGRESS_QPS:
		queryItem = fmt.Sprintf("count(1) / %d", params.Interval)
	case SLS_INGRESS_LATENCY_AVG:
		queryItem = "avg(request_time) * 1000"
	case SLS_INGRESS_LATENCY_P50:
		queryItem = "approx_percentile(request_time, 0.50) * 1000"
	case SLS_INGRESS_LATENCY_P95:
		queryItem = "approx_percentile(request_time, 0.95) * 1000"
	case SLS_INGRESS_LATENCY_P9999:
		queryItem = "approx_percentile(request_time, 0.9999) * 1000"
	case SLS_INGRESS_LATENCY_P99:
		queryItem = "approx_percentile(request_time, 0.99) * 1000"
	case SLS_INGRESS_INFLOW:
		queryItem = fmt.Sprintf("sum(request_length) / %d", params.Interval)
	default:
		// add default action for unknown metric
		query = ""
	}
	query = fmt.Sprintf("* and proxy_upstream_name: %s | SELECT %s as value from log WHERE __time__ >= %d  and __time__ < %d", params.Route, queryItem, queryRealBegin, end)
	return
}

func (ss *SLSMetricSource) getSLSIngressMetrics(namespace string, requirements labels.Requirements, metricName string) (values []external_metrics.ExternalMetricValue, err error) {

	params, err := getSLSParams(requirements)
	if err != nil {
		return values, fmt.Errorf("failed to get sls params,because of %v", err)
	}

	client, err := ss.Client(params.Internal)
	if err != nil {
		log.Errorf("Failed to create sls client, because of %v", err)
		return values, err
	}

	begin, end, query := ss.getSLSIngressQuery(params, metricName)

	if query == "" {
		log.Errorf("The metric you specific is not supported.")
		return values, errors.New("MetricNotSupport")
	}

	var queryRsp *slssdk.GetLogsResponse
	for i := 0; i < params.MaxRetry; i++ {
		queryRsp, err = client.GetLogs(params.Project, params.LogStore, "", begin, end, query, 100, 0, false)

		if err != nil || len(queryRsp.Logs) == 0 {
			return values, err
		}

		// if there are too many logs in sls, query may be not completed, we should retry
		if !queryRsp.IsComplete() {
			continue
		}

		value := queryRsp.Logs[0]["value"]
		var valid = regexp.MustCompile("[0-9.]")
		array := valid.FindAllStringSubmatch(value, -1)

		valStr := ""
		for _, i := range array {
			if len(i) == 1 {
				valStr += i[0]
			}
		}

		if valStr == "" {
			valStr = "0"
		}

		val, err := strconv.ParseFloat(valStr, 64)

		if err != nil {
			return values, err
		}

		values = append(values, external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
			Timestamp:  metav1.Now(),
		})

		return values, err
	}
	return values, errors.New("Query sls timeout,it might because of too many logs.")
}
