// +build integration

package digitalocean

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/kubermatic/api"
)

func TestCreate(t *testing.T) {
	do := api.DigitaloceanCloudSpec{
		DC:    "fra1",
		Token: "c465373bf74b4d8eca066c71b172a5ba19ddf4c7910a9f5a7b6e39e26697c2d6",
		SSHKeys: []string{
			"c6:d8:74:95:a0:3c:48:c1:15:ab:19:db:52:cb:08:97",
		},
	}

	cluster := api.Cluster{
		Metadata: api.Metadata{
			Name: "test",
			UID:  "1",
		},
		Spec: api.ClusterSpec{
			Cloud: &api.CloudSpec{
				Digitalocean: &do,
			},
		},
	}

	var spec api.NodeSpec
	spec.OS = "coreos-stable"
	spec.Type = "512mb"

	provider := NewDigitialoceanCloudProvider()
	t.Log(provider.CreateNode(context.Background(), &cluster, &spec, 1))
}
