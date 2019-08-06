## CMS(k8s) External metrics

#### Global Params
all metrics need the global params.

| global params       | description              | example            | required | 
| ------------------- | ------------------------ | ------------------ | -------- | 
| k8s.cluster.id      | the cluster id of Aliyun Container Service. | c7689a1dcf77c42a3b26114f851fa8fef | True | 
| k8s.workload.type   | kind of reference Object.| Deployment(default value)| False | 
| k8s.workload.namespace| namespace of reference Object. | default (default value) | False | 
| k8s.workload.name   | name of reference Object | demo | True | 

#### Metrics List

| metric name                  | description                               | extra params |
| ---------------------------- | ----------------------------------------- | ------------ |
| k8s_workload_cpu_util             | average cpu util per minute                      | None         |
| k8s_workload_cpu_limit             | cpu limit                       | None         |
| k8s_workload_cpu_request              | cpu request      | None         |
| k8s_workload_memory_usage              | memory usage              | None         |
| k8s_workload_memory_request      | memory request                        | None         |
| k8s_workload_memory_limit         | memory limit            | None         |
| k8s_workload_memory_working_set | working set                 | None         |
| k8s_workload_memory_rss                   | rss                                       | None         |
| k8s_workload_memory_cache                    | cache                             | None         |
| k8s_workload_network_tx_rate             | network transaction rate                   | None         |
| k8s_workload_network_rx_rate             | network receice rate                  | None         |
| k8s_workload_network_tx_errors             | tx errors                  | None         |
| k8s_workload_network_rx_errors             | rx errors                   | None         |
#### Demo

```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: cms-cpu-hpa
  namespace: kube-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1beta2
    kind: Deployment
    name: arms-springboot-demo-hanyan-system
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: External
      external:
        metric:
          name: k8s_workload_cpu_util
          selector:
            matchLabels:
              k8s.cluster.id: "c7689a1dcf77c42a3b26114f851fa8fef"
              k8s.workload.type: "Deployment"
              k8s.workload.name: "demo"
        target:
          type: AverageValue
          averageValue: 10m
```



