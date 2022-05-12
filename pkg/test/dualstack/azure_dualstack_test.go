//go:build dualstack

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package dualstack

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/util"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
)

var operatingSystems = map[string]models.OperatingSystemSpec{
	"centos":  centos(),
	"flatcar": flatcar(),
	"rhel":    rhel(),
	"sles":    sles(),
	"ubuntu":  ubuntu(),
}

var cloudProviders = map[string]cloudProvider{
	"azure": azure{},
}

var cnis = map[string]models.CNIPluginSettings{
	"cilium": cilium(),
	"canal":  canal(),
}

func TestCloudClusterIPFamily(t *testing.T) {
	// export KUBERMATIC_API_ENDPOINT=https://dev.kubermatic.io
	// export KKP_API_TOKEN=<steal token>
	token := os.Getenv("KKP_API_TOKEN")

	_ = cnis

	if token == "" {
		var err error
		token, err = utils.RetrieveMasterToken(context.Background())
		if err != nil {
			t.Fatalf("failed to retrieve master token: %v", err)
		}
	} else {
		t.Logf("token found in env")
	}

	apicli := utils.NewTestClient(token, t)

	tests := []struct {
		cloudName           string
		osName              string
		cni                 string
		ipFamily            util.IPFamily
		skipNodes           bool
		skipHostNetworkPods bool
		disabledReason      string
	}{
		{
			cloudName:           "azure",
			osName:              "centos",
			cni:                 "cilium",
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
			disabledReason:      "fails due to https://github.com/kubermatic/kubermatic/issues/9222",
		},
		{
			cloudName:           "azure",
			osName:              "centos",
			cni:                 "canal",
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudName:           "azure",
			osName:              "flatcar",
			cni:                 "cilium",
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
			disabledReason:      "fails due to https://github.com/kubermatic/kubermatic/issues/9798",
		},
		{
			cloudName:           "azure",
			osName:              "rhel",
			cni:                 "cilium",
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
			disabledReason:      "cilium-agent crashing",
		},
		{
			cloudName:           "azure",
			osName:              "sles",
			cni:                 "cilium",
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
			disabledReason:      "not supported",
		},
		{
			cloudName:           "azure",
			osName:              "ubuntu",
			cni:                 "cilium",
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
	}

	var mu sync.Mutex

	for _, test := range tests {
		name := fmt.Sprintf("c-%s-%s-%s", test.cloudName, test.osName, test.ipFamily)

		if test.disabledReason != "" {
			t.Logf("test %q disabled because: %s", name, test.disabledReason)
			continue
		}

		cloud := cloudProviders[test.cloudName]
		cloudSpec := cloud.CloudSpec()
		nodeSpec := cloud.NodeSpec()
		osSpec := operatingSystems[test.osName]
		clusterSpec := defaultClusterRequest()

		t.Run(name, func(t *testing.T) {
			clusterSpec := clusterSpec.WithName(name).
				WithCloud(cloudSpec).
				WithNode(nodeSpec).
				WithOS(osSpec)
			spec := models.CreateClusterSpec(clusterSpec)

			mu.Lock()
			name := fmt.Sprintf("%s-%s", name, rand.String(4))
			config, _, cleanup, err := createUsercluster(t, apicli, name, spec)
			mu.Unlock()
			if err != nil {
				respErr := new(project.CreateClusterV2Default)
				if errors.As(err, &respErr) {
					errData, err := respErr.GetPayload().MarshalBinary()
					if err != nil {
						t.Fatalf("failed to marshal error response")
					}
					t.Fatalf(string(errData))
				}
				t.Fatalf("failed to create cluster: %v", err)
			}
			defer func() {
				mu.Lock()
				cleanup()
				mu.Unlock()
			}()

			userclusterClient, err := kubernetes.NewForConfig(config)
			if err != nil {
				t.Fatalf("failed to create usercluster client: %s", err)
			}

			t.Logf("waiting for nodes to come up")
			_, err = checkNodeReadiness(t, userclusterClient)
			if err != nil {
				t.Fatalf("nodes never became ready: %v", err)
			}

			t.Logf("sleeping for 2m...")
			time.Sleep(time.Minute * 2)

			err = waitForPods(t, userclusterClient, kubeSystem, "app", []string{
				"coredns", "konnectivity-agent", "kube-proxy", "metrics-server",
				"node-local-dns", "user-ssh-keys-agent",
			})

			if err != nil {
				t.Fatalf("pods never became ready: %v", err)
			}

			testUserCluster(t, userclusterClient, test.ipFamily, test.skipNodes, test.skipHostNetworkPods)

		})
	}
}

func waitForPods(t *testing.T, client *kubernetes.Clientset, namespace string, key string, names []string) error {
	t.Log("checking pod readiness...", namespace, key, names)

	return wait.Poll(30*time.Second, 15*time.Minute, func() (bool, error) {
		r, err := labels.NewRequirement(key, selection.In, names)
		if err != nil {
			t.Logf("failed to build requirement: %v", err)
			return false, nil
		}
		l := labels.NewSelector().Add(*r)
		pods, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: l.String(),
		})
		if err != nil {
			t.Logf("failed to get pod list: %s", err)
			return false, nil
		}

		if len(pods.Items) == 0 {
			t.Logf("no pods found")
			return false, nil
		}

		if !allPodsHealthy(t, pods) {
			t.Logf("not all pods healthy yet...")
			return false, nil
		}

		t.Logf("all pods healthy")

		return true, nil
	})
}

func allPodsHealthy(t *testing.T, pods *corev1.PodList) bool {
	allHealthy := true
	for _, pod := range pods.Items {
		podHealthy := true
		if pod.Status.Phase != corev1.PodRunning {
			podHealthy = false
			t.Log("not running", pod.Name, pod.Status.Phase)
		}
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodReady {
				if c.Status != corev1.ConditionTrue {
					podHealthy = false
					t.Log("not ready", pod.Name, c.Type, c.Status)
				}
			} else if c.Type == corev1.ContainersReady {
				if c.Status != corev1.ConditionTrue {
					podHealthy = false
					t.Log("not container ready", pod.Name, c.Type, c.Status)
				}
			}
		}

		if !podHealthy {
			t.Logf("%q not healthy", pod.Name)
		}

		allHealthy = allHealthy && podHealthy
	}

	return allHealthy
}

func checkNodeReadiness(t *testing.T, userClient *kubernetes.Clientset) (string, error) {
	expectedNodes := 1
	var nodeIP string

	err := wait.Poll(30*time.Second, 15*time.Minute, func() (bool, error) {
		nodes, err := userClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("failed to get nodes list: %s", err)
			return false, nil
		}
		if len(nodes.Items) != expectedNodes {
			t.Logf("node count: %d, expected: %d", len(nodes.Items), expectedNodes)
			return false, nil
		}

		readyNodeCount := 0

		for _, node := range nodes.Items {
			for _, c := range node.Status.Conditions {
				if c.Type == corev1.NodeReady {
					readyNodeCount++
				}
			}
		}

		if readyNodeCount != expectedNodes {
			t.Logf("%d out of %d nodes are ready", readyNodeCount, expectedNodes)
			return false, nil
		}

		for _, addr := range nodes.Items[0].Status.Addresses {
			if addr.Type == corev1.NodeExternalIP {
				nodeIP = addr.Address
				break
			}
		}
		return true, nil
	})

	return nodeIP, err
}

type cloudProvider interface {
	NodeSpec() models.NodeCloudSpec
	CloudSpec() models.CloudSpec
}

type azure struct{}

var _ cloudProvider = azure{}

func (a azure) NodeSpec() models.NodeCloudSpec {
	return models.NodeCloudSpec{
		Azure: &models.AzureNodeSpec{
			AssignAvailabilitySet: true,
			AssignPublicIP:        true,
			DataDiskSize:          int32(30),
			OSDiskSize:            70,
			Size:                  pointer.String("Standard_B2s"),
		},
	}
}

func (a azure) CloudSpec() models.CloudSpec {
	return models.CloudSpec{
		DatacenterName: "azure-westeurope",
		Azure: &models.AzureCloudSpec{
			ClientID:        os.Getenv("AZURE_CLIENT_ID"),
			ClientSecret:    os.Getenv("AZURE_CLIENT_SECRET"),
			SubscriptionID:  os.Getenv("AZURE_SUBSCRIPTION_ID"),
			TenantID:        os.Getenv("AZURE_TENANT_ID"),
			LoadBalancerSKU: "standard",
		},
	}
}

func defaultClusterRequest() createClusterRequest {
	clusterSpec := models.CreateClusterSpec{}
	clusterSpec.Cluster = &models.Cluster{
		Type: "kubernetes",
		Name: "test-default-azure-cluster",
		Spec: &models.ClusterSpec{
			Cloud: &models.CloudSpec{
				DatacenterName: "azure-westeurope",
				Azure: &models.AzureCloudSpec{
					ClientID:        os.Getenv("AZURE_CLIENT_ID"),
					ClientSecret:    os.Getenv("AZURE_CLIENT_SECRET"),
					SubscriptionID:  os.Getenv("AZURE_SUBSCRIPTION_ID"),
					TenantID:        os.Getenv("AZURE_TENANT_ID"),
					LoadBalancerSKU: "standard",
				},
			},
			CniPlugin: &models.CNIPluginSettings{
				Version: "v1.11",
				Type:    "cilium",
			},
			ClusterNetwork: &models.ClusterNetworkingConfig{
				NodeCIDRMaskSizeIPV4: 24,
				// {"cidrBlocks":["172.25.0.0/16","fd00::/104"]},
				// "services":{"cidrBlocks":["10.240.16.0/20","fd03::/120"]}
				NodeCIDRMaskSizeIPV6: 105,
				ProxyMode:            "ebpf",
				IPFamily:             "IPv4+IPv6",
				Pods: &models.NetworkRanges{
					CIDRBlocks: []string{"172.25.0.0/16", "fd00::/104"},
				},
				Services: &models.NetworkRanges{
					CIDRBlocks: []string{"10.240.16.0/20", "fd03::/120"},
				},
				KonnectivityEnabled: true,
			},
			Version: "1.22.7",
		},
	}

	clusterSpec.NodeDeployment = &models.NodeDeployment{
		Spec: &models.NodeDeploymentSpec{
			Replicas: pointer.Int32(1),
			Template: &models.NodeSpec{
				SSHUserName: "root",
				Cloud: &models.NodeCloudSpec{
					Azure: &models.AzureNodeSpec{
						AssignAvailabilitySet: true,
						AssignPublicIP:        true,
						DataDiskSize:          int32(30),
						OSDiskSize:            70,
						Size:                  pointer.String("Standard_B2s"),
					},
				},
				OperatingSystem: &models.OperatingSystemSpec{
					Ubuntu: &models.UbuntuSpec{
						DistUpgradeOnBoot: false,
					},
				},
			},
		},
		Status: nil,
	}

	return createClusterRequest(clusterSpec)
}

func (c createClusterRequest) WithCNI(cni models.CNIPluginSettings) createClusterRequest {
	c.Cluster.Spec.CniPlugin = &cni
	return c
}

func (c createClusterRequest) WithOS(os models.OperatingSystemSpec) createClusterRequest {
	c.NodeDeployment.Spec.Template.OperatingSystem = &os
	return c
}

func (c createClusterRequest) WithNode(node models.NodeCloudSpec) createClusterRequest {
	c.NodeDeployment.Spec.Template.Cloud = &node
	return c
}

func (c createClusterRequest) WithCloud(cloud models.CloudSpec) createClusterRequest {
	c.Cluster.Spec.Cloud = &cloud
	return c
}

func (c createClusterRequest) WithName(name string) createClusterRequest {
	c.Cluster.Name = name
	return c
}

func (c createClusterRequest) WithNetworkConfig(netConfig models.ClusterNetworkingConfig) createClusterRequest {
	c.Cluster.Spec.ClusterNetwork = &netConfig
	return c
}

type createClusterRequest models.CreateClusterSpec

func ubuntu() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Ubuntu: &models.UbuntuSpec{},
	}
}

func rhel() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Rhel: &models.RHELSpec{},
	}
}

func sles() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Sles: &models.SLESSpec{},
	}
}

func centos() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Centos: &models.CentOSSpec{},
	}
}

func flatcar() models.OperatingSystemSpec {
	return models.OperatingSystemSpec{
		Flatcar: &models.FlatcarSpec{},
	}
}

func cilium() models.CNIPluginSettings {
	return models.CNIPluginSettings{
		Version: "v1.11",
		Type:    "cilium",
	}
}

func canal() models.CNIPluginSettings {
	return models.CNIPluginSettings{
		Type:    "canal",
		Version: "v3.22",
	}
}

func createUsercluster(t *testing.T, apicli *utils.TestClient, projectName string, clusterSpec models.CreateClusterSpec) (*rest.Config, string, func(), error) {
	var teardowns []func()
	cleanup := func() {
		n := len(teardowns)
		for i := range teardowns {
			teardowns[n-1-i]()
		}
	}

	// create a project
	proj, err := apicli.CreateProject(projectName)
	if err != nil {
		return nil, "", nil, err
	}
	teardowns = append(teardowns, func() {
		err := apicli.DeleteProject(proj.ID)
		if err != nil {
			t.Errorf("failed to delete project %s: %s", proj.ID, err)
		}
	})

	// create a usercluster on aws
	resp, err := apicli.GetKKPAPIClient().Project.CreateClusterV2(&project.CreateClusterV2Params{
		Body:       &clusterSpec,
		ProjectID:  proj.ID,
		Context:    context.Background(),
		HTTPClient: http.DefaultClient,
	}, apicli.GetBearerToken())
	if err != nil {
		return nil, "", nil, err
	}

	cluster := resp.Payload
	teardowns = append(teardowns, func() {
		_, err := apicli.GetKKPAPIClient().Project.DeleteClusterV2(&project.DeleteClusterV2Params{
			DeleteLoadBalancers: pointer.Bool(true),
			DeleteVolumes:       pointer.Bool(true),
			ClusterID:           cluster.ID,
			ProjectID:           proj.ID,
			Context:             context.Background(),
			HTTPClient:          http.DefaultClient,
		}, apicli.GetBearerToken())
		if err != nil {
			t.Errorf("failed to delete cluster %s/%s: %s", proj.ID, cluster.ID, err)
		}
	})

	// try to get kubeconfig
	var userconfig string
	err = wait.Poll(30*time.Second, 30*time.Minute, func() (bool, error) {
		t.Logf("trying to get kubeconfig...")
		// construct clients
		resp, err := apicli.GetKKPAPIClient().Project.GetClusterKubeconfigV2(&project.GetClusterKubeconfigV2Params{
			ClusterID:  cluster.ID,
			ProjectID:  proj.ID,
			Context:    context.Background(),
			HTTPClient: http.DefaultClient,
		}, apicli.GetBearerToken())
		if err != nil {
			t.Logf("error trying to get kubeconfig: %s", err)
			return false, nil
		}

		userconfig = string(resp.Payload)

		return true, nil
	})
	if err != nil {
		return nil, "", nil, err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(userconfig))
	if err != nil {
		t.Fatalf("failed to build config: %s", err)
	}

	return config, cluster.ID, cleanup, nil
}
