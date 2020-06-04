package slb

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/utils"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"

	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	SLB_L4_TRAFFIC_RX             = "slb_l4_traffic_rx"
	SLB_L4_TRAFFIC_TX             = "slb_l4_traffic_tx"
	SLB_L4_PACKET_TX              = "slb_l4_packet_tx"
	SLB_L4_PACKET_RX              = "slb_l4_packet_rx"
	SLB_L4_ACTIVE_CONNECTION      = "slb_l4_active_connection"
	SLB_L4_MAX_CONNECTION         = "slb_l4_max_connection"
	SLB_L4_CONNECTION_UTILIZATION = "slb_l4_connection_utilization"
	SLB_L7_QPS                    = "slb_l7_qps"
	SLB_L7_RT                     = "slb_l7_rt"
	SLB_L7_STATUS_2XX             = "slb_l7_status_2xx"
	SLB_L7_STATUS_3XX             = "slb_l7_status_3xx"
	SLB_L7_STATUS_4XX             = "slb_l7_status_4xx"
	SLB_L7_STATUS_5XX             = "slb_l7_status_5xx"
	SLB_L7_UPSTREAM_4XX           = "slb_l7_upstream_4xx"
	SLB_L7_UPSTREAM_5XX           = "slb_l7_upstream_5xx"
	SLB_L7_UPSTREAM_RT            = "slb_l7_upstream_rt"

	//Global Params
	SLB_INSTANCE_ID = "slb.instance.id"
	SLB_PORT        = "slb.instance.port"
	SLB_PERIOD      = "slb.period"

	MIN_PERIOD = 60
)

type SLBMetricSource struct{}

//list all external metric
func (sb *SLBMetricSource) GetExternalMetricInfoList() []p.ExternalMetricInfo {
	metricInfoList := make([]p.ExternalMetricInfo, 0)
	var MetricArray = []string{
		SLB_L4_TRAFFIC_RX,
		SLB_L4_TRAFFIC_TX,
		SLB_L4_PACKET_TX,
		SLB_L4_PACKET_RX,
		SLB_L4_ACTIVE_CONNECTION,
		SLB_L4_MAX_CONNECTION,
		SLB_L4_CONNECTION_UTILIZATION,
		SLB_L7_QPS,
		SLB_L7_RT,
		SLB_L7_STATUS_2XX,
		SLB_L7_STATUS_3XX,
		SLB_L7_STATUS_4XX,
		SLB_L7_STATUS_5XX,
		SLB_L7_UPSTREAM_4XX,
		SLB_L7_UPSTREAM_5XX,
		SLB_L7_UPSTREAM_RT,
	}
	for _, metric := range MetricArray {
		metricInfoList = append(metricInfoList, p.ExternalMetricInfo{
			Metric: metric,
		})
	}
	return metricInfoList
}

//according to the incoming label, get the metric..
func (sb *SLBMetricSource) GetExternalMetric(info p.ExternalMetricInfo, namespace string, requirements labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {
	switch info.Metric {
	case SLB_L4_TRAFFIC_RX:
		values, err = sb.getSLBMetrics(namespace, "TrafficRXNew", SLB_L4_TRAFFIC_RX, requirements)
	case SLB_L4_TRAFFIC_TX:
		values, err = sb.getSLBMetrics(namespace, "TrafficTXNew", SLB_L4_TRAFFIC_TX, requirements)
	case SLB_L4_PACKET_TX:
		values, err = sb.getSLBMetrics(namespace, "PacketTX", SLB_L4_PACKET_TX, requirements)
	case SLB_L4_PACKET_RX:
		values, err = sb.getSLBMetrics(namespace, "PacketRX", SLB_L4_PACKET_RX, requirements)
	case SLB_L4_ACTIVE_CONNECTION:
		values, err = sb.getSLBMetrics(namespace, "ActiveConnection", SLB_L4_ACTIVE_CONNECTION, requirements)
	case SLB_L4_MAX_CONNECTION:
		values, err = sb.getSLBMetrics(namespace, "MaxConnection", SLB_L4_MAX_CONNECTION, requirements)
	case SLB_L4_CONNECTION_UTILIZATION:
		values, err = sb.getSLBMetrics(namespace, "InstanceMaxConnectionUtilization", SLB_L4_CONNECTION_UTILIZATION, requirements)
	case SLB_L7_QPS:
		values, err = sb.getSLBMetrics(namespace, "Qps", SLB_L7_QPS, requirements)
	case SLB_L7_RT:
		values, err = sb.getSLBMetrics(namespace, "Rt", SLB_L7_RT, requirements)
	case SLB_L7_STATUS_2XX:
		values, err = sb.getSLBMetrics(namespace, "StatusCode2xx", SLB_L7_STATUS_2XX, requirements)
	case SLB_L7_STATUS_3XX:
		values, err = sb.getSLBMetrics(namespace, "StatusCode3xx", SLB_L7_STATUS_3XX, requirements)
	case SLB_L7_STATUS_4XX:
		values, err = sb.getSLBMetrics(namespace, "StatusCode4xx", SLB_L7_STATUS_4XX, requirements)
	case SLB_L7_STATUS_5XX:
		values, err = sb.getSLBMetrics(namespace, "StatusCode5xx", SLB_L7_STATUS_5XX, requirements)
	case SLB_L7_UPSTREAM_4XX:
		values, err = sb.getSLBMetrics(namespace, "UpstreamCode4xx", SLB_L7_UPSTREAM_4XX, requirements)
	case SLB_L7_UPSTREAM_5XX:
		values, err = sb.getSLBMetrics(namespace, "UpstreamCode5xx", SLB_L7_UPSTREAM_5XX, requirements)
	case SLB_L7_UPSTREAM_RT:
		values, err = sb.getSLBMetrics(namespace, "UpstreamRt", SLB_L7_UPSTREAM_RT, requirements)
	}
	if err != nil {
		log.Warningf("Failed to GetExternalMetric %s,because of %v", info.Metric, err)
	}
	return values, err
}

//the client of slb
func (sb *SLBMetricSource) Client() (client *cms.Client, err error) {

	accessUserInfo, err := utils.GetAccessUserInfo()
	if err != nil {
		log.Errorf("Failed to get accessUserInfo,because of %v.", err)
		return nil, err
	}

	if strings.HasPrefix(accessUserInfo.AccessKeyId, "STS.") {
		client, err = cms.NewClientWithStsToken(accessUserInfo.Region, accessUserInfo.AccessKeyId, accessUserInfo.AccessKeySecret, accessUserInfo.Token)
	} else {
		client, err = cms.NewClientWithAccessKey(accessUserInfo.Region, accessUserInfo.AccessKeyId, accessUserInfo.AccessKeySecret)

	}
	return client, err

}

// Global params
type SLBGlobalParams struct {
	InstanceId string `json:"instanceId"`
	Port       string `json:"port"`
}

//
func NewSLBMetricSource() *SLBMetricSource {
	return &SLBMetricSource{}
}

type SLBParams struct {
	SLBGlobalParams
	Period int
}

//get the slb specific metric values
func (sms *SLBMetricSource) getSLBMetrics(namespace, metric, externalMetric string, requirements labels.Requirements) (values []external_metrics.ExternalMetricValue, err error) {
	namespace = "acs_slb_dashboard"

	params, err := getSLBParams(requirements)
	if err != nil {
		return values, fmt.Errorf("failed to get slb params,because of %v", err)
	}

	client, err := sms.Client()
	if err != nil {
		log.Errorf("Failed to create slb client,because of %v", err)
		return values, err
	}

	request := cms.CreateDescribeMetricListRequest()
	request.Scheme = "https"
	request.Namespace = namespace
	request.MetricName = metric

	//time range
	endTime := time.Now().Add(-2 * time.Minute)
	startTime := endTime.Add(-1 * time.Duration(params.Period) * time.Second)
	//make ensure that the starttime minus Endtime is greater than period.
	err = utils.JudgeWithPeriod(startTime, endTime, params.Period)
	if err != nil {
		return values, err
	}

	request.StartTime = startTime.Format(utils.DEFAULT_TIME_FORMAT)
	request.EndTime = endTime.Format(utils.DEFAULT_TIME_FORMAT)

	dimensions, err := createDimensions(params.InstanceId, params.Port)
	if err != nil {
		log.Errorf("Dimensions conversion to json failed: %v", err)
		return values, err
	}
	request.Dimensions = dimensions
	response, err := client.DescribeMetricList(request)
	if err != nil {
		log.Errorf("Failed to get slb response,err: %v", err)
		return values, err
	}

	metricValue, err := getMetricFromDataPoints(response.Datapoints)
	if err != nil {
		log.Errorf("Failed to get slb metrics from api,because of %v", err)
		return values, err
	}
	values = append(values, external_metrics.ExternalMetricValue{
		MetricName: externalMetric,
		Value:      *resource.NewQuantity(int64(metricValue), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	})
	return values, nil
}

//request.Dimensions
type Dimensions struct {
	InstanceId string `json:"instanceId"`
	Port       string `json:"port"`
}

func createDimensions(instanceId, port string) (string, error) {
	dimensions := &Dimensions{instanceId, port}
	dimensionsByte, err := json.Marshal(dimensions)
	if err != nil {
		log.Errorf("dimensions To json err: %v", err)
		return "", err
	}
	return string(dimensionsByte), nil
}

//get the slb Params
func getSLBParams(requirements labels.Requirements) (params *SLBParams, err error) {
	params = &SLBParams{
		Period: MIN_PERIOD,
	}
	for _, r := range requirements {

		if len(r.Values().List()) <= 0 {
			continue
		}

		value := r.Values().List()[0]

		switch r.Key() {
		case SLB_INSTANCE_ID:
			params.InstanceId = value
		case SLB_PORT:
			params.Port = value
		case SLB_PERIOD:
			if params.Period, err = strconv.Atoi(value); err != nil {
				log.Errorf("Failed to parse period and skip,because of %v", err)
				continue
			}
		}
	}
	if params.InstanceId == "" || params.Port == "" {
		return params, errors.New("InstanceId and Port must be provide")
	}

	if params.Period < MIN_PERIOD {
		log.Warningf("The period you specific is too low and use MIN_PERIOD(%d) as default", MIN_PERIOD)
		params.Period = MIN_PERIOD
	}

	return params, nil
}

type DataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Vip       string  `json:"vip,omitempty"`
	Average   float64 `json:"Average"`
	Minimum   float64 `json:"Minimum"`
	Maximum   float64 `json:"Maximum"`
}

// extract metric data points
func getMetricFromDataPoints(datapoints string) (value float64, err error) {
	if datapoints == "" {
		return 0, errors.New("NoMetricData")
	}

	points := make([]DataPoint, 0)

	err = json.Unmarshal([]byte(datapoints), &points)

	if err != nil || len(points) == 0 {
		return 0, err
	}

	return points[len(points)-1].Average, nil
}
