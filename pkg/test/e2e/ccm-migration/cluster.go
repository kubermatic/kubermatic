/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ccmmigration

import (
	"context"
	"fmt"
	"time"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterReadinessCheckPeriod = 10 * time.Second
	clusterReadinessTimeout     = 10 * time.Minute

	machineDeploymentName      = "ccm-migration-e2e"
	machineDeploymentNamespace = "kube-system"
)

// ClusterJig helps setting up a user cluster for testing.
type ClusterJig struct {
	Log            *zap.SugaredLogger
	Name           string
	DatacenterName string
	Version        semver.Semver
	SeedClient     ctrlruntimeclient.Client
	Cluster        *kubermaticv1.Cluster
}

type credentials struct {
	authURL        string
	username       string
	password       string
	tenant         string
	domain         string
	region         string
	floatingIPPool string
	network        string
}

func (c *ClusterJig) SetUp(cloudSpec kubermaticv1.CloudSpec, osCredentials credentials) error {
	c.Log.Debugw("Setting up new cluster", "name", c.Name)

	if err := c.createSecret(cloudSpec.Openstack.CredentialsReference.Name, osCredentials); err != nil {
		return err
	}
	c.Log.Debugw("secret created", "name", cloudSpec.Openstack.CredentialsReference.Name)

	if err := c.createCluster(cloudSpec); err != nil {
		return nil
	}
	c.Log.Debugw("Cluster created", "name", c.Name)

	return c.waitForClusterControlPlaneReady(c.Cluster)
}

func (c *ClusterJig) CreateMachineDeployment(userClient ctrlruntimeclient.Client, osCredentials credentials) error {
	providerSpec := fmt.Sprintf(`{"cloudProvider": "openstack","cloudProviderSpec": {"identityEndpoint": "%s","username": "%s","password": "%s", "tenantName": "%s", "region": "%s", "domainName": "%s", "floatingIPPool": "%s", "network": "%s", "image": "machine-controller-e2e-ubuntu", "flavor": "m1.small"},"operatingSystem": "ubuntu","operatingSystemSpec":{"distUpgradeOnBoot": false,"disableAutoUpdate": true}}`,
		osCredentials.authURL,
		osCredentials.username,
		osCredentials.password,
		osCredentials.tenant,
		osCredentials.region,
		osCredentials.domain,
		osCredentials.floatingIPPool,
		osCredentials.network)

	machineDeployment := &clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineDeploymentName,
			Namespace: machineDeploymentNamespace,
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": machineDeploymentName,
				},
			},
			Template: clusterv1alpha1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": machineDeploymentName,
					},
				},
				Spec: clusterv1alpha1.MachineSpec{
					ProviderSpec: clusterv1alpha1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(providerSpec),
						},
					},
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: "1.20.0",
					},
				},
			},
		},
	}
	return userClient.Create(context.TODO(), machineDeployment)
}

func (c *ClusterJig) createSecret(secretName string, osCredentials credentials) error {
	secretData := map[string][]byte{
		resources.OpenstackUsername:                    []byte(osCredentials.username),
		resources.OpenstackPassword:                    []byte(osCredentials.password),
		resources.OpenstackTenant:                      []byte(osCredentials.tenant),
		resources.OpenstackTenantID:                    []byte(""),
		resources.OpenstackDomain:                      []byte(osCredentials.domain),
		resources.OpenstackApplicationCredentialID:     []byte(""),
		resources.OpenstackApplicationCredentialSecret: []byte(""),
		resources.OpenstackToken:                       []byte(""),
	}
	credentialSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: resources.KubermaticNamespace,
			Labels: map[string]string{
				"name": secretName,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	}

	if err := c.SeedClient.Create(context.TODO(), credentialSecret); err != nil {
		return errors.Wrap(err, "failed to create credential secret")
	}

	return nil
}

func (c *ClusterJig) createCluster(cloudSpec kubermaticv1.CloudSpec) error {
	c.Cluster = &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: cloudSpec,
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"10.240.16.0/20"},
				},
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"172.25.0.0/16"},
				},
				ProxyMode: "ipvs",
			},
			ComponentsOverride: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{
					EndpointReconcilingDisabled: pointer.BoolPtr(true),
					DeploymentSettings: kubermaticv1.DeploymentSettings{
						Replicas: pointer.Int32Ptr(1),
					},
				},
				ControllerManager: kubermaticv1.ControllerSettings{
					DeploymentSettings: kubermaticv1.DeploymentSettings{
						Replicas: pointer.Int32Ptr(1),
					},
				},
				Etcd: kubermaticv1.EtcdStatefulSetSettings{
					ClusterSize: pointer.Int32Ptr(1),
				},
				Scheduler: kubermaticv1.ControllerSettings{
					DeploymentSettings: kubermaticv1.DeploymentSettings{
						Replicas: pointer.Int32Ptr(1),
					},
				},
			},
			EnableUserSSHKeyAgent: pointer.BoolPtr(false),
			ExposeStrategy:        kubermaticv1.ExposeStrategyTunneling,
			HumanReadableName:     "test",
			Version:               c.Version,
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: fmt.Sprintf("cluster-%s", c.Name),
			UserEmail:     "e2e@test.com",
		},
	}
	if err := c.SeedClient.Create(context.TODO(), c.Cluster); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}

// CleanUp deletes the cluster.
func (c *ClusterJig) CleanUp() error {
	clusterClientProvider, err := clusterclient.NewExternal(c.SeedClient)
	if err != nil {
		return errors.Wrap(err, "failed to get user cluster client provider")
	}

	userClient, err := clusterClientProvider.GetClient(context.TODO(), c.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get user cluster client")
	}

	// Skip eviction to speed up the clean up process
	nodes := &corev1.NodeList{}
	if err := userClient.List(context.TODO(), nodes); err != nil {
		return errors.Wrap(err, "failed to list user cluster nodes")
	}

	for _, node := range nodes.Items {
		nodeKey := ctrlruntimeclient.ObjectKey{Name: node.Name}

		retErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			n := corev1.Node{}
			if err := userClient.Get(context.TODO(), nodeKey, &n); err != nil {
				return err
			}

			if n.Annotations == nil {
				n.Annotations = map[string]string{}
			}
			n.Annotations["kubermatic.io/skip-eviction"] = "true"
			return userClient.Update(context.TODO(), &n)
		})
		if retErr != nil {
			return errors.Wrapf(retErr, "failed to annotate node %s", node.Name)
		}
	}

	// Delete MachineDeployment and Cluster
	deleteTimeout := 15 * time.Minute
	return wait.PollImmediate(5*time.Second, deleteTimeout, func() (bool, error) {
		mdKey := ctrlruntimeclient.ObjectKey{Name: machineDeploymentName, Namespace: machineDeploymentNamespace}

		md := &clusterv1alpha1.MachineDeployment{}
		err := userClient.Get(context.TODO(), mdKey, md)
		if err == nil {
			if md.DeletionTimestamp != nil {
				return false, nil
			}
			err := userClient.Delete(context.TODO(), md)
			if err != nil {
				return false, errors.Wrap(err, "failed to delete user cluster machinedeployment")
			}
			return false, nil
		} else if err != nil && !k8serrors.IsNotFound(err) {
			return false, errors.Wrap(err, "failed to get user cluster machinedeployment")
		}

		clusters := &kubermaticv1.ClusterList{}
		err = c.SeedClient.List(context.TODO(), clusters)
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
			err := c.SeedClient.Delete(context.TODO(), c.Cluster)
			if err != nil {
				return false, errors.Wrap(err, "failed to delete user cluster")
			}
		}

		return false, nil
	})
}

func (c *ClusterJig) waitForClusterControlPlaneReady(cl *kubermaticv1.Cluster) error {
	return wait.PollImmediate(clusterReadinessCheckPeriod, clusterReadinessTimeout, func() (bool, error) {
		if err := c.SeedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: c.Name, Namespace: cl.Namespace}, cl); err != nil {
			return false, err
		}
		_, cond := kubermaticv1helper.GetClusterCondition(cl, kubermaticv1.ClusterConditionSeedResourcesUpToDate)
		if cond != nil && cond.Status == corev1.ConditionTrue {
			return true, nil
		}
		return false, nil
	})
}
