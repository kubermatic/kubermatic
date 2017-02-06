package baremetal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

const (
	appJSON        = "application/json"
	clusterNameKey = "bm-cluster-name"
)

type baremetal struct {
	datacenters map[string]provider.DatacenterMeta
}

// NewCloudProvider returns a new bare-metal provider.
func NewCloudProvider(datacenters map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &baremetal{
		datacenters: datacenters,
	}
}

func (b *baremetal) InitializeCloudSpec(c *api.Cluster) error {
	if c.Spec.Cloud.BareMetal != nil && c.Spec.Cloud.BareMetal.Name != "" {
		return nil
	}

	cfg := c.GetKubeconfig()

	jcfg, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal kubeconfig to json: %v", err)
	}

	ycfg, err := yaml.JSONToYAML(jcfg)
	if err != nil {
		return fmt.Errorf("failed to convert kubeconfig from json to yaml: %v", err)
	}

	c.Spec.Cloud.BareMetal = &api.BareMetalCloudSpec{
		Name: c.Metadata.Name,
	}

	Cluster := struct {
		Name         string `json:"name"`
		APIServerURL string `json:"apiserver_url"`
		Kubeconfig   string `json:"kubeconfig"`
	}{
		Name:         c.Metadata.Name,
		APIServerURL: c.Address.URL,
		Kubeconfig:   string(ycfg),
	}

	data, err := json.Marshal(Cluster)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster: %v", err)
	}

	resp, err := http.Post(b.getURL(c, "/clusters"), appJSON, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create cluster provider: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("got unexpected status code. Expected: %d Got: %s", http.StatusCreated, getLogableResponse(resp, ""))
	}

	return nil
}

func (*baremetal) MarshalCloudSpec(cs *api.CloudSpec) (annotations map[string]string, err error) {
	annotations = map[string]string{
		clusterNameKey: cs.BareMetal.Name,
	}
	return annotations, nil
}

func (*baremetal) UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error) {
	cs := api.CloudSpec{BareMetal: &api.BareMetalCloudSpec{}}

	name, ok := annotations[clusterNameKey]
	if !ok {
		return nil, fmt.Errorf("couldn't find key %q in annotations while unmarshalling CloudSpec", clusterNameKey)
	}
	cs.BareMetal.Name = name

	return &cs, nil
}

func (b *baremetal) CreateNodes(ctx context.Context, c *api.Cluster, _ *api.NodeSpec, num int) ([]*api.Node, error) {
	var nodes []*api.Node
	data, err := json.Marshal(struct {
		Number int `json:"number"`
	}{num})
	if err != nil {
		return nodes, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(b.getURL(c, fmt.Sprintf("/clusters/%s/nodes", c.Metadata.Name)), appJSON, bytes.NewReader(data))
	if err != nil {
		return nodes, fmt.Errorf("failed sending request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nodes, fmt.Errorf("got unexpected status code. Expected: %d Got: %s", http.StatusCreated, getLogableResponse(resp, ""))
	}

	var createdNodes []api.BareMetalNodeSpec

	defer func(r *http.Response) {
		err = r.Body.Close()
		if err != nil {
			glog.Errorf("failed to close response body: %v", err)
		}
	}(resp)

	err = json.NewDecoder(resp.Body).Decode(resp.Body)
	if err != nil {
		return nodes, fmt.Errorf("failed to decode response body: %v. response: %s", err, getLogableResponse(resp, ""))
	}
	for _, n := range createdNodes {
		createdNode := &api.Node{
			Metadata: api.Metadata{
				Name: n.ID,
				UID:  n.ID,
			},
			Status: api.NodeStatus{
				Addresses: api.NodeAddresses{
					Public: n.RemoteAddress,
				},
			},
			Spec: api.NodeSpec{
				DatacenterName: c.Spec.Cloud.DatacenterName,
				BareMetal:      &n,
			},
		}
		nodes = append(nodes, createdNode)
	}
	return nodes, nil
}

func (b *baremetal) Nodes(_ context.Context, c *api.Cluster) ([]*api.Node, error) {
	resp, err := http.Get(b.getURL(c, fmt.Sprintf("/clusters/%s/nodes", c.Metadata.Name)))
	if err != nil {
		return nil, fmt.Errorf("failed sending request: %v", err)
	}

	var providerNodes []api.BareMetalNodeSpec

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading body from response: %v", err)
	}
	err = json.Unmarshal(body, &providerNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v. response: %s", err, getLogableResponse(resp, string(body)))
	}

	var nodes []*api.Node
	for _, pn := range providerNodes {
		node := &api.Node{
			Metadata: api.Metadata{
				Name: pn.ID,
				UID:  pn.ID,
			},
			Status: api.NodeStatus{
				Addresses: api.NodeAddresses{},
			},
			Spec: api.NodeSpec{
				DatacenterName: c.Spec.Cloud.DatacenterName,
				BareMetal:      &pn,
			},
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (b *baremetal) DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error {
	client := &http.Client{}
	for _, uid := range UIDs {
		req, err := http.NewRequest(http.MethodDelete, b.getURL(c, fmt.Sprintf("/clusters/%s/nodes/%s", c.Metadata.Name, uid)), nil)
		if err != nil {
			return fmt.Errorf("failed creating request: %v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed sending request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("got unexpected status code. Expected: %d Got: %s", http.StatusOK, getLogableResponse(resp, ""))
		}
	}
	return nil
}

func (b *baremetal) CleanUp(c *api.Cluster) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodDelete, b.getURL(c, fmt.Sprintf("/clusters/%s", c.Metadata.Name)), nil)
	if err != nil {
		return fmt.Errorf("failed creating request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed sending request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got unexpected status code. Expected: %d Got: %s", http.StatusOK, getLogableResponse(resp, ""))
	}
	return nil
}

func getLogableResponse(r *http.Response, body string) string {
	if body == "" {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			glog.Errorf("failed to get body from response: %v", err)
			return ""
		}
		body = string(b)
	}

	return fmt.Sprintf("%s %s %d %s", r.Request.Method, r.Request.URL.String(), r.StatusCode, body)
}

func (b *baremetal) getURL(c *api.Cluster, p string) string {
	u, _ := url.Parse(b.datacenters[c.Spec.Cloud.DatacenterName].Spec.BareMetal.URL)
	u.Path = p
	return u.String()
}
