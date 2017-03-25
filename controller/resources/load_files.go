package resources

import (
	"fmt"
	"net"
	"net/url"
	"path"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/template"
	"github.com/kubermatic/api/provider"
	extensionsv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

// LoadDeploymentFile loads a k8s yaml deloyment from disk and returns a Deployment struct
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
