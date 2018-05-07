package resources

import (
	"fmt"
	"path"

	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8stemplate "github.com/kubermatic/kubermatic/api/pkg/template/kubernetes"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
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
	//SchedulerDeploymentName is the name for the scheduler deployment
	SchedulerDeploymentName = "scheduler"
	//MachineControllerDeploymentName is the name for the machine-controller deployment
	MachineControllerDeploymentName = "machine-controller"
	//OpenVPNServerDeploymentName is the name for the openvpn server deployment
	OpenVPNServerDeploymentName = "openvpn-server"

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
	//PrometheusServiceName is the name for the prometheus service
	PrometheusServiceName = "prometheus"
	//SchedulerServiceName is the name for the scheduler service
	SchedulerServiceName = "scheduler"
	//OpenVPNServerServiceName is the name for the openvpn server service
	OpenVPNServerServiceName = "openvpn-server"

	//AdminKubeconfigSecretName is the name for the secret containing the private ca key
	AdminKubeconfigSecretName = "admin-kubeconfig"
	//CAKeySecretName is the name for the secret containing the private ca key
	CAKeySecretName = "ca-key"
	//CACertSecretName is the name for the secret containing the ca.crt
	CACertSecretName = "ca-cert"
	//ApiserverTLSSecretName is the name for the secrets required for the apiserver tls
	ApiserverTLSSecretName = "apiserver-tls"
	//KubeletClientCertificatesSecretName is the name for the secret containing the kubelet client certificates
	KubeletClientCertificatesSecretName = "kubelet-client-certificates"
	//ServiceAccountKeySecretName is the name for the secret containing the service account key
	ServiceAccountKeySecretName = "service-account-key"
	//TokensSecretName is the name for the secret containing the user tokens
	TokensSecretName = "tokens"
	//OpenVPNServerCertificatesSecretName is the name for the secret containing the openvpn server certificates
	OpenVPNServerCertificatesSecretName = "openvpn-server-certificates"
	//OpenVPNClientCertificatesSecretName is the name for the secret containing the openvpn client certificates
	OpenVPNClientCertificatesSecretName = "openvpn-client-certificates"

	//CloudConfigConfigMapName is the name for the configmap containing the cloud-config
	CloudConfigConfigMapName = "cloud-config"
	//OpenVPNClientConfigConfigMapName is the name for the configmap containing the openvpn client config used within the user cluster
	OpenVPNClientConfigConfigMapName = "openvpn-client-configs"

	//EtcdOperatorServiceAccountName is the name for the etcd-operator serviceaccount
	EtcdOperatorServiceAccountName = "etcd-operator"
	//PrometheusServiceAccountName is the name for the Prometheus serviceaccount
	PrometheusServiceAccountName = "prometheus"

	//PrometheusName is the name for the Prometheus
	PrometheusName = "prometheus"

	//PrometheusRoleName is the name for the Prometheus role
	PrometheusRoleName = "prometheus"

	//PrometheusRoleBindingName is the name for the Prometheus rolebinding
	PrometheusRoleBindingName = "prometheus"

	//EtcdOperatorClusterRoleBindingName is the name for the etcd-operator clusterrolebinding
	EtcdOperatorClusterRoleBindingName = "etcd-operator"

	//ApiserverServiceMonitorName is the name for the apiserver servicemonitor
	ApiserverServiceMonitorName = "apiserver"
	//ControllerManagerServiceMonitorName is the name for the controller manager servicemonitor
	ControllerManagerServiceMonitorName = "controller-manager"
	//EtcdServiceMonitorName is the name for the etcd servicemonitor
	EtcdServiceMonitorName = "etcd"
	//KubeStateMetricsServiceMonitorName is the name for the kube state metrics servicemonitor
	KubeStateMetricsServiceMonitorName = "kube-state-metrics"
	//MachineControllerServiceMonitorName is the name for the machine controller servicemonitor
	MachineControllerServiceMonitorName = "machine-controller"
	//SchedulerServiceMonitorName is the name for the scheduler servicemonitor
	SchedulerServiceMonitorName = "scheduler"
)

const (
	// CAKeySecretKey ca.key
	CAKeySecretKey = "ca.key"
	// CACertSecretKey ca.crt
	CACertSecretKey = "ca.crt"
	// ApiserverTLSKeySecretKey apiserver-tls.key
	ApiserverTLSKeySecretKey = "apiserver-tls.key"
	// ApiserverTLSCertSecretKey apiserver-tls.crt
	ApiserverTLSCertSecretKey = "apiserver-tls.crt"
	// KubeletClientKeySecretKey kubelet-client.key
	KubeletClientKeySecretKey = "kubelet-client.key"
	// KubeletClientCertSecretKey kubelet-client.crt
	KubeletClientCertSecretKey = "kubelet-client.crt"
	// ServiceAccountKeySecretKey sa.key
	ServiceAccountKeySecretKey = "sa.key"
	// AdminKubeconfigSecretKey admin-kubeconfig
	AdminKubeconfigSecretKey = "admin-kubeconfig"
	// TokensSecretKey tokens.csv
	TokensSecretKey = "tokens.csv"
	// OpenVPNServerKeySecretKey server.key
	OpenVPNServerKeySecretKey = "server.key"
	// OpenVPNServerCertSecretKey server.crt
	OpenVPNServerCertSecretKey = "server.crt"
	// OpenVPNInternalClientKeySecretKey client.key
	OpenVPNInternalClientKeySecretKey = "client.key"
	// OpenVPNInternalClientCertSecretKey client.crt
	OpenVPNInternalClientCertSecretKey = "client.crt"
)

// TemplateData is a group of data required for template generation
type TemplateData struct {
	Cluster         *kubermaticv1.Cluster
	Version         *apiv1.MasterVersion
	DC              *provider.DatacenterMeta
	SecretLister    corev1lister.SecretLister
	ConfigMapLister corev1lister.ConfigMapLister
	ServiceLister   corev1lister.ServiceLister
}

// GetClusterRef returns a instance of a OwnerReference for the Cluster in the TemplateData
func (d *TemplateData) GetClusterRef() metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(d.Cluster, gv.WithKind("Cluster"))
}

// Int32 returns a pointer to of the int32 value passed in.
func Int32(v int32) *int32 {
	return &v
}

// Int64 returns a pointer to of the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// Bool returns a pointer to of the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// NewTemplateData returns an instance of TemplateData
func NewTemplateData(
	cluster *kubermaticv1.Cluster,
	version *apiv1.MasterVersion,
	dc *provider.DatacenterMeta,
	secretLister corev1lister.SecretLister,
	configMapLister corev1lister.ConfigMapLister,
	serviceLister corev1lister.ServiceLister) *TemplateData {
	return &TemplateData{
		Cluster:         cluster,
		DC:              dc,
		Version:         version,
		ConfigMapLister: configMapLister,
		SecretLister:    secretLister,
		ServiceLister:   serviceLister,
	}
}

// SecretRevision returns the resource version of the secret specified by name. A empty string will be returned in case of an error
func (d *TemplateData) SecretRevision(name string) string {
	secret, err := d.SecretLister.Secrets(d.Cluster.Status.NamespaceName).Get(name)
	if err != nil {
		runtime.HandleError(fmt.Errorf("could not get secret %s from lister for cluster %s: %v", name, d.Cluster.Name, err))
		return ""
	}
	return secret.ResourceVersion
}

// ConfigMapRevision returns the resource version of the configmap specified by name. A empty string will be returned in case of an error
func (d *TemplateData) ConfigMapRevision(name string) string {
	cm, err := d.ConfigMapLister.ConfigMaps(d.Cluster.Status.NamespaceName).Get(name)
	if err != nil {
		runtime.HandleError(fmt.Errorf("could not get configmap %s from lister for cluster %s: %v", name, d.Cluster.Name, err))
		return ""
	}
	return cm.ResourceVersion
}

// ProviderName returns the name of the clusters providerName
func (d *TemplateData) ProviderName() string {
	p, err := provider.ClusterCloudProviderName(d.Cluster.Spec.Cloud)
	if err != nil {
		glog.V(0).Infof("could not identify cloud provider: %v", err)
	}
	return p
}

// GetApiserverExternalNodePort returns the nodeport of the external apiserver service
func (d *TemplateData) GetApiserverExternalNodePort() string {
	s, err := d.ServiceLister.Services(d.Cluster.Status.NamespaceName).Get(ApiserverExternalServiceName)
	if err != nil {
		if !errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("failed to get NodePort for external apiserver service: %v", err))
			return "ERROR"
		}

		return ""
	}
	return fmt.Sprintf("%d", s.Spec.Ports[0].NodePort)
}

// LoadDeploymentFile loads a k8s yaml deployment from disk and returns a Deployment struct
func LoadDeploymentFile(data *TemplateData, masterResourcesPath, yamlFile string) (*appsv1.Deployment, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, yamlFile))
	if err != nil {
		return nil, "", err
	}

	var dep appsv1.Deployment
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

// LoadRoleFile loads a role from disk, sets the namespace and returns it
func LoadRoleFile(data *TemplateData, app, masterResourcesPath string) (*rbacv1.Role, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-role.yaml"))
	if err != nil {
		return nil, "", err
	}

	var r rbacv1.Role
	json, err := t.Execute(data, &r)
	return &r, json, err
}

// LoadRoleBindingFile loads a role binding from disk, sets the namespace and returns it
func LoadRoleBindingFile(data *TemplateData, app, masterResourcesPath string) (*rbacv1.RoleBinding, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-rolebinding.yaml"))
	if err != nil {
		return nil, "", err
	}

	var r rbacv1.RoleBinding
	json, err := t.Execute(data, &r)
	return &r, json, err
}

// LoadClusterRoleBindingFile loads a role binding from disk, sets the namespace and returns it
func LoadClusterRoleBindingFile(data *TemplateData, app, masterResourcesPath string) (*rbacv1.ClusterRoleBinding, string, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-clusterrolebinding.yaml"))
	if err != nil {
		return nil, "", err
	}

	var r rbacv1.ClusterRoleBinding
	json, err := t.Execute(data, &r)
	return &r, json, err
}

// LoadMachineFile parses and returns the given machine manifest
func LoadMachineFile(filename string, c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey, version *apiv1.MasterVersion) (*machinev1alpha1.Machine, error) {
	t, err := k8stemplate.ParseFile(filename)
	if err != nil {
		return nil, err
	}

	data := struct {
		Cluster    *kubermaticv1.Cluster
		Node       *apiv2.Node
		Datacenter provider.DatacenterMeta
		Name       string
		Keys       []*kubermaticv1.UserSSHKey
		Version    *apiv1.MasterVersion
	}{
		Cluster:    c,
		Node:       node,
		Datacenter: dc,
		Keys:       keys,
		Version:    version,
	}

	var machine machinev1alpha1.Machine
	_, err = t.Execute(data, &machine)
	return &machine, err
}
