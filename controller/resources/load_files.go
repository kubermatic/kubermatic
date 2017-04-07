package resources

import (
	"fmt"
	"net"
	"net/url"
	"path"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/template"
	"github.com/kubermatic/api/extensions/etcd-cluster"
	"github.com/kubermatic/api/provider"
	"k8s.io/client-go/pkg/api/v1"
	extensionsv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

// LoadDeploymentFile loads a k8s yaml deployment from disk and returns a Deployment struct
func LoadDeploymentFile(c *api.Cluster, v *api.MasterVersion, masterResourcesPath, dc, yamlFile string) (*extensionsv1beta1.Deployment, error) {
	p, err := provider.ClusterCloudProviderName(c.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("could not identify cloud provider: %v", err)
	}
	data := struct {
		DC               string
		AdvertiseAddress string
		Cluster          *api.Cluster
		Version          *api.MasterVersion
		CloudProvider    string
	}{
		DC:            dc,
		Cluster:       c,
		Version:       v,
		CloudProvider: p,
	}

	u, err := url.Parse(c.Address.URL)
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, err
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}
	data.AdvertiseAddress = addrs[0]

	t, err := template.ParseFiles(path.Join(masterResourcesPath, yamlFile))
	if err != nil {
		return nil, err
	}

	var dep extensionsv1beta1.Deployment
	err = t.Execute(data, &dep)
	return &dep, err
}

// LoadServiceFile returns the service for the given cluster and app
func LoadServiceFile(c *api.Cluster, app, masterResourcesPath string) (*v1.Service, error) {
	t, err := template.ParseFiles(path.Join(masterResourcesPath, app+"-service.yaml"))
	if err != nil {
		return nil, err
	}

	var service v1.Service

	data := struct {
		Cluster *api.Cluster
	}{
		Cluster: c,
	}

	err = t.Execute(data, &service)

	return &service, err
}

// LoadIngressFile returns the ingress for the given cluster and app
func LoadIngressFile(c *api.Cluster, app, masterResourcesPath, dc, externalURL string) (*extensionsv1beta1.Ingress, error) {
	t, err := template.ParseFiles(path.Join(masterResourcesPath, app+"-ingress.yaml"))
	if err != nil {
		return nil, err
	}
	var ingress extensionsv1beta1.Ingress
	data := struct {
		DC          string
		ClusterName string
		ExternalURL string
	}{
		DC:          dc,
		ClusterName: c.Metadata.Name,
		ExternalURL: externalURL,
	}
	err = t.Execute(data, &ingress)

	if err != nil {
		return nil, err
	}

	return &ingress, err
}

// LoadPVCFile returns the PVC for the given cluster & app
func LoadPVCFile(c *api.Cluster, app, masterResourcesPath string) (*v1.PersistentVolumeClaim, error) {
	t, err := template.ParseFiles(path.Join(masterResourcesPath, app+"-pvc.yaml"))
	if err != nil {
		return nil, err
	}

	var pvc v1.PersistentVolumeClaim
	data := struct {
		ClusterName string
	}{
		ClusterName: c.Metadata.Name,
	}
	err = t.Execute(data, &pvc)
	return &pvc, err
}

// LoadAwsCloudConfigConfigMap returns the aws cloud config configMap for the cluster
func LoadAwsCloudConfigConfigMap(c *api.Cluster) (*v1.ConfigMap, error) {
	cm := v1.ConfigMap{}
	cm.Name = "aws-cloud-config"
	cm.APIVersion = "v1"
	cm.Data = map[string]string{
		"aws-cloud-config": fmt.Sprintf(`
[global]
zone=%s
VPC=%s
kubernetesclustertag=%s
disablesecuritygroupingress=false
disablestrictzonecheck=true`, c.Spec.Cloud.AWS.AvailabilityZone, c.Spec.Cloud.AWS.VPCId, c.Metadata.Name),
	}
	return &cm, nil

}

// LoadEtcdClustertFile loads a etcd-operator tpr from disk and returns a Cluster tpr struct
func LoadEtcdClustertFile(c *api.Cluster, v *api.MasterVersion, masterResourcesPath, dc, yamlFile string) (*etcd_cluster.Cluster, error) {

	data := struct {
		ClusterName string
	}{
		ClusterName: c.Metadata.Name,
	}

	t, err := template.ParseFiles(path.Join(masterResourcesPath, yamlFile))
	if err != nil {
		return nil, err
	}

	var etcd etcd_cluster.Cluster
	err = t.Execute(data, &etcd)
	return &etcd, err
}
