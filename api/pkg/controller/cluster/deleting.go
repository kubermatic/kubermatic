package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cc *controller) syncDeletingCluster(c *kubermaticv1.Cluster) (changedC *kubermaticv1.Cluster, err error) {
	changedC, err = cc.deletingNodeCleanup(c)
	if err != nil || changedC != nil {
		return changedC, err
	}
	changedC, err = cc.deletingCloudProviderCleanup(c)
	if err != nil || changedC != nil {
		return changedC, err
	}
	changedC, err = cc.deletingNamespaceCleanup(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	return nil, cc.deletingClusterResource(c)
}

func (cc *controller) deletingNodeCleanup(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(c, nodeDeletionFinalizer) {
		return nil, nil
	}

	clusterClient, err := c.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster client: %v", err)
	}

	nodes, err := clusterClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %v", err)
	}

	if len(nodes.Items) == 0 {
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, nodeDeletionFinalizer)
		return c, nil
	}

	err = clusterClient.CoreV1().Nodes().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to delete nodes: %v", err)
	}

	return nil, nil
}

func (cc *controller) deletingCloudProviderCleanup(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(c, cloudProviderCleanupFinalizer) {
		return nil, nil
	}

	_, cp, err := provider.ClusterCloudProvider(cc.cps, c)
	if err != nil {
		return nil, err
	}

	if err = cp.CleanUp(c.Spec.Cloud); err != nil {
		return nil, err
	}

	c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, cloudProviderCleanupFinalizer)
	return c, nil
}

func (cc *controller) deletingNamespaceCleanup(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(c, namespaceDeletionFinalizer) {
		return nil, nil
	}

	ns, err := cc.seedInformerGroup.NamespaceInformer.Lister().Get(c.Status.NamespaceName)
	// Only delete finalizer if namespace is really gone
	if err != nil {
		if errors.IsNotFound(err) {
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, namespaceDeletionFinalizer)
			return c, nil
		}
		return nil, err
	}

	if ns.DeletionTimestamp == nil {
		return nil, cc.client.CoreV1().Namespaces().Delete(c.Status.NamespaceName, &metav1.DeleteOptions{})
	}

	return nil, nil
}

func (cc *controller) deletingClusterResource(c *kubermaticv1.Cluster) error {
	if len(c.Finalizers) != 0 {
		return nil
	}

	err := cc.masterCrdClient.KubermaticV1().Clusters().Delete(c.Name, &metav1.DeleteOptions{})
	// Only delete finalizer if namespace is really gone
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}
