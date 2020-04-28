package main

import (
	"context"

	"go.uber.org/zap"

	kubernetesdashboard "github.com/kubermatic/kubermatic/api/pkg/resources/kubernetes-dashboard"

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
		kubernetesdashboard.Namespace)
)

func deleteAllNonDefaultNamespaces(log *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	log.Info("Removing non-default namespaces...")

	return wait.Poll(defaultUserClusterPollInterval, defaultTimeout, func() (done bool, err error) {
		namespaceList := &corev1.NamespaceList{}
		ctx := context.Background()
		if err := client.List(ctx, namespaceList); err != nil {
			log.Errorw("Failed to list namespaces", zap.Error(err))
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

			// If it's not gone & the DeletionTimestamp is nil, delete it
			if namespace.DeletionTimestamp == nil {
				// make sure to create a new variable, or else subsequent With() calls will
				// *add* new attributes instead of overriding the existing namespace value
				log := log.With("namespace", namespace.Name)

				if err := client.Delete(ctx, &namespace); err != nil {
					log.Errorw("Failed to delete namespace", zap.Error(err))
				} else {
					log.Debug("Deleted namespace.")
				}
			}
		}

		return false, nil
	})
}
