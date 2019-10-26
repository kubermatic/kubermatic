package main

import (
	"context"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	protectedNamespaces = sets.NewString(
		metav1.NamespaceDefault,
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
		corev1.NamespaceNodeLease,
		"kubernetes-dashboard")
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

			// make sure to create a new variable, or else subsequent With() calls will
			// *add* new attributes instead of overriding the existing namespace-to-delete value
			log := log.With("namespace-to-delete", namespace.Name)

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
