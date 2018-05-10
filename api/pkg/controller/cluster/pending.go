package cluster

import (
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/go-test/deep"
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/prometheus"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	nodeDeletionFinalizer         = "kubermatic.io/delete-nodes"
	cloudProviderCleanupFinalizer = "kubermatic.io/cleanup-cloud-provider"
	namespaceDeletionFinalizer    = "kubermatic.io/delete-ns"

	annotationPrefix            = "kubermatic.io/"
	lastAppliedConfigAnnotation = annotationPrefix + "last-applied-configuration"

	emptyPatch = "{}"
)

func (cc *Controller) reconcileCluster(cluster *kubermaticv1.Cluster) error {
	if cluster.Spec.MasterVersion == "" {
		cluster.Spec.MasterVersion = cc.defaultMasterVersion.ID
	}

	if err := cc.ensureAutomaticMasterUpdate(cluster); err != nil {
		return err
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

	// check that all ConfigMap's are available
	if err := cc.ensureConfigMaps(cluster); err != nil {
		return err
	}

	// check that all deployments are available
	if err := cc.ensureDeployments(cluster); err != nil {
		return err
	}

	// check that all StatefulSet's are created
	if err := cc.ensureStatefulSets(cluster); err != nil {
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

	if cluster.Status.Health.Apiserver {
		if err := cc.ensureClusterReachable(cluster); err != nil {
			return err
		}

		if err := cc.launchingCreateClusterInfoConfigMap(cluster); err != nil {
			return err
		}

		if err := cc.launchingCreateOpenVPNClientCertificates(cluster); err != nil {
			return err
		}

		if err := cc.launchingCreateOpenVPNConfigMap(cluster); err != nil {
			return err
		}
	}

	if !allHealthy {
		glog.V(5).Infof("Cluster %q not yet healthy: %+v", cluster.Name, cluster.Status.Health)
		return nil
	}

	if cluster.Status.Phase == kubermaticv1.LaunchingClusterStatusPhase {
		cluster.Status.Phase = kubermaticv1.RunningClusterStatusPhase
	}

	return nil
}

func (cc *Controller) getClusterTemplateData(c *kubermaticv1.Cluster) (*resources.TemplateData, error) {
	version, found := cc.versions[c.Spec.MasterVersion]
	if !found {
		return nil, fmt.Errorf("failed to get version %s", c.Spec.MasterVersion)
	}

	dc, found := cc.dcs[c.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", c.Spec.Cloud.DatacenterName)
	}

	return resources.NewTemplateData(
		c,
		version,
		&dc,
		cc.SecretLister,
		cc.ConfigMapLister,
		cc.ServiceLister,
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

func (cc *Controller) ensureAutomaticMasterUpdate(c *kubermaticv1.Cluster) error {
	updateVersion, err := version.BestAutomaticUpdate(c.Spec.MasterVersion, cc.updates)
	if err != nil {
		return err
	}

	if updateVersion != nil {
		// start automatic update
		glog.V(4).Infof("applying automatic update cluster %s. From version %s to %s", c.Name, c.Spec.MasterVersion, updateVersion.To)
		c.Spec.MasterVersion = updateVersion.To
	}
	return nil
}

// ensureNamespaceExists will create the cluster namespace
func (cc *Controller) ensureNamespaceExists(c *kubermaticv1.Cluster) error {
	if c.Status.NamespaceName == "" {
		c.Status.NamespaceName = fmt.Sprintf("cluster-%s", c.Name)
	}

	if _, err := cc.NamespaceLister.Get(c.Status.NamespaceName); !errors.IsNotFound(err) {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            c.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{cc.getOwnerRefForCluster(c)},
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

// ensureAddress will set the cluster hostname and the url under which the apiserver will be reachable
func (cc *Controller) ensureAddress(c *kubermaticv1.Cluster) error {
	c.Address.ExternalName = fmt.Sprintf("%s.%s.%s", c.Name, cc.dc, cc.externalURL)

	//Always update the ip
	resolvedIPs, err := net.LookupIP(c.Address.ExternalName)
	if err != nil {
		return fmt.Errorf("failed to lookup ip address for %s: %v", c.Address.ExternalName, err)
	}
	if len(resolvedIPs) == 0 {
		return fmt.Errorf("no ip addresses found for %s: %v", c.Address.ExternalName, err)
	}
	ipList := sets.NewString()
	for _, ip := range resolvedIPs {
		if ip.To4() != nil {
			ipList.Insert(ip.String())
		}
	}
	ips := ipList.List()
	sort.Strings(ips)
	c.Address.IP = ips[0]

	s, err := cc.ServiceLister.Services(c.Status.NamespaceName).Get(resources.ApiserverExternalServiceName)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return nil
	}
	c.Address.URL = fmt.Sprintf("https://%s:%d", c.Address.ExternalName, int(s.Spec.Ports[0].NodePort))

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
	//We need to follow a specific order here...
	//And maps in go are not sorted
	type secretOp struct {
		name string
		gen  func(*kubermaticv1.Cluster, *corev1.Secret) (*corev1.Secret, string, error)
	}
	ops := []secretOp{
		{resources.CAKeySecretName, cc.getRootCAKeySecret},
		{resources.CACertSecretName, cc.getRootCACertSecret},
		{resources.ApiserverTLSSecretName, cc.getApiserverServingCertificatesSecret},
		{resources.KubeletClientCertificatesSecretName, cc.getKubeletClientCertificatesSecret},
		{resources.ServiceAccountKeySecretName, cc.getServiceAccountKeySecret},
		{resources.AdminKubeconfigSecretName, cc.getAdminKubeconfigSecret},
		{resources.TokensSecretName, cc.getTokenUsersSecret},
		{resources.OpenVPNServerCertificatesSecretName, cc.getOpenVPNServerCertificates},
		{resources.OpenVPNClientCertificatesSecretName, cc.getOpenVPNInternalClientCertificates},
	}

	for _, op := range ops {
		exists := false
		existingSecret, err := cc.SecretLister.Secrets(c.Status.NamespaceName).Get(op.name)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to get secret %s from lister: %v", op.name, err)
			}
		} else {
			exists = true
		}

		generatedSecret, currentJSON, err := op.gen(c, existingSecret)
		if err != nil {
			return fmt.Errorf("failed to generate Secret %s: %v", op.name, err)
		}
		generatedSecret.Annotations[lastAppliedConfigAnnotation] = currentJSON
		generatedSecret.Name = op.name

		if !exists {
			if _, err = cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Create(generatedSecret); err != nil {
				return fmt.Errorf("failed to create secret for %s: %v", op.name, err)
			}

			secretExistsInLister := func() (bool, error) {
				_, err = cc.SecretLister.Secrets(c.Status.NamespaceName).Get(generatedSecret.Name)
				if err != nil {
					if os.IsNotExist(err) {
						return false, nil
					}
					runtime.HandleError(fmt.Errorf("failed to check if a created secret %s/%s got published to lister: %v", c.Status.NamespaceName, generatedSecret.Name, err))
					return false, nil
				}
				return true, nil
			}

			if err := wait.Poll(100*time.Millisecond, 30*time.Second, secretExistsInLister); err != nil {
				return fmt.Errorf("failed waiting for secret '%s' to exist in the lister: %v", generatedSecret.Name, err)
			}
			continue
		} else {
			if existingSecret.Annotations[lastAppliedConfigAnnotation] != currentJSON {
				patch, err := getPatch(existingSecret, generatedSecret)
				if err != nil {
					return err
				}
				if _, err := cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Patch(op.name, types.MergePatchType, patch); err != nil {
					return fmt.Errorf("failed to patch secret '%s': %v", op.name, err)
				}
			}
		}
	}

	return nil
}

func (cc *Controller) ensureServices(c *kubermaticv1.Cluster) error {
	services := map[string]func(data *resources.TemplateData, app, masterResourcesPath string) (*corev1.Service, string, error){
		resources.ApiserverInternalServiceName: resources.LoadServiceFile,
		resources.ApiserverExternalServiceName: resources.LoadServiceFile,
		resources.OpenVPNServerServiceName:     resources.LoadServiceFile,
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
	names := []string{
		resources.EtcdOperatorServiceAccountName,
		resources.PrometheusServiceAccountName,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}
	ref := data.GetClusterRef()

	for _, name := range names {
		var existing *corev1.ServiceAccount
		sa := resources.ServiceAccount(name, &ref, nil)

		if existing, err = cc.ServiceAccountLister.ServiceAccounts(c.Status.NamespaceName).Get(sa.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.CoreV1().ServiceAccounts(c.Status.NamespaceName).Create(sa); err != nil {
				return fmt.Errorf("failed to create ServiceAccount %s: %v", sa.Name, err)
			}
			continue
		}

		// We update the existing SA
		sa = resources.ServiceAccount(name, &ref, existing.DeepCopy())
		if diff := deep.Equal(sa, existing); diff == nil {
			continue
		}
		if _, err = cc.kubeClient.CoreV1().ServiceAccounts(c.Status.NamespaceName).Update(sa); err != nil {
			return fmt.Errorf("failed to patch ServiceAccount %s: %v", sa.Name, err)
		}
	}

	return nil
}

func (cc *Controller) ensureRoles(c *kubermaticv1.Cluster) error {
	roles := map[string]func(data *resources.TemplateData, app, masterResourcesPath string) (*rbacv1.Role, string, error){
		resources.PrometheusRoleName: resources.LoadRoleFile,
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
				if _, err := cc.kubeClient.RbacV1().Roles(c.Status.NamespaceName).Create(generatedRole); err != nil {
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
	roleBindings := map[string]func(data *resources.TemplateData, app, masterResourcesPath string) (*rbacv1.RoleBinding, string, error){
		resources.PrometheusRoleBindingName: resources.LoadRoleBindingFile,
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
				if _, err := cc.kubeClient.RbacV1().RoleBindings(c.Status.NamespaceName).Create(generatedRoleBinding); err != nil {
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
	clusterRoleBindings := map[string]func(data *resources.TemplateData, app, masterResourcesPath string) (*rbacv1.ClusterRoleBinding, string, error){
		resources.EtcdOperatorClusterRoleBindingName: resources.LoadClusterRoleBindingFile,
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
				if _, err = cc.kubeClient.RbacV1().ClusterRoleBindings().Create(generatedClusterRoleBinding); err != nil {
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
			if _, err = cc.kubeClient.RbacV1().ClusterRoleBindings().Patch(generatedClusterRoleBinding.Name, types.MergePatchType, patch); err != nil {
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
		resources.EtcdOperatorDeploymentName:      masterVersion.EtcdOperatorDeploymentYaml,
		resources.ApiserverDeploymenName:          masterVersion.ApiserverDeploymentYaml,
		resources.ControllerManagerDeploymentName: masterVersion.ControllerDeploymentYaml,
		resources.SchedulerDeploymentName:         masterVersion.SchedulerDeploymentYaml,
		resources.AddonManagerDeploymentName:      masterVersion.AddonManagerDeploymentYaml,
		resources.MachineControllerDeploymentName: masterVersion.MachineControllerDeploymentYaml,
		resources.OpenVPNServerDeploymentName:     masterVersion.OpenVPNServerDeploymentYaml,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for name, yamlFile := range deps {
		generatedDeployment, lastApplied, err := resources.LoadDeploymentFile(data, cc.masterResourcesPath, yamlFile)
		if err != nil {
			return fmt.Errorf("failed to generate Deployment %s: %v", name, err)
		}
		generatedDeployment.Annotations[lastAppliedConfigAnnotation] = lastApplied
		generatedDeployment.Name = name

		deployment, err := cc.DeploymentLister.Deployments(c.Status.NamespaceName).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				if _, err = cc.kubeClient.AppsV1().Deployments(c.Status.NamespaceName).Create(generatedDeployment); err != nil {
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
	creators := []resources.ConfigMapCreator{
		cloudconfig.ConfigMap,
		openvpn.ConfigMap,
		prometheus.ConfigMap,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *corev1.ConfigMap
		cm, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		if existing, err = cc.ConfigMapLister.ConfigMaps(c.Status.NamespaceName).Get(cm.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.CoreV1().ConfigMaps(c.Status.NamespaceName).Create(cm); err != nil {
				return fmt.Errorf("failed to create ConfigMap %s: %v", cm.Name, err)
			}
			continue
		}

		cm, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		if diff := deep.Equal(cm, existing); diff == nil {
			continue
		}

		if _, err = cc.kubeClient.CoreV1().ConfigMaps(c.Status.NamespaceName).Update(cm); err != nil {
			return fmt.Errorf("failed to update ConfigMap %s: %v", cm.Name, err)
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

	generatedEtcd, lastApplied, err := resources.LoadEtcdClusterFile(data, cc.masterResourcesPath, masterVersion.EtcdClusterYaml)
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

func (cc *Controller) ensureStatefulSets(c *kubermaticv1.Cluster) error {
	creators := []func(data *resources.TemplateData, existing *v1.StatefulSet) (*v1.StatefulSet, error){
		prometheus.StatefulSet,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *v1.StatefulSet
		set, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build StatefulSet: %v", err)
		}

		if existing, err = cc.StatefulSetLister.StatefulSets(c.Status.NamespaceName).Get(set.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.AppsV1().StatefulSets(c.Status.NamespaceName).Create(set); err != nil {
				return fmt.Errorf("failed to create StatefulSet %s: %v", set.Name, err)
			}
			continue
		}

		set, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build StatefulSet: %v", err)
		}

		if diff := deep.Equal(set, existing); diff == nil {
			continue
		}

		if _, err = cc.kubeClient.AppsV1().StatefulSets(c.Status.NamespaceName).Update(set); err != nil {
			return fmt.Errorf("failed to patch StatefulSet %s: %v", set.Name, err)
		}
	}

	return nil
}

func (cc *Controller) createStrategicMergePatch(modified interface{}, original interface{}, dataStruct interface{}) ([]byte, error) {
	mb, err := json.Marshal(modified)
	if err != nil {
		return nil, err
	}

	ob, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}

	return strategicpatch.CreateTwoWayMergePatch(ob, mb, dataStruct)
}
