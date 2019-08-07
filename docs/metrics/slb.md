## SLB External metrics

#### Global Params

all metrics need the global params.

| global params       | description              | example            | required | 
| ------------------- | ------------------------ | ------------------ | -------- | 
| slb.instance.id     | The ID of a SLB instance.| lb-2zelc9ml3tr1cnsir6ep2 | True | 
| slb.instance.port   | The port of SLB instance.| 80                 | True | 

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
apiVersion: apps/v1beta2 # for versions before 1.8.0 use apps/v1beta1
kind: Deployment
metadata:
  name: nginx-deployment-basic
  labels:
    app: nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9 # replace it with your exactly <image_name:tags>
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
    - port: 80
      protocol: TCP
      targetPort: 80
  selector:
    app: nginx
  sessionAffinity: None
  type: LoadBalancer
---
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: slb-hpa
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
          name: slb_l4_active_connection
          selector:
            matchLabels:
              # slb.instance.id: "lb-2ze2locy5fk8at1cfx47y"
              slb.instance.id: ""
              # slb.instance.port: "80"
              slb.instance.port: ""
        target:
          type: Value
          value: 100
```



