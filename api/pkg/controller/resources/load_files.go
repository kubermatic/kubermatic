package resources

import (
	"fmt"
	"path"
	"strconv"

	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	prometheusv1 "github.com/kubermatic/kubermatic/api/pkg/crd/prometheus/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8stemplate "github.com/kubermatic/kubermatic/api/pkg/template/kubernetes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"

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
	//TokenUsersSecretName is the name for the secret containing the user tokens
	TokenUsersSecretName = "token-users"
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

const (
	minNodePort = 30000
	maxNodePort = 32767
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
			runtime.HandleError(fmt.Errorf("failed to get NodePort for external apiserver service"))
			return "UNKNOWN"
		}
		return "UNKNOWN"
	}
	return fmt.Sprintf("%d", s.Spec.Ports[0].NodePort)
}

// GetApiserverExternalNodePortOrGetFree returns the nodeport of the external apiserver service or returns the next free one
func (d *TemplateData) GetApiserverExternalNodePortOrGetFree() string {
	p := d.GetApiserverExternalNodePort()
	if p == "UNKNOWN" {
		return d.GetFreeNodePort()
	}
	return p
}

// GetFreeNodePort returns the next free nodeport
func (d *TemplateData) GetFreeNodePort() string {
	services, err := d.ServiceLister.List(labels.Everything())
	if err != nil {
		runtime.HandleError(fmt.Errorf("failed to get free NodePort"))
		return "UNKNOWN"
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
			return strconv.Itoa(i)
		}
	}

	runtime.HandleError(fmt.Errorf("no free nodeports available"))
	return "UNKNOWN"
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
