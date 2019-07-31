package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	provider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (d *Deletion) cleanUpCredentialsSecrets(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// If no relevant finalizer exists, directly return
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.CredentialsSecretsCleanupFinalizer) {
		return nil
	}

	if err := d.deleteSecret(ctx, cluster); err != nil {
		return err
	}

	return d.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(c, kubermaticapiv1.CredentialsSecretsCleanupFinalizer)
	})
}

func (d *Deletion) deleteSecret(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	secretName := getSecretName(cluster)
	if secretName == "" {
		return nil
	}

	secret := &corev1.Secret{}
	err := d.seedClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: resources.KubermaticNamespace}, secret)
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to get Secret %s/%s: %v", resources.KubermaticNamespace, secretName, err)
	}

	if err == nil {
		if err := d.seedClient.Delete(ctx, secret); err != nil {
			return fmt.Errorf("failed to delete Secret '%s/%s': %v", secret.Namespace, secret.Name, err)
		}
	}
	return nil
}

func getSecretName(cluster *kubermaticv1.Cluster) string {
	if cluster.Spec.Cloud.Packet != nil {
		return fmt.Sprintf("%s-packet-%s", provider.CredentialPrefix, cluster.Name)
	}
	return ""
}
