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

