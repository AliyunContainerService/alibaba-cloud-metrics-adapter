package sls

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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

	client, err := ss.Client(params.Project, params.Internal)
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

func (ss *SLSMetricSource) getSLSIngressPredictQuery(params *SLSIngressParams, metricName string) (begin int64, end int64, query string) {
	now := time.Now().Unix()

	switch metricName {
	case SLS_INGRESS_QPM:
		end = (now / 60) * 60
		begin = (now / 60 - 10) * 60
		var querySearch string = fmt.Sprintf(`proxy_upstream_name: '%v' or proxy_alternative_upstream_name: '%v'`, params.Route, params.Route)
		query = fmt.Sprintf(`%v | select array_agg(to_unixtime(time)) as ts, array_agg(num) as ds from ( select date_trunc('minute', __time__) as time, COUNT(*) as num from log group by time ) limit 1000`, querySearch)
	case SLS_INGRESS_PREDICT:
		end = now / 60 * 60 - 60
		begin = end - 60
		if len(params.Route) == 0 {
			params.Route = "*"
		}
		var querySearch string = fmt.Sprintf(`__tag__:__model_type__: predict and '%v'`, params.Route)
		var queryItem string = fmt.Sprintf("json_extract(result, '$.ts') as ts, json_extract(result, '$.ds') as ds")
		var filterItem string = fmt.Sprintf("json_extract_scalar(meta, '$.logstore_name') = '%v' and json_extract_scalar(meta, '$.project_name') = '%v'", params.LogStore, params.Project)
		query = fmt.Sprintf("%v | SELECT %v from log WHERE __time__ >= %d  and __time__ < %d and %v limit 1000", querySearch, queryItem, begin, end, filterItem)
	}
	return
}

func (ss *SLSMetricSource) getSLSIngressPredictMetrics(namespace string, requirements labels.Requirements, metricName string) (values []external_metrics.ExternalMetricValue, err error) {
	params, err := getSLSParams(requirements)
	if err != nil {
		return values, fmt.Errorf("failed to get sls params,because of %v", err)
	}

	client, err := ss.Client(params.Project, params.Internal)
	if err != nil {
		log.Errorf("Failed to create sls client, because of %v", err)
		return values, err
	}

	begin, end, query := ss.getSLSIngressPredictQuery(params, metricName)
	fmt.Println("begin", begin, "end", end, "query", query)

	if query == "" {
		log.Errorf("The metric you specific is not supported.")
		return values, errors.New("MetricNotSupport")
	}

	var queryRsp *slssdk.GetLogsResponse
	for i := 0; i < params.MaxRetry; i++ {
		queryRsp, err = client.GetLogs(params.Project, params.MlLogStore, "", begin, end, query, 100, 0, false)

		if err != nil || len(queryRsp.Logs) == 0 {
			return values, err
		}

		// if there are too many logs in sls, query may be not completed, we should retry
		if !queryRsp.IsComplete() {
			continue
		}

		expectScore := -1.0
		for _, logCell := range queryRsp.Logs {
			ds := logCell["ds"]
			content := ds[1:len(ds)-1]
			if len(content) <= 0 {
				if expectScore <= 0.0 {
					expectScore = 0.0
				}
			} else {
				dsString := strings.Split(ds[1:len(ds)-1], ",")

				// 计算未来几分钟的最大值，需要提前预留对应的资源
				maxValue := -1.0
				n := len(dsString)
				for i := 0; i < n; i++ {
					if v, e := strconv.ParseFloat(dsString[i], 64); e == nil {
						if maxValue < v {
							maxValue = v
						}
					} else {
						err = e
					}
				}
				if maxValue > expectScore {
					expectScore = maxValue
				}
			}
		}

		fmt.Println("Project", params.Project, "MlLogStore", params.MlLogStore, "expectScore", expectScore, "timestamp", metav1.Now())
		if err != nil {
			return values, err
		}

		values = append(values, external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(expectScore), resource.DecimalSI),
			Timestamp:  metav1.Now(),
		})

		return values, err
	}
	return values, errors.New("Query sls timeout,it might because of too many logs.")
}