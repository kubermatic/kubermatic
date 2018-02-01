package resources

import (
	"bytes"
	"fmt"
	"net"
	"path"
	"text/template"

	"github.com/Masterminds/sprig"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8stemplate "github.com/kubermatic/kubermatic/api/pkg/template/kubernetes"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
)

const (
	// EtcdClusterName is the name of the etcd cluster
	EtcdClusterName = "etcd-cluster"
)

// LoadDeploymentFile loads a k8s yaml deployment from disk and returns a Deployment struct
func LoadDeploymentFile(c *kubermaticv1.Cluster, v *apiv1.MasterVersion, masterResourcesPath, yamlFile string) (*v1beta1.Deployment, error) {
	p, err := provider.ClusterCloudProviderName(c.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("could not identify cloud provider: %v", err)
	}
	data := struct {
		DC               string
		AdvertiseAddress string
		Cluster          *kubermaticv1.Cluster
		Version          *apiv1.MasterVersion
		CloudProvider    string
	}{
		Cluster:       c,
		Version:       v,
		CloudProvider: p,
	}

	addrs, err := net.LookupHost(c.Address.ExternalName)
	if err != nil {
		return nil, err
	}
	data.AdvertiseAddress = addrs[0]

	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, yamlFile))
	if err != nil {
		return nil, err
	}

	var dep v1beta1.Deployment
	err = t.Execute(data, &dep)
	return &dep, err
}

// LoadServiceFile returns the service for the given cluster and app
func LoadServiceFile(c *kubermaticv1.Cluster, app, masterResourcesPath string) (*corev1.Service, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-service.yaml"))
	if err != nil {
		return nil, err
	}

	var service corev1.Service

	data := struct {
		Cluster *kubermaticv1.Cluster
	}{
		Cluster: c,
	}

	err = t.Execute(data, &service)

	return &service, err
}

// LoadSecretFile returns the secret for the given cluster and app
func LoadSecretFile(c *kubermaticv1.Cluster, app, masterResourcesPath string) (*corev1.Secret, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-secret.yaml"))
	if err != nil {
		return nil, err
	}

	var secret corev1.Secret
	data := struct {
		Cluster *kubermaticv1.Cluster
	}{
		Cluster: c,
	}

	err = t.Execute(data, &secret)

	return &secret, err
}

// LoadIngressFile returns the ingress for the given cluster and app
func LoadIngressFile(c *kubermaticv1.Cluster, app, masterResourcesPath string) (*v1beta1.Ingress, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-ingress.yaml"))
	if err != nil {
		return nil, err
	}
	var ingress v1beta1.Ingress
	data := struct {
		Cluster *kubermaticv1.Cluster
	}{
		Cluster: c,
	}
	err = t.Execute(data, &ingress)

	if err != nil {
		return nil, err
	}

	return &ingress, err
}

// LoadPVCFile returns the PVC for the given cluster & app
func LoadPVCFile(c *kubermaticv1.Cluster, app, masterResourcesPath string) (*corev1.PersistentVolumeClaim, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-pvc.yaml"))
	if err != nil {
		return nil, err
	}

	var pvc corev1.PersistentVolumeClaim
	data := struct {
		ClusterName string
	}{
		ClusterName: c.Name,
	}
	err = t.Execute(data, &pvc)
	return &pvc, err
}

// LoadAwsCloudConfigConfigMap returns the aws cloud config configMap for the cluster
func LoadAwsCloudConfigConfigMap(c *kubermaticv1.Cluster, dc *provider.DatacenterMeta, version *apiv1.MasterVersion) (*corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{}
	cm.Name = "cloud-config"
	cm.APIVersion = "v1"
	cm.Data = map[string]string{
		"config": fmt.Sprintf(`
[global]
zone=%s
VPC=%s
kubernetesclustertag=%s
disablesecuritygroupingress=false
SubnetID=%s
RouteTableID=%s
disablestrictzonecheck=true`,
			c.Spec.Cloud.AWS.AvailabilityZone,
			c.Spec.Cloud.AWS.VPCID,
			c.Name,
			c.Spec.Cloud.AWS.SubnetID,
			c.Spec.Cloud.AWS.RouteTableID,
		),
	}
	return &cm, nil
}

// LoadOpenstackCloudConfigConfigMap returns the aws cloud config configMap for the cluster
func LoadOpenstackCloudConfigConfigMap(c *kubermaticv1.Cluster, dc *provider.DatacenterMeta, version *apiv1.MasterVersion) (*corev1.ConfigMap, error) {
	tmpl := `[Global]
auth-url = "{{ .DC.Spec.Openstack.AuthURL }}"
username = "{{ .Cluster.Spec.Cloud.Openstack.Username }}"
password = "{{ .Cluster.Spec.Cloud.Openstack.Password }}"
domain-name= "{{ .Cluster.Spec.Cloud.Openstack.Domain }}"
tenant-name = "{{ .Cluster.Spec.Cloud.Openstack.Tenant }}"

[BlockStorage]
trust-device-path = false
bs-version = "v2"
{{- if eq (substr 0 4 (index .Version.Values "k8s-version")) "v1.9" }}
ignore-volume-az = {{ .DC.Spec.Openstack.IgnoreVolumeAZ }}
{{- end }}
`

	t, err := template.New("cloud-config").Funcs(sprig.TxtFuncMap()).Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}

	data := struct {
		Cluster *kubermaticv1.Cluster
		DC      *provider.DatacenterMeta
		Version *apiv1.MasterVersion
	}{
		Cluster: c,
		DC:      dc,
		Version: version,
	}

	b := &bytes.Buffer{}
	err = t.Execute(b, data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %v", err)
	}

	cm := corev1.ConfigMap{}
	cm.Name = "cloud-config"
	cm.APIVersion = "v1"
	cm.Data = map[string]string{
		"config": b.String(),
	}
	return &cm, nil
}

// LoadEtcdClusterFile loads a etcd-operator crd from disk and returns a Cluster crd struct
func LoadEtcdClusterFile(v *apiv1.MasterVersion, masterResourcesPath, yamlFile string) (*etcdoperatorv1beta2.EtcdCluster, error) {

	data := struct {
		Version *apiv1.MasterVersion
	}{
		Version: v,
	}

	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, yamlFile))
	if err != nil {
		return nil, err
	}

	var c etcdoperatorv1beta2.EtcdCluster
	err = t.Execute(data, &c)
	return &c, err
}

// LoadServiceAccountFile loads a service account from disk and returns it
func LoadServiceAccountFile(app, masterResourcesPath string) (*corev1.ServiceAccount, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-serviceaccount.yaml"))
	if err != nil {
		return nil, err
	}

	var sa corev1.ServiceAccount
	err = t.Execute(nil, &sa)
	return &sa, err
}

// LoadClusterRoleBindingFile loads a role binding from disk, sets the namespace and returns it
func LoadClusterRoleBindingFile(ns, app, masterResourcesPath string) (*rbacv1beta1.ClusterRoleBinding, error) {
	t, err := k8stemplate.ParseFile(path.Join(masterResourcesPath, app+"-rolebinding.yaml"))
	if err != nil {
		return nil, err
	}

	data := struct {
		Namespace string
	}{
		Namespace: ns,
	}

	var r rbacv1beta1.ClusterRoleBinding
	err = t.Execute(data, &r)
	return &r, err
}
