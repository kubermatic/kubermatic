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

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	k8csemverv1 "k8c.io/kubermatic/v2/pkg/semver/v1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	Version        k8csemverv1.Semver
	SeedClient     ctrlruntimeclient.Client
	Cluster        *kubermaticv1.Cluster
}

type credentials struct {
	authURL        string
	username       string
	password       string
	project        string
	domain         string
	region         string
	floatingIPPool string
	network        string
}

func (c *ClusterJig) SetUp(ctx context.Context, cloudSpec kubermaticv1.CloudSpec, osCredentials credentials) error {
	c.Log.Debugw("Setting up new cluster", "name", c.Name)

	if err := c.createSecret(ctx, cloudSpec.Openstack.CredentialsReference.Name, osCredentials); err != nil {
		return err
	}
	c.Log.Debugw("secret created", "name", cloudSpec.Openstack.CredentialsReference.Name)

	project, err := c.createProject(ctx)
	if err != nil {
		return err
	}
	c.Log.Debugw("Project created", "name", project.Name)

	if err := c.createCluster(ctx, project, cloudSpec); err != nil {
		return err
	}
	c.Log.Debugw("Cluster created", "name", c.Name)

	return c.waitForClusterControlPlaneReady(ctx, c.Cluster)
}

func (c *ClusterJig) CreateMachineDeployment(ctx context.Context, userClient ctrlruntimeclient.Client, osCredentials credentials) error {
	providerSpec := fmt.Sprintf(`{"cloudProvider": "openstack","cloudProviderSpec": {"identityEndpoint": "%s","username": "%s","password": "%s", "tenantName": "%s", "region": "%s", "domainName": "%s", "floatingIPPool": "%s", "network": "%s", "image": "machine-controller-e2e-ubuntu", "flavor": "m1.small"},"operatingSystem": "ubuntu","operatingSystemSpec":{"distUpgradeOnBoot": false,"disableAutoUpdate": true}}`,
		osCredentials.authURL,
		osCredentials.username,
		osCredentials.password,
		osCredentials.project,
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
	return userClient.Create(ctx, machineDeployment)
}

func (c *ClusterJig) createSecret(ctx context.Context, secretName string, osCredentials credentials) error {
	secretData := map[string][]byte{
		resources.OpenstackUsername:                    []byte(osCredentials.username),
		resources.OpenstackPassword:                    []byte(osCredentials.password),
		resources.OpenstackProject:                     []byte(osCredentials.project),
		resources.OpenstackProjectID:                   []byte(""),
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

	if err := c.SeedClient.Create(ctx, credentialSecret); err != nil {
		return fmt.Errorf("failed to create credential secret: %w", err)
	}

	return nil
}

func (c *ClusterJig) createProject(ctx context.Context) (*kubermaticv1.Project, error) {
	project := &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "proj1234",
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: "test project",
		},
	}

	if err := c.SeedClient.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

func (c *ClusterJig) createCluster(ctx context.Context, project *kubermaticv1.Project, cloudSpec kubermaticv1.CloudSpec) error {
	c.Cluster = &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Name,
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: project.Name,
			},
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
	}
	if err := c.SeedClient.Create(ctx, c.Cluster); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	waiter := reconciling.WaitUntilObjectExistsInCacheConditionFunc(ctx, c.SeedClient, c.Log, ctrlruntimeclient.ObjectKeyFromObject(c.Cluster), c.Cluster)
	if err := wait.Poll(100*time.Millisecond, 5*time.Second, waiter); err != nil {
		return fmt.Errorf("failed waiting for the new cluster to appear in the cache: %w", err)
	}

	if err := kubermaticv1helper.UpdateClusterStatus(ctx, c.SeedClient, c.Cluster, func(c *kubermaticv1.Cluster) {
		c.Status.UserEmail = "e2e@test.com"
	}); err != nil {
		return fmt.Errorf("failed to update cluster status: %w", err)
	}

	return nil
}

// CleanUp deletes the cluster.
func (c *ClusterJig) CleanUp(ctx context.Context) error {
	clusterClientProvider, err := clusterclient.NewExternal(c.SeedClient)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client provider: %w", err)
	}

	userClient, err := clusterClientProvider.GetClient(ctx, c.Cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	// Skip eviction to speed up the clean up process
	nodes := &corev1.NodeList{}
	if err := userClient.List(ctx, nodes); err != nil {
		return fmt.Errorf("failed to list user cluster nodes: %w", err)
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
			return fmt.Errorf("failed to annotate node %s: %w", node.Name, retErr)
		}
	}

	// Delete MachineDeployment and Cluster
	deleteTimeout := 15 * time.Minute
	return wait.PollImmediate(5*time.Second, deleteTimeout, func() (bool, error) {
		mdKey := ctrlruntimeclient.ObjectKey{Name: machineDeploymentName, Namespace: machineDeploymentNamespace}

		md := &clusterv1alpha1.MachineDeployment{}
		err := userClient.Get(ctx, mdKey, md)
		if err == nil {
			if md.DeletionTimestamp != nil {
				return false, nil
			}
			err := userClient.Delete(ctx, md)
			if err != nil {
				return false, fmt.Errorf("failed to delete user cluster machinedeployment: %w", err)
			}
			return false, nil
		} else if err != nil && !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("failed to get user cluster machinedeployment: %w", err)
		}

		clusters := &kubermaticv1.ClusterList{}
		err = c.SeedClient.List(ctx, clusters)
		if err != nil {
			return false, fmt.Errorf("failed to list user clusters: %w", err)
		}

		if len(clusters.Items) == 0 {
			return true, nil
		}
		if len(clusters.Items) > 1 {
			return false, fmt.Errorf("there is more than one user cluster, expected one or zero cluster: %w", err)
		}

		if clusters.Items[0].DeletionTimestamp == nil {
			err := c.SeedClient.Delete(ctx, c.Cluster)
			if err != nil {
				return false, fmt.Errorf("failed to delete user cluster: %w", err)
			}
		}

		return false, nil
	})
}

func (c *ClusterJig) waitForClusterControlPlaneReady(ctx context.Context, cl *kubermaticv1.Cluster) error {
	c.Log.Debug("Waiting for control plane to become ready...")

	return wait.PollImmediate(clusterReadinessCheckPeriod, clusterReadinessTimeout, func() (bool, error) {
		if err := c.SeedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: c.Name, Namespace: cl.Namespace}, cl); err != nil {
			return false, err
		}

		return cl.Status.Conditions[kubermaticv1.ClusterConditionSeedResourcesUpToDate].Status == corev1.ConditionTrue, nil
	})
}
