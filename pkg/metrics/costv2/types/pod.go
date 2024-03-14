package costv2

import (
	"fmt"
)

type Pod struct {
	Window      Window
	Key         PodMeta
	Node        string
	Allocations *Allocation
}

type PodMeta struct {
	Namespace string
	Pod       string
}

func (m PodMeta) String() string {
	return fmt.Sprintf("%s/%s", m.Namespace, m.Pod)
}
