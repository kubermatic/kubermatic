package cluster

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	controllerresources "github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	prometheusv1 "github.com/kubermatic/kubermatic/api/pkg/crd/prometheus/v1"
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

func (cc *Controller) reconcileCluster(cluster *kubermaticv1.Cluster) error {
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

	// check that all roles are created
	if err := cc.ensureRoles(cluster); err != nil {
		return err
	}

	// check that all role bindings are created
	if err := cc.ensureRoleBindings(cluster); err != nil {
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

	if err := cc.ensurePrometheus(cluster); err != nil {
		return err
	}

	if err := cc.ensureServiceMonitors(cluster); err != nil {
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

func (cc *Controller) getClusterTemplateData(c *kubermaticv1.Cluster) (*controllerresources.TemplateData, error) {
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

func (cc *Controller) ensureCloudProviderIsInitialize(cluster *kubermaticv1.Cluster) error {
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
func (cc *Controller) ensureNamespaceExists(c *kubermaticv1.Cluster) error {
	name := fmt.Sprintf("cluster-%s", c.Name)
	if _, err := cc.NamespaceLister.Get(name); !errors.IsNotFound(err) {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if _, err := cc.kubeClient.CoreV1().Namespaces().Create(ns); err != nil {
		return fmt.Errorf("failed to create namespace %s: %v", name, err)
	}

	if !kuberneteshelper.HasFinalizer(c, namespaceDeletionFinalizer) {
		c.Finalizers = append(c.Finalizers, namespaceDeletionFinalizer)
	}

	c.Status.NamespaceName = name
	return nil
}

func (cc *Controller) getFreeNodePort() (int, error) {
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
func (cc *Controller) ensureAddress(c *kubermaticv1.Cluster) error {
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

func (cc *Controller) ensureSecrets(c *kubermaticv1.Cluster) error {
	resources := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*corev1.Secret, string, error){
		controllerresources.ApiserverSecretName:           controllerresources.LoadSecretFile,
		controllerresources.ControllerManagerSecretName:   controllerresources.LoadSecretFile,
		controllerresources.ApiserverTokenUsersSecretName: controllerresources.LoadSecretFile,
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

func (cc *Controller) ensureServices(c *kubermaticv1.Cluster) error {
	services := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*corev1.Service, string, error){
		controllerresources.ApiserverInternalServiceName: controllerresources.LoadServiceFile,
		controllerresources.ApiserverExternalServiceName: controllerresources.LoadServiceFile,
		controllerresources.ControllerManagerServiceName: controllerresources.LoadServiceFile,
		controllerresources.KubeStateMetricsServiceName:  controllerresources.LoadServiceFile,
		controllerresources.MachineControllerServiceName: controllerresources.LoadServiceFile,
		controllerresources.SchedulerServiceName:         controllerresources.LoadServiceFile,
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

func (cc *Controller) ensureCheckServiceAccounts(c *kubermaticv1.Cluster) error {
	serviceAccounts := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*corev1.ServiceAccount, string, error){
		controllerresources.EtcdOperatorServiceAccountName: controllerresources.LoadServiceAccountFile,
		controllerresources.PrometheusServiceAccountName:   controllerresources.LoadServiceAccountFile,
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
		generatedServiceAccount.Name = name

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

func (cc *Controller) ensureRoles(c *kubermaticv1.Cluster) error {
	roles := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*rbacv1beta1.Role, string, error){
		controllerresources.PrometheusRoleName: controllerresources.LoadRoleFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range roles {
		generatedRole, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate role %s: %v", name, err)
		}
		generatedRole.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedRole.Name = name

		role, err := cc.RoleLister.Roles(c.Status.NamespaceName).Get(generatedRole.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err := cc.kubeClient.RbacV1beta1().Roles(c.Status.NamespaceName).Create(generatedRole); err != nil {
					return fmt.Errorf("failed to create role for %s: %v", name, err)
				}
				continue
			} else {
				return err
			}
		}
		if role.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(role, generatedRole)
			if err != nil {
				return err
			}
			if _, err = cc.kubeClient.RbacV1beta1().Roles(c.Status.NamespaceName).Patch(generatedRole.Name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to patch role for %s: %v", name, err)
			}
		}
	}

	return nil
}

func (cc *Controller) ensureRoleBindings(c *kubermaticv1.Cluster) error {
	roleBindings := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*rbacv1beta1.RoleBinding, string, error){
		controllerresources.PrometheusRoleBindingName: controllerresources.LoadRoleBindingFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range roleBindings {
		generatedRoleBinding, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate RoleBinding %s: %v", name, err)
		}
		generatedRoleBinding.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedRoleBinding.Name = name

		roleBinding, err := cc.RoleBindingLister.RoleBindings(c.Status.NamespaceName).Get(generatedRoleBinding.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err := cc.kubeClient.RbacV1beta1().RoleBindings(c.Status.NamespaceName).Create(generatedRoleBinding); err != nil {
					return fmt.Errorf("failed to create roleBinding for %s: %v", name, err)
				}
				continue
			} else {
				return err
			}
		}
		if roleBinding.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(roleBinding, generatedRoleBinding)
			if err != nil {
				return err
			}
			if _, err = cc.kubeClient.RbacV1beta1().RoleBindings(c.Status.NamespaceName).Patch(generatedRoleBinding.Name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to patch roleBinding for %s: %v", name, err)
			}
		}
	}

	return nil
}

func (cc *Controller) ensureClusterRoleBindings(c *kubermaticv1.Cluster) error {
	clusterRoleBindings := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*rbacv1beta1.ClusterRoleBinding, string, error){
		controllerresources.EtcdOperatorClusterRoleBindingName: controllerresources.LoadClusterRoleBindingFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range clusterRoleBindings {
		generatedClusterRoleBinding, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate ClusterRoleBinding %s: %v", name, err)
		}
		generatedClusterRoleBinding.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedClusterRoleBinding.Name = fmt.Sprintf("cluster-%s-etcd-operator", c.Name)

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

func (cc *Controller) ensureDeployments(c *kubermaticv1.Cluster) error {
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

func (cc *Controller) ensureConfigMaps(c *kubermaticv1.Cluster) error {
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

func (cc *Controller) ensureEtcdCluster(c *kubermaticv1.Cluster) error {
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

func (cc *Controller) ensurePrometheus(c *kubermaticv1.Cluster) error {
	proms := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*prometheusv1.Prometheus, string, error){
		controllerresources.PrometheusName: controllerresources.LoadPrometheusFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range proms {
		generatedPrometheus, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate Prometheus %s: %v", name, err)
		}
		generatedPrometheus.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedPrometheus.Name = name

		prometheus, err := cc.PrometheusLister.Prometheuses(c.Status.NamespaceName).Get(generatedPrometheus.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				_, err := cc.kubermaticClient.MonitoringV1().Prometheuses(c.Status.NamespaceName).Create(generatedPrometheus)
				if err != nil {
					return fmt.Errorf("failed to create prometheus resource: %v", err)
				}
				return nil
			}
			return err
		}
		if prometheus.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(prometheus, generatedPrometheus)
			if err != nil {
				return err
			}
			if _, err := cc.kubermaticClient.MonitoringV1().Prometheuses(c.Status.NamespaceName).Patch(generatedPrometheus.Name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to create patch for prometheus resource: %v", err)
			}
		}
	}
	return nil
}

func (cc *Controller) ensureServiceMonitors(c *kubermaticv1.Cluster) error {
	sms := map[string]func(data *controllerresources.TemplateData, app, masterResourcesPath string) (*prometheusv1.ServiceMonitor, string, error){
		controllerresources.ApiserverServiceMonitorName:         controllerresources.LoadServiceMonitorFile,
		controllerresources.ControllerManagerServiceMonitorName: controllerresources.LoadServiceMonitorFile,
		controllerresources.EtcdServiceMonitorName:              controllerresources.LoadServiceMonitorFile,
		controllerresources.KubeStateMetricsServiceMonitorName:  controllerresources.LoadServiceMonitorFile,
		controllerresources.MachineControllerServiceMonitorName: controllerresources.LoadServiceMonitorFile,
		controllerresources.SchedulerServiceMonitorName:         controllerresources.LoadServiceMonitorFile,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, gen := range sms {
		generatedServiceMonitor, lastApplied, err := gen(data, name, cc.masterResourcesPath)
		if err != nil {
			return fmt.Errorf("failed to generate service monitor %s: %v", name, err)
		}
		generatedServiceMonitor.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedServiceMonitor.Name = name

		serviceMonitor, err := cc.ServiceMonitorLister.ServiceMonitors(c.Status.NamespaceName).Get(generatedServiceMonitor.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				_, err := cc.kubermaticClient.MonitoringV1().ServiceMonitors(c.Status.NamespaceName).Create(generatedServiceMonitor)
				if err != nil {
					return fmt.Errorf("failed to create service monitor resource: %v", err)
				}
				return nil
			}
			return err
		}
		if serviceMonitor.Annotations[lastAppliedConfigAnnotation] != lastApplied {
			patch, err := getPatch(serviceMonitor, generatedServiceMonitor)
			if err != nil {
				return err
			}
			if _, err := cc.kubermaticClient.MonitoringV1().ServiceMonitors(c.Status.NamespaceName).Patch(serviceMonitor.Name, types.MergePatchType, patch); err != nil {
				return fmt.Errorf("failed to create patch for service monitor resource: %v", err)
			}
		}
	}

	return nil
}
