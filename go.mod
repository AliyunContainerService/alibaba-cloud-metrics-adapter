module github.com/AliyunContainerService/alibaba-cloud-metrics-adapter

go 1.13

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.258
	github.com/aliyun/aliyun-log-go-sdk v0.1.10
	github.com/denverdino/aliyungo v0.0.0-20200609114633-3b95b3216337
	github.com/kubernetes-incubator/custom-metrics-apiserver v0.0.0-20200323093244-5046ce1afe6b
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/common v0.10.0
	github.com/stretchr/testify v1.4.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/component-base v0.17.3
	k8s.io/klog v1.0.0
	k8s.io/metrics v0.17.3
)

replace (
	// forced by the inclusion of sigs.k8s.io/metrics-server's use of this in their go.mod
	k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1 => ./localvendor/k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1
	sigs.k8s.io/metrics-server v0.3.7 => sigs.k8s.io/metrics-server v0.0.0-20200406215547-5fcf6956a533
)
