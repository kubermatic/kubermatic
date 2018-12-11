package cluster

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster/resources/openvpn"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// clusterIsReachable checks if the cluster is reachable via its external name
func (cc *Controller) clusterIsReachable(c *kubermaticv1.Cluster) (bool, error) {
	client, err := cc.userClusterConnProvider.GetClient(c)
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

func (cc *Controller) launchingCreateOpenVPNClientCertificates(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	caKp, err := resources.GetOpenVPNCA(c, cc.secretLister)
	if err != nil {
		return err
	}

	existing, err := client.CoreV1().Secrets(metav1.NamespaceSystem).Get(resources.OpenVPNClientCertificatesSecretName, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}

		// Secret does not exist -> Create it
		secret, err := openvpn.UserClusterClientCertificate(nil, caKp)
		if err != nil {
			return fmt.Errorf("failed to build Secret %s: %v", secret.Name, err)
		}

		if _, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Create(secret); err != nil {
			return fmt.Errorf("failed to create Secret %s: %v", secret.Name, err)
		}
		return nil
	}

	// Secret already exists, see if we need to update it
	secret, err := openvpn.UserClusterClientCertificate(existing.DeepCopy(), caKp)
	if err != nil {
		return fmt.Errorf("failed to build Secret: %v", err)
	}

	if equal := resources.DeepEqual(existing, secret); equal {
		return nil
	}

	if _, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Update(secret); err != nil {
		return fmt.Errorf("failed to update Secret %s: %v", secret.Name, err)
	}
	return nil
}
