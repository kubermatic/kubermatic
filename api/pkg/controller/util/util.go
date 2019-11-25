package util

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EnqueueClusterForNamespacedObject enqueues the cluster that owns a namespaced object, if any
// It is used by various controllers to react to changes in the resources in the cluster namespace
func EnqueueClusterForNamespacedObject(client ctrlruntimeclient.Client) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		if err := client.List(context.Background(), clusterList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %v", err))
			return []reconcile.Request{}
		}
		for _, cluster := range clusterList.Items {
			if cluster.Status.NamespaceName == a.Meta.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}
		return []reconcile.Request{}
	})}
}

// EnqueueClusterForNamespacedObjectWithSeedName enqueues the cluster that owns a namespaced object,
// if any. The seedName is put into the namespace field
// It is used by various controllers to react to changes in the resources in the cluster namespace
func EnqueueClusterForNamespacedObjectWithSeedName(client ctrlruntimeclient.Client, seedName string) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		if err := client.List(context.Background(), clusterList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %v", err))
			return []reconcile.Request{}
		}
		for _, cluster := range clusterList.Items {
			if cluster.Status.NamespaceName == a.Meta.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{
					Namespace: seedName,
					Name:      cluster.Name,
				}}}
			}
		}
		return []reconcile.Request{}
	})}
}

// EnqueueClusterScopedObjectWithSeedName enqueues a cluster-scoped object with the seedName
// as namespace. If it gets an object with a non-empty name, it will log an error and not enqueue
// anything.
func EnqueueClusterScopedObjectWithSeedName(seedName string) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		if a.Meta.GetNamespace() != "" {
			utilruntime.HandleError(fmt.Errorf("EnqueueClusterScopedObjectWithSeedName was used with namespace scoped object %s/%s of type %T", a.Meta.GetNamespace(), a.Meta.GetName(), a.Object))
		}
		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Namespace: seedName,
			Name:      a.Meta.GetName(),
		}}}
	})}
}

// EnqueueConst enqueues a constant. It is meant for controllers that don't have a parent object
// they could enc and instead reconcile everything at once.
// The queueKey will be defaulted if empty
func EnqueueConst(queueKey string) *handler.EnqueueRequestsFromMapFunc {
	if queueKey == "" {
		queueKey = "const"
	}

	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Name:      queueKey,
				Namespace: "",
			}}}
	})}
}

// SupportsFailureDomainZoneAntiAffinity checks if there are any nodes with the
// TopologyKeyFailureDomainZone label.
func SupportsFailureDomainZoneAntiAffinity(ctx context.Context, client ctrlruntimeclient.Client) (bool, error) {
	selector, err := labels.Parse(resources.TopologyKeyFailureDomainZone)
	if err != nil {
		return false, fmt.Errorf("failed to parse selector: %v", err)
	}
	opts := &ctrlruntimeclient.ListOptions{
		LabelSelector: selector,
		Raw: &metav1.ListOptions{
			Limit: 1,
		},
	}

	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, nodeList, opts); err != nil {
		return false, fmt.Errorf("failed to list nodes having the %s label: %v", resources.TopologyKeyFailureDomainZone, err)
	}

	return len(nodeList.Items) != 0, nil
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
		return true, fmt.Errorf("failed to list clusters: %v", err)
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
