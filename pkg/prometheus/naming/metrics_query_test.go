package naming

import (
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/prometheus"
	"testing"

	pmodel "github.com/prometheus/common/model"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
)

type resourceConverterMock struct {
	namespaced bool
}

// ResourcesForSeries is a mock that returns a single group resource,
// namely the series as a resource itself.
func (rcm *resourceConverterMock) ResourcesForSeries(series prometheus.Series) (res []schema.GroupResource, namespaced bool) {
	return []schema.GroupResource{{Resource: series.Name}}, rcm.namespaced
}

// LabelForResource is a mock that returns the label name,
// simply by taking the given resource.
func (rcm *resourceConverterMock) LabelForResource(gr schema.GroupResource) (pmodel.LabelName, error) {
	return pmodel.LabelName(gr.Resource), nil
}

type checkFunc func(prometheus.Selector, error) error

func hasError(want error) checkFunc {
	return func(_ prometheus.Selector, got error) error {
		if want != got {
			return fmt.Errorf("got error %v, want %v", got, want)
		}
		return nil
	}
}

func hasSelector(want string) checkFunc {
	return func(got prometheus.Selector, _ error) error {
		if prometheus.Selector(want) != got {
			return fmt.Errorf("got selector %q, want %q", got, want)
		}
		return nil
	}
}

func checks(cs ...checkFunc) checkFunc {
	return func(s prometheus.Selector, e error) error {
		for _, c := range cs {
			if err := c(s, e); err != nil {
				return err
			}
		}
		return nil
	}
}

func TestBuildSelector(t *testing.T) {
	mustNewQuery := func(queryTemplate string, namespaced bool) MetricsQuery {
		mq, err := NewMetricsQuery(queryTemplate, &resourceConverterMock{namespaced})
		if err != nil {
			t.Fatal(err)
		}
		return mq
	}

	mustNewLabelRequirement := func(key string, op selection.Operator, vals []string) *labels.Requirement {
		req, err := labels.NewRequirement(key, op, vals)
		if err != nil {
			t.Fatal(err)
		}
		return req
	}

	tests := []struct {
		name string
		mq   MetricsQuery

		series         string
		resource       schema.GroupResource
		namespace      string
		extraGroupBy   []string
		metricSelector labels.Selector
		names          []string

		check checkFunc
	}{
		{
			name: "series",

			mq:             mustNewQuery(`series <<.Series>>`, false),
			metricSelector: labels.NewSelector(),
			series:         "foo",

			check: checks(
				hasError(nil),
				hasSelector("series foo"),
			),
		},

		{
			name: "multiple LabelMatchers values",

			mq:             mustNewQuery(`<<.LabelMatchers>>`, false),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},
			names:          []string{"bar", "baz"},

			check: checks(
				hasError(nil),
				hasSelector(`resource=~"bar|baz"`),
			),
		},

		{
			name: "single LabelMatchers value",

			mq:             mustNewQuery(`<<.LabelMatchers>>`, false),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},
			names:          []string{"bar"},

			check: checks(
				hasError(nil),
				hasSelector(`resource="bar"`),
			),
		},

		{
			name: "LabelMatchers with additional metrics filter",

			mq: mustNewQuery(`<<.LabelMatchers>>`, false),
			metricSelector: labels.NewSelector().Add(
				*mustNewLabelRequirement("metric1", selection.Equals, []string{"value1"}),
			),
			resource: schema.GroupResource{Group: "group", Resource: "resource"},
			names:    []string{"bar"},

			check: checks(
				hasError(nil),
				hasSelector(`metric1="value1",resource="bar"`),
			),
		},

		{
			name: "single LabelValuesByName value",

			mq:             mustNewQuery(`<<index .LabelValuesByName "resource">>`, false),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},
			names:          []string{"bar"},

			check: checks(
				hasError(nil),
				hasSelector("bar"),
			),
		},

		{
			name: "multiple LabelValuesByName values",

			mq:             mustNewQuery(`<<index .LabelValuesByName "resource">>`, false),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},
			names:          []string{"bar", "baz"},

			check: checks(
				hasError(nil),
				hasSelector("bar|baz"),
			),
		},

		{
			name: "multiple LabelValuesByName values with namespace",

			mq:             mustNewQuery(`<<index .LabelValuesByName "namespaces">> <<index .LabelValuesByName "resource">>`, true),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},
			namespace:      "default",
			names:          []string{"bar", "baz"},

			check: checks(
				hasError(nil),
				hasSelector("default bar|baz"),
			),
		},

		{
			name: "single GroupBy value",

			mq:             mustNewQuery(`<<.GroupBy>>`, false),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},

			check: checks(
				hasError(nil),
				hasSelector("resource"),
			),
		},

		{
			name: "multiple GroupBy values",

			mq:             mustNewQuery(`<<.GroupBy>>`, false),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},
			extraGroupBy:   []string{"extra", "groups"},

			check: checks(
				hasError(nil),
				hasSelector("resource,extra,groups"),
			),
		},

		{
			name: "single GroupBySlice value",

			mq:             mustNewQuery(`<<.GroupBySlice>>`, false),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},

			check: checks(
				hasError(nil),
				hasSelector("[resource]"),
			),
		},

		{
			name: "multiple GroupBySlice values",

			mq:             mustNewQuery(`<<.GroupBySlice>>`, false),
			metricSelector: labels.NewSelector(),
			resource:       schema.GroupResource{Group: "group", Resource: "resource"},
			extraGroupBy:   []string{"extra", "groups"},

			check: checks(
				hasError(nil),
				hasSelector("[resource extra groups]"),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			selector, err := tc.mq.Build(tc.series, tc.resource, tc.namespace, tc.extraGroupBy, tc.metricSelector, tc.names...)

			if err := tc.check(selector, err); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestBuildExternalSelector(t *testing.T) {
	mustNewQuery := func(queryTemplate string) MetricsQuery {
		mq, err := NewMetricsQuery(queryTemplate, &resourceConverterMock{true})
		if err != nil {
			t.Fatal(err)
		}
		return mq
	}

	mustNewLabelRequirement := func(key string, op selection.Operator, vals []string) *labels.Requirement {
		req, err := labels.NewRequirement(key, op, vals)
		if err != nil {
			t.Fatal(err)
		}
		return req
	}

	tests := []struct {
		name string
		mq   MetricsQuery

		series         string
		namespace      string
		groupBy        string
		groupBySlice   []string
		metricSelector labels.Selector

		check checkFunc
	}{
		{
			name: "series",

			mq:             mustNewQuery(`series <<.Series>>`),
			series:         "foo",
			metricSelector: labels.NewSelector(),

			check: checks(
				hasError(nil),
				hasSelector("series foo"),
			),
		},
		{
			name: "single GroupBy value",

			mq:             mustNewQuery(`<<.GroupBy>>`),
			groupBy:        "foo",
			metricSelector: labels.NewSelector(),

			check: checks(
				hasError(nil),
				hasSelector("foo"),
			),
		},
		{
			name: "multiple GroupBySlice values",

			mq:             mustNewQuery(`<<.GroupBySlice>>`),
			groupBySlice:   []string{"foo", "bar"},
			metricSelector: labels.NewSelector(),

			check: checks(
				hasError(nil),
				hasSelector("[foo bar]"),
			),
		},
		{
			name: "multiple GroupBySlice values with namespace",

			mq:             mustNewQuery(`<<index .LabelValuesByName "namespaces">> <<.GroupBySlice>>`),
			namespace:      "default",
			groupBySlice:   []string{"foo", "bar"},
			metricSelector: labels.NewSelector(),

			check: checks(
				hasError(nil),
				hasSelector("default [foo bar]"),
			),
		},
		{
			name: "single LabelMatchers value",

			mq: mustNewQuery(`<<.LabelMatchers>>`),
			metricSelector: labels.NewSelector().Add(
				*mustNewLabelRequirement("foo", selection.Equals, []string{"bar"}),
			),

			check: checks(
				hasError(nil),
				hasSelector(`foo="bar"`),
			),
		},
		{
			name: "single LabelMatchers value with namespace",

			mq:        mustNewQuery(`<<.LabelMatchers>>`),
			namespace: "default",
			metricSelector: labels.NewSelector().Add(
				*mustNewLabelRequirement("foo", selection.Equals, []string{"bar"}),
			),

			check: checks(
				hasError(nil),
				hasSelector(`foo="bar",namespaces="default"`),
			),
		},
		{
			name: "multiple LabelMatchers value",

			mq: mustNewQuery(`<<.LabelMatchers>>`),
			metricSelector: labels.NewSelector().Add(
				*mustNewLabelRequirement("foo", selection.Equals, []string{"bar"}),
				*mustNewLabelRequirement("qux", selection.In, []string{"bar", "baz"}),
			),

			check: checks(
				hasError(nil),
				hasSelector(`foo="bar",qux=~"bar|baz"`),
			),
		},
		{
			name: "single LabelValuesByName value",

			mq: mustNewQuery(`<<.LabelValuesByName>>`),
			metricSelector: labels.NewSelector().Add(
				*mustNewLabelRequirement("foo", selection.Equals, []string{"bar"}),
			),

			check: checks(
				hasError(nil),
				hasSelector("map[foo:bar]"),
			),
		},
		{
			name: "single LabelValuesByName with multiple selectors",

			mq: mustNewQuery(`<<.LabelValuesByName>>`),
			metricSelector: labels.NewSelector().Add(
				*mustNewLabelRequirement("foo", selection.In, []string{"bar", "baz"}),
			),

			check: checks(
				hasError(nil),
				hasSelector("map[foo:bar|baz]"),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			selector, err := tc.mq.BuildExternal(tc.series, tc.namespace, tc.groupBy, tc.groupBySlice, tc.metricSelector)
			t.Logf("selector: '%v'", selector)

			if err := tc.check(selector, err); err != nil {
				t.Error(err)
			}
		})
	}
}
