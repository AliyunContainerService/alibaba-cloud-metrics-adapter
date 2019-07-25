## SLS External metrics 

#### Global Params 

all metrics need the global params.
* sls.project:  The project name of a SLS instance.
* sls.logstore: The specific logStore of a SLS project.  

optional params 
* sls.ingress.route:    ingress route  
* sls.query.period:     query interval seconds, min val 15s
* sls.query.delay:      query delay seconds, default 0s
* sls.query.retry:      max retry, default 5
 

#### Metrics List 

| metric name     | description                     |     
| --------------- | ------------------------------- | 
| sls_ingress_qps | QPS of a specific ingress route |
| sls_ingress_latency_avg | latency of all requests |
| sls_ingress_latency_p50 | latency of 50% requests|
| sls_ingress_latency_p99 | latency of 99% requests |
| sls_ingress_latency_p9999 | latency of 99.99% requests |
| sls_ingress_inflow | inflow bandwidth of ingress |

#### Demo  
```
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: ingress-qps-hpa
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
              sls.logstore: "nginx-ingress"
              sls.ingress.route: "default-kubecon-springboot-demo-6666"
        target:
          type: AverageValue
          averageValue: 10
```



