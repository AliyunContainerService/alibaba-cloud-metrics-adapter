## AHAS Sentinel External metrics

#### Global Params

All metrics need the global params.

| Global params       | Description              | Example            | Required | Default value |
| ------------------- | ------------------------ | ------------------ | -------- | ------------- | 
| `ahas.sentinel.app.name` | The name of your service in AHAS | sentinel-console | True |  |
| `ahas.sentinel.namespace` | The namespace of your service in AHAS | staging | False | default |
| `ahas.sentinel.stat.period` | The statistic period of request count (in second) | 5 | False | 1 |

#### Metrics List

| metric name                  | description                               | extra params |
| ---------------------------- | ----------------------------------------- | ------------ |
| ahas_sentinel_total_qps             | total QPS                       | None         |
| ahas_sentinel_pass_qps             | passed QPS                       | None         |
| ahas_sentinel_block_qps              | blocked QPS      | None         |
| ahas_sentinel_avg_rt              | average response time              | None         |

