package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanUpCredentialsSecrets(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if err := d.deleteSecret(ctx, cluster); err != nil {
		return err
	}

	oldCluster := cluster.DeepCopy()
	kuberneteshelper.RemoveFinalizer(cluster, kubermaticapiv1.CredentialsSecretsCleanupFinalizer)
	return d.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (d *Deletion) deleteSecret(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	secretName := cluster.GetSecretName()
	if secretName == "" {
		return nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: secretName, Namespace: resources.KubermaticNamespace}
	err := d.seedClient.Get(ctx, name, secret)
	// Its already gone
	if kerrors.IsNotFound(err) {
		return nil
	}

	// Something failed while loading the secret
	if err != nil {
		return fmt.Errorf("failed to get Secret %q: %v", name.String(), err)
	}

	if err := d.seedClient.Delete(ctx, secret); err != nil {
		return fmt.Errorf("failed to delete Secret %q: %v", name.String(), err)
	}

	// We successfully deleted the secret
	return nil
}
