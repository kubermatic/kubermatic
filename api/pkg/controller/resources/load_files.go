package resources

import (
	"fmt"
	"path"

	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	prometheusv1 "github.com/kubermatic/kubermatic/api/pkg/crd/prometheus/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8stemplate "github.com/kubermatic/kubermatic/api/pkg/template/kubernetes"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	corev1lister "k8s.io/client-go/listers/core/v1"
)

const (
	// EtcdClusterName is the name of the etcd cluster
	EtcdClusterName = "etcd-cluster"

	//AddonManagerDeploymentName is the name for the addon-manager deployment
	AddonManagerDeploymentName = "addon-manager"
	//ApiserverDeploymenName is the name for the apiserver deployment
	ApiserverDeploymenName = "apiserver"
	//ControllerManagerDeploymentName is the name for the controller manager deployment
	ControllerManagerDeploymentName = "controller-manager"
	//EtcdOperatorDeploymentName is the name for the etcd-operator deployment
	EtcdOperatorDeploymentName = "etcd-operator"
	//NodeControllerDeploymentName is the name for the node-controller deployment
	NodeControllerDeploymentName = "node-controller"
	//SchedulerDeploymentName is the name for the scheduler deployment
	SchedulerDeploymentName = "scheduler"
	//MachineControllerDeploymentName is the name for the machine-controller deployment
	MachineControllerDeploymentName = "machine-controller"

	//ApiserverExternalServiceName is the name for the external apiserver service
	ApiserverExternalServiceName = "apiserver-external"
	//ApiserverInternalServiceName is the name for the internal apiserver service
	ApiserverInternalServiceName = "apiserver"
	//ControllerManagerServiceName is the name for the controller manager service
	ControllerManagerServiceName = "controller-manager"
	//KubeStateMetricsServiceName is the name for the kube-state-metrics service
	KubeStateMetricsServiceName = "kube-state-metrics"
	//MachineControllerServiceName is the name for the machine controller service
	MachineControllerServiceName = "machine-controller"
	//SchedulerServiceName is the name for the scheduler service
	SchedulerServiceName = "scheduler"

	//ApiserverSecretName is the name for the secrets required for the apiserver
	ApiserverSecretName = "apiserver"
	//ControllerManagerSecretName is the name for the secrets required for the controller manager
	ControllerManagerSecretName = "controller-manager"
	//ApiserverTokenUsersSecretName is the name for the token-users secret
	ApiserverTokenUsersSecretName = "token-users"

	//CloudConfigConfigMapName is the name for the configmap containing the cloud-config
	CloudConfigConfigMapName = "cloud-config"

	//EtcdOperatorServiceAccountName is the name for the etcd-operator serviceaccount
	EtcdOperatorServiceAccountName = "etcd-operator"
	//PrometheusServiceAccountName is the name for the Prometheus serviceaccount
	PrometheusServiceAccountName = "prometheus"

	//PrometheusRoleName is the name for the Prometheus role
	PrometheusRoleName = "prometheus"

	//PrometheusRoleBindingName is the name for the Prometheus rolebinding
	PrometheusRoleBindingName = "prometheus"

	//EtcdOperatorClusterRoleBindingName is the name for the etcd-operator clusterrolebinding
	EtcdOperatorClusterRoleBindingName = "etcd-operator"
)

// TemplateData is a group of data required for template generation
type TemplateData struct {
	Cluster         *kubermaticv1.Cluster
	Version         *apiv1.MasterVersion
	DC              *provider.DatacenterMeta
	SecretLister    corev1lister.SecretLister
	ConfigMapLister corev1lister.ConfigMapLister
}

// NewTemplateData returns an instance of TemplateData
func NewTemplateData(
	cluster *kubermaticv1.Cluster,
	version *apiv1.MasterVersion,
	dc *provider.DatacenterMeta,
	secretLister corev1lister.SecretLister,
	configMapLister corev1lister.ConfigMapLister) *TemplateData {
	return &TemplateData{
		Cluster:         cluster,
		DC:              dc,
		Version:         version,
		ConfigMapLister: configMapLister,
		SecretLister:    secretLister,
	}
}

// TokenCSVRevision returns the resource version of the token-users secret for the cluster
func (d *TemplateData) TokenCSVRevision() string {
	secret, err := d.SecretLister.Secrets(d.Cluster.Status.NamespaceName).Get(ApiserverTokenUsersSecretName)
	if err != nil {
		glog.V(0).Infof("could not get token-users secret from lister: %v", err)
		return ""
	}
	return secret.ResourceVersion
}

// CloudConfigRevision returns the resource version of the cloud-config configmap for the cluster
func (d *TemplateData) CloudConfigRevision() string {
	configmap, err := d.ConfigMapLister.ConfigMaps(d.Cluster.Status.NamespaceName).Get(CloudConfigConfigMapName)
	if err != nil {
		glog.V(0).Infof("could not get cloud-config configmap from lister: %v", err)
		return ""
	}
	return configmap.ResourceVersion
}

// ProviderName returns the name of the clusters providerName
func (d *TemplateData) ProviderName() string {
	p, err := provider.ClusterCloudProviderName(d.Cluster.Spec.Cloud)
	if err != nil {
		glog.V(0).Infof("could not identify cloud provider: %v", err)
	}
	return p
}

// LoadDeploymentFile loads a k8s yaml deployment from disk and returns a Deployment struct
func LoadDeploymentFile(data *TemplateData, masterResourcesPath, yamlFile string) (*v1beta1.Deployment, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, yamlFile))
	if err != nil {
		return nil, "", err
	}

	var dep v1beta1.Deployment
	json, err := t.Execute(data, &dep)
	return &dep, json, err
}

// LoadServiceFile returns the service for the given cluster and app
func LoadServiceFile(data *TemplateData, app, masterResourcesPath string) (*corev1.Service, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-service.yaml"))
	if err != nil {
		return nil, "", err
	}

	var service corev1.Service
	json, err := t.Execute(data, &service)
	return &service, json, err
}

// LoadSecretFile returns the secret for the given cluster and app
func LoadSecretFile(data *TemplateData, app, masterResourcesPath string) (*corev1.Secret, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-secret.yaml"))
	if err != nil {
		return nil, "", err
	}

	var secret corev1.Secret
	json, err := t.Execute(data, &secret)
	return &secret, json, err
}

// LoadConfigMapFile returns the configmap for the given cluster and app
func LoadConfigMapFile(data *TemplateData, app, masterResourcesPath string) (*corev1.ConfigMap, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-configmap.yaml"))
	if err != nil {
		return nil, "", err
	}

	var secret corev1.ConfigMap
	json, err := t.Execute(data, &secret)
	return &secret, json, err
}

// LoadIngressFile returns the ingress for the given cluster and app
func LoadIngressFile(data *TemplateData, app, masterResourcesPath string) (*v1beta1.Ingress, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-ingress.yaml"))
	if err != nil {
		return nil, "", err
	}
	var ingress v1beta1.Ingress
	json, err := t.Execute(data, &ingress)
	return &ingress, json, err
}

// LoadPVCFile returns the PVC for the given cluster & app
func LoadPVCFile(data *TemplateData, app, masterResourcesPath string) (*corev1.PersistentVolumeClaim, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-pvc.yaml"))
	if err != nil {
		return nil, "", err
	}

	var pvc corev1.PersistentVolumeClaim
	json, err := t.Execute(data, &pvc)
	return &pvc, json, err
}

// LoadEtcdClusterFile loads a etcd-operator crd from disk and returns a Cluster crd struct
func LoadEtcdClusterFile(data *TemplateData, masterResourcesPath, yamlFile string) (*etcdoperatorv1beta2.EtcdCluster, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, yamlFile))
	if err != nil {
		return nil, "", err
	}

	var c etcdoperatorv1beta2.EtcdCluster
	json, err := t.Execute(data, &c)
	return &c, json, err
}

// LoadServiceAccountFile loads a service account from disk and returns it
func LoadServiceAccountFile(data *TemplateData, app, masterResourcesPath string) (*corev1.ServiceAccount, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-serviceaccount.yaml"))
	if err != nil {
		return nil, "", err
	}

	var sa corev1.ServiceAccount
	json, err := t.Execute(data, &sa)
	return &sa, json, err
}

// LoadRoleFile loads a role from disk, sets the namespace and returns it
func LoadRoleFile(data *TemplateData, app, masterResourcesPath string) (*rbacv1beta1.Role, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-role.yaml"))
	if err != nil {
		return nil, "", err
	}

	var r rbacv1beta1.Role
	json, err := t.Execute(data, &r)
	return &r, json, err
}

// LoadRoleBindingFile loads a role binding from disk, sets the namespace and returns it
func LoadRoleBindingFile(data *TemplateData, app, masterResourcesPath string) (*rbacv1beta1.RoleBinding, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-rolebinding.yaml"))
	if err != nil {
		return nil, "", err
	}

	var r rbacv1beta1.RoleBinding
	json, err := t.Execute(data, &r)
	return &r, json, err
}

// LoadClusterRoleBindingFile loads a role binding from disk, sets the namespace and returns it
func LoadClusterRoleBindingFile(data *TemplateData, app, masterResourcesPath string) (*rbacv1beta1.ClusterRoleBinding, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-clusterrolebinding.yaml"))
	if err != nil {
		return nil, "", err
	}

	var r rbacv1beta1.ClusterRoleBinding
	json, err := t.Execute(data, &r)
	return &r, json, err
}

// LoadPrometheusFile loads a prometheus crd from disk and returns a Cluster crd struct
func LoadPrometheusFile(data *TemplateData, app, masterResourcesPath string) (*prometheusv1.Prometheus, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, "prometheus.yaml"))
	if err != nil {
		return nil, "", err
	}

	var p prometheusv1.Prometheus
	json, err := t.Execute(data, &p)
	return &p, json, err
}

// LoadServiceMonitorFile loads a service monitor crd from disk and returns a Cluster crd struct
func LoadServiceMonitorFile(data *TemplateData, app, masterResourcesPath string) (*prometheusv1.ServiceMonitor, string, error) {
	filename := fmt.Sprintf("prometheus-service-monitor-%s.yaml", app)
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, filename))
	if err != nil {
		return nil, "", err
	}

	var sm prometheusv1.ServiceMonitor
	json, err := t.Execute(data, &sm)
	return &sm, json, err
}
