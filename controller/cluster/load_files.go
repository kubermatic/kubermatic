package cluster

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/cluster/template"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/controller/template"

	"k8s.io/client-go/pkg/api/v1"
	extensionsv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func loadServiceFile(cc *clusterController, c *api.Cluster, s string) (*v1.Service, error) {
	t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-service.yaml"))
	if err != nil {
		return nil, err
	}

	var service v1.Service

	data := struct {
		SecurePort int
	}{
		SecurePort: c.Address.NodePort,
	}

	err = t.Execute(data, &service)

	return &service, err
}

func loadIngressFile(cc *clusterController, c *api.Cluster, s string) (*extensionsv1beta1.Ingress, error) {
	t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-ingress.yaml"))
	if err != nil {
		return nil, err
	}
	var ingress extensionsv1beta1.Ingress
	data := struct {
		DC          string
		ClusterName string
		ExternalURL string
	}{
		DC:          cc.dc,
		ClusterName: c.Metadata.Name,
		ExternalURL: cc.externalURL,
	}
	err = t.Execute(data, &ingress)

	if err != nil {
		return nil, err
	}

	return &ingress, err
}

func loadDeploymentFile(cc *clusterController, c *api.Cluster, s string) (*extensionsv1beta1.Deployment, error) {
	t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-dep.yaml"))
	if err != nil {
		return nil, err
	}

	var dep extensionsv1beta1.Deployment
	data := struct {
		DC          string
		ClusterName string
		Cluster     *api.Cluster
	}{
		DC:          cc.dc,
		ClusterName: c.Metadata.Name,
		Cluster:     c,
	}
	err = t.Execute(data, &dep)
	return &dep, err
}

func loadDeploymentFileControllerManager(cc *clusterController, c *api.Cluster, s string) (*extensionsv1beta1.Deployment, error) {
	if nil == c.Spec.Cloud {
		return loadDeploymentFile(cc, c, s)
	}

	cloud, err := provider.ClusterCloudProviderName(c.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud provider name from cloud: %v", err)
	}
	filename := fmt.Sprintf("%s-%s-dep.yaml", s, strings.ToLower(cloud))
	file := path.Join(cc.masterResourcesPath, filename)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		glog.Infof("No cloud provider specific deployment found for %q", filename)
		return loadDeploymentFile(cc, c, s)
	}

	return loadDeploymentFile(cc, c, fmt.Sprintf("%s-%s", s, strings.ToLower(cloud)))
}

func loadApiserver(cc *clusterController, c *api.Cluster, s string) (*extensionsv1beta1.Deployment, error) {
	var data struct {
		AdvertiseAddress string
		SecurePort       int
	}

	if cc.overwriteHost == "" {
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
	} else {
		data.AdvertiseAddress = cc.overwriteHost
	}
	data.SecurePort = c.Address.NodePort

	t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-dep.yaml"))
	if err != nil {
		return nil, err
	}

	var dep extensionsv1beta1.Deployment
	err = t.Execute(data, &dep)
	return &dep, err
}

func loadPVCFile(cc *clusterController, c *api.Cluster, s string) (*v1.PersistentVolumeClaim, error) {
	t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-pvc.yaml"))
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

func loadAwsCloudConfigConfigMap(cc *clusterController, c *api.Cluster, s string) (*v1.ConfigMap, error) {
	cm := v1.ConfigMap{}
	cm.Name = "aws-cloud-config"
	cm.APIVersion = "v1"
	cm.Data = map[string]string{
		"aws-cloud-config": fmt.Sprintf(`
[global]
zone=%s
kubernetesclustertag=
disablesecuritygroupingress=false
disablestrictzonecheck=true`, c.Spec.Cloud.AWS.AvailabilityZone),
	}
	return &cm, nil

}
