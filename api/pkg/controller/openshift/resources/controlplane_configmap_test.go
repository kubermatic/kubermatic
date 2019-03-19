package resources

import (
	"context"
	"log"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeMasterConfigData struct {
	cluster  *kubermaticv1.Cluster
	nodePort int32
}

func (d *fakeMasterConfigData) Cluster() *kubermaticv1.Cluster {
	return d.cluster
}

func (d *fakeMasterConfigData) GetApiserverExternalNodePort(context.Context) (int32, error) {
	return d.nodePort, nil
}

func TestGetMasterConfig(t *testing.T) {
	ctx := context.Background()
	data := &fakeMasterConfigData{
		nodePort: 32593,
		cluster: &kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "aws-1",
			},
			Address: kubermaticv1.ClusterAddress{
				URL: "https://henrik-os1.europe-west3-c.dev.kubermatic.io:32593",
				IP:  "35.198.93.90",
			},
			Status: kubermaticv1.ClusterStatus{
				NamespaceName: "cluster-aws-1",
			},
			Spec: kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"10.10.10.0/24"},
					},
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"172.25.0.0/16"},
					},
					DNSDomain: "cluster.local",
				},
				Cloud: kubermaticv1.CloudSpec{
					DatacenterName: "aws-dummy",
					AWS: &kubermaticv1.AWSCloudSpec{
						AccessKeyID:     "access-key-id",
						SecretAccessKey: "secret-access-key",
					},
				},
			},
		},
	}

	config, err := getMasterConfig(ctx, data, "apiserver")
	if err != nil {
		log.Fatal(err)
	}
	t.Log(config)
}
