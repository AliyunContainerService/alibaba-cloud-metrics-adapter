## SLB External metrics

#### Global Params

all metrics need the global params.

- slb.instance.id: The ID of a SLB instance.
- slb.instance.port：The port of SLB instance.

#### Metrics List

| metric name                  | description                               | extra params |
| ---------------------------- | ----------------------------------------- | ------------ |
| slb_l4_traffic_rx             | Inflows per second                        | None         |
| slb_l4_traffic_tx             | Outflow per second                        | None         |
| slb_l4_packet_tx              | Number of packets inflows per second      | None         |
| slb_l4_packet_rx              | Number of packets per second              | None         |
| slb_l4_active_connection      | Active connections                        | None         |
| slb_l4_max_connection         | Maximum number of connections             | None         |
| slb_l4_connection_utilization | Maximum connection usage                  | None         |
| slb_l7_qps                   | QPS                                       | None         |
| slb_l7_rt                    | Request delay                             | None         |
| slb_l7_status_2xx             | 2xx request(per second)                   | None         |
| slb_l7_status_3xx             | 3xx request(per second)                   | None         |
| slb_l7_status_4xx             | 4xx request(per second)                   | None         |
| slb_l7_status_5xx             | 5xx request(per second)                   | None         |
| slb_l7_upstream_4xx           | Upstream service 4xx request (per second) | None         |
| slb_l7_upstream_5xx           | Upstream service 5xx request (per second) | None         |
| slb_l7_upstream_rt            | Upstream service rt                       | None         |

#### Demo

```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: kubecon-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1beta2
    kind: Deployment
    name: nginx-deployment-basic
  minReplicas: 5
  maxReplicas: 10
  metrics:
    - type: External
      external:
        metric:
          name: slb_l7_status_2xx
          selector:
            matchLabels:
              slb.instance.id: "lb-2zedu6pk8bryv2z4hnrif"
              slb.instance.port: "80"
        target:
          type: AverageValue
          averageValue: 100m
```



