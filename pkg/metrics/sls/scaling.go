package sls

import (
	"fmt"
	"strconv"
	"time"

	slssdk "github.com/aliyun/aliyun-log-go-sdk"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	log "k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const MLLogstore string = "internal-ml-log"
const MLPredQuery string = `
	* and series_prediction and "__tag__:__job_name__":%s | select time, value from (
		(select max(__time__) as pred_time, max_by("__tag__:__batch_id__", __time__) as batch_id from log where cast(json_extract(result, '$.entity') as varchar)='%s' and cast(json_extract(result, '$.metric') as varchar)='%s' and not result_type='prediction_error') t1 
			join 
		(select "__tag__:__batch_id__" as batch_id, cast(json_extract(result, '$.time') as bigint) as time, cast(json_extract(result, '$.expect_value') as double) as value 
			from log where cast(json_extract(result, '$.entity') as varchar)='%s' and cast(json_extract(result, '$.metric') as varchar)='%s' limit 100000) t2 
	on t1.batch_id=t2.batch_id) where time > %d order by time
`

type SLSScalingParams struct {
	SLSGlobalParams
}

func (ss *SLSMetricSource) getSLSScalingQuery(params *SLSScalingParams, metricName string) (begin int64, end int64, query string, err error) {
	now := time.Now().Unix()
	begin = now - int64(params.Interval)
	end = now

	switch metricName {
	case SLS_SCALING:
		query = fmt.Sprintf(MLPredQuery, params.JobName, params.Entity, params.Metric, params.Entity, params.Metric, now)
	default:
		err = fmt.Errorf("failed to get ml prediction query: unsupported metric %s(qps)", metricName)
		log.Errorf(err.Error())
	}
	return begin, end, query, err
}

func (ss *SLSMetricSource) getSLSScalingMetrics(namespace string, requirements labels.Requirements, metricName string) (values []external_metrics.ExternalMetricValue, err error) {

	params, err := getSLSScalingParams(requirements)
	if err != nil {
		log.Errorf("failed to get scaling metrics for sls scaling params error: %v", err)
		return values, err
	}

	client, err := ss.Client(params.Internal)
	if err != nil {
		log.Errorf("failed to get scaling metrics for sls client error: %v", err)
		return values, err
	}

	begin, end, query, err := ss.getSLSScalingQuery(params, metricName)
	if err != nil {
		return values, err
	}

	var resp *slssdk.GetLogsResponse
	for i := 0; i < params.MaxRetry; i++ {
		resp, err = client.GetLogs(params.Project, params.LogStore, "", begin, end, query, 10000, 0, false)
		if err != nil || len(resp.Logs) == 0 {
			return values, err
		}

		if !resp.IsComplete() {
			continue
		}

		// just reture the next metric
		log := resp.Logs[0]
		ts, err := strconv.ParseInt(log["time"], 10, 64)
		if err != nil {
			return values, err
		}
		val, err := strconv.ParseFloat(log["value"], 64)
		if err != nil {
			return values, err
		}
		values = append(values, external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
			Timestamp:  metav1.Unix(ts, 0),
		})
		break
	}
	return values, nil
}
