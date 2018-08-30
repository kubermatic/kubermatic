package cluster

import (
	"crypto/x509"
	"fmt"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/cert"
	certutil "k8s.io/client-go/util/cert"
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

// Creates cluster-info ConfigMap in customer cluster
//see https://kubernetes.io/docs/admin/bootstrap-tokens/
func (cc *Controller) launchingCreateClusterInfoConfigMap(c *kubermaticv1.Cluster) error {
	caKp, err := resources.GetClusterRootCA(c, cc.secretLister)
	if err != nil {
		return err
	}

	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	name := "cluster-info"
	_, err = client.CoreV1().ConfigMaps(metav1.NamespacePublic).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			config := clientcmdapi.Config{}
			config.Clusters = map[string]*clientcmdapi.Cluster{
				"": {
					Server: c.Address.URL,
					CertificateAuthorityData: cert.EncodeCertPEM(caKp.Cert),
				},
			}
			cm := v1.ConfigMap{}
			cm.Name = name
			bconfig, err := clientcmd.Write(config)
			if err != nil {
				return fmt.Errorf("failed to encode kubeconfig: %v", err)
			}
			cm.Data = map[string]string{"kubeconfig": string(bconfig)}
			_, err = client.CoreV1().ConfigMaps(metav1.NamespacePublic).Create(&cm)
			if err != nil {
				return fmt.Errorf("failed to create cluster-info configmap in client cluster: %v", err)
			}
		} else {
			return fmt.Errorf("failed to load cluster-info configmap from client cluster: %v", err)
		}
	}

	_, err = client.RbacV1beta1().Roles(metav1.NamespacePublic).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			role := &v1beta1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Rules: []v1beta1.PolicyRule{
					{
						APIGroups:     []string{""},
						ResourceNames: []string{name},
						Resources:     []string{"configmaps"},
						Verbs:         []string{"get"},
					},
				},
			}
			if _, err = client.RbacV1beta1().Roles(metav1.NamespacePublic).Create(role); err != nil {
				return fmt.Errorf("failed to create cluster-info role")
			}
		} else {
			return fmt.Errorf("failed to load cluster-info role from client cluster: %v", err)
		}
	}

	_, err = client.RbacV1beta1().RoleBindings(metav1.NamespacePublic).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			rolebinding := &v1beta1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				RoleRef: v1beta1.RoleRef{
					Name:     name,
					Kind:     "Role",
					APIGroup: v1beta1.GroupName,
				},
				Subjects: []v1beta1.Subject{
					{
						APIGroup: v1beta1.GroupName,
						Kind:     v1beta1.UserKind,
						Name:     "system:anonymous",
					},
				},
			}
			if _, err = client.RbacV1beta1().RoleBindings(metav1.NamespacePublic).Create(rolebinding); err != nil {
				return fmt.Errorf("failed to create cluster-info rolebinding")
			}
		} else {
			return fmt.Errorf("failed to load cluster-info rolebinding from client cluster: %v", err)
		}
	}

	return nil
}

func (cc *Controller) launchingCreateOpenVPNClientCertificates(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	name := "openvpn-client-certificates"
	_, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			caKp, err := resources.GetClusterRootCA(c, cc.secretLister)
			if err != nil {
				return err
			}

			clientKey, err := certutil.NewPrivateKey()
			if err != nil {
				return fmt.Errorf("unable to create a server private key: %v", err)
			}

			clientConfig := certutil.Config{
				CommonName: "user-cluster-client",
				AltNames:   certutil.AltNames{},
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}
			clientCert, err := certutil.NewSignedCert(clientConfig, clientKey, caKp.Cert, caKp.Key)
			if err != nil {
				return fmt.Errorf("unable to sign the server certificate: %v", err)
			}

			secret := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				StringData: map[string]string{
					"ca.crt":     string(certutil.EncodeCertPEM(caKp.Cert)),
					"client.crt": string(certutil.EncodeCertPEM(clientCert)),
					"client.key": string(certutil.EncodePrivateKeyPEM(clientKey)),
				},
			}
			_, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Create(&secret)
			if err != nil {
				return fmt.Errorf("failed to create openvpn secret: %v", err)
			}
		} else {
			return fmt.Errorf("failed to load openvpn secret from client cluster: %v", err)
		}
	}

	return nil
}
