package baremetal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	clusterNameKey = "bm-cluster-name"
)

type baremetal struct {
	datacenters map[string]provider.DatacenterMeta
	client      *http.Client
}

// NewCloudProvider returns a new bare-metal provider.
func NewCloudProvider(datacenters map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &baremetal{
		datacenters: datacenters,
		client:      http.DefaultClient,
	}
}

func (b *baremetal) getAuthenticatedRequest(c *api.Cluster, method, path string, body io.Reader) (*http.Request, error) {
	bmSpec := b.datacenters[c.Spec.Cloud.DatacenterName].Spec.BareMetal
	u, _ := url.Parse(bmSpec.URL)
	u.Path = path

	r, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if bmSpec.AuthUser != "" || bmSpec.AuthPassword != "" {
		r.SetBasicAuth(bmSpec.AuthUser, bmSpec.AuthPassword)
	}
	return r, nil
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
		Name               string `json:"name"`
		APIServerURL       string `json:"apiserver_url"`
		Kubeconfig         string `json:"kubeconfig"`
		ApiserverSSHPubKey string `json:"apiserver_ssh_pub_key"`
		EtcdURL            string `json:"etcd_url"`
		RootCACert         string `json:"root_ca_cert"`
	}{
		Name:               c.Metadata.Name,
		APIServerURL:       c.Address.URL,
		Kubeconfig:         base64.StdEncoding.EncodeToString(ycfg),
		ApiserverSSHPubKey: c.Status.ApiserverSSH,
		EtcdURL:            c.Address.EtcdURL,
		RootCACert:         c.Status.RootCA.Cert.Base64(),
	}

	data, err := json.Marshal(Cluster)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster: %v", err)
	}

	r, err := b.getAuthenticatedRequest(c, http.MethodPost, "/clusters", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create cluster create request: %v", err)
	}
	resp, err := b.client.Do(r)
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
	for r := 0; r <= num; r++ {
		clientKC, err := c.CreateKeyCert(provider.ShortUID(5), []string{})
		if err != nil {
			return nodes, err
		}

		var createNodesReq = struct {
			ClientKey  string `json:"client_key"`
			ClientCert string `json:"client_cert"`
		}{
			ClientKey:  clientKC.Key.Base64(),
			ClientCert: clientKC.Cert.Base64(),
		}

		data, err := json.Marshal(createNodesReq)
		if err != nil {
			return nodes, fmt.Errorf("failed to marshal create node request: %v", err)
		}
		r, err := b.getAuthenticatedRequest(c, http.MethodPost, fmt.Sprintf("/clusters/%s/add_node", c.Metadata.Name), bytes.NewReader(data))
		if err != nil {
			return nodes, fmt.Errorf("failed to create assign node request: %v", err)
		}
		resp, err := b.client.Do(r)
		if err != nil {
			return nodes, fmt.Errorf("failed sending assign nodes request: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
				return nodes, errors.New("not enough free nodes available")
			}
			return nodes, fmt.Errorf("got unexpected status code. Expected: %d Got: %s", http.StatusCreated, getLogableResponse(resp, ""))
		}

		var createdNode api.BareMetalNodeSpec
		err = json.NewDecoder(resp.Body).Decode(&createdNode)
		_ = resp.Close
		if err != nil {
			return nodes, fmt.Errorf("failed to decode response body: %v. response: %s", err, getLogableResponse(resp, ""))
		}

		nodes = append(nodes, &api.Node{
			Metadata: api.Metadata{
				Name: createdNode.ID,
				UID:  createdNode.ID,
			},
			Status: api.NodeStatus{
				Addresses: api.NodeAddresses{
					Public: createdNode.PublicIP,
				},
			},
			Spec: api.NodeSpec{
				DatacenterName: c.Spec.Cloud.DatacenterName,
				BareMetal:      &createdNode,
			},
		})
	}
	return nodes, nil
}

func (b *baremetal) Nodes(_ context.Context, c *api.Cluster) ([]*api.Node, error) {
	r, err := b.getAuthenticatedRequest(c, http.MethodGet, fmt.Sprintf("/clusters/%s/nodes", c.Metadata.Name), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster create request: %v", err)
	}
	resp, err := b.client.Do(r)
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
	for _, uid := range UIDs {
		r, err := b.getAuthenticatedRequest(c, http.MethodDelete, fmt.Sprintf("/clusters/%s/nodes/%s", c.Metadata.Name, uid), nil)
		if err != nil {
			return fmt.Errorf("failed to create cluster create request: %v", err)
		}
		resp, err := b.client.Do(r)
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
	r, err := b.getAuthenticatedRequest(c, http.MethodDelete, fmt.Sprintf("/clusters/%s", c.Metadata.Name), nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster create request: %v", err)
	}
	resp, err := b.client.Do(r)
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
