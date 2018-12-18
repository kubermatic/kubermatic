package main

import (
	"fmt"

	"github.com/sirupsen/logrus"

	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/node/eviction"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

var (
	protectedNamespaces = sets.NewString(metav1.NamespaceDefault, metav1.NamespaceSystem, metav1.NamespacePublic)
)

func deleteAllNonDefaultNamespaces(log *logrus.Entry, kubeClient kubernetes.Interface) error {
	return wait.Poll(defaultUserClusterPollInterval, defaultTimeout, func() (done bool, err error) {
		namespaceList, err := kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
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
			log = log.WithFields(logrus.Fields{"namespace-to-delete": namespace.Name})

			// If its not gone & the DeletionTimestamp is nil, delete it
			if namespace.DeletionTimestamp == nil {
				if err := kubeClient.CoreV1().Namespaces().Delete(namespace.Name, nil); err != nil {
					log.Errorf("Failed to delete namespace: %v", err)
				} else {
					log.Debugf("Called delete on namespace")
				}
			}
		}
		return false, nil
	})
}

func tryToDeleteClusterWithRetries(log *logrus.Entry, cluster *kubermaticv1.Cluster, clusterClientProvider *clusterclient.Provider, kubermaticClient kubermaticclientset.Interface) error {
	const maxAttempts = 5
	return retryNAttempts(maxAttempts, func(attempt int) error {
		err := tryToDeleteCluster(log, cluster, clusterClientProvider, kubermaticClient)
		if err != nil {
			log.Warnf("[Attempt %d/%d] Failed to delete Cluster: %v", attempt, maxAttempts, err)
		}
		return err
	})
}

// tryToDeleteCluster will try to delete all potential orphaned cloud provider resources like LB's & PVC's
// After deleting them it will delete the kubermatic cluster object
func tryToDeleteCluster(log *logrus.Entry, cluster *kubermaticv1.Cluster, clusterClientProvider *clusterclient.Provider, kubermaticClient kubermaticclientset.Interface) error {
	log.Infof("Trying to delete cluster...")

	kubeClient, err := clusterClientProvider.GetClient(cluster)
	if err != nil {
		return fmt.Errorf("failed to get the client for the cluster: %v", err)
	}

	if err := deleteAllNonDefaultNamespaces(log, kubeClient); err != nil {
		return fmt.Errorf("failed to delete all namespaces: %v", err)
	}

	// Disable eviction on all nodes
	nodes, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	for _, node := range nodes.Items {
		log.Debugf("Disabling eviction on node '%s' ...", node.Name)
		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// Get latest version of the node to prevent conflict errors
			currentNode, err := kubeClient.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if currentNode.Annotations == nil {
				currentNode.Annotations = map[string]string{}
			}
			node.Annotations[eviction.SkipEvictionAnnotationKey] = "true"

			currentNode, err = kubeClient.CoreV1().Nodes().Update(&node)
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to add the annotation '%s=true' to node '%s': %v", eviction.SkipEvictionAnnotationKey, node.Name, err)
		}
	}

	log.Debug("Calling delete on the cluster resource...")
	if err := kubermaticClient.KubermaticV1().Clusters().Delete(cluster.Name, &metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete cluster %s: %v. Make sure to delete it manually afterwards", cluster.Name, err)
	}

	log.Info("Finished deleting cluster")
	return nil
}
