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
	"k8c.io/kubermatic/v2/pkg/resources"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterReadinessCheckPeriod = 10 * time.Second
	clusterReadinessTimeout     = 10 * time.Minute
)

// ClusterJig helps setting up a user cluster for testing.
type ClusterJig struct {
	Log            *zap.SugaredLogger
	Name           string
	DatacenterName string
	Version        semver.Semver
	SeedClient     ctrlruntimeclient.Client
	userClient     ctrlruntimeclient.Client

	Cluster *kubermaticv1.Cluster
}

type credentials struct {
	username string
	password string
	tenant   string
	domain   string
}

func (c *ClusterJig) SetUp(cloudSpec kubermaticv1.CloudSpec, osCredentials credentials) error {
	c.Log.Debugw("Creating cluster", "name", c.Name)

	if err := c.createSecret(cloudSpec.Openstack.CredentialsReference.Name, osCredentials); err != nil {
		return err
	}
	if err := c.createCluster(cloudSpec); err != nil {
		return nil
	}
	return c.waitForClusterControlPlaneReady(c.Cluster)
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
					ClusterSize: 1,
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
	return c.SeedClient.Delete(context.TODO(), c.Cluster)
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
