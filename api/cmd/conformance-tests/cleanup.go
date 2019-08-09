package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/node/eviction"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	protectedNamespaces = sets.NewString(metav1.NamespaceDefault, metav1.NamespaceSystem, metav1.NamespacePublic, corev1.NamespaceNodeLease)
)

func deleteAllNonDefaultNamespaces(log *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	return wait.Poll(defaultUserClusterPollInterval, defaultTimeout, func() (done bool, err error) {
		namespaceList := &corev1.NamespaceList{}
		ctx := context.Background()
		if err := client.List(ctx, &ctrlruntimeclient.ListOptions{}, namespaceList); err != nil {
			log.Errorf("failed to list namespaces: %v", err)
			return false, nil
		}

		// This check assumes no one deleted one of the protected namespaces
		if len(namespaceList.Items) <= protectedNamespaces.Len() {
			return true, nil
		}

		for _, namespace := range namespaceList.Items {
			if protectedNamespaces.Has(namespace.Name) {
				continue
			}
			log = log.With("namespace-to-delete", namespace.Name)

			// If its not gone & the DeletionTimestamp is nil, delete it
			if namespace.DeletionTimestamp == nil {
				if err := client.Delete(ctx, &namespace); err != nil {
					log.Errorf("Failed to delete namespace: %v", err)
				} else {
					log.Debugf("Called delete on namespace")
				}
			}
		}
		return false, nil
	})
}

func tryToDeleteClusterWithRetries(log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, clusterClientProvider clusterclient.UserClusterConnectionProvider, seedClusterClient ctrlruntimeclient.Client) error {
	const maxAttempts = 5
	return retryNAttempts(maxAttempts, func(attempt int) error {
		err := tryToDeleteCluster(log, cluster, clusterClientProvider, seedClusterClient)
		if err != nil {
			log.Warnf("[Attempt %d/%d] Failed to delete Cluster: %v", attempt, maxAttempts, err)
		}
		return err
	})
}

// tryToDeleteCluster will try to delete all potential orphaned cloud provider resources like LB's & PVC's
// After deleting them it will delete the kubermatic cluster object
func tryToDeleteCluster(log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, clusterClientProvider clusterclient.UserClusterConnectionProvider, seedClusterClient ctrlruntimeclient.Client) error {
	log.Infof("Trying to delete cluster...")

	userClusterClient, err := clusterClientProvider.GetClient(cluster)
	if err != nil {
		return fmt.Errorf("failed to get the client for the cluster: %v", err)
	}

	if err := deleteAllNonDefaultNamespaces(log, userClusterClient); err != nil {
		return fmt.Errorf("failed to delete all namespaces: %v", err)
	}

	// Disable eviction on all nodes
	nodeList := &corev1.NodeList{}
	ctx := context.Background()
	if err := userClusterClient.List(ctx, &ctrlruntimeclient.ListOptions{}, nodeList); err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	for _, node := range nodeList.Items {
		log.Debugf("Disabling eviction on node '%s' ...", node.Name)
		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// Get latest version of the node to prevent conflict errors
			currentNode := &corev1.Node{}
			if err := userClusterClient.Get(ctx, types.NamespacedName{Name: node.Name}, currentNode); err != nil {
				return err
			}
			if currentNode.Annotations == nil {
				currentNode.Annotations = map[string]string{}
			}
			currentNode.Annotations[eviction.SkipEvictionAnnotationKey] = "true"

			return userClusterClient.Update(ctx, currentNode)
		})
		if err != nil {
			return fmt.Errorf("failed to add the annotation '%s=true' to node '%s': %v", eviction.SkipEvictionAnnotationKey, node.Name, err)
		}
	}

	log.Debug("Calling delete on the cluster resource...")
	if err := seedClusterClient.Delete(context.Background(), cluster); err != nil {
		return fmt.Errorf("failed to delete cluster %s: %v. Make sure to delete it manually afterwards", cluster.Name, err)
	}

	log.Info("Finished deleting cluster")
	return nil
}
