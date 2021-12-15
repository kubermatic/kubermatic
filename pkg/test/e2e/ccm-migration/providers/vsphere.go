package providers

import (
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	types2 "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	vsphereSecretPrefixName = "credentials-vsphere"
)

type VsphereClusterJig struct {
	CommonClusterJig

	log *zap.SugaredLogger

	Credentials VsphereCredentialsType
}

func NewClusterJigVsphere(seedClient ctrlruntimeclient.Client, clusterName string, version semver.Semver, credentials VsphereCredentialsType) *VsphereClusterJig {
	return &VsphereClusterJig{
		CommonClusterJig: CommonClusterJig{
			name:           clusterName,
			DatacenterName: credentials.Datacenter,
			Version:        version,
			SeedClient:     seedClient,
		},
		log:         e2eutils.DefaultLogger,
		Credentials: credentials,
	}
}

func (c *VsphereClusterJig) Setup() error {
	c.log.Debugw("Setting up new cluster", "name", c.Name)

	if err := c.generateAndCreateSecret(vsphereSecretPrefixName, c.Credentials.GenerateSecretData()); err != nil {
		return errors.Wrap(err, "failed to create credential secret")
	}
	c.log.Debugw("secret created", "name", fmt.Sprintf("%s-%s", vsphereSecretPrefixName, c.name))

	if err := c.generateAndCreateCluster(kubermaticv1.CloudSpec{
		DatacenterName: c.DatacenterName,
		VSphere: &kubermaticv1.VSphereCloudSpec{
			CredentialsReference: &types2.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s-%s", vsphereSecretPrefixName, c.name),
					Namespace: resources.KubermaticNamespace,
				},
			},
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create user cluster")
	}
	c.log.Debugw("Cluster created", "name", c.Name)

	return c.waitForClusterControlPlaneReady()
}

func (c *VsphereClusterJig) CreateMachineDeployment(userClient ctrlruntimeclient.Client) error {
	if err := c.generateAndCCreateMachineDeployment(userClient, c.Credentials.GenerateProviderSpec()); err != nil {
		return errors.Wrap(err, "failed to create machine deployment")
	}
	return nil
}

func (c *VsphereClusterJig) Cleanup(userClient ctrlruntimeclient.Client) error {
	return c.cleanUp(userClient)
}

func (c *VsphereClusterJig) CheckComponents() (bool, error) {
	return true, nil
}

func (c *VsphereClusterJig) Name() string {
	return c.name
}

func (c *VsphereClusterJig) Seed() ctrlruntimeclient.Client {
	return c.SeedClient
}

func (c *VsphereClusterJig) Log() *zap.SugaredLogger {
	return c.log
}

type VsphereCredentialsType struct {
	AuthURL    string
	Username   string
	Password   string
	Datacenter string
}

func (vc *VsphereCredentialsType) GenerateSecretData() map[string][]byte {
	return map[string][]byte{
		resources.VsphereUsername:                    []byte(vc.Username),
		resources.VspherePassword:                    []byte(vc.Password),
		resources.VsphereInfraManagementUserUsername: []byte(""),
		resources.VsphereInfraManagementUserPassword: []byte(""),
	}
}

func (vc *VsphereCredentialsType) GenerateProviderSpec() []byte {
	cloudProviderSpec := fmt.Sprintf(`{"allowInsecure":false,"cluster":"%s","cpus":2,"datacenter":"dc-1","datastore":"exsi-nas","datastoreCluster":"","diskSizeGB":10,"memoryMB":4096,"folder":"/dc-1/vm/kubermatic/6cdxcas3ff","templateVMName":"machine-controller-e2e-ubuntu"}`, "cl-1")
	return []byte(fmt.Sprintf(`{"cloudProvider":"vsphere","cloudProviderSpec":%s,"operatingSystem":"ubuntu","operatingSystemSpec":{"distUpgradeOnBoot":false},"sshPublicKeys":["ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDfWMWB244VDl+EL8f5OG5zYbu+eW1PeOAKrfd6c8GP+xQfO/cgvyF1u495eS2Ef+aLIsD09h/dwfCefW9WtQ12kgNpOneqEBhkhwW/1DIcB6Or63SxEapt9mqullSc6WtcwEoRaT+Ro0o3TuZ6xW7RBLFMcU3Zp2PM2WvN/B67X8agMxqYVFw/T94tpYGKSIOV03a/PTWN9Er2zCEcsVu4XEShtUHO1wOrGVOOsfk1hd27o1odRPpBNL+6DbXQBhQRrS45NTeIIsIECccSpNBX3WdAh+sasFgfWhap1ZNU/Je1lygM81ChwUvldydgE8ccL4oeLX+S8om2VAQeUkllaKuO22SJMaGzIFm5ZQY8yjzOGkAABuDHmB31knDGTCAQ0l+XTjN+ucbhKdQi645Ar/leLV93TXyKxKCDBxLp22gDWP2YIsS0mw6eqiiuEQu4a0QjFegfRTMPM3K1g7i7evYUAlpVR5Bq4gK52t6z+Ev1Z2frYSA0flTBCHX7v4k= mlavacca@Thinkpad-mattia"]}`,
		cloudProviderSpec))
}
