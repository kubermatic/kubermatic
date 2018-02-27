package cluster

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	controllerresources "github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"

	corev1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	nodeDeletionFinalizer         = "kubermatic.io/delete-nodes"
	cloudProviderCleanupFinalizer = "kubermatic.io/cleanup-cloud-provider"
	namespaceDeletionFinalizer    = "kubermatic.io/delete-ns"

	minNodePort = 30000
	maxNodePort = 32767

	annotationPrefix            = "kubermatic.io/"
	lastAppliedConfigAnnotation = annotationPrefix + "last-applied-configuration"
)

func (cc *ClusterController) reconcileCluster(cluster *kubermaticv1.Cluster) error {
	if cluster.Spec.MasterVersion == "" {
		cluster.Spec.MasterVersion = cc.defaultMasterVersion.ID
	}

	// Create the namespace
	if err := cc.ensureNamespaceExists(cluster); err != nil {
		return err
	}

	// Setup required infrastructure at cloud provider
	if err := cc.ensureCloudProviderIsInitialize(cluster); err != nil {
		return err
	}

	// Set the hostname & url
	if err := cc.ensureAddress(cluster); err != nil {
		return err
	}

	// Generate the kubelet and admin token
	if err := cc.ensureTokens(cluster); err != nil {
		return err
	}

	// Create the root ca
	if err := cc.ensureRootCA(cluster); err != nil {
		return err
	}

	// Create the certificates
	if err := cc.ensureCertificates(cluster); err != nil {
		return err
	}

	// Create the service account key
	if err := cc.ensureCreateServiceAccountKey(cluster); err != nil {
		return err
	}

	// Create the ssh keys for the apiserver
	if err := cc.ensureApiserverSSHKeypair(cluster); err != nil {
		return err
	}

	// check that all service accounts are created
	if err := cc.ensureCheckServiceAccounts(cluster); err != nil {
		return err
	}

	// check that all role bindings are created
	if err := cc.ensureClusterRoleBindings(cluster); err != nil {
		return err
	}

	// check that all services are available
	if err := cc.ensureServices(cluster); err != nil {
		return err
	}

	// check that all secrets are available
	if err := cc.ensureSecrets(cluster); err != nil {
		return err
	}

	// check that all configmaps are available
	if err := cc.ensureConfigMaps(cluster); err != nil {
		return err
	}

	// check that all deployments are available
	if err := cc.ensureDeployments(cluster); err != nil {
		return err
	}

	// check that the etcd-cluster cr is available
	if err := cc.ensureEtcdCluster(cluster); err != nil {
		return err
	}

	allHealthy, health, err := cc.clusterHealth(cluster)
	if err != nil {
		return err
	}
	cluster.Status.Health = health

	if !allHealthy {
		glog.V(5).Infof("Cluster %q not yet healthy: %+v", cluster.Name, cluster.Status.Health)
		return nil
	}

	if err := cc.ensureClusterReachable(cluster); err != nil {
		return err
	}

	if err := cc.launchingCreateClusterInfoConfigMap(cluster); err != nil {
		return err
	}

	if cluster.Status.Phase == kubermaticv1.LaunchingClusterStatusPhase {
		cluster.Status.Phase = kubermaticv1.RunningClusterStatusPhase
	}

	return nil
}

func (cc *ClusterController) getClusterTemplateData(c *kubermaticv1.Cluster) (*controllerresources.TemplateData, error) {
	version, found := cc.versions[c.Spec.MasterVersion]
	if !found {
		return nil, fmt.Errorf("failed to get version %s", c.Spec.MasterVersion)
	}

	dc, found := cc.dcs[c.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", c.Spec.Cloud.DatacenterName)
	}

	return controllerresources.NewTemplateData(
		c,
		version,
		&dc,
		cc.SecretLister,
		cc.ConfigMapLister,
	), nil
}

func (cc *ClusterController) ensureCloudProviderIsInitialize(cluster *kubermaticv1.Cluster) error {
	_, prov, err := provider.ClusterCloudProvider(cc.cps, cluster)
	if err != nil {
		return err
	}

	cloud, err := prov.InitializeCloudProvider(cluster.Spec.Cloud, cluster.Name)
	if err != nil {
		return err
	}
	if cloud != nil {
		cluster.Spec.Cloud = cloud
	}

	if !kuberneteshelper.HasFinalizer(cluster, cloudProviderCleanupFinalizer) {
		cluster.Finalizers = append(cluster.Finalizers, cloudProviderCleanupFinalizer)
	}

	return nil
}

// ensureNamespaceExists will create the cluster namespace
func (cc *ClusterController) ensureNamespaceExists(c *kubermaticv1.Cluster) error {
	if _, err := cc.NamespaceLister.Get(c.Status.NamespaceName); !errors.IsNotFound(err) {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Status.NamespaceName,
		},
	}
	if _, err := cc.kubeClient.CoreV1().Namespaces().Create(ns); err != nil {
		return fmt.Errorf("failed to create namespace %s: %v", c.Status.NamespaceName, err)
	}

	if !kuberneteshelper.HasFinalizer(c, namespaceDeletionFinalizer) {
		c.Finalizers = append(c.Finalizers, namespaceDeletionFinalizer)
	}

	return nil
}

func (cc *ClusterController) getFreeNodePort() (int, error) {
	services, err := cc.ServiceLister.List(labels.Everything())
	if err != nil {
		return 0, err
	}
	allocatedPorts := map[int]struct{}{}

	for _, s := range services {
		for _, p := range s.Spec.Ports {
			if p.NodePort != 0 {
				allocatedPorts[int(p.NodePort)] = struct{}{}
			}
		}
	}

	for i := minNodePort; i < maxNodePort; i++ {
		if _, exists := allocatedPorts[i]; !exists {
			return i, nil
		}
	}

	return 0, fmt.Errorf("no free nodeport left")
}

// ensureAddress will set the cluster hostname and the url under which the apiserver will be reachable
func (cc *ClusterController) ensureAddress(c *kubermaticv1.Cluster) error {
	if c.Address.ExternalName == "" {
		c.Address.ExternalName = fmt.Sprintf("%s.%s.%s", c.Name, cc.dc, cc.externalURL)
	}

	if c.Address.ExternalPort == 0 {
		port, err := cc.getFreeNodePort()
		if err != nil {
			return fmt.Errorf("failed to get nodeport: %v", err)
		}
		c.Address.ExternalPort = port
	}

	if c.Address.URL == "" {
		c.Address.URL = fmt.Sprintf("https://%s:%d", c.Address.ExternalName, c.Address.ExternalPort)
	}

	//Always update the ip
	ips, err := net.LookupIP(c.Address.ExternalName)
	if err != nil {
		return fmt.Errorf("failed to lookup ip address for %s: %v", c.Address.ExternalName, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("no ip addresses found for %s: %v", c.Address.ExternalName, err)
	}
	c.Address.IP = ips[0].String()

	return nil
}

func getPatch(currentObj, updateObj metav1.Object) ([]byte, error) {
	currentData, err := json.Marshal(currentObj)
	if err != nil {
		return nil, err
	}

	modifiedData, err := json.Marshal(updateObj)
	if err != nil {
		return nil, err
	}

	originalData, exists := currentObj.GetAnnotations()[lastAppliedConfigAnnotation]
	if !exists {
		glog.V(2).Infof("no last applied found in annotation %s for %s/%s", lastAppliedConfigAnnotation, currentObj.GetNamespace(), currentObj.GetName())
	}

	return jsonmergepatch.CreateThreeWayJSONMergePatch([]byte(originalData), modifiedData, currentData)
}

func (cc *ClusterController) ensureSecrets(c *kubermaticv1.Cluster) error {
	generateTokensSecret := func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*corev1.Secret, string, error) {
		tokens := []controllerresources.Token{
			{
				Token:  c.Address.AdminToken,
				Name:   "admin",
				UserID: "10000",
				Group:  "system:masters",
			},
			{
				Token:  c.Address.KubeletToken,
				Name:   "kubelet-bootstrap",
				UserID: "10001",
				Group:  "system:bootstrappers",
			},
		}
		return controllerresources.GenerateTokenCSV(controllerresources.ApiserverTokenUsersSecretName, tokens)
	}

	resources := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*corev1.Secret, string, error){
		controllerresources.ApiserverSecretName:           controllerresources.LoadSecretFile,
		controllerresources.ControllerManagerSecretName:   controllerresources.LoadSecretFile,
		controllerresources.ApiserverTokenUsersSecretName: generateTokensSecret,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range resources {
		generatedSecret, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate Secret %s: %v", name, err)
		}
		generatedSecret.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedSecret.Name = name

		secret, err := cc.SecretLister.Secrets(c.Status.NamespaceName).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err = cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Create(generatedSecret); err != nil {
					return fmt.Errorf("failed to create secret for %s: %v", name, err)
				}
				continue
			} else {
				return err
			}
		}
		if secret.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(secret, generatedSecret)
			if err != nil {
				return err
			}
			if _, err := cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Patch(name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to patch secret for %s: %v", name, err)
			}
		}
	}

	return nil
}

func (cc *ClusterController) ensureServices(c *kubermaticv1.Cluster) error {
	services := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*corev1.Service, string, error){
		controllerresources.ApiserverInternalServiceName: controllerresources.LoadServiceFile,
		controllerresources.ApiserverExternalServiceName: controllerresources.LoadServiceFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range services {
		generatedService, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate Service %s: %v", name, err)
		}
		generatedService.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedService.Name = name

		service, err := cc.ServiceLister.Services(c.Status.NamespaceName).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err = cc.kubeClient.CoreV1().Services(c.Status.NamespaceName).Create(generatedService); err != nil {
					return fmt.Errorf("failed to create service for %s: %v", name, err)
				}
				continue
			} else {
				return err
			}
		}
		if service.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(service, generatedService)
			if err != nil {
				return err
			}
			if _, err = cc.kubeClient.CoreV1().Services(c.Status.NamespaceName).Patch(name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to patch service for %s: %v", name, err)
			}
		}
	}

	return nil
}

func (cc *ClusterController) ensureCheckServiceAccounts(c *kubermaticv1.Cluster) error {
	serviceAccounts := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*corev1.ServiceAccount, string, error){
		controllerresources.EtcdOperatorServiceAccountName: controllerresources.LoadServiceAccountFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range serviceAccounts {
		generatedServiceAccount, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate ServiceAccount %s: %v", name, err)
		}
		generatedServiceAccount.Annotations[lastAppliedConfigAnnotation] = lastApplied

		serviceAccount, err := cc.ServiceAccountLister.ServiceAccounts(c.Status.NamespaceName).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err = cc.kubeClient.CoreV1().ServiceAccounts(c.Status.NamespaceName).Create(generatedServiceAccount); err != nil {
					return fmt.Errorf("failed to create serviceAccount for %s: %v", name, err)
				}
				continue
			} else {
				return err
			}
		}
		if serviceAccount.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(serviceAccount, generatedServiceAccount)
			if err != nil {
				return err
			}
			if _, err = cc.kubeClient.CoreV1().ServiceAccounts(c.Status.NamespaceName).Patch(name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to patch serviceAccount for %s: %v", name, err)
			}
		}
	}

	return nil
}

func (cc *ClusterController) ensureClusterRoleBindings(c *kubermaticv1.Cluster) error {
	roleBindings := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*rbacv1beta1.ClusterRoleBinding, string, error){
		"etcd-operator": controllerresources.LoadClusterRoleBindingFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range roleBindings {
		generatedClusterRoleBinding, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate ClusterRoleBinding %s: %v", name, err)
		}
		generatedClusterRoleBinding.Annotations[lastAppliedConfigAnnotation] = lastApplied

		clusterRoleBinding, err := cc.ClusterRoleBindingLister.Get(generatedClusterRoleBinding.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err = cc.kubeClient.RbacV1beta1().ClusterRoleBindings().Create(generatedClusterRoleBinding); err != nil {
					return fmt.Errorf("failed to create clusterRoleBinding for %s: %v", name, err)
				}
				continue
			} else {
				return err
			}
		}
		if clusterRoleBinding.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(clusterRoleBinding, generatedClusterRoleBinding)
			if err != nil {
				return err
			}
			if _, err = cc.kubeClient.RbacV1beta1().ClusterRoleBindings().Patch(generatedClusterRoleBinding.Name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to patch clusterRoleBinding for %s: %v", name, err)
			}
		}
	}

	return nil
}

func (cc *ClusterController) ensureDeployments(c *kubermaticv1.Cluster) error {
	masterVersion, found := cc.versions[c.Spec.MasterVersion]
	if !found {
		return fmt.Errorf("unknown new cluster %q master version %q", c.Name, c.Spec.MasterVersion)
	}

	deps := map[string]string{
		controllerresources.EtcdOperatorDeploymentName:      masterVersion.EtcdOperatorDeploymentYaml,
		controllerresources.ApiserverDeploymenName:          masterVersion.ApiserverDeploymentYaml,
		controllerresources.ControllerManagerDeploymentName: masterVersion.ControllerDeploymentYaml,
		controllerresources.SchedulerDeploymentName:         masterVersion.SchedulerDeploymentYaml,
		controllerresources.NodeControllerDeploymentName:    masterVersion.NodeControllerDeploymentYaml,
		controllerresources.AddonManagerDeploymentName:      masterVersion.AddonManagerDeploymentYaml,
		controllerresources.MachineControllerDeploymentName: masterVersion.MachineControllerDeploymentYaml,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, yamlFile := range deps {
		generatedDeployment, lastApplied, err := controllerresources.LoadDeploymentFile(data, cc.masterResourcesPath, yamlFile)
		if err != nil {
			return fmt.Errorf("failed to generate Deployment %s: %v", name, err)
		}
		generatedDeployment.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedDeployment.Name = name

		deployment, err := cc.DeploymentLister.Deployments(c.Status.NamespaceName).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err = cc.kubeClient.ExtensionsV1beta1().Deployments(c.Status.NamespaceName).Create(generatedDeployment); err != nil {
					return fmt.Errorf("failed to create deployment for %s: %v", name, err)
				}
				continue
			} else {
				return err
			}
		}
		if deployment.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(deployment, generatedDeployment)
			if err != nil {
				return err
			}
			if _, err = cc.kubeClient.ExtensionsV1beta1().Deployments(c.Status.NamespaceName).Patch(name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to patch deployment for %s: %v", name, err)
			}
		}
	}

	return nil
}

func (cc *ClusterController) ensureConfigMaps(c *kubermaticv1.Cluster) error {
	cms := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*corev1.ConfigMap, string, error){
		controllerresources.CloudConfigConfigMapName: controllerresources.LoadConfigMapFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range cms {
		generatedConfigMap, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate ConfigMap %s: %v", name, err)
		}
		generatedConfigMap.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedConfigMap.Name = name

		configMap, err := cc.ConfigMapLister.ConfigMaps(c.Status.NamespaceName).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err = cc.kubeClient.CoreV1().ConfigMaps(c.Status.NamespaceName).Create(generatedConfigMap); err != nil {
					return fmt.Errorf("failed to create configMap for %s: %v", name, err)
				}
				continue
			} else {
				return err
			}
		}
		if configMap.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(configMap, generatedConfigMap)
			if err != nil {
				return err
			}
			if _, err = cc.kubeClient.CoreV1().ConfigMaps(c.Status.NamespaceName).Patch(name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to patch configMap for %s: %v", name, err)
			}
		}
	}

	return nil
}

func (cc *ClusterController) ensureEtcdCluster(c *kubermaticv1.Cluster) error {
	masterVersion, found := cc.versions[c.Spec.MasterVersion]
	if !found {
		return fmt.Errorf("unknown new cluster %q master version %q", c.Name, c.Spec.MasterVersion)
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	generatedEtcd, lastApplied, err := controllerresources.LoadEtcdClusterFile(data, cc.masterResourcesPath, masterVersion.EtcdClusterYaml)
	if err != nil {
		return fmt.Errorf("failed to load etcd-cluster: %v", err)
	}
	generatedEtcd.Annotations[lastAppliedConfigAnnotation] = lastApplied

	etcd, err := cc.EtcdClusterLister.EtcdClusters(c.Status.NamespaceName).Get(generatedEtcd.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = cc.kubermaticClient.EtcdV1beta2().EtcdClusters(c.Status.NamespaceName).Create(generatedEtcd)
			if err != nil {
				return fmt.Errorf("failed to create etcd-cluster resource: %v", err)
			}
			return nil
		}
		return err
	}
	if etcd.Annotations[lastAppliedConfigAnnotation] != lastApplied {
		patch, err := getPatch(etcd, generatedEtcd)
		if err != nil {
			return err
		}
		if _, err = cc.kubermaticClient.EtcdV1beta2().EtcdClusters(c.Status.NamespaceName).Patch(generatedEtcd.Name, types.MergePatchType, patch); err != nil {
			return fmt.Errorf("failed to create patch etcd-cluster resource: %v", err)
		}
	}

	return nil
}
