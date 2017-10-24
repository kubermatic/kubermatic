package cluster

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	extensionv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nodeDeletionFinalizer         = "kubermatic.io/delete-nodes"
	cloudProviderCleanupFinalizer = "kubermatic.io/cleanup-cloud-provider"
	namespaceDeletionFinalizer    = "kubermatic.io/delete-ns"
)

func (cc *controller) syncPendingCluster(c *kubermaticv1.Cluster) (changedC *kubermaticv1.Cluster, err error) {
	if c.Spec.MasterVersion == "" {
		c.Spec.MasterVersion = cc.defaultMasterVersion.ID
	}

	//Every function with the prefix 'pending' *WILL* modify the cluster struct and cause an update
	//Every function with the prefix 'launching' *WONT* modify the cluster struct and should not cause an update
	// Setup required infrastructure at cloud provider
	changedC, err = cc.pendingInitializeCloudProvider(c)
	if err != nil {
		return nil, err
	}

	// Add finalizers
	changedC, err = cc.pendingRegisterFinalizers(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// Set the hostname & url
	changedC, err = cc.pendingCreateAddresses(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// Generate the kubelet and admin token
	changedC, err = cc.pendingCreateTokens(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// Create the root ca
	changedC, err = cc.pendingCreateRootCA(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// Create the certificates
	changedC, err = cc.pendingCreateCertificates(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// Create the service account key
	changedC, err = cc.pendingCreateServiceAccountKey(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// Create the ssh keys for the apiserver
	changedC, err = cc.pendingCreateApiserverSSHKeys(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// Create the namespace
	err = cc.launchingCreateNamespace(c)
	if err != nil {
		return nil, err
	}

	// Create secret for user tokens
	err = cc.launchingCheckTokenUsers(c)
	if err != nil {
		return nil, err
	}

	// check that all service accounts are created
	err = cc.launchingCheckServiceAccounts(c)
	if err != nil {
		return nil, err
	}

	// check that all role bindings are created
	err = cc.launchingCheckClusterRoleBindings(c)
	if err != nil {
		return nil, err
	}

	// check that all services are available
	err = cc.launchingCheckServices(c)
	if err != nil {
		return nil, err
	}

	err = cc.launchingCheckSecrets(c)
	if err != nil {
		return nil, err
	}

	err = cc.launchingCheckConfigMaps(c)
	if err != nil {
		return nil, err
	}

	// check that all deployments are available
	err = cc.launchingCheckDeployments(c)
	if err != nil {
		return nil, err
	}

	// check that all deployments are available
	err = cc.launchingCheckIngress(c)
	if err != nil {
		return nil, err
	}

	// check that all deployments are available
	err = cc.launchingCheckEtcdCluster(c)
	if err != nil {
		return nil, err
	}

	c.Status.LastTransitionTime = metav1.Now()
	c.Status.Phase = kubermaticv1.LaunchingClusterStatusPhase
	return c, nil
}

func (cc *controller) pendingInitializeCloudProvider(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	_, prov, err := provider.ClusterCloudProvider(cc.cps, cluster)
	if err != nil {
		return nil, err
	}

	cloud, err := prov.Initialize(cluster.Spec.Cloud, cluster.Name)
	if err != nil {
		return nil, err
	}
	cluster.Spec.Cloud = cloud
	return cluster, nil
}

// launchingCreateNamespace will create the cluster namespace
func (cc *controller) launchingCreateNamespace(c *kubermaticv1.Cluster) error {
	_, err := cc.seedInformerGroup.NamespaceInformer.Lister().Get(c.Status.NamespaceName)
	if !errors.IsNotFound(err) {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Status.NamespaceName,
		},
	}
	_, err = cc.client.CoreV1().Namespaces().Create(ns)
	return err
}

// pendingRegisterFinalizers adds all finalizers we need
func (cc *controller) pendingRegisterFinalizers(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var updated bool

	finalizers := []string{
		nodeDeletionFinalizer,
		cloudProviderCleanupFinalizer,
		namespaceDeletionFinalizer,
	}

	for _, f := range finalizers {
		if !kuberneteshelper.HasFinalizer(c, f) {
			c.Finalizers = append(c.Finalizers, f)
			updated = true
		}
	}

	if updated {
		glog.V(4).Infof("Added finalizers to cluster %s", c.Name)
		return c, nil
	}
	return nil, nil
}

// pendingCreateAddresses will set the cluster hostname and the url under which the apiserver will be reachable
func (cc *controller) pendingCreateAddresses(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var updated bool

	if c.Address.ExternalName == "" {
		c.Address.ExternalName = fmt.Sprintf("%s.%s.%s", c.Name, cc.dc, cc.externalURL)
		updated = true
	}

	if c.Address.ExternalPort == 0 {
		c.Address.ExternalPort = cc.apiserverExternalPort
		updated = true
	}

	if c.Address.URL == "" {
		c.Address.URL = fmt.Sprintf("https://%s:%d", c.Address.ExternalName, c.Address.ExternalPort)
		updated = true
	}

	if updated {
		glog.V(4).Infof("Set address for cluster %s to %s", c.Name, c.Address.URL)
		return c, nil
	}
	return nil, nil
}

func (cc *controller) launchingCheckSecrets(c *kubermaticv1.Cluster) error {
	secrets := map[string]func(c *kubermaticv1.Cluster, app, masterResourcesPath string) (*corev1.Secret, error){
		"apiserver":          resources.LoadSecretFile,
		"controller-manager": resources.LoadSecretFile,
	}

	ns := c.Status.NamespaceName
	for s, gen := range secrets {
		_, err := cc.seedInformerGroup.SecretInformer.Lister().Secrets(ns).Get(s)
		if !errors.IsNotFound(err) {
			return err
		}

		secret, err := gen(c, s, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate %s: %v", s, err)
		}

		_, err = cc.client.CoreV1().Secrets(ns).Create(secret)
		if err != nil {
			return fmt.Errorf("failed to create secret for %s: %v", s, err)
		}
	}

	return nil
}

func (cc *controller) launchingCheckServices(c *kubermaticv1.Cluster) error {
	services := map[string]func(c *kubermaticv1.Cluster, app, masterResourcesPath string) (*corev1.Service, error){
		"apiserver": resources.LoadServiceFile,
	}

	ns := c.Status.NamespaceName
	for s, gen := range services {
		_, err := cc.seedInformerGroup.ServiceInformer.Lister().Services(ns).Get(s)
		if !errors.IsNotFound(err) {
			return err
		}

		service, err := gen(c, s, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate service %s: %v", s, err)
		}

		_, err = cc.client.CoreV1().Services(ns).Create(service)
		if err != nil {
			return fmt.Errorf("failed to create service %s: %v", s, err)
		}
	}

	return nil
}

func (cc *controller) launchingCheckServiceAccounts(c *kubermaticv1.Cluster) error {
	serviceAccounts := map[string]func(app, masterResourcesPath string) (*corev1.ServiceAccount, error){
		"etcd-operator": resources.LoadServiceAccountFile,
	}

	ns := c.Status.NamespaceName
	for s, gen := range serviceAccounts {
		_, err := cc.seedInformerGroup.ServiceAccountInformer.Lister().ServiceAccounts(ns).Get(s)
		if !errors.IsNotFound(err) {
			return err
		}

		sa, err := gen(s, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate service account %s: %v", s, err)
		}

		_, err = cc.client.CoreV1().ServiceAccounts(ns).Create(sa)
		if err != nil {
			return fmt.Errorf("failed to create service account %s: %v", s, err)
		}
	}

	return nil
}

func (cc *controller) launchingCheckTokenUsers(c *kubermaticv1.Cluster) error {
	ns := c.Status.NamespaceName
	name := "token-users"
	_, err := cc.seedInformerGroup.SecretInformer.Lister().Secrets(ns).Get(name)
	if !errors.IsNotFound(err) {
		return err
	}

	buffer := bytes.Buffer{}
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{c.Address.KubeletToken, "kubelet-bootstrap", "10001", "system:bootstrappers"}); err != nil {
		return err
	}
	if err := writer.Write([]string{c.Address.AdminToken, "admin", "10000", "system:masters"}); err != nil {
		return err
	}
	writer.Flush()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tokens.csv": buffer.Bytes(),
		},
	}

	_, err = cc.client.CoreV1().Secrets(ns).Create(secret)
	if err != nil {
		return fmt.Errorf("failed to create user token secret: %v", err)
	}
	return nil
}

func (cc *controller) launchingCheckClusterRoleBindings(c *kubermaticv1.Cluster) error {
	roleBindings := map[string]func(namespace, app, masterResourcesPath string) (*rbacv1beta1.ClusterRoleBinding, error){
		"etcd-operator": resources.LoadClusterRoleBindingFile,
	}

	ns := c.Status.NamespaceName
	for s, gen := range roleBindings {
		binding, err := gen(ns, s, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate cluster role binding %s: %v", s, err)
		}

		_, err = cc.seedInformerGroup.ClusterRoleBindingInformer.Lister().Get(binding.ObjectMeta.Name)
		if !errors.IsNotFound(err) {
			return err
		}

		_, err = cc.client.RbacV1beta1().ClusterRoleBindings().Create(binding)
		if err != nil {
			return fmt.Errorf("failed to create cluster role binding %s: %v", s, err)
		}
	}

	return nil
}

func (cc *controller) launchingCheckDeployments(c *kubermaticv1.Cluster) error {
	ns := c.Status.NamespaceName
	masterVersion, found := cc.versions[c.Spec.MasterVersion]
	if !found {
		return fmt.Errorf("unknown new cluster %q master version %q", c.Name, c.Spec.MasterVersion)
	}

	deps := map[string]string{
		"etcd-operator":      masterVersion.EtcdOperatorDeploymentYaml,
		"apiserver":          masterVersion.ApiserverDeploymentYaml,
		"controller-manager": masterVersion.ControllerDeploymentYaml,
		"scheduler":          masterVersion.SchedulerDeploymentYaml,
		"node-controller":    masterVersion.NodeControllerDeploymentYaml,
		"addon-manager":      masterVersion.AddonManagerDeploymentYaml,
	}

	for name, yamlFile := range deps {
		dep, err := resources.LoadDeploymentFile(c, masterVersion, cc.masterResourcesPath, cc.dc, yamlFile)
		if err != nil {
			return fmt.Errorf("failed to generate deployment %q: %v", name, err)
		}

		_, err = cc.seedInformerGroup.DeploymentInformer.Lister().Deployments(ns).Get(name)
		if !errors.IsNotFound(err) {
			return err
		}

		_, err = cc.client.ExtensionsV1beta1().Deployments(ns).Create(dep)
		if err != nil {
			return fmt.Errorf("failed to create deployment %q: %v", name, err)
		}
	}

	return nil
}

func (cc *controller) launchingCheckConfigMaps(c *kubermaticv1.Cluster) error {
	ns := c.Status.NamespaceName

	var dc *provider.DatacenterMeta
	cms := map[string]func(c *kubermaticv1.Cluster, datacenter *provider.DatacenterMeta) (*corev1.ConfigMap, error){}
	if c.Spec.Cloud != nil {
		cdc, found := cc.dcs[c.Spec.Cloud.DatacenterName]
		if !found {
			return fmt.Errorf("invalid datacenter %q", c.Spec.Cloud.DatacenterName)
		}
		dc = &cdc

		if c.Spec.Cloud.AWS != nil {
			cms["cloud-config"] = resources.LoadAwsCloudConfigConfigMap
		}
		if c.Spec.Cloud.Openstack != nil {
			cms["cloud-config"] = resources.LoadOpenstackCloudConfigConfigMap
		}
	}

	for s, gen := range cms {
		_, err := cc.seedInformerGroup.ConfigMapInformer.Lister().ConfigMaps(ns).Get(s)
		if !errors.IsNotFound(err) {
			return err
		}

		cm, err := gen(c, dc)
		if err != nil {
			return fmt.Errorf("failed to generate cm %s: %v", s, err)
		}

		_, err = cc.client.CoreV1().ConfigMaps(ns).Create(cm)
		if err != nil {
			return fmt.Errorf("failed to create cm %s; %v", s, err)
		}
	}

	return nil
}

func (cc *controller) launchingCheckIngress(c *kubermaticv1.Cluster) error {
	ingress := map[string]func(c *kubermaticv1.Cluster, app, masterResourcesPath string) (*extensionv1beta1.Ingress, error){
		"apiserver": resources.LoadIngressFile,
	}

	ns := c.Status.NamespaceName
	for s, gen := range ingress {
		_, err := cc.seedInformerGroup.IngressInformer.Lister().Ingresses(ns).Get(s)
		if !errors.IsNotFound(err) {
			return err
		}

		ingress, err := gen(c, s, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate %s: %v", s, err)
		}

		_, err = cc.client.ExtensionsV1beta1().Ingresses(ns).Create(ingress)
		if err != nil {
			return fmt.Errorf("failed to create ingress %s: %v", s, err)
		}
	}
	return nil
}

func (cc *controller) launchingCheckEtcdCluster(c *kubermaticv1.Cluster) error {
	ns := c.Status.NamespaceName
	masterVersion, found := cc.versions[c.Spec.MasterVersion]
	if !found {
		return fmt.Errorf("unknown new cluster %q master version %q", c.Name, c.Spec.MasterVersion)
	}

	etcd, err := resources.LoadEtcdClusterFile(masterVersion, cc.masterResourcesPath, masterVersion.EtcdClusterYaml)
	if err != nil {
		return fmt.Errorf("failed to load etcd-cluster: %v", err)
	}

	_, err = cc.seedInformerGroup.EtcdClusterInformer.Lister().EtcdClusters(ns).Get(etcd.ObjectMeta.Name)
	if !errors.IsNotFound(err) {
		return err
	}

	_, err = cc.seedCrdClient.EtcdV1beta2().EtcdClusters(ns).Create(etcd)
	if err != nil {
		return fmt.Errorf("failed to create etcd-cluster definition (crd): %v", err)
	}

	return nil
}
