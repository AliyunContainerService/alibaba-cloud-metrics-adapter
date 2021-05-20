package prometheus

import (
	"context"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/utils"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// queryLatency is the total latency of any query going through the
	// various endpoints (query, range-query, series).  It includes some deserialization
	// overhead and HTTP overhead.
	queryLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cmgateway_prometheus_query_latency_seconds",
			Help:    "Prometheus client query latency in seconds.  Broken down by target prom endpoint and target server",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 10),
		},
		[]string{"endpoint", "server"},
	)
)

func init() {
	prometheus.MustRegister(queryLatency)
}

// instrumentedClient is a client.GenericAPIClient which instruments calls to Do,
// capturing request latency.
type instrumentedGenericClient struct {
	serverName string
	client     GenericAPIClient
}

func (c *instrumentedGenericClient) Do(ctx context.Context, verb, endpoint string, query string) (utils.APIResponse, error) {
	startTime := time.Now()
	var err error
	defer func() {
		endTime := time.Now()
		// skip calls where we don't make the actual request
		if err != nil {
			if _, wasAPIErr := err.(*utils.Error); !wasAPIErr {
				// TODO: measure API errors by code?
				return
			}
		}
		queryLatency.With(prometheus.Labels{"endpoint": endpoint, "server": c.serverName}).Observe(endTime.Sub(startTime).Seconds())
	}()

	var resp utils.APIResponse
	resp, err = c.client.Do(ctx, verb, endpoint, query)
	return resp, err
}

func InstrumentGenericAPIClient(client GenericAPIClient, serverName string) GenericAPIClient {
	return &instrumentedGenericClient{
		serverName: serverName,
		client:     client,
	}
}
