package baremetal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"golang.org/x/net/context"
)

const serviceURL = "baremetal-provider.api.svc.cluster.local"
const appJSON = "application/json"

type baremetal struct {
}

func (*baremetal) InitializeCloudSpec(cl *api.Cluster) error {
	cl.Spec.Cloud.BareMetal = &api.BareMetalCloudSpec{
		Name: cl.Metadata.Name,
	}

	url := path.Join(serviceURL, "/clusters")
	data, err := json.Marshal(cl.Spec.Cloud.BareMetal)
	if err != nil {
		return err
	}

	_, err = http.Post(url, appJSON, bytes.NewReader(data))
	if err != nil {
		return err
	}
	return nil
}

func (*baremetal) MarshalCloudSpec(cls *api.CloudSpec) (annotations map[string]string, err error) {
	annotations = map[string]string{
		"name": cls.BareMetal.Name,
	}
	return annotations, nil
}

func (*baremetal) UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error) {
	var cl *api.CloudSpec

	name, ok := annotations["name"]
	if !ok {
		return nil, errors.New("Couldn't find key 'name' while unmarshalling CloudSpec")
	}
	cl.BareMetal.Name = name

	return cl, nil
}

func (*baremetal) CreateNodes(ctx context.Context, cl *api.Cluster, _ *api.NodeSpec, num int) ([]*api.Node, error) {
	url := path.Join(serviceURL, fmt.Sprintf("/clusters/%s/nodes", cl.Metadata.Name))
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
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error creating node got status %d: %s", resp.StatusCode, string(body))
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
				Addresses: map[string]string{
					"public": n.RemoteAddress,
				},
			},
			Spec: api.NodeSpec{
				DatacenterName: cl.Spec.Cloud.DatacenterName,
				BareMetal:      &n,
			},
		}
		nodes = append(nodes, createdNode)
	}
	return nodes, nil
}

func (*baremetal) Nodes(_ context.Context, cl *api.Cluster) ([]*api.Node, error) {
	resp, err := http.Get(path.Join(serviceURL, fmt.Sprintf("/clusters/%s/nodes", cl.Metadata.Name)))
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
		//uid := strings.Split(b.ID, "-")[2]
		node := &api.Node{
			Metadata: api.Metadata{
				Name: b.ID,
				UID:  b.ID,
			},
			Status: api.NodeStatus{
				Addresses: map[string]string{},
			},
			Spec: api.NodeSpec{
				DatacenterName: cl.Spec.Cloud.DatacenterName,
				BareMetal:      b,
			},
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (*baremetal) DeleteNodes(ctx context.Context, cl *api.Cluster, UIDs []string) error {
	for _, uid := range UIDs {
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodDelete, path.Join(fmt.Sprintf("/clusters/%s/nodes/%s", cl.Metadata.Name, uid)), nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("got status code %s from baremetal service delting node", resp.Status)
		}
	}
	return nil
}

func (b *baremetal) CleanUp(cl *api.Cluster) error {
	nodes, err := b.Nodes(context.Background(), cl)
	if err != nil {
		return err
	}

	var UIDs []string
	for _, n := range nodes {
		// Node UID name pattern = "%s"
		// Not shown in the customer dashboard ?
		// TODO(realfake): Adopt Node UID naming pattern
		UIDs = append(UIDs, n.Metadata.Name)
	}
	return b.DeleteNodes(context.Background(), cl, UIDs)
}
