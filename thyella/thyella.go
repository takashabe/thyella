package thyella

import (
	"context"
	"fmt"
	"log"

	"github.com/K0kubun/pp"
)

// Thyella provide purge
type Thyella struct {
	KaasClient KaasProvider
	K8sClient  K8sAccessor
}

// Purge purge nodes.
func (p Thyella) Purge(cluster string, nps []string) error {
	ctx := context.Background()

	if len(nps) == 0 {
		return nil
	}

	nodes, err := p.K8sClient.GetNodeList(ctx)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return nil
	}

	// find a node from all nodes.
	nodeEachPools := make(map[string]*Node)
	for _, n := range nodes {
		if _, ok := nodeEachPools[n.NodePool]; ok {
			continue
		}
		nodeEachPools[n.NodePool] = n
	}

	n, ok, err := p.purgeInGroup(ctx, cluster, nps, nodes, nodeEachPools)
	if err != nil {
		return err
	}
	if ok {
		log.Printf("purge node: %s\n", n.Name)
	}
	return nil
}

func (p Thyella) purgeInGroup(ctx context.Context, cluster string, group []string, nodes []*Node, nodeEachPools map[string]*Node) (*Node, bool, error) {
	npg := NodePoolGroup{
		NodePools: make([]*NodePool, 0),
	}
	for _, pool := range group {
		if _, ok := nodeEachPools[pool]; !ok {
			continue
		}

		np, err := p.KaasClient.GetNodePool(ctx, cluster, pool, nodes)
		if err != nil {
			return nil, false, err
		}
		npg.NodePools = append(npg.NodePools, np)
	}

	log.Printf("processing node-pool group: %s\n", npg)

	ready := true
	for _, np := range npg.NodePools {
		if !np.AllGreen() {
			log.Printf("skipped: %s is unhealthy.\n", np.Name)
			ready = false
		}
	}
	if !ready {
		return nil, false, nil
	}

	// purge node in non-preemptible pool
	if np, ok := npg.GetPoolWithPreemptible(false); ok {
		if !np.IsMinimumNodes() {
			target, _ := np.GetMaxAgeNodeWithBalance()
			if err := p.K8sClient.Purge(ctx, target); err != nil {
				return nil, false, fmt.Errorf("failed to purge node: %s %w", target.Name, err)
			}
			if err := p.KaasClient.DeleteInstance(ctx, cluster, target); err != nil {
				return nil, false, fmt.Errorf("failed to delete instance: %s %w", target.Name, err)
			}
			return target, true, nil
		}
	}

	// purge node in preemptible pool
	if np, ok := npg.GetPoolWithPreemptible(true); ok {
		if target, ok := np.GetMaxAgeNode(); ok {
			if err := p.K8sClient.Purge(ctx, target); err != nil {
				return nil, false, fmt.Errorf("failed to purge node: %s %w", target.Name, err)
			}
			if err := p.KaasClient.DeleteInstance(ctx, cluster, target); err != nil {
				return nil, false, fmt.Errorf("failed to delete instance: %s %w", target.Name, err)
			}
			return target, true, nil
		}
	}

	// not found a purgeable node
	return nil, false, nil
}
