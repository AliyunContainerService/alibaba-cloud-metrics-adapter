package prometheus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/utils"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/prometheus/common/model"
	"k8s.io/klog"
)

// APIClient is a raw client to the Prometheus Query API.
// It knows how to appropriately deal with generic Prometheus API
// responses, but does not know the specifics of different endpoints.
// You can use this to call query endpoints not represented in Client.
type GenericAPIClient interface {
	// Do makes a request to the Prometheus HTTP API against a particular endpoint.  Query
	// parameters should be in `query`, not `endpoint`.  An error will be returned on HTTP
	// status errors or errors making or unmarshalling the request, as well as when the
	// response has a Status of ResponseError.
	Do(ctx context.Context, verb, endpoint string, query string) (utils.APIResponse, error)
}

// httpAPIClient is a GenericAPIClient implemented in terms of an underlying http.Client.
type httpAPIClient struct {
	client  *http.Client
	baseURL *url.URL
}

func (c *httpAPIClient) Do(ctx context.Context, verb, endpoint string, query string) (utils.APIResponse, error) {
	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, endpoint)
	u.RawQuery = strings.Replace(query, " ", "", -1)
	req, err := http.NewRequest(verb, u.String(), nil)
	if err != nil {
		return utils.APIResponse{}, fmt.Errorf("error constructing HTTP request to Prometheus: %v", err)
	}
	req.WithContext(ctx)

	resp, err := c.client.Do(req)
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	if err != nil {
		return utils.APIResponse{}, err
	}

	if klog.V(6) {
		klog.Infof("%s %s %s", verb, u.String(), resp.Status)
	}

	code := resp.StatusCode

	// codes that aren't 2xx, 400, 422, or 503 won't return JSON objects
	if code/100 != 2 && code != 400 && code != 422 && code != 503 {
		return utils.APIResponse{}, &utils.Error{
			Type: utils.ErrBadResponse,
			Msg:  fmt.Sprintf("unknown response code %d", code),
		}
	}

	var body io.Reader = resp.Body
	if klog.V(8) {
		data, err := ioutil.ReadAll(body)
		if err != nil {
			return utils.APIResponse{}, fmt.Errorf("unable to log response body: %v", err)
		}
		klog.Infof("Response Body: %s", string(data))
		body = bytes.NewReader(data)
	}

	var res utils.APIResponse
	if err = json.NewDecoder(body).Decode(&res); err != nil {
		return utils.APIResponse{}, &utils.Error{
			Type: utils.ErrBadResponse,
			Msg:  err.Error(),
		}
	}

	if res.Status == utils.ResponseError {
		return res, &utils.Error{
			Type: res.ErrorType,
			Msg:  res.Error,
		}
	}

	return res, nil
}

// NewGenericAPIClient builds a new generic Prometheus API client for the given base URL and HTTP Client.
func NewGenericAPIClient(client *http.Client, baseURL *url.URL) GenericAPIClient {
	return &httpAPIClient{
		client:  client,
		baseURL: baseURL,
	}
}

const (
	queryURL      = "/api/v1/query"
	queryRangeURL = "/api/v1/query_range"
	seriesURL     = "/api/v1/series"
)

// queryClient is a Client that connects to the Prometheus HTTP API.
type queryClient struct {
	api GenericAPIClient
}

// NewClientForAPI creates a Client for the given generic Prometheus API client.
func NewClientForAPI(client GenericAPIClient) Client {
	return &queryClient{
		api: client,
	}
}

// NewClient creates a Client for the given HTTP client and base URL (the location of the Prometheus server).
func NewClient(client *http.Client, baseURL *url.URL) Client {
	genericClient := NewGenericAPIClient(client, baseURL)
	return NewClientForAPI(genericClient)
}

func (h *queryClient) Series(ctx context.Context, interval model.Interval, selectors ...Selector) ([]Series, error) {
	var expr string

	if interval.Start != 0 {
		expr += fmt.Sprintf("&start=%s", interval.Start.String())
	}
	expr += fmt.Sprintf("&end=%s", time.Now().Add(-8*time.Hour).Format(utils.PROM_FORMAT))

	for _, selector := range selectors {
		expr += fmt.Sprintf("&match[]=%s", string(selector))
	}

	res, err := h.api.Do(ctx, "GET", seriesURL, expr)

	if err != nil {
		return nil, err
	}

	var seriesRes []Series
	err = json.Unmarshal(res.Data, &seriesRes)
	return seriesRes, err
}

func (h *queryClient) Query(ctx context.Context, t model.Time, query Selector) (QueryResult, error) {
	var expr string
	expr += fmt.Sprintf("&query=%s", string(query))

	if t != 0 {
		expr += fmt.Sprintf("&time=%s", t.String())
	}
	if timeout, hasTimeout := timeoutFromContext(ctx); hasTimeout {
		expr += fmt.Sprintf("&timeout=%s", model.Duration(timeout).String())
	}

	res, err := h.api.Do(ctx, "GET", queryURL, expr)
	if err != nil {
		return QueryResult{}, err
	}

	var queryRes QueryResult
	err = json.Unmarshal(res.Data, &queryRes)
	return queryRes, err
}

func (h *queryClient) QueryRange(ctx context.Context, r Range, query Selector) (QueryResult, error) {
	var expr string
	expr += fmt.Sprintf("&query=%s", string(query))

	if r.Start != 0 {
		expr += fmt.Sprintf("&start=%s", r.Start.String())
	}
	if r.End != 0 {
		expr += fmt.Sprintf("&end=%s", r.End.String())
	}
	if r.Step != 0 {
		expr += fmt.Sprintf("&step=%s", model.Duration(r.Step).String())
	}
	if timeout, hasTimeout := timeoutFromContext(ctx); hasTimeout {
		expr += fmt.Sprintf("&timeout=%s", model.Duration(timeout).String())
	}

	res, err := h.api.Do(ctx, "GET", queryRangeURL, expr)

	if err != nil {
		return QueryResult{}, err
	}

	var queryRes QueryResult
	err = json.Unmarshal(res.Data, &queryRes)
	return queryRes, err
}

// timeoutFromContext checks the context for a deadline and calculates a "timeout" duration from it,
// when present
func timeoutFromContext(ctx context.Context) (time.Duration, bool) {
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		return time.Now().Sub(deadline), true
	}

	return time.Duration(0), false
}
