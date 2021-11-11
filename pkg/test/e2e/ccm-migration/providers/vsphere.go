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

func (osc *VsphereCredentialsType) GenerateSecretData() map[string][]byte {
	return map[string][]byte{
		resources.VsphereUsername:                    []byte(osc.Username),
		resources.VspherePassword:                    []byte(osc.Password),
		resources.VsphereInfraManagementUserUsername: []byte(""),
		resources.VsphereInfraManagementUserPassword: []byte(""),
	}
}

func (osc *VsphereCredentialsType) GenerateProviderSpec() []byte {
	return []byte(fmt.Sprintf(`{"cloudProvider": "vsphere","cloudProviderSpec": {"identityEndpoint": "%s","Username": "%s","Password": "%s", "image": "machine-controller-e2e-ubuntu", "flavor": "m1.small"},"operatingSystem": "ubuntu","operatingSystemSpec":{"distUpgradeOnBoot": false,"disableAutoUpdate": true}}`,
		osc.AuthURL,
		osc.Username,
		osc.Password))
}
