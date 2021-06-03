package prometheus

import (
	"context"
	"fmt"
	pmodel "github.com/prometheus/common/model"
)

// FakePrometheusClient is a fake instance of prom.Client
type FakePrometheusClient struct {
	// AcceptableInterval is the interval in which to return queries
	AcceptableInterval pmodel.Interval
	// ErrQueries are queries that result in an error (whether from Query or Series)
	ErrQueries map[Selector]error
	// Series are non-error responses to partial Series calls
	SeriesResults map[Selector][]Series
	// QueryResults are non-error responses to Query
	QueryResults map[Selector]QueryResult
}

func (c *FakePrometheusClient) Series(_ context.Context, interval pmodel.Interval, selectors ...Selector) ([]Series, error) {
	if (interval.Start != 0 && interval.Start < c.AcceptableInterval.Start) || (interval.End != 0 && interval.End > c.AcceptableInterval.End) {
		return nil, fmt.Errorf("interval [%v, %v] for query is outside range [%v, %v]", interval.Start, interval.End, c.AcceptableInterval.Start, c.AcceptableInterval.End)
	}
	res := []Series{}
	for _, sel := range selectors {
		if err, found := c.ErrQueries[sel]; found {
			return nil, err
		}
		if series, found := c.SeriesResults[sel]; found {
			res = append(res, series...)
		}
	}

	return res, nil
}

func (c *FakePrometheusClient) Query(_ context.Context, t pmodel.Time, query Selector) (QueryResult, error) {
	if t < c.AcceptableInterval.Start || t > c.AcceptableInterval.End {
		return QueryResult{}, fmt.Errorf("time %v for query is outside range [%v, %v]", t, c.AcceptableInterval.Start, c.AcceptableInterval.End)
	}

	if err, found := c.ErrQueries[query]; found {
		return QueryResult{}, err
	}

	if res, found := c.QueryResults[query]; found {
		return res, nil
	}

	return QueryResult{
		Type:   pmodel.ValVector,
		Vector: &pmodel.Vector{},
	}, nil
}

func (c *FakePrometheusClient) QueryRange(_ context.Context, r Range, query Selector) (QueryResult, error) {
	return QueryResult{}, nil
}
