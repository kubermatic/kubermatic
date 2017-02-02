package cluster

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
	texttemplate "text/template"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/cluster/template"
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

	file := path.Join(cc.masterResourcesPath, fmt.Sprintf("%s-%s-dep.yaml", s, strings.ToLower(c.Spec.Cloud.Name)))
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return loadDeploymentFile(cc, c, s)
	}

	return loadDeploymentFile(cc, c, fmt.Sprintf("%s-%s", s, strings.ToLower(c.Spec.Cloud.Name)))
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
		addrs, err := net.LookupHost(u.Host)
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
	var conf bytes.Buffer
	cfgt, err := texttemplate.ParseFiles(path.Join(cc.masterResourcesPath, "aws-cloud-config.cfg"))
	if err != nil {
		return nil, err
	}

	if err := cfgt.Execute(&conf, struct{ Zone string }{Zone: c.Spec.Cloud.Region}); err != nil {
		return nil, err
	}

	t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-cm.yaml"))
	if err != nil {
		return nil, err
	}

	var cm v1.ConfigMap
	data := struct {
		Conf string
	}{
		Conf: conf.String(),
	}
	err = t.Execute(data, &cm)
	return &cm, err
}
