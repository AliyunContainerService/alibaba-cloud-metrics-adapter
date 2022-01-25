package utils

import (
	"context"
	"net/url"
	"time"
	prom "sigs.k8s.io/prometheus-adapter/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// queryLatency is the total latency of any query going through the
	// various endpoints (query, range-query, series).  It includes some deserialization
	// overhead and HTTP overhead.
	queryLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cmgateway_prometheus_query_latency_seconds",
			Help:    "Prometheus client query latency in seconds.  Broken down by target prometheus endpoint and target server",
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
	client     prom.GenericAPIClient
}

func (c *instrumentedGenericClient) Do(ctx context.Context, verb, endpoint string, query url.Values) (prom.APIResponse, error) {
	startTime := time.Now()
	var err error
	defer func() {
		endTime := time.Now()
		// skip calls where we don't make the actual request
		if err != nil {
			if _, wasAPIErr := err.(*Error); !wasAPIErr {
				// TODO: measure API errors by code?
				return
			}
		}
		queryLatency.With(prometheus.Labels{"endpoint": endpoint, "server": c.serverName}).Observe(endTime.Sub(startTime).Seconds())
	}()

	resp, err := c.client.Do(ctx, verb, endpoint, query)
	return resp, err
}

func InstrumentGenericAPIClient(client prom.GenericAPIClient, serverName string) prom.GenericAPIClient {
	return &instrumentedGenericClient{
		serverName: serverName,
		client:     client,
	}
}
