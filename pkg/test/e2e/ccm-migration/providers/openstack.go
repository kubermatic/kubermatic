package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	types2 "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/ccm-migration/utils"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type OpenstackClusterJig struct {
	CommonClusterJig

	Log *zap.SugaredLogger

	OpenstackCredentials OpenstackCredentialsType
}

func (c *OpenstackClusterJig) Setup() error {
	c.Log.Debugw("Setting up new cluster", "name", c.Name)

	if err := c.generateAndCreateSecret(c.OpenstackCredentials.GenerateSecretData()); err != nil {
		return errors.Wrap(err, "failed to create credential secret")
	}
	c.Log.Debugw("secret created", "name", fmt.Sprintf("credential-openstack-%s", c.Name))

	if err := c.generateAndCreateCluster(kubermaticv1.CloudSpec{
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			CredentialsReference: &types2.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name:      fmt.Sprintf("credential-openstack-%s", c.Name),
					Namespace: resources.KubermaticNamespace,
				},
			},
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create user cluster")
	}
	c.Log.Debugw("Cluster created", "name", c.Name)

	return c.waitForClusterControlPlaneReady()
}

func (c *OpenstackClusterJig) CreateMachineDeployment(userClient ctrlruntimeclient.Client) error {
	if err := c.generateAndCCreateMachineDeployment(userClient, c.OpenstackCredentials.GenerateProviderSpec()); err != nil {
		return errors.Wrap(err, "failed to create machine deployment")
	}
	return nil
}

func (c *OpenstackClusterJig) CheckComponents() (bool, error) {
	return true, nil
}

// CleanUp deletes the cluster.
func (c *OpenstackClusterJig) CleanUp(userClient ctrlruntimeclient.Client) error {
	ctx := context.TODO()

	cluster := &kubermaticv1.Cluster{}
	if err := c.SeedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: c.Name, Namespace: ""}, cluster); err != nil {
		return errors.Wrap(err, "failed to get user cluster")
	}

	// Skip eviction to speed up the clean up process
	nodes := &corev1.NodeList{}
	if err := userClient.List(ctx, nodes); err != nil {
		return errors.Wrap(err, "failed to list user cluster nodes")
	}

	for _, node := range nodes.Items {
		nodeKey := ctrlruntimeclient.ObjectKey{Name: node.Name}

		retErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			n := corev1.Node{}
			if err := userClient.Get(ctx, nodeKey, &n); err != nil {
				return err
			}

			if n.Annotations == nil {
				n.Annotations = map[string]string{}
			}
			n.Annotations["kubermatic.io/skip-eviction"] = "true"
			return userClient.Update(ctx, &n)
		})
		if retErr != nil {
			return errors.Wrapf(retErr, "failed to annotate node %s", node.Name)
		}
	}

	// Delete MachineDeployment and Cluster
	deleteTimeout := 15 * time.Minute
	return wait.PollImmediate(5*time.Second, deleteTimeout, func() (bool, error) {
		mdKey := ctrlruntimeclient.ObjectKey{Name: utils.MachineDeploymentName, Namespace: utils.MachineDeploymentNamespace}

		md := &clusterv1alpha1.MachineDeployment{}
		err := userClient.Get(ctx, mdKey, md)
		if err == nil {
			if md.DeletionTimestamp != nil {
				return false, nil
			}
			err := userClient.Delete(ctx, md)
			if err != nil {
				return false, errors.Wrap(err, "failed to delete user cluster machinedeployment")
			}
			return false, nil
		} else if err != nil && !k8serrors.IsNotFound(err) {
			return false, errors.Wrap(err, "failed to get user cluster machinedeployment")
		}

		clusters := &kubermaticv1.ClusterList{}
		err = c.SeedClient.List(ctx, clusters)
		if err != nil {
			return false, errors.Wrap(err, "failed to list user clusters")
		}

		if len(clusters.Items) == 0 {
			return true, nil
		}
		if len(clusters.Items) > 1 {
			return false, errors.Wrap(err, "there is more than one user cluster, expected one or zero cluster")
		}

		if clusters.Items[0].DeletionTimestamp == nil {
			cluster := &kubermaticv1.Cluster{}
			if err := c.SeedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: c.Name, Namespace: ""}, cluster); err != nil {
				return false, errors.Wrap(err, "failed to get user cluster")
			}
			err := c.SeedClient.Delete(ctx, cluster)
			if err != nil {
				return false, errors.Wrap(err, "failed to delete user cluster")
			}
		}

		return false, nil
	})
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
