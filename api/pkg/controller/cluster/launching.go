package cluster

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	controllerresources "github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func (cc *Controller) clusterHealth(c *kubermaticv1.Cluster) (bool, *kubermaticv1.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(c.Name)
	health := kubermaticv1.ClusterHealth{
		ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{},
	}

	type depInfo struct {
		healthy  *bool
		minReady int32
	}

	healthMapping := map[string]*depInfo{
		controllerresources.ApiserverDeploymenName:          {healthy: &health.Apiserver, minReady: 1},
		controllerresources.ControllerManagerDeploymentName: {healthy: &health.Controller, minReady: 1},
		controllerresources.SchedulerDeploymentName:         {healthy: &health.Scheduler, minReady: 1},
		controllerresources.NodeControllerDeploymentName:    {healthy: &health.NodeController, minReady: 1},
		controllerresources.MachineControllerDeploymentName: {healthy: &health.MachineController, minReady: 1},
	}

	for name := range healthMapping {
		healthy, err := cc.healthyDeployment(ns, name, healthMapping[name].minReady)
		if err != nil {
			return false, nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthy = healthy
	}

	var err error
	health.Etcd, err = cc.healthyEtcd(ns, resources.EtcdClusterName)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get etcd health: %v", err)
	}

	return health.AllHealthy(), &health, nil
}

// ensureClusterReachable checks if the cluster is reachable via its external name
func (cc *Controller) ensureClusterReachable(c *kubermaticv1.Cluster) error {
	client, err := c.GetClient()
	if err != nil {
		return err
	}
	_, err = client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		glog.V(5).Infof("Cluster %q not yet reachable: %v", c.Name, err)
		return nil
	}

	// Only add the node deletion finalizer when the cluster is actually running
	// Otherwise we fail to delete the nodes and are stuck in a loop
	if !kuberneteshelper.HasFinalizer(c, nodeDeletionFinalizer) {
		c.Finalizers = append(c.Finalizers, nodeDeletionFinalizer)
	}

	return nil
}

// Creates cluster-info ConfigMap in customer cluster
//see https://kubernetes.io/docs/admin/bootstrap-tokens/
func (cc *Controller) launchingCreateClusterInfoConfigMap(c *kubermaticv1.Cluster) error {
	client, err := c.GetClient()
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
					CertificateAuthorityData: c.Status.RootCA.Cert,
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
					APIGroup: "rbac.authorization.k8s.io",
				},
				Subjects: []v1beta1.Subject{
					{
						APIGroup: "rbac.authorization.k8s.io",
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

// Creates secret containing a public key. Used by the apiserver bridge server
func (cc *Controller) launchingCreateApiserverBridgePublicKeySecret(c *kubermaticv1.Cluster) error {
	client, err := c.GetClient()
	if err != nil {
		return err
	}

	name := "apiserver-bridge-server-authorized-keys"
	_, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			secret := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				StringData: map[string]string{
					"authorized_keys": string(c.Status.ApiserverSSHKey.PublicKey),
				},
			}
			_, err = client.CoreV1().Secrets(metav1.NamespaceSystem).Create(&secret)
			if err != nil {
				return fmt.Errorf("failed to create public key secret in client cluster: %v", err)
			}
		} else {
			return fmt.Errorf("failed to load public key secret from client cluster: %v", err)
		}
	}

	return nil
}
