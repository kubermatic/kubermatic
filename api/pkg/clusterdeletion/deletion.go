package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deletedLBAnnotationName = "kubermatic.io/cleaned-up-loadbalancers"
)

func New(seedClient, userClusterClient controllerruntimeclient.Client) *Deletion {
	return &Deletion{
		seedClient:        seedClient,
		userClusterClient: userClusterClient,
	}
}

type Deletion struct {
	seedClient        controllerruntimeclient.Client
	userClusterClient controllerruntimeclient.Client
}

// CleanupCluster is responsible for cleaning up a cluster.
func (d *Deletion) CleanupCluster(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Delete Volumes and LB's inside the user cluster
	if err := d.cleanupInClusterResources(ctx, cluster); err != nil {
		return err
	}
	if err := d.cleanupNodes(ctx, cluster); err != nil {
		return err
	}

	// If we still have nodes, we must not cleanup other infrastructure at the cloud provider
	if kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
		return nil
	}

	if err := d.cleanUpCredentialsSecrets(ctx, cluster); err != nil {
		return err
	}

	return nil
}

func (d *Deletion) cleanupInClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	shouldDeleteLBs := kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.InClusterLBCleanupFinalizer)
	shouldDeletePVs := kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.InClusterPVCleanupFinalizer)

	// If no relevant finalizer exists, directly return
	if !shouldDeleteLBs && !shouldDeletePVs {
		return nil
	}

	// We'll set this to true in case we deleted something. This is meant to requeue as long as all resources are really gone
	// We'll use it for LB's and PV's as well, so the Kubernetes controller manager does the cleanup of all resources in parallel
	var deletedSomeResource bool

	if shouldDeleteLBs {
		deletedSomeLBs, err := d.cleanupLBs(ctx, cluster)
		if err != nil {
			return fmt.Errorf("failed to cleanup LBs: %v", err)
		}
		deletedSomeResource = deletedSomeResource || deletedSomeLBs
	}

	if shouldDeletePVs {
		deletedSomeVolumes, err := d.cleanupVolumes(ctx, cluster)
		if err != nil {
			return fmt.Errorf("failed to cleanup LBs: %v", err)
		}
		deletedSomeResource = deletedSomeResource || deletedSomeVolumes
	}

	// If we deleted something it is implied that there was still something left. Just return
	// here so the finalizers stay, it will make the cluster controller requeue us after a delay
	// This also means that we may end up issuing multiple DELETE calls against the same ressource
	// if cleaning up takes some time, but that shouldn't cause any harm
	// We also need to return when something was deleted so the checkIfAllLoadbalancersAreGone
	// call gets an updated version of the cluster from the lister
	if deletedSomeResource {
		return nil
	}

	lbsAreGone, err := d.checkIfAllLoadbalancersAreGone(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to check if all Loadbalancers are gone: %v", err)
	}
	// Return so we check again later
	if !lbsAreGone {
		return nil
	}

	return d.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(c, kubermaticapiv1.InClusterLBCleanupFinalizer)
		kuberneteshelper.RemoveFinalizer(c, kubermaticapiv1.InClusterPVCleanupFinalizer)
	})
}

func (d *Deletion) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster)) error {
	// Store it here because it may be unset later on if an update request failed
	name := cluster.Name
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		//Get latest version
		if err := d.seedClient.Get(ctx, types.NamespacedName{Name: name}, cluster); err != nil {
			return err
		}
		// Apply modifications
		modify(cluster)
		// Update the cluster
		return d.seedClient.Update(ctx, cluster)
	})
}
