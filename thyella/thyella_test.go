package thyella

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	type Args struct {
		cluster string
		nps     []string
	}

	var (
		nodeA = &Node{Name: "na", NodePool: "pa", Ready: true}
		nodeB = &Node{Name: "nb", NodePool: "pb", Ready: true}
		nodeC = &Node{Name: "nc", NodePool: "pc", Ready: true}

		nodes = []*Node{nodeA, nodeB, nodeC}

		args = Args{
			cluster: "cluster",
			nps: []string{
				"pa",
				"pb",
			},
		}
	)

	tests := []struct {
		name     string
		input    Args
		wantMock func(*MockKaasProvider, *MockK8sAccessor)
	}{
		{
			name:  "should purge 'pa' and 'pb'",
			input: args,
			wantMock: func(kaas *MockKaasProvider, k8s *MockK8sAccessor) {
				k8s.EXPECT().GetNodeList(ctx).Return([]*Node{nodeA, nodeB, nodeC}, nil)

				kaas.EXPECT().GetNodePool(ctx, "cluster", "pa", nodes).Return(&NodePool{
					Name:         "pa",
					Nodes:        []*Node{nodeA},
					MinNodeCount: 0,
					Status:       statusNodePoolStable,
				}, nil)
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pb", nodes).Return(&NodePool{
					Name:        "pb",
					Nodes:       []*Node{nodeB},
					Preemptible: true,
					Status:      statusNodePoolStable,
				}, nil)
				k8s.EXPECT().Purge(ctx, nodeA).Return(nil)
				kaas.EXPECT().DeleteInstance(ctx, "cluster", nodeA).Return(nil)
			},
		},
		{
			name:  "should purge only the 'pa'",
			input: args,
			wantMock: func(kaas *MockKaasProvider, k8s *MockK8sAccessor) {
				k8s.EXPECT().GetNodeList(ctx).Return([]*Node{nodeA, nodeB, nodeC}, nil)

				kaas.EXPECT().GetNodePool(ctx, "cluster", "pa", nodes).Return(&NodePool{
					Name:         "pa",
					Nodes:        []*Node{nodeA},
					MinNodeCount: 0,
					Status:       statusNodePoolStable,
				}, nil)
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pb", nodes).Return(&NodePool{
					Name:        "pb",
					Nodes:       []*Node{nodeB},
					Preemptible: true,
					Status:      statusNodePoolStable,
				}, nil)
				k8s.EXPECT().Purge(ctx, nodeA).Return(nil)
				kaas.EXPECT().DeleteInstance(ctx, "cluster", nodeA).Return(nil)
			},
		},
		{
			name:  "should non purge when running the minimum nodes",
			input: args,
			wantMock: func(kaas *MockKaasProvider, k8s *MockK8sAccessor) {
				k8s.EXPECT().GetNodeList(ctx).Return([]*Node{nodeA, nodeB, nodeC}, nil)

				kaas.EXPECT().GetNodePool(ctx, "cluster", "pa", nodes).Return(&NodePool{
					Name:         "pa",
					Nodes:        []*Node{nodeA},
					MinNodeCount: 1,
					ZoneURLs:     []string{"1"},
					Status:       statusNodePoolStable,
				}, nil)
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pb", nodes).Return(&NodePool{
					Name:         "pb",
					Nodes:        []*Node{nodeB},
					MinNodeCount: 1,
					ZoneURLs:     []string{"1"},
					Status:       statusNodePoolStable,
				}, nil)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockKaasClient := NewMockKaasProvider(ctrl)
			mockK8sClient := NewMockK8sAccessor(ctrl)
			tt.wantMock(mockKaasClient, mockK8sClient)

			thyella := Thyella{
				KaasClient: mockKaasClient,
				K8sClient:  mockK8sClient,
			}

			err := thyella.Purge(tt.input.cluster, tt.input.nps)
			assert.NoError(t, err)
		})
	}
}

func TestPurgeInGroup(t *testing.T) {
	ctx := context.Background()

	type input struct {
		group []string
		nodes []*Node
		nep   map[string]*Node
	}

	var (
		nodeA = &Node{Name: "na", NodePool: "pa", Ready: true, Zone: "za"}
		nodeB = &Node{Name: "nb", NodePool: "pb", Ready: true, Zone: "zb"}
		// not ready
		nodeC = &Node{Name: "nc", NodePool: "pc"}

		nodes = []*Node{nodeA, nodeB, nodeC}
	)

	tests := []struct {
		name     string
		input    input
		wantMock func(*MockKaasProvider, *MockK8sAccessor)
		wantNode *Node
	}{
		{
			name: "should purge 'nodeA' when the non-preemptible pool is enough nodes",
			input: input{
				group: []string{"pa", "pb"},
				nodes: []*Node{nodeA, nodeB, nodeC},
				nep: map[string]*Node{
					"pa": nodeA,
					"pb": nodeB,
				},
			},
			wantMock: func(kaas *MockKaasProvider, k8s *MockK8sAccessor) {
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pa", nodes).Return(&NodePool{
					Name:         "pa",
					MinNodeCount: 0,
					Nodes:        []*Node{nodeA},
					ZoneURLs:     []string{"1"},
					Status:       statusNodePoolStable,
				}, nil)
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pb", nodes).Return(&NodePool{
					Name:        "pb",
					Nodes:       []*Node{nodeB},
					Preemptible: true,
					ZoneURLs:    []string{"1"},
					Status:      statusNodePoolStable,
				}, nil)
				k8s.EXPECT().Purge(ctx, nodeA).Return(nil)
				kaas.EXPECT().DeleteInstance(ctx, "cluster", nodeA).Return(nil)
			},
			wantNode: nodeA,
		},
		{
			name: "should purge 'nodeB' when the non-preemptible pool is minimum nodes",
			input: input{
				group: []string{"pa", "pb"},
				nodes: []*Node{nodeA, nodeB, nodeC},
				nep: map[string]*Node{
					"pa": nodeA,
					"pb": nodeB,
				},
			},
			wantMock: func(kaas *MockKaasProvider, k8s *MockK8sAccessor) {
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pa", nodes).Return(&NodePool{
					Name:         "pa",
					MinNodeCount: 2,
					Nodes:        []*Node{nodeA},
					ZoneURLs:     []string{"1"},
					Status:       statusNodePoolStable,
				}, nil)
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pb", nodes).Return(&NodePool{
					Name:         "pb",
					Nodes:        []*Node{nodeB},
					Preemptible:  true,
					MinNodeCount: 0,
					ZoneURLs:     []string{"1"},
					Status:       statusNodePoolStable,
				}, nil)
				k8s.EXPECT().Purge(ctx, nodeB).Return(nil)
				kaas.EXPECT().DeleteInstance(ctx, "cluster", nodeB).Return(nil)
			},
			wantNode: nodeB,
		},
		{
			name: "should non purge when unhealthy node",
			input: input{
				group: []string{"pa", "pb"},
				nodes: []*Node{nodeA, nodeB, nodeC},
				nep: map[string]*Node{
					"pa": nodeA,
					"pb": nodeB,
				},
			},
			wantMock: func(kaas *MockKaasProvider, k8s *MockK8sAccessor) {
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pa", nodes).Return(&NodePool{
					Name:     "pa",
					Nodes:    []*Node{nodeA, nodeC},
					ZoneURLs: []string{"1"},
					Status:   statusNodePoolStable,
				}, nil)
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pb", nodes).Return(&NodePool{
					Name:        "pb",
					Nodes:       []*Node{nodeB},
					Preemptible: true,
					ZoneURLs:    []string{"1"},
					Status:      statusNodePoolStable,
				}, nil)
			},
			wantNode: nil,
		},
		{
			name: "should non purge when unhealthy node pool",
			input: input{
				group: []string{"pa", "pb"},
				nodes: []*Node{nodeA, nodeB, nodeC},
				nep: map[string]*Node{
					"pa": nodeA,
					"pb": nodeB,
				},
			},
			wantMock: func(kaas *MockKaasProvider, k8s *MockK8sAccessor) {
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pa", nodes).Return(&NodePool{
					Name:     "pa",
					Nodes:    []*Node{nodeA},
					ZoneURLs: []string{"1"},
					Status:   statusNodePoolStable,
				}, nil)
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pb", nodes).Return(&NodePool{
					Name:        "pb",
					Nodes:       []*Node{nodeB},
					Preemptible: true,
					Status:      "unkwnon",
				}, nil)
			},
			wantNode: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockKaasClient := NewMockKaasProvider(ctrl)
			mockK8sClient := NewMockK8sAccessor(ctrl)
			tt.wantMock(mockKaasClient, mockK8sClient)

			purger := Thyella{
				KaasClient: mockKaasClient,
				K8sClient:  mockK8sClient,
			}

			got, _, err := purger.purgeInGroup(ctx, "cluster", tt.input.group, tt.input.nodes, tt.input.nep)
			assert.NoError(t, err)
			if tt.wantNode == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.wantNode, got)
			}
		})
	}
}

func TestPurgeInGroupWithBalanced(t *testing.T) {
	ctx := context.Background()

	type input struct {
		group []string
		nodes []*Node
		nep   map[string]*Node
	}

	var (
		// zone:za
		nodeA = &Node{Name: "na", NodePool: "pa", Ready: true, Age: time.Duration(1), Zone: "za"}
		nodeB = &Node{Name: "nb", NodePool: "pa", Ready: true, Age: time.Duration(2), Zone: "za"}
		// zone:zb
		nodeC = &Node{Name: "nc", NodePool: "pa", Ready: true, Age: time.Duration(3), Zone: "zb"}

		nodes = []*Node{nodeA, nodeB, nodeC}
	)

	tests := []struct {
		name     string
		input    input
		wantMock func(*MockKaasProvider, *MockK8sAccessor)
	}{
		{
			// name: "nodeBがpurgeされる(非preemptible poolはzone内balanceを保つように動く)",
			name: "should purge 'nodeB' to while keeping balance",
			input: input{
				group: []string{"pa"},
				nodes: []*Node{nodeA, nodeB, nodeC},
				nep:   map[string]*Node{"pa": nodeA},
			},
			wantMock: func(kaas *MockKaasProvider, k8s *MockK8sAccessor) {
				kaas.EXPECT().GetNodePool(ctx, "cluster", "pa", nodes).Return(&NodePool{
					Name:         "pa",
					Nodes:        []*Node{nodeA, nodeB, nodeC},
					MinNodeCount: 0,
					ZoneURLs:     []string{"1", "2"},
					Status:       statusNodePoolStable,
				}, nil)
				k8s.EXPECT().Purge(ctx, nodeB).Return(nil)
				kaas.EXPECT().DeleteInstance(ctx, "cluster", nodeB).Return(nil)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockKaasClient := NewMockKaasProvider(ctrl)
			mockK8sClient := NewMockK8sAccessor(ctrl)
			tt.wantMock(mockKaasClient, mockK8sClient)

			purger := Thyella{
				KaasClient: mockKaasClient,
				K8sClient:  mockK8sClient,
			}

			_, _, err := purger.purgeInGroup(ctx, "cluster", tt.input.group, tt.input.nodes, tt.input.nep)
			assert.NoError(t, err)
		})
	}
}
