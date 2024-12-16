package costv2

import (
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/klog/v2"
	"strings"
)

type Filter struct {
	Cluster        []string
	Namespace      []string
	ControllerName []string
	ControllerKind []string
	Pod            []string
	Label          map[string][]string
}

func (f *Filter) GetLabelSelectorStr() string {
	var requirements []labels.Requirement

	addInRequirement := func(key string, values []string) error {
		if len(values) == 0 {
			klog.Infof("filter value is empty: %s", key)
			return nil
		}
		r, err := labels.NewRequirement(key, selection.In, values)
		if err != nil {
			klog.Errorf("failed to parse filter %s to requirement, error: %v", key, err)
			return err
		}
		requirements = append(requirements, *r)
		return nil
	}

	if f.Label != nil {
		for key, values := range f.Label {
			if values == nil {
				continue
			}
			if err := addInRequirement("label_"+key, values); err != nil {
				return ""
			}
		}
	}

	if err := addInRequirement("cluster", f.Cluster); err != nil {
		return ""
	}

	if err := addInRequirement("namespace", f.Namespace); err != nil {
		return ""
	}

	if err := addInRequirement("created_by_name", f.ControllerName); err != nil {
		return ""
	}

	if err := addInRequirement("created_by_kind", f.ControllerKind); err != nil {
		return ""
	}

	if err := addInRequirement("pod", f.Pod); err != nil {
		return ""
	}

	if requirements == nil {
		klog.Infof("filter is empty, do not need to parse.")
		return ""
	}

	selector := labels.NewSelector().Add(requirements...).String()
	klog.Infof("get filter label selector str: %s", selector)
	return selector
}

// parseFilterParts Split the filter string
func parseFilterParts(filterStr string) []string {
	prefixes := []string{"namespace:", "controllerName:", "controllerKind:", "pod:", "label["}

	// for each prefix, the prefix in the filter string is replaced with a special symbol("\x1f") plus prefix for segmentation.
	for _, prefix := range prefixes {
		filterStr = strings.Replace(filterStr, prefix, "\x1f"+prefix, -1)
	}

	return strings.Split(filterStr, "\x1f")
}

// parse str "a","b" to []string{a,b}
func parseValueList(values string) []string {
	valueList := strings.Split(values, ",")
	for i, value := range valueList {
		valueList[i] = strings.Trim(strings.Trim(value, " "), `"`)
	}
	return valueList
}

// ParseFilter Parses the given string to *Filter
func ParseFilter(filterStr string) (*Filter, error) {
	filter := &Filter{}

	filterParts := parseFilterParts(filterStr)
	klog.Infof("split the filterStr to filterParts: %v", filterParts)

	for _, part := range filterParts {
		if part == "" {
			continue
		}

		// handles the contents of fields inside ""
		part = strings.Trim(part, " ")
		kv := strings.SplitN(part, `:"`, 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid filter format: %s", part)
		}

		key := strings.Trim(kv[0], `"`)
		values := strings.Trim(kv[1], `+`)

		switch {
		case strings.HasPrefix(key, "cluster"):
			filter.Cluster = parseValueList(values)
		case strings.HasPrefix(key, "namespace"):
			filter.Namespace = parseValueList(values)
		case strings.HasPrefix(key, "controllerName"):
			filter.ControllerName = parseValueList(values)
		case strings.HasPrefix(key, "controllerKind"):
			filter.ControllerKind = parseValueList(values)
		case strings.HasPrefix(key, "pod"):
			filter.Pod = parseValueList(values)
		case strings.HasPrefix(key, "label["):
			filter.Label = make(map[string][]string)
			labelKey := strings.TrimPrefix(key, "label[")
			labelKey = strings.TrimSuffix(labelKey, "]")
			filter.Label[labelKey] = []string{strings.Trim(values, `"`)}
		default:
			return nil, fmt.Errorf("unsupported filter key: %s", key)
		}
	}

	if filter.ControllerKind != nil {
		for i, kind := range filter.ControllerKind {
			switch strings.ToLower(kind) {
			case "deployment":
				filter.ControllerKind[i] = "ReplicaSet"
			case "daemonset":
				filter.ControllerKind[i] = "DaemonSet"
			case "statefulset":
				filter.ControllerKind[i] = "StatefulSet"
			case "job":
				filter.ControllerKind[i] = "Job"
			case "replicaset":
				filter.ControllerKind[i] = "ReplicaSet"
			default:
				return nil, fmt.Errorf("unsupported controller kind: %s", kind)
			}
		}
	}

	return filter, nil
}

func (f *Filter) IsNonClusterEmpty() bool {
	return len(f.Namespace) == 0 &&
		len(f.ControllerName) == 0 &&
		len(f.ControllerKind) == 0 &&
		len(f.Pod) == 0 &&
		len(f.Label) == 0
}
