package baremetal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

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
	cfg := c.GetKubeconfig()

	jcfg, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	ycfg, err := yaml.JSONToYAML(jcfg)
	if err != nil {
		return err
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
		return err
	}

	resp, err := http.Post(b.datacenters[c.Spec.Cloud.DatacenterName].Spec.BareMetal.URL+"/clusters", appJSON, bytes.NewReader(data))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		glog.Errorf("Got bad response from bare-metal provider during cluster creation: %s", getLogableResponse(resp))
		return errors.New("provider returned not successful status code")
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
		return nil, fmt.Errorf("Couldn't find key %q while unmarshalling CloudSpec", clusterNameKey)
	}
	cs.BareMetal.Name = name

	return &cs, nil
}

func (b *baremetal) CreateNodes(ctx context.Context, c *api.Cluster, _ *api.NodeSpec, num int) ([]*api.Node, error) {
	url := b.datacenters[c.Spec.Cloud.DatacenterName].Spec.BareMetal.URL + fmt.Sprintf("/clusters/%s/nodes", c.Metadata.Name)
	var nodes []*api.Node
	data, err := json.Marshal(struct {
		Number int `json:"number"`
	}{num})
	if err != nil {
		return nodes, err
	}

	resp, err := http.Post(url, appJSON, bytes.NewReader(data))
	if err != nil {
		return nodes, err
	}

	if resp.StatusCode != http.StatusCreated {
		glog.Errorf("Got bad response from bare-metal provider during node creation: %s", getLogableResponse(resp))
		return nodes, errors.New("provider returned not successful status code")
	}

	var createdNodes []api.BareMetalNodeSpec

	defer func(r *http.Response) {
		err = r.Body.Close()
		if err != nil {
			glog.Error(err)
		}
	}(resp)

	err = json.NewDecoder(resp.Body).Decode(resp.Body)
	if err != nil {
		return nodes, err
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
	resp, err := http.Get(b.datacenters[c.Spec.Cloud.DatacenterName].Spec.BareMetal.URL + fmt.Sprintf("/clusters/%s/nodes", c.Metadata.Name))
	if err != nil {
		return nil, err
	}

	var bareNodes []*api.BareMetalNodeSpec
	err = json.NewDecoder(resp.Body).Decode(bareNodes)
	if err != nil {
		return nil, err
	}

	var nodes []*api.Node
	for _, b := range bareNodes {
		node := &api.Node{
			Metadata: api.Metadata{
				Name: b.ID,
				UID:  b.ID,
			},
			Status: api.NodeStatus{
				Addresses: api.NodeAddresses{},
			},
			Spec: api.NodeSpec{
				DatacenterName: c.Spec.Cloud.DatacenterName,
				BareMetal:      b,
			},
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (b *baremetal) DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error {
	client := &http.Client{}
	for _, uid := range UIDs {
		req, err := http.NewRequest(http.MethodDelete, b.datacenters[c.Spec.Cloud.DatacenterName].Spec.BareMetal.URL+fmt.Sprintf("/clusters/%s/nodes/%s", c.Metadata.Name, uid), nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			glog.Errorf("Got bad response from bare-metal provider during delete node: %s", getLogableResponse(resp))
			return errors.New("provider returned not successful status code")
		}
	}
	return nil
}

func (b *baremetal) CleanUp(c *api.Cluster) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodDelete, b.datacenters[c.Spec.Cloud.DatacenterName].Spec.BareMetal.URL+fmt.Sprintf("/clusters/%s", c.Metadata.Name), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		glog.Errorf("Got bad response from bare-metal provider during cleanup: %s", getLogableResponse(resp))
		return errors.New("provider returned not successful status code")
	}
	return nil
}

func getLogableResponse(r *http.Response) string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("failed to get body from response: %v", err)
		return ""
	}

	return fmt.Sprintf("StatusCode=%d, Body=%d", r.StatusCode, string(body))
}
