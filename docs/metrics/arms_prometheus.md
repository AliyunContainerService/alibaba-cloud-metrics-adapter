## Walkthrough

### Install
First,you need fill your arms-prometheus's api in deploy.yaml,then
```shell script
kubectl apply -f deploy/deploy.yaml 
``` 
### Verify 
```shell script
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1" 
```
feedback look like this:
```json
{
  "kind": "APIResourceList",
  "apiVersion": "v1",
  "groupVersion": "custom.metrics.k8s.io/v1beta1",
  "resources": [
    {
      "name": "pods/http_requests_total",
      "singularName": "",
      "namespaced": true,
      "kind": "MetricValueList",
      "verbs": ["get"]
    },
    {
      "name": "namespaces/http_requests_total",
      "singularName": "",
      "namespaced": false,
      "kind": "MetricValueList",
      "verbs": ["get"]
    }
  ]
}
```
### Usage
1.deploy the Application
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-app
  labels:
    app: sample-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sample-app
  template:
    metadata:
      labels:
        app: sample-app
    spec:
      containers:
      - image: luxas/autoscale-demo:v0.1.2
        name: metrics-provider
        ports:
        - name: http
          containerPort: 8080
```
```shell script
$ kubectl create -f sample-app.deploy.yaml
$ kubectl create service clusterip sample-app --tcp=80:8080
```
2.Add ServiceMonitor in arms-prometheus
```helmyaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  #  填写一个唯一名称
  name: sample-app
  #  填写目标命名空间
  namespace: default
spec:
  endpoints:
  - interval: 30s
    #  填写service.yaml中Prometheus Exporter对应的Port的Name字段的值
    port: 80-8080
    #  填写Prometheus Exporter对应的Path的值
    path: /metrics
  namespaceSelector:
    any: true
    #  Nginx Demo的命名空间
  selector:
    matchLabels:
      #  填写service.yaml的Label字段的值以定位目标service.yaml
      app: sample-app
```
3.modify configuration
```yaml
rules:
- seriesQuery: 'http_requests_total{kubernetes_namespace!="",kubernetes_pod_name!=""}'
  resources:
    overrides:
      kubernetes_namespace: {resource: "namespace"}
      kubernetes_pod_name: {resource: "pod"}
  name:
    matches: "^(.*)_total"
    as: "${1}_per_second"
  metricsQuery: 'sum(rate(<<.Series>>{<<.LabelMatchers>>}[2m])) by (<<.GroupBy>>)'
```
4.deploy the HPA
```yaml
kind: HorizontalPodAutoscaler
apiVersion: autoscaling/v2beta1
metadata:
  name: sample-app
spec:
  scaleTargetRef:
    # point the HPA at the sample application
    # you created above
    apiVersion: apps/v1
    kind: Deployment
    name: sample-app
  # autoscale between 1 and 10 replicas
  minReplicas: 1
  maxReplicas: 10
  metrics:
  # use a "Pods" metric, which takes the average of the
  # given metric across all pods controlled by the autoscaling target
  - type: Pods
    pods:
      # use the metric that you used above: pods/http_requests
      metricName: http_requests_per_second
      # target 500 milli-requests per second,
      # which is 1 request every two seconds
      targetAverageValue: 50m
```
5.Stress test 
```shell script
$ ab -c 50 -n 2000 ClusterIP(sample-app):80
$ kubectl get hpa sample-app
```
### Complete configuration
```yaml
##config DEMO
rules:
# this rule matches cumulative cAdvisor metrics measured in seconds
- seriesQuery: '{__name__=~"^container_.*",container_name!="POD",namespace!="",pod_name!=""}'
  resources:
    # skip specifying generic resource<->label mappings, and just
    # attach only pod and namespace resources by mapping label names to group-resources
    overrides:
      namespace: {resource: "namespace"},
      pod_name: {resource: "pod"},
  # specify that the `container_` and `_seconds_total` suffixes should be removed.
  # this also introduces an implicit filter on metric family names
  name: 
    # we use the value of the capture group implicitly as the API name
    # we could also explicitly write `as: "$1"`
    matches: "^container_(.*)_seconds_total$"
  # specify how to construct a query to fetch samples for a given series
  # This is a Go template where the `.Series` and `.LabelMatchers` string values
  # are available, and the delimiters are `<<` and `>>` to avoid conflicts with
  # the prom query language
  metricsQuery: "sum(rate(<<.Series>>{<<.LabelMatchers>>,container_name!="POD"}[2m])) by (<<.GroupBy>>)"

# this rule matches cumulative cAdvisor metrics not measured in seconds
- seriesQuery: '{__name__=~"^container_.*_total",container_name!="POD",namespace!="",pod_name!=""}'
  resources:
    overrides:
      namespace: {resource: "namespace"},
      pod_name: {resource: "pod"},
  seriesFilters:
  # since this is a superset of the query above, we introduce an additional filter here
  - isNot: "^container_.*_seconds_total$"
  name: {matches: "^container_(.*)_total$"}
  metricsQuery: "sum(rate(<<.Series>>{<<.LabelMatchers>>,container_name!="POD"}[2m])) by (<<.GroupBy>>)"

# this rule matches cumulative non-cAdvisor metrics
- seriesQuery: '{namespace!="",__name__!="^container_.*"}'
  name: {matches: "^(.*)_total$"}
  resources:
    # specify an a generic mapping between resources and labels.  This
    # is a template, like the `metricsQuery` template, except with the `.Group`
    # and `.Resource` strings available.  It will also be used to match labels,
    # so avoid using template functions which truncate the group or resource.
    # Group will be converted to a form acceptible for use as a label automatically.
    template: "<<.Resource>>"
    # if we wanted to, we could also specify overrides here
  metricsQuery: "sum(rate(<<.Series>>{<<.LabelMatchers>>,container_name!="POD"}[2m])) by (<<.GroupBy>>)"

# this rule matches only a single metric, explicitly naming it something else
# It's series query *must* return only a single metric family
- seriesQuery: 'cheddar{sharp="true"}'
  # this metric will appear as "cheesy_goodness" in the custom metrics API
  name: {as: "cheesy_goodness"}
  resources:
    overrides:
      # this should still resolve in our cluster
      brand: {group: "cheese.io", resource: "brand"}
  metricQuery: 'count(cheddar{sharp="true"})'
```
Each rule can be broken down into roughly four parts:

- *Metrics Discovery*, which specifies how the adapter should find all Prometheus metrics for this rule.

- *Bound resource*, which specifies how the adapter should determine which Kubernetes resources a particular metric is associated with.

- *ReName*, which specifies how the adapter should expose the metric in the custom metrics API.

- *Querying*, which specifies how a request for a particular metric on one or more Kubernetes objects should be turned into a query to Prometheus.

### Metrics Discovery
Metrics Discovery governs the process of finding the metrics that you want to expose in the custom metrics API. There are two fields that factor into discovery: seriesQuery and seriesFilters.

seriesQuery specifies Prometheus series query (as passed to the /api/v1/series endpoint in Prometheus) to use to find some set of Prometheus series. The adapter will strip the label values from this series, and then use the resulting metric-name-label-names combinations later on.

In many cases, seriesQuery will be sufficient to narrow down the list of Prometheus series. However, sometimes (especially if two rules might otherwise overlap), it's useful to do additional filtering on metric names. In this case, seriesFilters can be used. After the list of series is returned from seriesQuery, each series has its metric name filtered through any specified filters.

Filters may be either:
- `is: <regex>`, which matches any series whose name matches the specified
  regex.

- `isNot: <regex>`, which matches any series whose name does not match the
  specified regex.
For example:
```yaml
# match all cAdvisor metrics that aren't measured in seconds
seriesQuery: '{__name__=~"^container_.*_total",container_name!="POD",namespace!="",pod_name!=""}'
seriesFilters:
  - isNot: "^container_.*_seconds_total"
```  
### Bound resource
Bound resource governs the process of figuring out which Kubernetes resources a particular metric could be attached to. The resources field controls this process.

There are two ways to associate resources with a particular metric. In both cases, the value of the label becomes the name of the particular object.

One way is to specify that any label name that matches some particular pattern refers to some group-resource based on the label name. This can be done using the template field. The pattern is specified as a Go template, with the Group and Resource fields representing group and resource. You don't necessarily have to use the Group field (in which case the group is guessed by the system). For instance:
```yaml
# any label `kube_<group>_<resource>` becomes <group>.<resource> in Kubernetes
resources:
  template: "kube_<<.Group>>_<<.Resource>>"
```
The other way is to specify that some particular label represents some particular Kubernetes resource. This can be done using the overrides field. Each override maps a Prometheus label to a Kubernetes group-resource. For instance:
```yaml
# the microservice label corresponds to the apps.deployment resource
resources:
  overrides:
    microservice: {group: "apps", resource: "deployment"}
```
These two can be combined, so you can specify both a template and some individual overrides.

The resources mentioned can be any resource available in your kubernetes cluster, as long as you've got a corresponding label.
### ReName
ReName governs the process of converting a Prometheus metric name into a metric in the custom metrics API, and vice versa. It's controlled by the name field.

Naming is controlled by specifying a pattern to extract an API name from a Prometheus name, and potentially a transformation on that extracted value.

The pattern is specified in the matches field, and is just a regular expression. If not specified, it defaults to .*.

The transformation is specified by the as field. You can use any capture groups defined in the matches field. If the matches field doesn't contain capture groups, the as field defaults to $0. If it contains a single capture group, the as field defautls to $1. Otherwise, it's an error not to specify the as field.

For example:
```yaml
# match turn any name <name>_total to <name>_per_second
# e.g. http_requests_total becomes http_requests_per_second
name:
  matches: "^(.*)_total$"
  as: "${1}_per_second"
```
### Querying
Querying governs the process of actually fetching values for a particular metric. It's controlled by the metricsQuery field.

The metricsQuery field is a Go template that gets turned into a Prometheus query, using input from a particular call to the custom metrics API. A given call to the custom metrics API is distilled down to a metric name, a group-resource, and one or more objects of that group-resource. These get turned into the following fields in the template:
- `Series`: the metric name
- `LabelMatchers`: a comma-separated list of label matchers matching the
  given objects.  Currently, this is the label for the particular
  group-resource, plus the label for namespace, if the group-resource is
  namespaced.
- `GroupBy`: a comma-separated list of labels to group by.  Currently,
  this contains the group-resource label used in `LabelMatchers`.
For instance, suppose we had a series http_requests_total (exposed as http_requests_per_second in the API) with labels service, pod, ingress, namespace, and verb. The first four correspond to Kubernetes resources. Then, if someone requested the metric pods/http_request_per_second for the pods pod1 and pod2 in the somens namespace, we'd have:
- `Series: "http_requests_total"`
- `LabelMatchers: "pod=~\"pod1|pod2",namespace="somens"`
- `GroupBy`: `pod`
Additionally, there are two advanced fields that are "raw" forms of other
fields:

- `LabelValuesByName`: a map mapping the labels and values from the
  `LabelMatchers` field.  The values are pre-joined by `|`
  (for used with the `=~` matcher in Prometheus).
- `GroupBySlice`: the slice form of `GroupBy`.

In general, you'll probably want to use the `Series`, `LabelMatchers`, and
`GroupBy` fields.  The other two are for advanced usage.

The query is expected to return one value for each object requested.  The
adapter will use the labels on the returned series to associate a given
series back to its corresponding object.

For example:

```yaml
# convert cumulative cAdvisor metrics into rates calculated over 2 minutes
metricsQuery: "sum(rate(<<.Series>>{<<.LabelMatchers>>,container_name!="POD"}[2m])) by (<<.GroupBy>>)"
```