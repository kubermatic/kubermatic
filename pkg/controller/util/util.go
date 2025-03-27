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

package util

import (
	"context"
	"fmt"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/workqueue"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EnqueueClusterForNamespacedObject enqueues the cluster that owns a namespaced object, if any
// It is used by various controllers to react to changes in the resources in the cluster namespace.
func EnqueueClusterForNamespacedObject(client ctrlruntimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		cluster, err := kubernetes.ClusterFromNamespace(ctx, client, a.GetNamespace())
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %w", err))
			return []reconcile.Request{}
		}

		if cluster != nil {
			return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
		}

		return []reconcile.Request{}
	})
}

// EnqueueClusterForNamespacedObjectWithSeedName enqueues the cluster that owns a namespaced object,
// if any. The seedName is put into the namespace field
// It is used by various controllers to react to changes in the resources in the cluster namespace.
func EnqueueClusterForNamespacedObjectWithSeedName(client ctrlruntimeclient.Client, seedName string, clusterSelector labels.Selector) handler.EventHandler {
	return TypedEnqueueClusterForNamespacedObjectWithSeedName[ctrlruntimeclient.Object](client, seedName, clusterSelector)
}

func TypedEnqueueClusterForNamespacedObjectWithSeedName[T ctrlruntimeclient.Object](client ctrlruntimeclient.Client, seedName string, clusterSelector labels.Selector) handler.TypedEventHandler[T, reconcile.Request] {
	return handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, a T) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		listOpts := &ctrlruntimeclient.ListOptions{
			LabelSelector: clusterSelector,
		}

		if err := client.List(ctx, clusterList, listOpts); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %w", err))
			return []reconcile.Request{}
		}

		for _, cluster := range clusterList.Items {
			if cluster.Status.NamespaceName == a.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{
					Namespace: seedName,
					Name:      cluster.Name,
				}}}
			}
		}
		return []reconcile.Request{}
	})
}

// EnqueueClusterScopedObjectWithSeedName enqueues a cluster-scoped object with the seedName
// as namespace. If it gets an object with a non-empty name, it will log an error and not enqueue
// anything.
func EnqueueClusterScopedObjectWithSeedName(seedName string) handler.EventHandler {
	return TypedEnqueueClusterScopedObjectWithSeedName[ctrlruntimeclient.Object](seedName)
}

func TypedEnqueueClusterScopedObjectWithSeedName[T ctrlruntimeclient.Object](seedName string) handler.TypedEventHandler[T, reconcile.Request] {
	return handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, a T) []reconcile.Request {
		if a.GetNamespace() != "" {
			utilruntime.HandleError(fmt.Errorf("EnqueueClusterScopedObjectWithSeedName was used with namespace scoped object %s/%s of type %T", a.GetNamespace(), a.GetName(), a))
		}

		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Namespace: seedName,
			Name:      a.GetName(),
		}}}
	})
}

// EnqueueProjectForCluster returns an event handler that creates a reconcile
// request for the project of a cluster, based on the ProjectIDLabelKey label.
func EnqueueProjectForCluster() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(_ context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		cluster, ok := a.(*kubermaticv1.Cluster)
		if !ok {
			return nil
		}

		projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
		if projectID == "" {
			return nil // should never happen, a webhook ensures a proper label
		}

		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Name: projectID,
		}}}
	})
}

const (
	CreateOperation = "create"
	UpdateOperation = "update"
	DeleteOperation = "delete"
)

// EnqueueObjectWithOperation enqueues a the namespaced name for any resource. It puts
// the current operation (create, update, delete) as the namespace and "namespace/name"
// of the object into the name of the reconcile request.
func EnqueueObjectWithOperation() handler.EventHandler {
	return handler.Funcs{
		CreateFunc: func(_ context.Context, e event.TypedCreateEvent[ctrlruntimeclient.Object], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: CreateOperation,
					Name:      fmt.Sprintf("%s/%s", e.Object.GetNamespace(), e.Object.GetName()),
				},
			})
		},

		UpdateFunc: func(_ context.Context, e event.TypedUpdateEvent[ctrlruntimeclient.Object], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: UpdateOperation,
					Name:      fmt.Sprintf("%s/%s", e.ObjectNew.GetNamespace(), e.ObjectNew.GetName()),
				},
			})
		},

		DeleteFunc: func(_ context.Context, e event.TypedDeleteEvent[ctrlruntimeclient.Object], queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: DeleteOperation,
					Name:      fmt.Sprintf("%s/%s", e.Object.GetNamespace(), e.Object.GetName()),
				},
			})
		},
	}
}

// EnqueueConst enqueues a constant. It is meant for controllers that don't have a parent object
// they could enc and instead reconcile everything at once.
// The queueKey will be defaulted if empty.
func EnqueueConst(queueKey string) handler.EventHandler {
	if queueKey == "" {
		queueKey = "const"
	}

	return handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o ctrlruntimeclient.Object) []reconcile.Request {
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Name:      queueKey,
				Namespace: "",
			}}}
	})
}

// ClusterAvailableForReconciling returns true if the given cluster can be reconciled. This is true if
// the cluster does not yet have the SeedResourcesUpToDate condition or if the concurrency limit of the
// controller is not yet reached. This ensures that not too many cluster updates are running at the same
// time, but also makes sure that un-UpToDate clusters will continue to be reconciled.
func ClusterAvailableForReconciling(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, concurrencyLimit int) (bool, error) {
	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionSeedResourcesUpToDate, corev1.ConditionTrue) {
		return true, nil
	}

	limitReached, err := ConcurrencyLimitReached(ctx, client, concurrencyLimit)
	return !limitReached, err
}

// ConcurrencyLimitReached checks all the clusters inside the seed cluster and checks for the
// SeedResourcesUpToDate condition. Returns true if the number of clusters without this condition
// is equal or larger than the given limit.
func ConcurrencyLimitReached(ctx context.Context, client ctrlruntimeclient.Client, limit int) (bool, error) {
	clusters := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, clusters); err != nil {
		return true, fmt.Errorf("failed to list clusters: %w", err)
	}

	finishedUpdatingClustersCount := 0
	for _, cluster := range clusters.Items {
		if cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionSeedResourcesUpToDate, corev1.ConditionTrue) {
			finishedUpdatingClustersCount++
		}
	}

	clustersUpdatingInProgressCount := len(clusters.Items) - finishedUpdatingClustersCount

	return clustersUpdatingInProgressCount >= limit, nil
}

func getCNIApplicationInstallation(ctx context.Context, userClusterClient ctrlruntimeclient.Client, cniType kubermaticv1.CNIPluginType) (*appskubermaticv1.ApplicationInstallation, error) {
	app := &appskubermaticv1.ApplicationInstallation{}
	switch cniType {
	case kubermaticv1.CNIPluginTypeCilium:
		name := kubermaticv1.CNIPluginTypeCilium.String()
		if err := userClusterClient.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: name}, app); err != nil {
			return nil, ctrlruntimeclient.IgnoreNotFound(err)
		}
		return app, nil
	}

	return nil, fmt.Errorf("unsupported CNI type: %s", cniType)
}

// IsCNIApplicationReady checks if the CNI application is deployed and ready.
func IsCNIApplicationReady(ctx context.Context, userClusterClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	if cluster.Spec.CNIPlugin == nil || !cni.IsManagedByAppInfra(cluster.Spec.CNIPlugin.Type, cluster.Spec.CNIPlugin.Version) {
		return true, nil // No CNI plugin or not managed by app infra, consider it ready
	}

	cniApp, err := getCNIApplicationInstallation(ctx, userClusterClient, cluster.Spec.CNIPlugin.Type)
	if err != nil {
		return false, err
	}
	// Check if the application is deployed and status is updated with app version.
	return cniApp != nil && cniApp.Status.ApplicationVersion != nil, nil
}

// NodesAvailable checks if any node object is already created.
func NodesAvailable(ctx context.Context, userClusterClient ctrlruntimeclient.Client) (bool, error) {
	nodeList := &corev1.NodeList{}
	if err := userClusterClient.List(ctx, nodeList, &ctrlruntimeclient.ListOptions{}); err != nil {
		return false, err
	}
	if len(nodeList.Items) < 1 {
		return false, nil
	}
	return true, nil
}
