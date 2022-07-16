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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deletedLBAnnotationName = "kubermatic.k8c.io/cleaned-up-loadbalancers"
)

func New(seedClient ctrlruntimeclient.Client, recorder record.EventRecorder, userClusterClientGetter func() (ctrlruntimeclient.Client, error)) *Deletion {
	return &Deletion{
		seedClient:              seedClient,
		recorder:                recorder,
		userClusterClientGetter: userClusterClientGetter,
	}
}

type Deletion struct {
	seedClient              ctrlruntimeclient.Client
	recorder                record.EventRecorder
	userClusterClientGetter func() (ctrlruntimeclient.Client, error)
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
		apiv1.InClusterLBCleanupFinalizer,
		apiv1.InClusterPVCleanupFinalizer) {
		d.recorder.Event(cluster, corev1.EventTypeNormal, "ClusterCleanup", "Cloud-provider resources have been deleted, waiting for them to be destroyed.")
		return nil
	}

	if err := d.cleanupEtcdBackupConfigs(ctx, cluster); err != nil {
		return err
	}

	if err := d.cleanupNodes(ctx, cluster); err != nil {
		return err
	}

	// Delete ClusterRoleBindings on for the cluster on the seed cluster
	if err := d.cleanupClusterRoleBindings(ctx, cluster); err != nil {
		return err
	}

	// If we still have nodes, we must not cleanup other infrastructure at the cloud provider
	if kuberneteshelper.HasFinalizer(cluster, apiv1.NodeDeletionFinalizer) {
		d.recorder.Event(cluster, corev1.EventTypeNormal, "ClusterCleanup", "Waiting for all nodes to be gone.")
		return nil
	}

	// We might need credentials for cloud provider cleanup. Since different cloud providers use different
	// finalizers, we need to ensure that the credentials are not removed until the cloud provider is cleaned up.
	// Cleanup for resources inside the cluster namespace is triggered by the namespace getting deleted, so this
	// must happen (and finish) before the credential secret can ultimately be deleted.
	cred := apiv1.CredentialsSecretsCleanupFinalizer
	ns := apiv1.NamespaceCleanupFinalizer
	if kuberneteshelper.HasFinalizer(cluster, ns) && kuberneteshelper.HasFinalizerSuperset(cluster, cred, ns) {
		if err := d.cleanUpNamespace(ctx, log, cluster); err != nil {
			return err
		}
	} else {
		d.recorder.Eventf(cluster, corev1.EventTypeNormal, "ClusterCleanup", "Waiting for all finalizers except %q to be removed before removing cluster namespace.", ns)
		return nil
	}

	// Now the cluster namespace is gone and we can remove the Secret.
	if kuberneteshelper.HasOnlyFinalizer(cluster, cred) {
		if err := d.cleanUpCredentialsSecrets(ctx, cluster); err != nil {
			return err
		}
	} else {
		d.recorder.Eventf(cluster, corev1.EventTypeNormal, "ClusterCleanup", "Waiting for all finalizers except %q and %q to be removed before removing credential Secret.", cred, ns)
	}

	return nil
}

func (d *Deletion) cleanupInClusterResources(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	shouldDeleteLBs := kuberneteshelper.HasFinalizer(cluster, apiv1.InClusterLBCleanupFinalizer)
	shouldDeletePVs := kuberneteshelper.HasFinalizer(cluster, apiv1.InClusterPVCleanupFinalizer)

	// If no relevant finalizer exists, directly return
	if !shouldDeleteLBs && !shouldDeletePVs {
		return nil
	}

	// Without a namespace, we cannot possibly construct a user cluster client into nothing.
	if cluster.Status.NamespaceName != "" {
		log = log.Named("in-cluster-resources")

		// We'll set this to true in case we deleted something. This is meant to requeue as long as all resources are really gone
		// We'll use it for LB's and PV's as well, so the Kubernetes controller manager does the cleanup of all resources in parallel
		var deletedSomeResource bool

		if shouldDeleteLBs {
			deletedSomeLBs, err := d.cleanupLBs(ctx, log, cluster)
			if err != nil {
				return fmt.Errorf("failed to cleanup LBs: %w", err)
			}
			deletedSomeResource = deletedSomeResource || deletedSomeLBs
		}

		if shouldDeletePVs {
			deletedSomeVolumes, err := d.cleanupVolumes(ctx, cluster)
			if err != nil {
				return fmt.Errorf("failed to cleanup PVs: %w", err)
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
			return fmt.Errorf("failed to check if all Loadbalancers are gone: %w", err)
		}
		// Return so we check again later
		if !lbsAreGone {
			return nil
		}
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, apiv1.InClusterLBCleanupFinalizer, apiv1.InClusterPVCleanupFinalizer)
}
