package sls

import (
	"fmt"
	"strconv"

	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/utils"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"k8s.io/apimachinery/pkg/labels"
	log "k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	p "sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

const (
	SLS_INGRESS_QPS           = "sls_ingress_qps"
	SLS_INGRESS_LATENCY_AVG   = "sls_ingress_latency_avg"
	SLS_INGRESS_LATENCY_P50   = "sls_ingress_latency_p50"
	SLS_INGRESS_LATENCY_P95   = "sls_ingress_latency_p95"
	SLS_INGRESS_LATENCY_P9999 = "sls_ingress_latency_p9999"
	SLS_INGRESS_LATENCY_P99   = "sls_ingress_latency_p99"
	SLS_INGRESS_INFLOW        = "sls_ingress_inflow" // byte per second

	SLS_SCALING = "sls_scaling"

	SLS_LABEL_PROJECT         = "sls.project"
	SLS_LABEL_LOGSTORE        = "sls.logstore"
	SLS_LABEL_QUERY_INTERVAL  = "sls.query.interval"  // query interval seconds, min val 15s
	SLS_LABEL_QUERY_DELAY     = "sls.query.delay"     // query delay seconds, default 0s
	SLS_LABEL_QUERY_MAX_RETRY = "sls.query.max_retry" // max retry, default 5
	SLS_LABEL_INGRESS_ROUTE   = "sls.ingress.route"   // e.g. namespace-svc-port
	SLS_LABEL_ETL_JOB         = "sls.etl.job"         // etl job name
	SLS_INTERNAL_ENDPOINT     = "sls.internal.endpoint"

	SLS_LABEL_SCALING_ENTITY = "sls.scaling.entity" // prediction entity for auto scaling
	SLS_LABEL_SCALING_METRIC = "sls.scaling.metric" // prediction metric for auto scaling

	MIN_INTERVAL      = 15   // minimum interval of ingress query
	MIN_PRED_INTERVAL = 3600 // minimum interval of scaling query
	MAX_RETRY_DEFAULT = 5
	DELAY_SECONDS     = 10
)

func isIngressRelated(metric string) bool {
	ingressRelated := map[string]bool{
		SLS_INGRESS_QPS:           true,
		SLS_INGRESS_LATENCY_AVG:   true,
		SLS_INGRESS_LATENCY_P50:   true,
		SLS_INGRESS_LATENCY_P95:   true,
		SLS_INGRESS_LATENCY_P99:   true,
		SLS_INGRESS_LATENCY_P9999: true,
		SLS_INGRESS_INFLOW:        true,
	}
	return ingressRelated[metric]
}

func isScalingRelated(metric string) bool {
	scalingRelated := map[string]bool{
		SLS_SCALING: true,
	}
	return scalingRelated[metric]
}

type SLSMetricSource struct{}

func (ss *SLSMetricSource) GetExternalMetricInfoList() []p.ExternalMetricInfo {
	metricInfoList := make([]p.ExternalMetricInfo, 0)

	// Ingress QPS
	metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
		Metric: SLS_INGRESS_QPS,
	})
	// Ingress latency avg
	metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
		Metric: SLS_INGRESS_LATENCY_AVG,
	})
	// Ingress latency 50%
	metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
		Metric: SLS_INGRESS_LATENCY_P50,
	})
	//Ingress latency 95%
	metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
		Metric: SLS_INGRESS_LATENCY_P95,
	})
	//Ingress latency 99.99%
	metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
		Metric: SLS_INGRESS_LATENCY_P9999,
	})
	// Ingress latency 99%
	metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
		Metric: SLS_INGRESS_LATENCY_P99,
	})
	// ingress inflow
	metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
		Metric: SLS_INGRESS_INFLOW,
	})
	// scaling qps
	metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
		Metric: SLS_SCALING,
	})
	return metricInfoList
}
func (ss *SLSMetricSource) GetExternalMetric(info p.ExternalMetricInfo, namespace string, requirements labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {
	if isIngressRelated(info.Metric) {
		values, err = ss.getSLSIngressMetrics(namespace, requirements, info.Metric)
		if err != nil {
			log.Warningf("Failed to GetExternalMetric %s,because of %v", info.Metric, err)
		}
	} else if isScalingRelated(info.Metric) {
		values, err = ss.getSLSScalingMetrics(namespace, requirements, info.Metric)
		if err != nil {
			log.Warningf("Failed to GetExternalMetric %s,because of %v", info.Metric, err)
		}
	}
	return values, err
}

// create client with specific project
func (ss *SLSMetricSource) Client(internal bool) (client sls.ClientInterface, err error) {

	accessUserInfo, err := utils.GetAccessUserInfo()
	if err != nil {
		log.Infof("Failed to GetAccessUserInfo,because of %v", err)
		return client, err
	}
	var endpoint string
	if internal {
		endpoint = fmt.Sprintf("%s-intranet.log.aliyuncs.com", accessUserInfo.Region)
	} else {
		endpoint = fmt.Sprintf("%s.log.aliyuncs.com", accessUserInfo.Region)
	}
	client = sls.CreateNormalInterface(endpoint, accessUserInfo.AccessKeyId, accessUserInfo.AccessKeySecret, accessUserInfo.Token)

	return client, nil
}

// get sls params from labels
func getSLSIngressParams(requirements labels.Requirements) (params *SLSIngressParams, err error) {
	// set default value
	params = &SLSIngressParams{
		SLSGlobalParams: SLSGlobalParams{
			Interval:     MIN_INTERVAL,
			MaxRetry:     MAX_RETRY_DEFAULT,
			DelaySeconds: 10,
			Internal:     true,
		},
	}
	for _, r := range requirements {

		if len(r.Values().List()) <= 0 {
			log.Warningf("You don't provide value of %s and skip.", r.Key())
			continue
		}

		value := r.Values().List()[0]

		switch r.Key() {
		case SLS_LABEL_PROJECT:
			params.Project = value
		case SLS_LABEL_LOGSTORE:
			params.LogStore = value
		case SLS_LABEL_INGRESS_ROUTE:
			params.Route = value
		case SLS_LABEL_QUERY_INTERVAL:
			if params.Interval, err = strconv.Atoi(value); err != nil {
				log.Errorf("Failed to parse %s,because of %v.", SLS_LABEL_QUERY_INTERVAL, err)
				return nil, err
			}
		case SLS_LABEL_QUERY_DELAY:
			if params.DelaySeconds, err = strconv.Atoi(value); err != nil {
				log.Errorf("Failed to parse %s,because of %v.", SLS_LABEL_QUERY_DELAY, err)
				return nil, err
			}
		case SLS_LABEL_QUERY_MAX_RETRY:
			if params.MaxRetry, err = strconv.Atoi(value); err != nil {
				log.Errorf("Failed to parse %s,because of %v", SLS_LABEL_QUERY_MAX_RETRY, err)
				return nil, err
			}
		case SLS_INTERNAL_ENDPOINT:
			if value != "" && value == "false" {
				params.Internal = false
			}
		}
	}

	if params.Project == "" || params.LogStore == "" {
		return params, fmt.Errorf("%s and %s must be provided", SLS_LABEL_PROJECT, SLS_LABEL_LOGSTORE)
	}
	if params.Interval < MIN_INTERVAL {
		log.Infof("The interval you specific is %d and less than the MIN_INTERVAL(%d).Use MIN_INTERVAL as default.", params.Interval, MIN_INTERVAL)
		params.Interval = MIN_INTERVAL
	}
	if params.MaxRetry < 1 {
		log.Infof("The MaxRetry you specific is %d and use MAX_RETRY_DEFAULT(%d) as default", params.MaxRetry, MAX_RETRY_DEFAULT)
		params.MaxRetry = MAX_RETRY_DEFAULT
	}

	if params.DelaySeconds < 0 {
		log.Infof("The DelaySeconds you specific is %d and use MAX_RETRY_DEFAULT(%d) as default", params.DelaySeconds, DELAY_SECONDS)
		params.DelaySeconds = DELAY_SECONDS
	}

	return params, nil
}

func getSLSScalingParams(requirements labels.Requirements) (params *SLSScalingParams, err error) {
	// set default value
	params = &SLSScalingParams{
		SLSGlobalParams: SLSGlobalParams{
			Internal: true,
		},
	}
	for _, r := range requirements {

		if len(r.Values().List()) <= 0 {
			log.Warningf("You don't provide value of %s and skip.", r.Key())
			continue
		}

		value := r.Values().List()[0]

		switch r.Key() {
		case SLS_LABEL_PROJECT:
			params.Project = value
		case SLS_LABEL_LOGSTORE:
			params.LogStore = value
		case SLS_LABEL_ETL_JOB:
			params.JobName = value
		case SLS_LABEL_SCALING_ENTITY:
			params.Entity = value
		case SLS_LABEL_SCALING_METRIC:
			params.Metric = value
		case SLS_LABEL_QUERY_INTERVAL:
			if params.Interval, err = strconv.Atoi(value); err != nil {
				log.Errorf("Failed to parse %s,because of %v.", SLS_LABEL_QUERY_INTERVAL, err)
				return nil, err
			}
		case SLS_LABEL_QUERY_MAX_RETRY:
			if params.MaxRetry, err = strconv.Atoi(value); err != nil {
				log.Errorf("Failed to parse %s,because of %v", SLS_LABEL_QUERY_MAX_RETRY, err)
				return nil, err
			}
		case SLS_INTERNAL_ENDPOINT:
			if value != "" && value == "false" {
				params.Internal = false
			}
		}
	}

	if params.Project == "" || params.LogStore == "" {
		return params, fmt.Errorf("%s and %s must be provided", SLS_LABEL_PROJECT, SLS_LABEL_LOGSTORE)
	}
	if params.JobName == "" || params.Entity == "" || params.Metric == "" {
		return params, fmt.Errorf("%s, %s and %s must be provided", SLS_LABEL_ETL_JOB, SLS_LABEL_SCALING_ENTITY, SLS_LABEL_SCALING_METRIC)
	}
	if params.Interval < MIN_PRED_INTERVAL {
		log.Infof("The interval you specific is %d and less than the MIN_PRED_INTERVAL(%d).Use MIN_PRED_INTERVAL as default.", params.Interval, MIN_PRED_INTERVAL)
		params.Interval = MIN_PRED_INTERVAL
	}
	if params.MaxRetry < 1 {
		log.Infof("The MaxRetry you specific is %d and use MAX_RETRY_DEFAULT(%d) as default", params.MaxRetry, MAX_RETRY_DEFAULT)
		params.MaxRetry = MAX_RETRY_DEFAULT
	}

	return params, nil
}

// Global params
type SLSGlobalParams struct {
	// common parameters
	Project  string
	LogStore string
	Internal bool
	MaxRetry int
	Interval int

	// ingress related parameters
	DelaySeconds int

	// scaling relate parameters
	JobName string
	Entity  string
	Metric  string
}

func NewSLSMetricSource() *SLSMetricSource {
	return &SLSMetricSource{}
}
