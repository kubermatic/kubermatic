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
	osSecretPrefixName = "credentials-openstack"
)

type OpenstackClusterJig struct {
	CommonClusterJig

	log *zap.SugaredLogger

	Credentials OpenstackCredentialsType
}

func NewClusterJigOpenstack(seedClient ctrlruntimeclient.Client, clusterName string, version semver.Semver, credentials OpenstackCredentialsType) *OpenstackClusterJig {
	return &OpenstackClusterJig{
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

func (c *OpenstackClusterJig) Setup() error {
	c.log.Debugw("Setting up new cluster", "name", c.Name)

	if err := c.generateAndCreateSecret(osSecretPrefixName, c.Credentials.GenerateSecretData()); err != nil {
		return errors.Wrap(err, "failed to create credential secret")
	}
	c.log.Debugw("secret created", "name", fmt.Sprintf("%s-%s", osSecretPrefixName, c.name))

	if err := c.generateAndCreateCluster(kubermaticv1.CloudSpec{
		DatacenterName: c.DatacenterName,
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			CredentialsReference: &types2.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s-%s", osSecretPrefixName, c.name),
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

func (c *OpenstackClusterJig) CreateMachineDeployment(userClient ctrlruntimeclient.Client) error {
	if err := c.generateAndCCreateMachineDeployment(userClient, c.Credentials.GenerateProviderSpec()); err != nil {
		return errors.Wrap(err, "failed to create machine deployment")
	}
	return nil
}

func (c *OpenstackClusterJig) Cleanup(userClient ctrlruntimeclient.Client) error {
	return c.cleanUp(userClient)
}

func (c *OpenstackClusterJig) CheckComponents() (bool, error) {
	return true, nil
}

func (c *OpenstackClusterJig) Name() string {
	return c.name
}

func (c *OpenstackClusterJig) Seed() ctrlruntimeclient.Client {
	return c.SeedClient
}

func (c *OpenstackClusterJig) Log() *zap.SugaredLogger {
	return c.log
}

type OpenstackCredentialsType struct {
	AuthURL        string
	Username       string
	Password       string
	Tenant         string
	Domain         string
	Region         string
	FloatingIPPool string
	Network        string
	Datacenter     string
}

func (osc *OpenstackCredentialsType) GenerateSecretData() map[string][]byte {
	return map[string][]byte{
		resources.OpenstackUsername:                    []byte(osc.Username),
		resources.OpenstackPassword:                    []byte(osc.Password),
		resources.OpenstackTenant:                      []byte(osc.Tenant),
		resources.OpenstackTenantID:                    []byte(""),
		resources.OpenstackDomain:                      []byte(osc.Domain),
		resources.OpenstackApplicationCredentialID:     []byte(""),
		resources.OpenstackApplicationCredentialSecret: []byte(""),
		resources.OpenstackToken:                       []byte(""),
	}
}

func (osc *OpenstackCredentialsType) GenerateProviderSpec() []byte {
	return []byte(fmt.Sprintf(`{"cloudProvider": "openstack","cloudProviderSpec": {"identityEndpoint": "%s","Username": "%s","Password": "%s", "tenantName": "%s", "Region": "%s", "domainName": "%s", "FloatingIPPool": "%s", "Network": "%s", "image": "machine-controller-e2e-ubuntu", "flavor": "m1.small"},"operatingSystem": "ubuntu","operatingSystemSpec":{"distUpgradeOnBoot": false,"disableAutoUpdate": true}}`,
		osc.AuthURL,
		osc.Username,
		osc.Password,
		osc.Tenant,
		osc.Region,
		osc.Domain,
		osc.FloatingIPPool,
		osc.Network))
}
