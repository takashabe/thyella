package thyella

import (
	"fmt"
	"strings"
	"time"
)

// Node represents node
type Node struct {
	Name     string
	NodePool string
	Zone     string
	Age      time.Duration
	Ready    bool
}

const statusNodePoolStable = "RUNNING"

// NodePool represents node-pool
type NodePool struct {
	Name         string
	Autoscale    bool
	MinNodeCount int
	Preemptible  bool
	Status       string
	ZoneURLs     []string
	Nodes        []*Node
}

func (np NodePool) relateNodes(nodes []*Node) []*Node {
	ret := make([]*Node, 0)
	for _, n := range nodes {
		if n.NodePool == np.Name {
			ret = append(ret, n)
		}
	}
	return ret
}

// GetMaxAgeNode returns max age node
func (np *NodePool) GetMaxAgeNode() (*Node, bool) {
	var max *Node
	for _, n := range np.Nodes {
		if max == nil || max.Age < n.Age {
			max = n
		}
	}
	return max, max != nil
}

// GetMaxAgeNodeWithBalance returns the longest-lived node so that it is even
// for each zone.
func (np *NodePool) GetMaxAgeNodeWithBalance() (*Node, bool) {
	nodeEachZone := make(map[string][]*Node, 0)
	for _, n := range np.Nodes {
		list, ok := nodeEachZone[n.Zone]
		if !ok {
			nodeEachZone[n.Zone] = []*Node{n}
			continue
		}
		nodeEachZone[n.Zone] = append(list, n)
	}

	maxNumZone := make([]*Node, 0)
	for _, ns := range nodeEachZone {
		if len(maxNumZone) < len(ns) {
			maxNumZone = ns
		}
	}

	var maxAge *Node
	for _, n := range maxNumZone {
		if maxAge == nil || maxAge.Age < n.Age {
			maxAge = n
		}
	}
	return maxAge, maxAge != nil
}

// IsMinimumNodes returns running nodes is minimum or not
func (np *NodePool) IsMinimumNodes() bool {
	readyCnt := 0
	for _, n := range np.Nodes {
		if n.Ready {
			readyCnt++
		}
	}
	minNodeWholeZone := np.MinNodeCount * len(np.ZoneURLs)
	return readyCnt <= minNodeWholeZone
}

// AllGreen returns available or not
func (np *NodePool) AllGreen() bool {
	if np.Status != statusNodePoolStable {
		return false
	}
	for _, n := range np.Nodes {
		if !n.Ready {
			return false
		}
	}
	return true
}

// NodePoolGroup represents node-pool group
// e.g. preemptible pool and non-preemptible pool
type NodePoolGroup struct {
	NodePools []*NodePool
}

// GetPoolWithPreemptible returns node pool specified preemptible status
func (npg NodePoolGroup) GetPoolWithPreemptible(preemptible bool) (*NodePool, bool) {
	for _, n := range npg.NodePools {
		if n.Preemptible == preemptible {
			return n, true
		}
	}
	return nil, false
}

func (npg NodePoolGroup) String() string {
	names := make([]string, 0)
	for _, np := range npg.NodePools {
		names = append(names, np.Name)
	}
	return fmt.Sprintf("[%s]", strings.Join(names, ","))
}
