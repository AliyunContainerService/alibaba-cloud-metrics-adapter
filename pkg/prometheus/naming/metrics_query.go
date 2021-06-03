package naming

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/pkg/prometheus"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
)

// MetricsQuery represents a compiled metrics query for some set of
// series that can be converted into an series of Prometheus expressions to
// be passed to a client.
type MetricsQuery interface {
	// Build constructs Prometheus expressions to represent this query
	// over the given group-resource.  If namespace is empty, the resource
	// is considered to be root-scoped.  extraGroupBy may be used for cases
	// where we need to scope down more specifically than just the group-resource
	// (e.g. container metrics).
	Build(series string, groupRes schema.GroupResource, namespace string, extraGroupBy []string, metricSelector labels.Selector, resourceNames ...string) (prometheus.Selector, error)
	BuildExternal(seriesName string, namespace string, groupBy string, groupBySlice []string, metricSelector labels.Selector) (prometheus.Selector, error)
}

// NewMetricsQuery constructs a new MetricsQuery by compiling the given Go template.
// The delimiters on the template are `<<` and `>>`, and it may use the following fields:
// - Series: the series in question
// - LabelMatchers: a pre-stringified form of the label matchers for the resources in the query
// - LabelMatchersByName: the raw map-form of the above matchers
// - GroupBy: the group-by clause to use for the resources in the query (stringified)
// - GroupBySlice: the raw slice form of the above group-by clause
func NewMetricsQuery(queryTemplate string, resourceConverter ResourceConverter) (MetricsQuery, error) {
	templ, err := template.New("metrics-query").Delims("<<", ">>").Parse(queryTemplate)
	if err != nil {
		return nil, fmt.Errorf("unable to parse metrics query template %q: %v", queryTemplate, err)
	}

	return &metricsQuery{
		resConverter: resourceConverter,
		template:     templ,
	}, nil
}

// metricsQuery is a MetricsQuery based on a compiled Go text template.
// with the delimiters as `<<` and `>>`, and the arguments found in
// queryTemplateArgs.
type metricsQuery struct {
	resConverter ResourceConverter
	template     *template.Template
}

// queryTemplateArgs contains the arguments for the template used in metricsQuery.
type queryTemplateArgs struct {
	Series            string
	LabelMatchers     string
	LabelValuesByName map[string]string
	GroupBy           string
	GroupBySlice      []string
}

type queryPart struct {
	labelName string
	values    []string
	operator  selection.Operator
}

func (q *metricsQuery) Build(series string, resource schema.GroupResource, namespace string, extraGroupBy []string, metricSelector labels.Selector, names ...string) (prometheus.Selector, error) {
	queryParts := q.createQueryPartsFromSelector(metricSelector)

	if namespace != "" {
		namespaceLbl, err := q.resConverter.LabelForResource(NsGroupResource)
		if err != nil {
			return "", err
		}

		queryParts = append(queryParts, queryPart{
			labelName: string(namespaceLbl),
			values:    []string{namespace},
			operator:  selection.Equals,
		})
	}

	exprs, valuesByName, err := q.processQueryParts(queryParts)
	if err != nil {
		return "", err
	}

	resourceLbl, err := q.resConverter.LabelForResource(resource)
	if err != nil {
		return "", err
	}

	matcher := prometheus.LabelEq
	targetValue := strings.Join(names, "|")

	if len(names) > 1 {
		matcher = prometheus.LabelMatches
	}

	exprs = append(exprs, matcher(string(resourceLbl), targetValue))
	valuesByName[string(resourceLbl)] = targetValue

	groupBy := make([]string, 0, len(extraGroupBy)+1)
	groupBy = append(groupBy, string(resourceLbl))
	groupBy = append(groupBy, extraGroupBy...)

	args := queryTemplateArgs{
		Series:            series,
		LabelMatchers:     strings.Join(exprs, ","),
		LabelValuesByName: valuesByName,
		GroupBy:           strings.Join(groupBy, ","),
		GroupBySlice:      groupBy,
	}
	queryBuff := new(bytes.Buffer)
	if err := q.template.Execute(queryBuff, args); err != nil {
		return "", err
	}

	if queryBuff.Len() == 0 {
		return "", fmt.Errorf("empty query produced by metrics query template")
	}

	return prometheus.Selector(queryBuff.String()), nil
}

func (q *metricsQuery) BuildExternal(seriesName string, namespace string, groupBy string, groupBySlice []string, metricSelector labels.Selector) (prometheus.Selector, error) {
	queryParts := []queryPart{}

	// Build up the query parts from the selector.
	queryParts = append(queryParts, q.createQueryPartsFromSelector(metricSelector)...)

	if namespace != "" {
		namespaceLbl, err := q.resConverter.LabelForResource(NsGroupResource)
		if err != nil {
			return "", err
		}

		queryParts = append(queryParts, queryPart{
			labelName: string(namespaceLbl),
			values:    []string{namespace},
			operator:  selection.Equals,
		})
	}

	// Convert our query parts into the types we need for our template.
	exprs, valuesByName, err := q.processQueryParts(queryParts)

	if err != nil {
		return "", err
	}

	args := queryTemplateArgs{
		Series:            seriesName,
		LabelMatchers:     strings.Join(exprs, ","),
		LabelValuesByName: valuesByName,
		GroupBy:           groupBy,
		GroupBySlice:      groupBySlice,
	}

	queryBuff := new(bytes.Buffer)
	if err := q.template.Execute(queryBuff, args); err != nil {
		return "", err
	}

	if queryBuff.Len() == 0 {
		return "", fmt.Errorf("empty query produced by metrics query template")
	}

	return prometheus.Selector(queryBuff.String()), nil
}

func (q *metricsQuery) createQueryPartsFromSelector(metricSelector labels.Selector) []queryPart {
	requirements, _ := metricSelector.Requirements()

	selectors := []queryPart{}
	for i := 0; i < len(requirements); i++ {
		selector := q.convertRequirement(requirements[i])

		selectors = append(selectors, selector)
	}

	return selectors
}

func (q *metricsQuery) convertRequirement(requirement labels.Requirement) queryPart {
	labelName := requirement.Key()
	values := requirement.Values().List()

	return queryPart{
		labelName: labelName,
		values:    values,
		operator:  requirement.Operator(),
	}
}

func (q *metricsQuery) processQueryParts(queryParts []queryPart) ([]string, map[string]string, error) {
	// We've take the approach here that if we can't perfectly map their query into a Prometheus
	// query that we should abandon the effort completely.
	// The concern is that if we don't get a perfect match on their query parameters, the query result
	// might contain unexpected data that would cause them to take an erroneous action based on the result.

	// Contains the expressions that we want to include as part of the query to Prometheus.
	// e.g. "namespace=my-namespace"
	// e.g. "some_label=some-value"
	var exprs []string

	// Contains the list of label values we're targeting, by namespace.
	// e.g. "some_label" => "value-one|value-two"
	valuesByName := map[string]string{}

	// Convert our query parts into template arguments.
	for _, qPart := range queryParts {
		// Be resilient against bad inputs.
		// We obviously can't generate label filters for these cases.
		if qPart.labelName == "" {
			return nil, nil, ErrLabelNotSpecified
		}

		if !q.operatorIsSupported(qPart.operator) {
			return nil, nil, ErrUnsupportedOperator
		}

		matcher, err := q.selectMatcher(qPart.operator, qPart.values)

		if err != nil {
			return nil, nil, err
		}

		targetValue, err := q.selectTargetValue(qPart.operator, qPart.values)
		if err != nil {
			return nil, nil, err
		}

		expression := matcher(qPart.labelName, targetValue)
		exprs = append(exprs, expression)
		valuesByName[qPart.labelName] = strings.Join(qPart.values, "|")
	}

	return exprs, valuesByName, nil
}

func (q *metricsQuery) selectMatcher(operator selection.Operator, values []string) (func(string, string) string, error) {

	numValues := len(values)
	if numValues == 0 {
		switch operator {
		case selection.Exists:
			return prometheus.LabelNeq, nil
		case selection.DoesNotExist:
			return prometheus.LabelEq, nil
		case selection.Equals, selection.DoubleEquals, selection.NotEquals, selection.In, selection.NotIn:
			return nil, ErrMalformedQuery
		}
	} else if numValues == 1 {
		switch operator {
		case selection.Equals, selection.DoubleEquals:
			return prometheus.LabelEq, nil
		case selection.NotEquals:
			return prometheus.LabelNeq, nil
		case selection.In, selection.Exists:
			return prometheus.LabelMatches, nil
		case selection.DoesNotExist, selection.NotIn:
			return prometheus.LabelNotMatches, nil
		}
	} else {
		// Since labels can only have one value, providing multiple
		// values results in a regex match, even if that's not what the user
		// asked for.
		switch operator {
		case selection.Equals, selection.DoubleEquals, selection.In, selection.Exists:
			return prometheus.LabelMatches, nil
		case selection.NotEquals, selection.DoesNotExist, selection.NotIn:
			return prometheus.LabelNotMatches, nil
		}
	}

	return nil, errors.New("operator not supported by query builder")
}

func (q *metricsQuery) selectTargetValue(operator selection.Operator, values []string) (string, error) {
	numValues := len(values)
	if numValues == 0 {
		switch operator {
		case selection.Exists, selection.DoesNotExist:
			// Return an empty string when values are equal to 0
			// When the operator is LabelNotMatches this will select series without the label
			// or with the label but a value of "".
			// When the operator is LabelMatches this will select series with the label
			// whose value is NOT "".
			return "", nil
		case selection.Equals, selection.DoubleEquals, selection.NotEquals, selection.In, selection.NotIn:
			return "", ErrMalformedQuery
		}
	} else if numValues == 1 {
		switch operator {
		case selection.Equals, selection.DoubleEquals, selection.NotEquals, selection.In, selection.NotIn:
			// Pass the value through as-is.
			// It's somewhat strange to do this for both the regex and equality
			// operators, but if we do it this way it gives the user a little more control.
			// They might choose to send an "IN" request and give a list of static values
			// or they could send a single value that's a regex, giving them a passthrough
			// for their label selector.
			return values[0], nil
		case selection.Exists, selection.DoesNotExist:
			return "", ErrQueryUnsupportedValues
		}
	} else {
		switch operator {
		case selection.Equals, selection.DoubleEquals, selection.NotEquals, selection.In, selection.NotIn:
			// Pass the value through as-is.
			// It's somewhat strange to do this for both the regex and equality
			// operators, but if we do it this way it gives the user a little more control.
			// They might choose to send an "IN" request and give a list of static values
			// or they could send a single value that's a regex, giving them a passthrough
			// for their label selector.
			return strings.Join(values, "|"), nil
		case selection.Exists, selection.DoesNotExist:
			return "", ErrQueryUnsupportedValues
		}
	}

	return "", errors.New("operator not supported by query builder")
}

func (q *metricsQuery) operatorIsSupported(operator selection.Operator) bool {
	return operator != selection.GreaterThan && operator != selection.LessThan
}
