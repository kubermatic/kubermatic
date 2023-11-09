package seedclusterresources

import (
	"context"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretReconciler returns a function to create the Secret containing the backup destination credentials.
func SecretReconciler(ctx context.Context, client ctrlruntimeclient.Client, data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return cloudCredentialsSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			refName := data.ClusterBackupConfig().Destination.Credentials.Name
			refNamespace := data.ClusterBackupConfig().Destination.Credentials.Namespace

			secret := &corev1.Secret{}
			if err := client.Get(ctx, types.NamespacedName{Name: refName, Namespace: refNamespace}, secret); err != nil {
				return nil, fmt.Errorf("failed to get backup destination credentials secret: %w", err)
			}
			cm.Data = secret.Data
			return cm, nil
		}
	}
}
