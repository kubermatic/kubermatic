package kubernetes

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	texttemplate "text/template"

	"github.com/go-test/deep"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Data struct {
	Cluster    *kubermaticv1.Cluster
	Node       *apiv2.Node
	Datacenter provider.DatacenterMeta
	Name       string
	Keys       []*kubermaticv1.UserSSHKey
	Version    *apiv1.MasterVersion
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		fixture  string
		data     interface{}
		ret      error
	}{
		{
			name:     "get valid machine.yaml for Digitalocean",
			filename: "../../../../config/kubermatic/static/nodes/machine.yaml",
			fixture:  "machine-digitalocean",
			data: Data{
				Cluster: &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "docluster-1a2b3c4d5e",
					},
					Address: &kubermaticv1.ClusterAddress{},
					Status:  kubermaticv1.ClusterStatus{},
					Spec: kubermaticv1.ClusterSpec{
						Cloud: &kubermaticv1.CloudSpec{
							DatacenterName: "do-fra1",
							Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
								Token: "digitalocean-token",
							},
						},
					},
				},
				Node: &apiv2.Node{
					Metadata: apiv2.ObjectMeta{
						Name: "docluster-1a2b3c4d5e-te5s7",
					},
					Spec: apiv2.NodeSpec{
						Cloud: apiv2.NodeCloudSpec{
							Digitalocean: &apiv2.DigitaloceanNodeSpec{
								Size:       "s-1vcpu-1gb",
								Backups:    false,
								IPv6:       false,
								Monitoring: true,
								Tags: []string{
									"digitalocean-example-tag-1",
									"digitalocean-example-tag-2",
								},
							},
						},
						OperatingSystem: apiv2.OperatingSystemSpec{
							Ubuntu: &apiv2.UbuntuSpec{
								DistUpgradeOnBoot: false,
							},
						},
						Versions: apiv2.NodeVersionInfo{
							Kubelet: "v1.9.6",
						},
					},
					Status: apiv2.NodeStatus{},
				},
				Datacenter: provider.DatacenterMeta{
					Location: "Frankfurt",
					Seed:     "europe-west3-c",
					Country:  "DE",
					Spec: provider.DatacenterSpec{
						Digitalocean: &provider.DigitaloceanSpec{
							Region: "fra1",
						},
					},
					IsSeed: false,
				},
				Keys: []*kubermaticv1.UserSSHKey{
					&kubermaticv1.UserSSHKey{
						Spec: kubermaticv1.SSHKeySpec{
							Owner:       "John Doe",
							Name:        "ssh-key-name",
							Fingerprint: "1234:56789:1234:56789",
							PublicKey:   "ssh-rsa TEST123test",
							Clusters: []string{
								"docluster-1a2b3c4d5e",
							},
						},
					},
				},
				Version: &apiv1.MasterVersion{},
			},
			ret: nil,
		},
		{
			name:     "get valid machine.yaml for AWS",
			filename: "../../../../config/kubermatic/static/nodes/machine.yaml",
			fixture:  "machine-aws",
			data: Data{
				Cluster: &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "awscluster-1a2b3c4d5e",
					},
					Address: &kubermaticv1.ClusterAddress{},
					Status:  kubermaticv1.ClusterStatus{},
					Spec: kubermaticv1.ClusterSpec{
						Cloud: &kubermaticv1.CloudSpec{
							DatacenterName: "aws-eu-central-1a",
							AWS: &kubermaticv1.AWSCloudSpec{
								AccessKeyID:         "aws-access-key-id",
								SecretAccessKey:     "aws-secret-access-key",
								VPCID:               "aws-vpc-ic",
								SubnetID:            "aws-subnet-id",
								RoleName:            "aws-role-name",
								RouteTableID:        "aws-route-table-id",
								InstanceProfileName: "aws-instance-profile-name",
								SecurityGroup:       "aws-security-group",
								SecurityGroupID:     "aws-security-group-id",
								AvailabilityZone:    "aws-availability-zone",
							},
						},
					},
				},
				Node: &apiv2.Node{
					Metadata: apiv2.ObjectMeta{
						Name: "awscluster-1a2b3c4d5e-te5s7",
					},
					Spec: apiv2.NodeSpec{
						Cloud: apiv2.NodeCloudSpec{
							AWS: &apiv2.AWSNodeSpec{
								InstanceType: "t2.micro",
								VolumeSize:   25,
								VolumeType:   "standard",
								AMI:          "aws-ami",
								Tags: map[string]string{
									"AWSExampleTagKey1": "AWSExampleTagValue1",
									"AWSExampleTagKey2": "AWSExampleTagValue2",
								},
							},
						},
						OperatingSystem: apiv2.OperatingSystemSpec{
							Ubuntu: &apiv2.UbuntuSpec{
								DistUpgradeOnBoot: false,
							},
						},
						Versions: apiv2.NodeVersionInfo{
							Kubelet: "v1.9.6",
						},
					},
					Status: apiv2.NodeStatus{},
				},
				Datacenter: provider.DatacenterMeta{
					Location: "Frankfurt",
					Seed:     "europe-west3-c",
					Country:  "DE",
					Spec: provider.DatacenterSpec{
						AWS: &provider.AWSSpec{
							Region:        "fra1",
							AMI:           "aws-ami",
							ZoneCharacter: "aws-zone-character",
						},
					},
					IsSeed: false,
				},
				Keys: []*kubermaticv1.UserSSHKey{
					&kubermaticv1.UserSSHKey{
						Spec: kubermaticv1.SSHKeySpec{
							Owner:       "John Doe",
							Name:        "ssh-key-name",
							Fingerprint: "1234:56789:1234:56789",
							PublicKey:   "ssh-rsa TEST123test",
							Clusters: []string{
								"awscluster-1a2b3c4d5e",
							},
						},
					},
					&kubermaticv1.UserSSHKey{
						Spec: kubermaticv1.SSHKeySpec{
							Owner:       "John Doe",
							Name:        "ssh-key-name-2",
							Fingerprint: "9876:54321:9876:54321",
							PublicKey:   "ssh-rsa TEST456test",
							Clusters: []string{
								"awscluster-1a2b3c4d5e",
							},
						},
					},
				},
				Version: &apiv1.MasterVersion{},
			},
			ret: nil,
		},
		{
			name:     "get valid machine.yaml for Openstack",
			filename: "../../../../config/kubermatic/static/nodes/machine.yaml",
			fixture:  "machine-openstack",
			data: Data{
				Cluster: &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "openstackcluster-1a2b3c4d5e",
					},
					Address: &kubermaticv1.ClusterAddress{},
					Status:  kubermaticv1.ClusterStatus{},
					Spec: kubermaticv1.ClusterSpec{
						Cloud: &kubermaticv1.CloudSpec{
							DatacenterName: "syseleven-dbl1",
							Openstack: &kubermaticv1.OpenstackCloudSpec{
								Username:             "os-username",
								Password:             "os-password",
								Tenant:               "os-tenant",
								Domain:               "os-domain",
								Network:              "os-network",
								SecurityGroups:       "os-security-groups",
								FloatingIPPool:       "os-floating-ip-pool",
								RouterID:             "os-router-id",
								SubnetID:             "os-subnet-id",
								NetworkCreated:       false,
								SecurityGroupCreated: false,
							},
						},
					},
				},
				Node: &apiv2.Node{
					Metadata: apiv2.ObjectMeta{
						Name: "openstackcluster-1a2b3c4d5e-te5s7",
					},
					Spec: apiv2.NodeSpec{
						Cloud: apiv2.NodeCloudSpec{
							Openstack: &apiv2.OpenstackNodeSpec{
								Flavor: "os-flavor",
								Image:  "os-image",
							},
						},
						OperatingSystem: apiv2.OperatingSystemSpec{
							Ubuntu: &apiv2.UbuntuSpec{
								DistUpgradeOnBoot: false,
							},
						},
						Versions: apiv2.NodeVersionInfo{
							Kubelet: "v1.9.6",
						},
					},
					Status: apiv2.NodeStatus{},
				},
				Datacenter: provider.DatacenterMeta{
					Location: "Frankfurt",
					Seed:     "europe-west3-c",
					Country:  "DE",
					Spec: provider.DatacenterSpec{
						Openstack: &provider.OpenstackSpec{
							AuthURL:          "os-auth-url",
							AvailabilityZone: "os-availability-zone",
							Region:           "os-region",
							IgnoreVolumeAZ:   false,
							DNSServers:       []string{},
						},
					},
					IsSeed: false,
				},
				Keys: []*kubermaticv1.UserSSHKey{
					&kubermaticv1.UserSSHKey{
						Spec: kubermaticv1.SSHKeySpec{
							Owner:       "John Doe",
							Name:        "ssh-key-name",
							Fingerprint: "1234:56789:1234:56789",
							PublicKey:   "ssh-rsa TEST123test",
							Clusters: []string{
								"openstackcluster-1a2b3c4d5e",
							},
						},
					},
				},
				Version: &apiv1.MasterVersion{},
			},
			ret: nil,
		},
		{
			name:     "get valid machine.yaml for Hetzner",
			filename: "../../../../config/kubermatic/static/nodes/machine.yaml",
			fixture:  "machine-hetzner",
			data: Data{
				Cluster: &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "hetznercluster-1a2b3c4d5e",
					},
					Address: &kubermaticv1.ClusterAddress{},
					Status:  kubermaticv1.ClusterStatus{},
					Spec: kubermaticv1.ClusterSpec{
						Cloud: &kubermaticv1.CloudSpec{
							DatacenterName: "hetzner-fsn1",
							Hetzner: &kubermaticv1.HetznerCloudSpec{
								Token: "hetzner-token",
							},
						},
					},
				},
				Node: &apiv2.Node{
					Metadata: apiv2.ObjectMeta{
						Name: "hetznercluster-1a2b3c4d5e-te5s7",
					},
					Spec: apiv2.NodeSpec{
						Cloud: apiv2.NodeCloudSpec{
							Hetzner: &apiv2.HetznerNodeSpec{
								Type: "hetzner-type",
							},
						},
						OperatingSystem: apiv2.OperatingSystemSpec{
							Ubuntu: &apiv2.UbuntuSpec{
								DistUpgradeOnBoot: false,
							},
						},
						Versions: apiv2.NodeVersionInfo{
							Kubelet: "v1.9.6",
						},
					},
					Status: apiv2.NodeStatus{},
				},
				Datacenter: provider.DatacenterMeta{
					Location: "Frankfurt",
					Seed:     "europe-west3-c",
					Country:  "DE",
					Spec: provider.DatacenterSpec{
						Hetzner: &provider.HetznerSpec{
							Datacenter: "hetzner-datacenter",
							Location:   "hetzner-location",
						},
					},
					IsSeed: false,
				},
				Keys: []*kubermaticv1.UserSSHKey{
					&kubermaticv1.UserSSHKey{
						Spec: kubermaticv1.SSHKeySpec{
							Owner:       "John Doe",
							Name:        "ssh-key-name",
							Fingerprint: "1234:56789:1234:56789",
							PublicKey:   "ssh-rsa TEST123test",
							Clusters: []string{
								"hetznercluster-1a2b3c4d5e",
							},
						},
					},
				},
				Version: &apiv1.MasterVersion{},
			},
			ret: nil,
		},
		{
			name:     "get valid machine.yaml for VSphere",
			filename: "../../../../config/kubermatic/static/nodes/machine.yaml",
			fixture:  "machine-vsphere",
			data: Data{
				Cluster: &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vsphere-1a2b3c4d5e",
					},
					Address: &kubermaticv1.ClusterAddress{},
					Status:  kubermaticv1.ClusterStatus{},
					Spec: kubermaticv1.ClusterSpec{
						Cloud: &kubermaticv1.CloudSpec{
							DatacenterName: "vsphere-dummy",
							VSphere: &kubermaticv1.VSphereCloudSpec{
								Username: "vsphere-username",
								Password: "vsphere-password",
							},
						},
					},
				},
				Node: &apiv2.Node{
					Metadata: apiv2.ObjectMeta{
						Name: "vsphere-1a2b3c4d5e-te5s7",
					},
					Spec: apiv2.NodeSpec{
						Cloud: apiv2.NodeCloudSpec{
							VSphere: &apiv2.VSphereNodeSpec{},
						},
						OperatingSystem: apiv2.OperatingSystemSpec{
							Ubuntu: &apiv2.UbuntuSpec{
								DistUpgradeOnBoot: false,
							},
						},
						Versions: apiv2.NodeVersionInfo{
							Kubelet: "v1.9.6",
						},
					},
					Status: apiv2.NodeStatus{},
				},
				Datacenter: provider.DatacenterMeta{
					Location: "Frankfurt",
					Seed:     "europe-west3-c",
					Country:  "DE",
					Spec: provider.DatacenterSpec{
						VSphere: &provider.VSphereSpec{
							Datastore:    "vsphere-datastore",
							Endpoint:     "vsphere-endpoint",
							ResourcePool: "vsphere-resource-pool",
							Datacenter:   "vsphere-datacenter",
						},
					},
					IsSeed: false,
				},
				Keys: []*kubermaticv1.UserSSHKey{
					&kubermaticv1.UserSSHKey{
						Spec: kubermaticv1.SSHKeySpec{
							Owner:       "John Doe",
							Name:        "ssh-key-name",
							Fingerprint: "1234:56789:1234:56789",
							PublicKey:   "ssh-rsa TEST123test",
							Clusters: []string{
								"vsphere-1a2b3c4d5e",
							},
						},
					},
				},
				Version: &apiv1.MasterVersion{},
			},
			ret: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := ioutil.ReadFile(test.filename)
			if err != nil {
				t.Errorf("failed to read file for '%v': %v", test.name, err)
			}

			tpl, err := texttemplate.New("base").Funcs(TxtFuncMap()).Parse(string(b))
			if err != nil {
				t.Errorf("failed to parse template for '%v': %v", test.name, err)
			}

			var buf bytes.Buffer
			err = tpl.Execute(&buf, test.data)
			bytes := buf.Bytes()

			if diff := deep.Equal(err, test.ret); diff != nil {
				t.Errorf("expected to get %v instead got: %v", test.ret, err)
			} else {
				fmt.Printf("\nparsed template for '%v':\n %v", test.name, string(bytes))
			}
		})
	}
}
