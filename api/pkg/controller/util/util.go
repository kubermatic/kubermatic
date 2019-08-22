package util

import (
	"context"
	"fmt"
	"k8s.io/api/apps/v1"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
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

// LimitConcurrentUpdates checks all the clusters inside the seed cluster and checks for
// ClusterConditionControllerUpdateInProgress. If the count condition true for each cluster was greater than
// the limit, it should return false to skip the current cluster update.
func LimitConcurrentUpdates(ctx context.Context, client ctrlruntimeclient.Client, limit int) (bool, error) {
	clusters := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, nil, clusters); err != nil {
		return false, fmt.Errorf("failed to list clusters: %v", err)
	}

	conditionsCount := 0
	for _, cluster := range clusters.Items {
		for _, condition := range cluster.Status.Conditions {
			if condition.Type == kubermaticv1.ClusterConditionControllerUpdateInProgress &&
				condition.Status == corev1.ConditionTrue {
				conditionsCount++
			}
		}
	}

	if conditionsCount < limit {
		return true, nil
	}

	return false, nil
}

// CheckClusterResourcesUpdatingStatus checks the status of the clusters StatefulSet or Deployments. If those resources
// are not in a ready state the, it will return True to indicate that, the cluster resources are still in an updating phase
// otherwise it returns false which means the cluster has finished updating those resources.
func CheckClusterResourcesUpdatingStatus(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) error {
	var (
		statefulSets = &v1.StatefulSetList{}
		opts         = &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
		deployments  = &v1.DeploymentList{}
	)

	if err := client.List(ctx, opts, statefulSets); err != nil {
		return fmt.Errorf("failed to list statefulSets: %v", err)
	}

	for _, statefulSet := range statefulSets.Items {
		if statefulSet.Status.Replicas != statefulSet.Status.UpdatedReplicas ||
			statefulSet.Status.Replicas != statefulSet.Status.ReadyReplicas {
			cluster.Status.SetClusterUpdateInProgressConditionTrue(fmt.Sprintf("cluster %v resources updates are in progress",
				cluster.Name))
			return client.Update(ctx, cluster)
		}
	}

	if err := client.List(ctx, opts, deployments); err != nil {
		return fmt.Errorf("failed to list deployments: %v", err)
	}

	for _, deployment := range deployments.Items {
		if deployment.Status.Replicas != deployment.Status.UpdatedReplicas ||
			deployment.Status.Replicas != deployment.Status.ReadyReplicas {
			cluster.Status.SetClusterUpdateInProgressConditionTrue(fmt.Sprintf("cluster %v resources updates are in progress",
				cluster.Name))
			return client.Update(ctx, cluster)
		}
	}

	cluster.Status.ClearCondition(kubermaticv1.ClusterConditionControllerUpdateInProgress)

	return client.Update(ctx, cluster)
}
