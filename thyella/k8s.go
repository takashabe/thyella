package thyella

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// kubeconfig auth via gcloud
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

//go:generate mockgen --package $GOPACKAGE -source $GOFILE -destination mock_$GOFILE

var now = time.Now()

// K8sAccessor wrapped raw k8s client
type K8sAccessor interface {
	GetNodeList(ctx context.Context) ([]*Node, error)
	Purge(ctx context.Context, node *Node) error
}

// K8sClient k8s client
type K8sClient struct {
	clientset *kubernetes.Clientset
}

// NewK8sClient returns initialized K8sClient
func NewK8sClient() (K8sAccessor, error) {
	config, err := getRestConfig()
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	client := K8sClient{
		clientset: cs,
	}
	return client, nil
}

func getRestConfig() (*rest.Config, error) {
	// local run
	if os.Getenv("THYELLA_LOCAL") != "" {
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// GetNodeList returns the nodes owned by the cluster
func (k8s K8sClient) GetNodeList(ctx context.Context) ([]*Node, error) {
	nl, err := k8s.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodes := make([]*Node, 0)
	for _, n := range nl.Items {
		labels := n.GetLabels()
		pool, ok := labels["cloud.google.com/gke-nodepool"]
		if !ok {
			continue
		}
		zone := labels["failure-domain.beta.kubernetes.io/zone"]

		ready := false
		condNum := len(n.Status.Conditions)
		if (condNum > 0) && (n.Status.Conditions[condNum-1].Type == "Ready") {
			// unschedulable flag is enable while draining
			if !n.Spec.Unschedulable {
				ready = true
			}
		}

		nodes = append(nodes, &Node{
			Name:     n.GetName(),
			NodePool: pool,
			Zone:     zone,
			Age:      now.Sub(n.GetCreationTimestamp().Time),
			Ready:    ready,
		})
	}

	return nodes, nil
}

// Purge drain & delete.
func (k8s K8sClient) Purge(ctx context.Context, node *Node) error {
	log.Printf("exec purge: %s/%s\n", node.NodePool, node.Name)

	if err := k8s.applyCordonOrUncordon(node, true); err != nil {
		return err
	}

	if err := k8s.drain(ctx, node); err != nil {
		e := fmt.Errorf("failed to drain: %w", err)
		if err := k8s.applyCordonOrUncordon(node, false); err != nil {
			e = fmt.Errorf("%+v: %w", e, err)
		}
		return e
	}

	if err := k8s.delete(ctx, node); err != nil {
		e := fmt.Errorf("failed to delete: %w", err)
		if err := k8s.applyCordonOrUncordon(node, false); err != nil {
			e = fmt.Errorf("%+v: %w", e, err)
		}
		return e
	}

	log.Printf("succeeded purge: %s/%s\n", node.NodePool, node.Name)
	return nil
}

const (
	EvictionKind        = "Eviction"
	EvictionSubresource = "pods/eviction"
)

func (k8s K8sClient) drain(ctx context.Context, node *Node) error {
	policy, err := k8s.policyVersion()
	if err != nil {
		return err
	}

	if err := k8s.evictPods(node, policy); err != nil {
		return err
	}

	return nil
}

func (k8s K8sClient) policyVersion() (string, error) {
	discoveryClient := k8s.clientset.Discovery()
	groupList, err := discoveryClient.ServerGroups()
	if err != nil {
		return "", err
	}

	foundPolicyGroup := false
	var policyGroupVersion string
	for _, group := range groupList.Groups {
		if group.Name == "policy" {
			foundPolicyGroup = true
			policyGroupVersion = group.PreferredVersion.GroupVersion
			break
		}
	}
	if !foundPolicyGroup {
		return "", nil
	}

	resourceList, err := discoveryClient.ServerResourcesForGroupVersion("v1")
	if err != nil {
		return "", err
	}
	for _, resource := range resourceList.APIResources {
		if resource.Name == EvictionSubresource && resource.Kind == EvictionKind {
			return policyGroupVersion, nil
		}
	}
	return "", nil
}

// applyCordonOrUncordon settings schedule flag.
// see. `kubectl [un]cordon <node>`
func (k8s K8sClient) applyCordonOrUncordon(node *Node, cordon bool) error {
	expect := "cordon"
	if !cordon {
		expect = "un" + expect
	}

	n, err := k8s.clientset.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if n.Spec.Unschedulable == cordon {
		log.Printf("already %s: %s\n", expect, node.Name)
		return nil
	}

	n.Spec.Unschedulable = cordon
	if _, err = k8s.clientset.CoreV1().Nodes().Update(n); err != nil {
		return err
	}

	log.Printf("%s: %s\n", expect, node.Name)
	return err
}

func (k8s K8sClient) evictPods(node *Node, policy string) error {
	pods, err := k8s.clientset.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}).String(),
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		eviction := &policyv1beta1.Eviction{
			TypeMeta: metav1.TypeMeta{
				APIVersion: policy,
				Kind:       EvictionKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
		}
		// TODO: error handling by response code
		if err = k8s.clientset.PolicyV1beta1().Evictions(eviction.Namespace).Evict(eviction); err != nil {
			return fmt.Errorf("failed to evict pod: %s %w", pod.Name, err)
		}
		log.Printf("evicted pod: %s\n", pod.GetName())
	}
	return nil
}

func (k8s K8sClient) delete(ctx context.Context, node *Node) error {
	n, err := k8s.clientset.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if !n.Spec.Unschedulable {
		return fmt.Errorf("detect schedulable flag, aborting delete node: %s %w", node.Name, err)
	}
	return k8s.clientset.CoreV1().Nodes().Delete(node.Name, &metav1.DeleteOptions{})
}
