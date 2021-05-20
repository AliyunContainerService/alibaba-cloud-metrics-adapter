```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: prometheus-hpa
  annotations:
    "prometheus.query": "rate(http_requests_total[1m])"
    "prometheus.metric.name": "http_requests_per_second"
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment-basic
  minReplicas: 1
  maxReplicas: 10
  metrics:
    - type: External
      external:
        metric:
          name: http_requests_per_second
        target:
          type: Value
          value: 5
```
      
