module github.com/AliyunContainerService/alibaba-cloud-metrics-adapter

go 1.13

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.258
	github.com/aliyun/aliyun-log-go-sdk v0.1.10
	github.com/denverdino/aliyungo v0.0.0-20200609114633-3b95b3216337
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/prometheus/client_golang v1.11.1
	github.com/prometheus/common v0.26.0
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/text v0.3.6
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
	k8s.io/component-base v0.22.0
	k8s.io/klog/v2 v2.40.1
	k8s.io/metrics v0.22.0
	sigs.k8s.io/custom-metrics-apiserver v1.22.0
	sigs.k8s.io/prometheus-adapter v0.9.1
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.9.0
)
