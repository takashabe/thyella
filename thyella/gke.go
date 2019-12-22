package thyella

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	"github.com/K0kubun/pp"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

//go:generate mockgen --package $GOPACKAGE -source $GOFILE -destination mock_$GOFILE

// KaasProvider wrapped GKE client
type KaasProvider interface {
	GetNodePool(ctx context.Context, clusterName, poolName string, nodes []*Node) (*NodePool, error)
	DeleteInstance(ctx context.Context, clusterName string, node *Node) error
}

// GKEClient gke client
type GKEClient struct {
	project string
	client  *container.ClusterManagerClient
}

// NewGKEClient returns initialized GKEClient
func NewGKEClient(project string) (*GKEClient, error) {
	cli, err := container.NewClusterManagerClient(context.Background())
	if err != nil {
		return nil, err
	}

	return &GKEClient{
		project: project,
		client:  cli,
	}, nil
}

// GetNodePool returns node-pool
func (gke GKEClient) GetNodePool(ctx context.Context, clusterName, poolName string, nodes []*Node) (*NodePool, error) {
	location, err := gke.getClusterLocation(ctx, gke.project, clusterName)
	if err != nil {
		pp.Println("4")
		return nil, err
	}

	uri := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s",
		gke.project, location, clusterName, poolName)
	req := &containerpb.GetNodePoolRequest{
		ProjectId: gke.project,
		Name:      uri,
	}
	res, err := gke.client.GetNodePool(ctx, req)
	if err != nil {
		pp.Println("5")
		return nil, err
	}

	ret := &NodePool{
		Name:         poolName,
		Autoscale:    res.GetAutoscaling().Enabled,
		MinNodeCount: int(res.GetAutoscaling().MinNodeCount),
		ZoneURLs:     res.GetInstanceGroupUrls(),
		Preemptible:  res.GetConfig().Preemptible,
		Status:       res.GetStatus().String(),
	}
	ret.Nodes = ret.relateNodes(nodes)
	return ret, nil
}

// DeleteInstance delete GCE instance.
func (gke GKEClient) DeleteInstance(ctx context.Context, clusterName string, node *Node) error {
	gCli, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		return fmt.Errorf("failed to google.DefaultClient: %w", err)
	}
	c, err := compute.New(gCli)
	if err != nil {
		return fmt.Errorf("failed to compute.New: %w", err)
	}

	_, err = c.Instances.Delete(gke.project, node.Zone, node.Name).Context(ctx).Do()
	return err
}

func (gke GKEClient) getClusterLocation(ctx context.Context, project, clusterName string) (string, error) {
	uri := fmt.Sprintf("projects/%s/locations/-", project)
	req := &containerpb.ListClustersRequest{
		ProjectId: project,
		Parent:    uri,
	}
	res, err := gke.client.ListClusters(ctx, req)
	if err != nil {
		return "", err
	}

	pp.Println(res.Clusters)
	pp.Println(clusterName)
	for _, c := range res.Clusters {
		if c.Name == clusterName {
			return c.Location, nil
		}
	}
	return "", fmt.Errorf("not found cluster(%s)", clusterName)
}
