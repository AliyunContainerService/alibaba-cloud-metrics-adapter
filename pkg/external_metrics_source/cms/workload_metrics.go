package cms

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	log "k8s.io/klog"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	// global const
	DEFAULT_ACS_KUBERNETES    = "acs_kubernetes"
	K8S_DEFAULT_WORKLOAD_TYPE = "Deployment"

	// params
	K8S_NAMESPACE     = "k8s.workload.namespace"
	K8S_WORKLOAD_TYPE = "k8s.workload.type"
	K8S_WORKLOAD_NAME = "k8s.workload.name"
	K8S_CLUSTER_ID    = "k8s.cluster.id"
	K8S_PERIOD        = "k8s.period"
	//
	MIN_PERIOD = 60
)

type DataPoint struct {
	Timestamp int64   `json:"timestamp"`
	UserId    string  `json:"userId"`
	GroupId   string  `json:"groupId"`
	Value     float64 `json:"value"`
	Sum       float64 `json:"Sum"`
	Average   float64 `json:"average"`
	Maximum   float64 `json:"maximum"`
	Minimum   float64 `json:"minimum"`
}

type CMSMetricParams struct {
	CMSGlobalParams
	Namespace    string
	ClusterId    string
	WorkloadType string
	WorkloadName string
}

type CMSGlobalParams struct {
	Period int
}

// get cms workload metrics
func (cs *CMSMetricSource) getCMSWorkLoadMetrics(namespace string, requires labels.Requirements, info p.ExternalMetricInfo) (values []external_metrics.ExternalMetricValue, err error) {
	log.V(4).Infof("Request to getCMSWorkLoadMetrics namespace: %s,requires: %s, metric: %s\n", namespace, requires, info.Metric)

	params, err := getCMSParams(namespace, requires)

	if err != nil {
		return values, fmt.Errorf("Failed to get CMS params, because of %v", err)
	}

	// get cluster id from group
	groupId, err := cs.getGroupIdByName(params)

	if err != nil || groupId <= 0 {
		return values, err
	}

	dataPoints, err := cs.getMetricListByGroupId(params, groupId, info.Metric)
	if err != nil {
		return values, err
	}

	if len(dataPoints) > 0 {
		values = append(values, external_metrics.ExternalMetricValue{
			MetricName: info.Metric,
			Timestamp:  metav1.Now(),
			Value:      *resource.NewQuantity(int64(dataPoints[len(dataPoints)-1].Sum), resource.DecimalSI),
		})
	}
	return values, err
}

func getCMSParams(namespace string, requirements labels.Requirements) (params *CMSMetricParams, err error) {
	params = &CMSMetricParams{
		Namespace:    namespace,
		WorkloadType: K8S_DEFAULT_WORKLOAD_TYPE,
	}
	for _, r := range requirements {

		if len(r.Values().List()) <= 0 {
			log.Warning("You don't specific any labels and skip")
			continue
		}

		value := r.Values().List()[0]

		switch r.Key() {
		case K8S_PERIOD:
			if params.Period, err = strconv.Atoi(value); err != nil {
				log.Warningf("Failed to parse period and use MIN_PERIOD(%d) as default", MIN_PERIOD)
				continue
			}
		case K8S_CLUSTER_ID:
			params.ClusterId = value
		case K8S_NAMESPACE:
			params.Namespace = value
		case K8S_WORKLOAD_TYPE:
			params.WorkloadType = value
		case K8S_WORKLOAD_NAME:
			params.WorkloadName = value
		}
	}

	if params.ClusterId == "" || params.WorkloadType == "" || params.WorkloadName == "" {
		return params, errors.New(fmt.Sprintf("%s %s %s must be provided", K8S_CLUSTER_ID, K8S_WORKLOAD_TYPE, K8S_WORKLOAD_NAME))
	}

	// avoid too short range of period
	if params.Period < MIN_PERIOD {
		params.Period = MIN_PERIOD
	}

	return params, nil
}

// get group id from meta
func (cs *CMSMetricSource) getGroupIdByName(params *CMSMetricParams) (groupId int64, err error) {

	//generate cms GroupName
	groupName := fmt.Sprintf("k8s-%s-%s-%s-%s", params.ClusterId, params.Namespace, params.WorkloadType, params.WorkloadName)

	request := cms.CreateDescribeMonitorGroupsRequest()
	request.Scheme = "https"
	request.PageSize = requests.NewInteger(1)
	request.GroupName = groupName
	request.SelectContactGroups = requests.NewBoolean(false)

	client, err := cs.Client()

	if err != nil {
		return 0, fmt.Errorf("failed to create cms client,because of %v", err)
	}

	response, err := client.DescribeMonitorGroups(request)

	if err != nil {
		return 0, fmt.Errorf("failed to query workload from cms api,because of %v", err)
	}

	if response.Success && response.Total == 1 {
		groups := response.Resources.Resource
		return groups[0].GroupId, err
	}

	return 0, err
}

func (cs *CMSMetricSource) getMetricListByGroupId(params *CMSMetricParams, groupId int64, metricName string) (values []DataPoint, err error) {
	request := cms.CreateDescribeMetricListRequest()
	request.Scheme = "https"

	// cms namespace not k8s namespace
	request.Namespace = DEFAULT_ACS_KUBERNETES
	request.MetricName = metricName

	// create dimensions
	dimensions := fmt.Sprintf("[{\"groupId\":\"%d\"}]", groupId)
	request.Dimensions = dimensions

	// time range
	startTime := time.Now().Add(-5 * time.Duration(params.Period) * time.Second).Format(utils.DEFAULT_TIME_FORMAT)
	endTime := time.Now().Format(utils.DEFAULT_TIME_FORMAT)

	request.StartTime = startTime
	request.EndTime = endTime

	client, err := cs.Client()

	if err != nil {
		log.Errorf("Failed to create cms client,because of %v", err)
		return
	}

	response, err := client.DescribeMetricList(request)

	if err != nil {
		log.Errorf("Failed to describe metric list,because of %v", err)
		return
	}
	if response.Success {
		dataPoint := response.Datapoints
		if dataPoint == "[]" {
			return values, fmt.Errorf("datapoint is empty %v", err)
		}

		var res []DataPoint

		err := json.Unmarshal([]byte(dataPoint), &res)
		if err != nil {
			return values, fmt.Errorf("json unmarshal datapoint exception %v", err)
		}
		return res, nil
	}
	return values, err
}

func (cs *CMSMetricSource) Client() (client *cms.Client, err error) {
	accessUserInfo, err := utils.GetAccessUserInfo()
	if err != nil {
		log.Errorf("Failed to create cms client,because of %v", err)
		return nil, err
	}

	if strings.HasPrefix(accessUserInfo.AccessKeyId, "STS.") {
		client, err = cms.NewClientWithStsToken(accessUserInfo.Region, accessUserInfo.AccessKeyId, accessUserInfo.AccessKeySecret, accessUserInfo.Token)
	} else {
		client, err = cms.NewClientWithAccessKey(accessUserInfo.Region, accessUserInfo.AccessKeyId, accessUserInfo.AccessKeySecret)

	}
	return client, err
}
