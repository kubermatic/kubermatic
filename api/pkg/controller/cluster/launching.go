package cluster

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// clusterIsReachable checks if the cluster is reachable via its external name
func (r *Reconciler) clusterIsReachable(c *kubermaticv1.Cluster) (bool, error) {
	client, err := r.userClusterConnProvider.GetClient(c)
	if err != nil {
		return false, err
	}

	_, err = client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		glog.V(5).Infof("Cluster %q not yet reachable: %v", c.Name, err)
		return false, nil
	}

	return true, nil
}

func (r *Reconciler) launchingCreateOpenVPNClientCertificates(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	client, err := r.userClusterConnProvider.GetClient(cluster)
	if err != nil {
		return err
	}

	// Pull from the seed cluster
	caKp, err := resources.GetOpenVPNCA(ctx, cluster, r)
	if err != nil {
		return err
	}

	createCertificate := openvpn.UserClusterClientCertificateCreator(caKp)

	existing, err := client.CoreV1().Secrets(metav1.NamespaceSystem).Get(resources.OpenVPNClientCertificatesSecretName, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		// Secret does not exist -> Create it
		secret, err := createCertificate(&corev1.Secret{})
		if err != nil {
			return fmt.Errorf("failed to build OpenVPN client key Secret: %v", err)
		}

		if _, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Create(secret); err != nil {
			return fmt.Errorf("failed to create Secret %s: %v", secret.Name, err)
		}
		return nil
	}

	// Secret already exists, see if we need to update it
	secret, err := createCertificate(existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build Secret: %v", err)
	}

	if equal := reconciling.DeepEqual(existing, secret); equal {
		return nil
	}

	if _, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Update(secret); err != nil {
		return fmt.Errorf("failed to update Secret %s: %v", secret.Name, err)
	}
	return nil
}
