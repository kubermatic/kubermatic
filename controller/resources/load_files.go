package resources

import (
	"net"
	"net/url"
	"path"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/template"

	extensionsv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func LoadDeploymentFile(c *api.Cluster, v *api.MasterVersion, masterResourcesPath, overwriteHost, dc, yamlFile string) (*extensionsv1beta1.Deployment, error) {
	data := struct {
		DC               string
		ClusterName      string
		AdvertiseAddress string
		SecurePort       int
		Version          *api.MasterVersion
	}{
		DC:          dc,
		ClusterName: c.Metadata.Name,
		SecurePort:  c.Address.NodePort,
		Version:     v,
	}

	if overwriteHost == "" {
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
		data.AdvertiseAddress = overwriteHost
	}

	t, err := template.ParseFiles(path.Join(masterResourcesPath, yamlFile))
	if err != nil {
		return nil, err
	}

	var dep extensionsv1beta1.Deployment
	err = t.Execute(data, &dep)
	return &dep, err
}
