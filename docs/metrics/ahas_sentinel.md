## AHAS Sentinel External metrics

#### Global Params

All metrics need the global params.

| Global params       | Description              | Example            | Required | Default value |
| ------------------- | ------------------------ | ------------------ | -------- | ------------- | 
| `ahas.sentinel.app` | The name of your service in AHAS | sentinel-console | True |  |
| `ahas.sentinel.namespace` | The namespace of your service in AHAS | staging | False | default |
| `ahas.sentinel.interval` | The query interval of request count (in second) | 5 | False | 10 |

Note that the `ahas.sentinel.app` is required, which should match the `project.name` property configured in AHAS Sentinel.

#### Metrics List

| metric name                  | description                               | extra params |
| ---------------------------- | ----------------------------------------- | ------------ |
| ahas_sentinel_total_qps             | total QPS                       | None         |
| ahas_sentinel_pass_qps             | passed QPS                       | None         |
| ahas_sentinel_block_qps              | blocked QPS (i.e. rejected by Sentinel)      | None         |
| ahas_sentinel_avg_rt              | average response time              | None         |

#### Example

To make the HPA enabled, please also [install the AHAS Sentinel pilot helm chart in ACK console](https://cs.console.aliyun.com/#/k8s/catalog/detail/incubator_ack-ahas-sentinel-pilot).

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: foo-service-cm-pilot
data:
  application.yaml: |
    spring:
      application:
        name: foo-service
    server:
      port: 8700
    eureka:
      instance:
        preferIpAddress: true
      client:
        enabled: false
    logging:
      file: /foo-service/logs/application.log
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-foo-on-pilot
  labels:
    name: agent-foo-on-pilot
spec:
  replicas: 1
  selector:
    matchLabels:
      name: agent-foo-on-pilot
  template:
    metadata:
      labels:
        name: agent-foo-on-pilot
      annotations:
        ahasPilotAutoEnable: "on"
        ahasAppName: "foo-service-on-pilot"
        ahasNamespace: "default"
    spec:
      containers:
        - name: master
          image: registry.cn-hangzhou.aliyuncs.com/sentinel-docker-repo/foo-service:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 8700
          volumeMounts:
            - name: foo-service-logs
              mountPath: /foo-service/logs
            - name: foo-service-config
              mountPath: /foo-service/config
          resources:
            limits:
              cpu: "0.5"
              memory: 500Mi
            requests:
              cpu: "0.5"
              memory: 500Mi
          env:
            - name: TEST_ENV
              value: "hello world"
      volumes:
        - name: foo-service-logs
          emptyDir: {}
        - name: foo-service-config
          configMap:
            name: foo-service-cm-pilot
            items:
              - key: application.yaml
                path: application.yml
---
apiVersion: v1
kind: Service
metadata:
  name: foo-service
  labels:
    name: foo-service
spec:
  ports:
    - port: 80
      targetPort: 8700
  selector:
    name: agent-foo-on-pilot
  type: LoadBalancer
  externalTrafficPolicy: Local
---
# To make the HPA enabled, we need to also install the AHAS Sentinel pilot helm chart in ACK console.
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: ahas-sentinel-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1beta2
    kind: Deployment
    name: agent-foo-on-pilot
  minReplicas: 1
  maxReplicas: 3
  metrics:
    - type: External
      external:
        metric:
          name: ahas_sentinel_total_qps
          selector:
            matchLabels:
            # If you're using AHAS Sentinel pilot, then the appName and namespace
            # can be retrieved from the annotation of target Deployment automatically.
            # ahas.sentinel.app: "foo-service-on-pilot"
            # ahas.sentinel.namespace: "default"
        target:
          type: Value
          # ahas_sentinel_total_qps > 30
          value: 30
```