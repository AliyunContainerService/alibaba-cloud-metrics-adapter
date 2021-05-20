package ahas

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/utils"
	ahas "github.com/aliyun/alibaba-cloud-sdk-go/services/ahas_openapi"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	log "k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	AHAS_SENTINEL_TOTAL_QPS = "ahas_sentinel_total_qps"
	AHAS_SENTINEL_PASS_QPS  = "ahas_sentinel_pass_qps"
	AHAS_SENTINEL_BLOCK_QPS = "ahas_sentinel_block_qps"
	AHAS_SENTINEL_AVG_RT    = "ahas_sentinel_avg_rt"

	NamespaceKey             = "ahas.sentinel.namespace"
	SentinelAppNameKey       = "ahas.sentinel.app"
	SentinelQueryIntervalKey = "ahas.sentinel.interval"
	SentinelQueryOffsetKey   = "ahas.sentinel.queryOffset"

	DefaultQueryInterval = 10
	DefaultQueryOffset   = 10
)

type AHASSentinelMetricSource struct{}

func (s *AHASSentinelMetricSource) GetExternalMetricInfoList() []provider.ExternalMetricInfo {
	metricInfoList := make([]p.ExternalMetricInfo, 0)
	var MetricArray = []string{
		AHAS_SENTINEL_PASS_QPS,
		AHAS_SENTINEL_BLOCK_QPS,
		AHAS_SENTINEL_TOTAL_QPS,
		AHAS_SENTINEL_AVG_RT,
	}
	for _, metric := range MetricArray {
		metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
			Metric: metric,
		})
	}
	return metricInfoList
}

func (s *AHASSentinelMetricSource) Name() string {
	return "ahas"
}

func (s *AHASSentinelMetricSource) GetExternalMetric(info provider.ExternalMetricInfo, namespace string, requirements labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {
	params, err := getAhasSentinelParams(requirements, namespace)
	if err != nil {
		return values, fmt.Errorf("failed to get AHAS Sentinel params, cause: %v", err)
	}

	client, err := s.createClient()
	if err != nil {
		log.Errorf("Failed to create AHAS Sentinel client, because of %v", err)
		return values, err
	}
	metricRequest := ahas.CreateGetSentinelAppSumMetricRequest()
	metricRequest.Namespace = params.AhasNamespace
	metricRequest.AppName = params.AppName
	interval := params.Interval
	queryOffset := params.QueryStartOffset
	endTimeStr := time.Now().Add(-1 * time.Duration(queryOffset) * time.Second).Format(utils.DEFAULT_TIME_FORMAT)
	startTimeStr := time.Now().Add(-1 * time.Duration(interval+queryOffset) * time.Second).Format(utils.DEFAULT_TIME_FORMAT)
	metricRequest.StartTime = startTimeStr
	metricRequest.EndTime = endTimeStr

	metrics, err := client.GetSentinelAppSumMetric(metricRequest)
	if err != nil {
		log.Errorf("Failed to get AHAS Sentinel response, err: %v", err)
		return values, err
	}
	count := resolveMetric(info, metrics)
	values = append(values, external_metrics.ExternalMetricValue{
		MetricName: info.Metric,
		Value:      *resource.NewQuantity(int64(count), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	})
	return values, nil
}

func resolveMetric(info provider.ExternalMetricInfo, response *ahas.GetSentinelAppSumMetricResponse) float64 {
	switch info.Metric {
	case AHAS_SENTINEL_TOTAL_QPS:
		return handleMetricAvg(response.MetricData.TotalCount, response)
	case AHAS_SENTINEL_PASS_QPS:
		return handleMetricAvg(response.MetricData.PassCount, response)
	case AHAS_SENTINEL_BLOCK_QPS:
		return handleMetricAvg(response.MetricData.BlockCount, response)
	case AHAS_SENTINEL_AVG_RT:
		return response.MetricData.AvgRt
	default:
		return 0
	}
}

func handleMetricAvg(count float64, response *ahas.GetSentinelAppSumMetricResponse) float64 {
	mc := response.MetricData.MachineCount
	if mc <= 0 {
		return 0
	} else {
		return count / float64(mc)
	}
}

func (s *AHASSentinelMetricSource) createClient() (client *ahas.Client, err error) {
	accessUserInfo, err := utils.GetAccessUserInfo()
	if err != nil {
		log.Errorf("Failed to get accessUserInfo, because of %v.", err)
		return nil, err
	}

	if strings.HasPrefix(accessUserInfo.AccessKeyId, "STS.") {
		client, err = ahas.NewClientWithStsToken(accessUserInfo.Region, accessUserInfo.AccessKeyId, accessUserInfo.AccessKeySecret, accessUserInfo.Token)
	} else {
		client, err = ahas.NewClientWithAccessKey(accessUserInfo.Region, accessUserInfo.AccessKeyId, accessUserInfo.AccessKeySecret)

	}
	return client, err
}

type AHASSentinelParams struct {
	SentinelGlobalParams
}

func getAhasSentinelParams(requirements labels.Requirements, k8sNamespace string) (params *AHASSentinelParams, err error) {
	params = &AHASSentinelParams{}
	for _, r := range requirements {

		if len(r.Values().List()) <= 0 {
			continue
		}

		value := r.Values().List()[0]

		switch r.Key() {
		case SentinelAppNameKey:
			params.AppName = value
		case NamespaceKey:
			params.AhasNamespace = value
		case SentinelQueryIntervalKey:
			if params.Interval, err = strconv.Atoi(value); err != nil {
				log.Errorf("Failed to parse AHAS Sentinel statistic interval, cause: %v", err)
				continue
			}
		case SentinelQueryOffsetKey:
			if params.QueryStartOffset, err = strconv.Atoi(value); err != nil {
				log.Errorf("Failed to parse AHAS Sentinel query start offset, cause: %v", err)
				continue
			}
		}
	}
	if len(params.AppName) <= 0 {
		// try to get Sentinel appName from pilot annotation.
		pilotMetadata, err := getPilotAnnotationMetadata(k8sNamespace)
		if err != nil {
			return params, err
		}
		if len(pilotMetadata.appName) <= 0 {
			return params, errors.New("appName in AHAS Sentinel is required")
		}
		// apply the configuration from pilot annotation.
		params.AppName = pilotMetadata.appName
		if len(params.AhasNamespace) <= 0 && len(pilotMetadata.namespace) > 0 {
			params.AhasNamespace = pilotMetadata.namespace
		}
	}
	if params.AhasNamespace == "" {
		params.AhasNamespace = "default"
	}

	if params.Interval <= 0 {
		if params.Interval < 0 {
			log.Warningf("The statistic interval you specific is too low and use 10s as default")
		}
		params.Interval = DefaultQueryInterval
	}

	if params.QueryStartOffset <= 0 {
		if params.QueryStartOffset < 0 {
			log.Warningf("Provided AHAS Sentinel query start offset (%d) is too low and use %ds as default", params.QueryStartOffset, DefaultQueryOffset)
		}
		params.QueryStartOffset = DefaultQueryOffset
	}

	return params, nil
}

// Global params
type SentinelGlobalParams struct {
	AppName          string
	AhasNamespace    string
	Interval         int
	QueryStartOffset int
}

func NewAHASSentinelMetricSource() *AHASSentinelMetricSource {
	return &AHASSentinelMetricSource{}
}
