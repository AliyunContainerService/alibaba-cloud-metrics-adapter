/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/metrics"
	p "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
)

type AlibabaCloudMetricsProvider struct {
	mapper     apimeta.RESTMapper
	kubeClient dynamic.Interface

	// external metrics manager
	eManager *metrics.ExternalMetricsManager

	// todo custom metrics manager
}

func NewAlibabaCloudProvider(mapper apimeta.RESTMapper, dynamicClient dynamic.Interface) (p.MetricsProvider, error) {
	return &AlibabaCloudMetricsProvider{
		mapper:     mapper,
		kubeClient: dynamicClient,
		eManager:   metrics.GetExternalMetricsManager(),
	}, nil
}
