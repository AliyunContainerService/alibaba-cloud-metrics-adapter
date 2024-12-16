package costv2

import (
	"fmt"
	"strings"
	"time"
)

type CostType string

const (
	AllocationPretaxAmount      CostType = "allocation_pretax_amount"
	AllocationPretaxGrossAmount CostType = "allocation_pretax_gross_amount"
	CostEstimated               CostType = "cost_estimated"

	SplitIdlePrefix   = "idle:"
	IdleSuffix        = "__idle__"
	UnallocatedSuffix = "__unallocated__"
)

type Allocation struct {
	Name       string                `json:"name"`
	Properties *AllocationProperties `json:"properties,omitempty"`
	//Window               *Window                `json:"window"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	//CPUCoreHours          float64   `json:"cpuCoreHours"`
	CPUCoreRequestAverage float64 `json:"cpuCoreRequestAverage"`
	CPUCoreUsageAverage   float64 `json:"cpuCoreUsageAverage"`
	//GPUHours               float64               `json:"gpuHours"`
	//RAMByteHours           float64 `json:"ramByteHours"`
	RAMBytesRequestAverage float64 `json:"ramByteRequestAverage"`
	RAMBytesUsageAverage   float64 `json:"ramByteUsageAverage"`
	Cost                   float64 `json:"cost"`
	CostRatio              float64 `json:"costRatio"`
	CustomCost             float64 `json:"customCost"`
}

type AllocationProperties struct {
	Cluster        string            `json:"cluster,omitempty"`
	Node           string            `json:"node,omitempty"`
	Controller     string            `json:"controller,omitempty"`
	ControllerKind string            `json:"controllerKind,omitempty"`
	Namespace      string            `json:"namespace,omitempty"`
	Pod            string            `json:"pod,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	ProviderID     string            `json:"providerID,omitempty"`
}

type AllocationSet map[string]*Allocation

// NewAllocationSet instantiates a new AllocationSet
func NewAllocationSet() *AllocationSet {
	as := AllocationSet(make(map[string]*Allocation))
	return &as
}

// IsEmpty returns true if the AllocationSet is nil, or if it contains
// zero allocations.
func (as *AllocationSet) IsEmpty() bool {
	if as == nil || len(*as) == 0 {
		return true
	}

	return false
}

// Set uses the given Allocation to overwrite the existing entry in the
// AllocationSet under the Allocation's name.
func (as *AllocationSet) Set(alloc *Allocation) error {
	if as.IsEmpty() {
		*as = make(map[string]*Allocation)
	}

	(*as)[alloc.Name] = alloc

	return nil
}

func (as *AllocationSet) AggregateBy(aggregateBy string, idleByNode bool) (*AllocationSet, error) {
	if as.IsEmpty() {
		return nil, nil
	}

	if aggregateBy == "" {
		return as, nil
	}

	aggSet := make(AllocationSet)

	for _, alloc := range *as {
		aggregateKey := ""
		if aggregateBy == "namespace" {
			aggregateKey = alloc.Properties.Namespace
		} else if aggregateBy == "controller" {
			if alloc.Properties.Controller != "<none>" && alloc.Properties.ControllerKind != "<none>" {
				aggregateKey = fmt.Sprintf("%s:%s", alloc.Properties.ControllerKind, alloc.Properties.Controller)
			}
		} else if aggregateBy == "controllerKind" {
			if alloc.Properties.ControllerKind != "<none>" {
				aggregateKey = alloc.Properties.ControllerKind
			}
		} else if aggregateBy == "node" {
			aggregateKey = alloc.Properties.Node
		} else if strings.HasPrefix(aggregateBy, "label:") {
			k := strings.TrimPrefix(aggregateBy, "label:")
			if v, ok := alloc.Properties.Labels[k]; ok {
				aggregateKey = v
			}
		} else {
			return nil, fmt.Errorf("invalid 'aggregate' parameter: %s", aggregateBy)
		}

		// idle cost
		if alloc.Name == IdleSuffix || strings.HasPrefix(alloc.Name, SplitIdlePrefix) {
			aggregateKey = alloc.Name
		}

		// unallocated cost
		if aggregateKey == "" {
			aggregateKey = UnallocatedSuffix
		}

		if v, ok := aggSet[aggregateKey]; !ok {
			aggSet[aggregateKey] = &Allocation{
				Name:                   aggregateKey,
				Start:                  alloc.Start,
				End:                    alloc.End,
				CPUCoreRequestAverage:  alloc.CPUCoreRequestAverage,
				CPUCoreUsageAverage:    alloc.CPUCoreUsageAverage,
				RAMBytesRequestAverage: alloc.RAMBytesRequestAverage,
				RAMBytesUsageAverage:   alloc.RAMBytesUsageAverage,
				Cost:                   alloc.Cost,
				CostRatio:              alloc.CostRatio,
				CustomCost:             alloc.CustomCost,
			}
		} else {
			v.CPUCoreRequestAverage += alloc.CPUCoreRequestAverage
			v.CPUCoreUsageAverage += alloc.CPUCoreUsageAverage
			v.RAMBytesRequestAverage += alloc.RAMBytesRequestAverage
			v.RAMBytesUsageAverage += alloc.RAMBytesUsageAverage
			v.Cost += alloc.Cost
			v.CostRatio += alloc.CostRatio
			v.CustomCost += alloc.CustomCost
		}
	}

	if aggregateBy == "node" && idleByNode {
		for k := range aggSet {
			if !strings.HasPrefix(k, SplitIdlePrefix) {
				continue
			}

			if alloc, ok := aggSet[strings.TrimPrefix(k, SplitIdlePrefix)]; ok {
				aggSet[k].Cost -= alloc.Cost
				aggSet[k].CostRatio -= alloc.CostRatio
			}
		}
	}

	return &aggSet, nil
}

type AllocationSetRange struct {
	Allocations []*AllocationSet `json:"data"`
}

// NewAllocationSetRange instantiates a new range composed of the given
// AllocationSets in the order provided.
func NewAllocationSetRange(allocs ...*AllocationSet) *AllocationSetRange {
	return &AllocationSetRange{
		Allocations: allocs,
	}
}

// AggregateBy aggregates each AllocationSet in the range by the given
// properties and options.
func (asr *AllocationSetRange) AggregateBy(aggregateBy string, idleByNode bool) error {
	asList := make([]*AllocationSet, 0)

	for _, as := range asr.Allocations {
		newAs, err := as.AggregateBy(aggregateBy, idleByNode)
		if err != nil {
			return err
		}
		asList = append(asList, newAs)
	}

	asr.Allocations = asList
	return nil
}

// Append appends the given AllocationSet to the end of the range. It does not
// validate whether or not that violates window continuity.
func (asr *AllocationSetRange) Append(that *AllocationSet) {
	asr.Allocations = append(asr.Allocations, that)
}
