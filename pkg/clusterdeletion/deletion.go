/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clusterdeletion

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deletedLBAnnotationName = "kubermatic.io/cleaned-up-loadbalancers"
)

func New(seedClient ctrlruntimeclient.Client, userClusterClientGetter func() (ctrlruntimeclient.Client, error), etcdBackupRestoreController bool) *Deletion {
	return &Deletion{
		seedClient:                  seedClient,
		userClusterClientGetter:     userClusterClientGetter,
		etcdBackupRestoreController: etcdBackupRestoreController,
	}
}

type Deletion struct {
	seedClient                  ctrlruntimeclient.Client
	userClusterClientGetter     func() (ctrlruntimeclient.Client, error)
	etcdBackupRestoreController bool
}

// CleanupCluster is responsible for cleaning up a cluster.
func (d *Deletion) CleanupCluster(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	log = log.Named("cleanup")

	// Delete OPA constraints first to make sure some rules dont block deletion
	if err := d.cleanupConstraints(ctx, cluster); err != nil {
		return err
	}

	// Delete Volumes and LB's inside the user cluster
	if err := d.cleanupInClusterResources(ctx, log, cluster); err != nil {
		return err
	}

	// If cleanup didn't finish we have to go back, because if there are controllers running
	// inside the cluster and we delete the nodes, we get stuck.
	if kuberneteshelper.HasAnyFinalizer(cluster,
		kubermaticapiv1.InClusterLBCleanupFinalizer,
		kubermaticapiv1.InClusterPVCleanupFinalizer) {
		return nil
	}

	if err := d.cleanupEtcdBackupConfigs(ctx, cluster, d.etcdBackupRestoreController); err != nil {
		return err
	}

	if err := d.cleanupNodes(ctx, cluster); err != nil {
		return err
	}

	// If we still have nodes, we must not cleanup other infrastructure at the cloud provider
	if kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
		return nil
	}

	// We might need credentials for cloud provider cleanup. Since different cloud providers use different
	// finalizers, we need to ensure that the credentials are not removed until the cloud provider is cleaned
	// up, or in other words, all other finalizers have been removed from the cluster, and the
	// CredentialsSecretsCleanupFinalizer is the only finalizer left.
	if kuberneteshelper.HasOnlyFinalizer(cluster, kubermaticapiv1.CredentialsSecretsCleanupFinalizer) {
		if err := d.cleanUpCredentialsSecrets(ctx, cluster); err != nil {
			return err
		}
	}

	return nil
}

func (d *Deletion) cleanupInClusterResources(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	log = log.Named("in-cluster-resources")

	shouldDeleteLBs := kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.InClusterLBCleanupFinalizer)
	shouldDeletePVs := kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.InClusterPVCleanupFinalizer)

	// If no relevant finalizer exists, directly return
	if !shouldDeleteLBs && !shouldDeletePVs {
		log.Debug("Skipping in-cluster-resources deletion. None of the in-cluster cleanup finalizers is set.")
		return nil
	}

	// We'll set this to true in case we deleted something. This is meant to requeue as long as all resources are really gone
	// We'll use it for LB's and PV's as well, so the Kubernetes controller manager does the cleanup of all resources in parallel
	var deletedSomeResource bool

	if shouldDeleteLBs {
		deletedSomeLBs, err := d.cleanupLBs(ctx, log, cluster)
		if err != nil {
			return fmt.Errorf("failed to cleanup LBs: %v", err)
		}
		deletedSomeResource = deletedSomeResource || deletedSomeLBs
	}

	if shouldDeletePVs {
		deletedSomeVolumes, err := d.cleanupVolumes(ctx, cluster)
		if err != nil {
			return fmt.Errorf("failed to cleanup PVs: %v", err)
		}
		deletedSomeResource = deletedSomeResource || deletedSomeVolumes
	}

	// If we deleted something it is implied that there was still something left. Just return
	// here so the finalizers stay, it will make the cluster controller requeue us after a delay
	// This also means that we may end up issuing multiple DELETE calls against the same resource
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

	oldCluster := cluster.DeepCopy()
	kuberneteshelper.RemoveFinalizer(cluster, kubermaticapiv1.InClusterLBCleanupFinalizer)
	kuberneteshelper.RemoveFinalizer(cluster, kubermaticapiv1.InClusterPVCleanupFinalizer)
	return d.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}
