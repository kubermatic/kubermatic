package util

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		if err := client.List(context.Background(), &ctrlruntimeclient.ListOptions{}, clusterList); err != nil {
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
	opts := &ctrlruntimeclient.ListOptions{
		Raw: &metav1.ListOptions{
			Limit: 1,
		},
	}
	if err := opts.SetLabelSelector(resources.TopologyKeyFailureDomainZone); err != nil {
		return false, fmt.Errorf("failed to set label selector: %v", err)
	}

	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, opts, nodeList); err != nil {
		return false, fmt.Errorf("failed to list nodes having the %s label: %v", resources.TopologyKeyFailureDomainZone, err)
	}

	return len(nodeList.Items) != 0, nil
}

// ClusterAvailableForReconciling returns true if the given cluster can be reconciled. This is true if
// the cluster does not yet have the SeedResourcesUpToDate condition or if the concurrency limit of the
// controller is not yet reached. This ensures that not too many cluster updates are running at the same
// time, but also makes sure that un-UpToDate clusters will continue to be reconciled.
func ClusterAvailableForReconciling(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, concurrencyLimit int) (bool, error) {
	if !cluster.Status.HasConditionValue(kubermaticv1.SeedResourcesUpToDate, corev1.ConditionTrue) {
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
	if err := client.List(ctx, &ctrlruntimeclient.ListOptions{}, clusters); err != nil {
		return true, fmt.Errorf("failed to list clusters: %v", err)
	}

	finishedUpdatingClustersCount := 0
	for _, cluster := range clusters.Items {
		if cluster.Status.HasConditionValue(kubermaticv1.SeedResourcesUpToDate, corev1.ConditionTrue) {
			finishedUpdatingClustersCount++
		}
	}

	clustersUpdatingInProgressCount := len(clusters.Items) - finishedUpdatingClustersCount

	return clustersUpdatingInProgressCount >= limit, nil
}

// SetSeedResourcesUpToDateCondition updates the cluster status condition based on the deployment and statefulSet
// replicas. if both statefulSet and deployment spec replica are equal to all replicas in the status object, then the
// SeedResourcesUpToDate, will be added to the cluster conditions.
func SetSeedResourcesUpToDateCondition(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client, successfullyReconciled bool) (hasUnfinishedUpdates bool, err error) {
	if successfullyReconciled {
		var (
			statefulSets = &appv1.StatefulSetList{}
			opts         = &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
			deployments  = &appv1.DeploymentList{}
		)

		if err := client.List(ctx, opts, statefulSets); err != nil {
			return hasUnfinishedUpdates, fmt.Errorf("failed to list statefulSets: %v", err)
		}

		for _, statefulSet := range statefulSets.Items {
			if statefulSet.Spec.Replicas != nil {
				if *statefulSet.Spec.Replicas != statefulSet.Status.UpdatedReplicas ||
					*statefulSet.Spec.Replicas != statefulSet.Status.CurrentReplicas ||
					*statefulSet.Spec.Replicas != statefulSet.Status.ReadyReplicas {
					hasUnfinishedUpdates = true
				}
			}
		}

		if err := client.List(ctx, opts, deployments); err != nil {
			return hasUnfinishedUpdates, fmt.Errorf("failed to list deployments: %v", err)
		}

		for _, deployment := range deployments.Items {
			if deployment.Spec.Replicas != nil {
				if *deployment.Spec.Replicas != deployment.Status.UpdatedReplicas ||
					*deployment.Spec.Replicas != deployment.Status.AvailableReplicas ||
					*deployment.Spec.Replicas != deployment.Status.ReadyReplicas {
					hasUnfinishedUpdates = true
				}
			}
		}

		if !hasUnfinishedUpdates {
			cluster.Status.
				SetClusterFinishedUpdatingSuccessfullyCondition(fmt.Sprintf("cluster %v has finished updating resources successfully",
					cluster.Name))
			return hasUnfinishedUpdates, client.Update(ctx, cluster)
		}
	}

	cluster.Status.ClearCondition(kubermaticv1.SeedResourcesUpToDate)
	return hasUnfinishedUpdates, client.Update(ctx, cluster)
}
