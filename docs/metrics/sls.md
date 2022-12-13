## SLS External metrics 

#### Global Params 

all metrics need the global params.

If your ingress not used session sticky, please refer to the configuration below.

| global params       | description              | example            | required | 
| ------------------- | ------------------------ | ------------------ | -------- | 
| sls.project         | The project name of a SLS instance. | k8s-log-c550367cdf1e84dfabab013b277cc6bc2" | True | 
| sls.logstore        | The specific logStore of a SLS project. | nginx-ingress  | True | 
| sls.ingress.route   | route of ingress(namespace-svc-port)| default-kubecon-springboot-demo-6666 | True | 
 
If your ingress used session sticky, please refer to the configuration below.

| global params       | description              | example            | required | 
| ------------------- | ------------------------ | ------------------ | -------- | 
| sls.project         | The project name of a SLS instance. | k8s-log-c550367cdf1e84dfabab013b277cc6bc2" | True | 
| sls.logstore        | The specific logStore of a SLS project. | nginx-ingress  | True | 
| sls.ingress.route   | route of ingress(sticky-namespace-servicename-containerport)| sticky-default-kubecon-springboot-demo-6666 | True | 

#### Metrics List 

| metric name     | description                     | extra params |     
| --------------- | ------------------------------- | ------------ |
| sls_alb_ingress_qps | QPS of a alb ingress route |  alb.ingress.route | 
| sls_ingress_qps | QPS of a specific ingress route |  sls.ingress.route | 
| sls_ingress_latency_avg | latency of all requests |  sls.ingress.route        | 
| sls_ingress_latency_p50 | latency of 50% requests|  sls.ingress.route        | 
| sls_ingress_latency_p95 | latency of 95% requests |  sls.ingress.route        | 
| sls_ingress_latency_p99 | latency of 99% requests |  sls.ingress.route        | 
| sls_ingress_latency_p9999 | latency of 99.99% requests |  sls.ingress.route        | 
| sls_ingress_inflow | inflow bandwidth of ingress |  sls.ingress.route        | 

#### Example1(Not use session sticky)
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
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: nginx
  namespace: default
spec:
  rules:
    - host: nginx.c550367cdf1e84dfabab013b277cc6bc2.cn-beijing.alicontainer.com
      http:
        paths:
          - backend:
              serviceName: nginx
              servicePort: 80
            path: /
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  namespace: default
spec:
  ports:
    - port: 80
      protocol: TCP
      targetPort: 80
  selector:
    app: nginx
  type: ClusterIP
---
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: ingress-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1beta2
    kind: Deployment
    name: nginx-deployment-basic
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: External
      external:
        metric:
          name: sls_ingress_qps
          selector:
            matchLabels:
              #sls.project: "k8s-log-c550367cdf1e84dfabab013b277cc6bc2"
              sls.project: ""
              #sls.logstore: "nginx-ingress"
              sls.logstore: ""
              #sls.ingress.route: "default-nginx-80"
              sls.ingress.route: ""
        target:
          type: AverageValue
          averageValue: 10
    - type: External
      external:
        metric:
          name: sls_ingress_latency_p9999
          selector:
            matchLabels:
              # default ingress log project is k8s-log-clusterId
              # sls.project: "k8s-log-c550367cdf1e84dfabab013b277cc6bc2"
              sls.project: ""
              # default ingress logstre is nginx-ingress
              # sls.logstore: "nginx-ingress"
              sls.logstore: ""
              # namespace-svc-port
              # sls.ingress.route: "default-nginx-80"
              sls.ingress.route: ""
        target:
          type: Value
          # sls_ingress_latency_p9999 > 10ms
          value: 10

```
#### Example2(Use session sticky)
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
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: nginx
  namespace: default
  annotations:
      ingress.kubernetes.io/affinity: "cookie"
      ingress.kubernetes.io/session-cookie-name: "route"
      ingress.kubernetes.io/session-cookie-hash: "sha1"
spec:
  rules:
    - host: nginx.c550367cdf1e84dfabab013b277cc6bc2.cn-beijing.alicontainer.com
      http:
        paths:
          - backend:
              serviceName: nginx
              servicePort: 80
            path: /
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  namespace: default
spec:
  ports:
    - port: 80
      protocol: TCP
      targetPort: 80
  selector:
    app: nginx
  type: ClusterIP
  sessionAffinity: ClientIP
---
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: ingress-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1beta2
    kind: Deployment
    name: nginx-deployment-basic
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: External
      external:
        metric:
          name: sls_ingress_qps
          selector:
            matchLabels:
              #sls.project: "k8s-log-c550367cdf1e84dfabab013b277cc6bc2"
              sls.project: ""
              #sls.logstore: "nginx-ingress"
              sls.logstore: ""
              #sls.ingress.route: "default-nginx-80"
              sls.ingress.route: ""
        target:
          type: AverageValue
          averageValue: 10
    - type: External
      external:
        metric:
          name: sls_ingress_latency_p9999
          selector:
            matchLabels:
              # default ingress log project is k8s-log-clusterId
              # sls.project: "k8s-log-c550367cdf1e84dfabab013b277cc6bc2"
              sls.project: ""
              # default ingress logstre is nginx-ingress
              # sls.logstore: "nginx-ingress"
              sls.logstore: ""
              # sticky-namespace-servicename-containerport  
              # sls.ingress.route: "sticky-default-nginx-80"
              sls.ingress.route: ""
        target:
          type: Value
          # sls_ingress_latency_p9999 > 10ms
          value: 10
```


