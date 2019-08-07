## alibaba-cloud-metrics-adapter 

[![License](https://img.shields.io/badge/license-Apache%202-4EB1BA.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)
[![Build Status](https://travis-ci.org/AliyunContainerService/alibaba-cloud-metrics-adapter.svg?branch=master)](https://travis-ci.org/AliyunContainerService/alibaba-cloud-metrics-adapter)


###  Overview 
An implementation of the Kubernetes [Custom Metrics API and External Metrics API](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#support-for-metrics-apis) for Alibaba Cloud.This adapter enables you to scale your application pods running on ACK using the Horizontal Pod Autoscaler (HPA) with External Metrics such as ingress QPS, ARMS jvm RT and so on.

### Installation 
```$xslt
kubectl apply -f deploy/deploy.yaml 
```
### Example 
HPA with external metric (sls_ingress_qps)
```$xslt
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: kubecon-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1beta2
    kind: Deployment
    name: kubecon-springboot-demo
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: External
      external:
        metric:
          name: sls_ingress_qps
          selector:
            matchLabels:
              sls.project: "k8s-log-c550367cdf1e84dfabab013b277cc6bc2"
              sls.logStore: "nginx-ingress"
              sls.ingress.route: "default-kubecon-springboot-demo-6666"
        target:
          type: AverageValue
          averageValue: 10
```
setup stress engine and watch the hpa output.

```$xslt
NAME          REFERENCE                            TARGETS      MINPODS   MAXPODS   REPLICAS   AGE
kubecon-hpa   Deployment/kubecon-springboot-demo   120/10 (avg)   2         10        8          6m3s
```

### Cloud Resource Metrics   
* <a href="docs/metrics/sls.md">Ingress（SLS)</a>
* <a href="docs/metrics/slb.md">SLB</a>
* <a href="docs/metrics/cms.md">CMS</a>


### Contributing 
Please check <a href="docs/CONTRIBUTING.md">CONTRIBUTING.md</a>

### License 
This software is released under the Apache 2.0 license.
