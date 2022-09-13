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
	"strings"
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

const (
	// Operating Systems.
	CentOS     string = "centos"
	Flatcar    string = "flatcar"
	RHEL       string = "rhel"
	SLES       string = "sles"
	Ubuntu     string = "ubuntu"
	RockyLinux string = "rockylinux"

	// CNI.
	Canal  string = "canal"
	Cilium string = "cilium"

	// Providers.
	Azure        string = "azure"
	GCP          string = "gcp"
	AWS          string = "aws"
	OpenStack    string = "openstack"
	Hetzner      string = "hetzner"
	DigitalOcean string = "do"
	Equinix      string = "equinix"
	VSphere      string = "vsphere"
)

var operatingSystems = map[string]func() models.OperatingSystemSpec{
	CentOS:     centos,
	Flatcar:    flatcar,
	RHEL:       rhel,
	SLES:       sles,
	Ubuntu:     ubuntu,
	RockyLinux: rockyLinux,
}

var cloudProviders = map[string]clusterSpec{
	Azure:        azure{},
	GCP:          gcp{},
	AWS:          aws{},
	OpenStack:    openstack{},
	Hetzner:      hetzner{},
	DigitalOcean: do{},
	Equinix:      equinix{},
	VSphere:      vsphere{},
}

var cnis = map[string]models.CNIPluginSettings{
	Cilium: cilium(),
	Canal:  canal(),
}

// TestCloudClusterIPFamily creates clusters and runs dualstack tests against them.
func TestCloudClusterIPFamily(t *testing.T) {
	// export KUBERMATIC_API_ENDPOINT=https://dev.kubermatic.io
	// export KKP_API_TOKEN=<steal token>
	token := os.Getenv("KKP_API_TOKEN")

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

	type testCase struct {
		cloudName              string
		osNames                []string
		cni                    string
		ipFamily               util.IPFamily
		skipNodes              bool
		skipHostNetworkPods    bool
		skipEgressConnectivity bool
	}

	tests := []testCase{
		{
			cloudName: Azure,
			osNames: []string{
				Flatcar,
				RHEL,
				RockyLinux,
				Ubuntu,
			},
			cni:      Cilium,
			ipFamily: util.DualStack,
		},
		{
			cloudName: Azure,
			osNames: []string{
				CentOS,
				Flatcar,
				RHEL,
				RockyLinux,
				Ubuntu,
			},
			cni:      Canal,
			ipFamily: util.DualStack,
		},
		{
			cloudName: AWS,
			osNames: []string{
				RHEL,
				Ubuntu,
				Flatcar,
				RockyLinux,
			},
			cni:                 Cilium,
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudName: AWS,
			osNames: []string{
				RHEL,
				Ubuntu,
				Flatcar,
				RockyLinux,
			},
			cni:                 Canal,
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudName: GCP,
			osNames: []string{
				Ubuntu,
			},
			cni:                 Cilium,
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudName: GCP,
			osNames: []string{
				Ubuntu,
			},
			cni:                 Canal,
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudName: OpenStack,
			osNames: []string{
				Ubuntu,
				Flatcar,
				RHEL,
			},
			cni:      Cilium,
			ipFamily: util.DualStack,
		},
		{
			cloudName: OpenStack,
			osNames: []string{
				Ubuntu,
				Flatcar,
				RHEL,
			},
			cni:      Canal,
			ipFamily: util.DualStack,
		},
		{
			cloudName: Hetzner,
			osNames: []string{
				Ubuntu,
				RockyLinux,
			},
			cni:      Cilium,
			ipFamily: util.DualStack,
		},
		{
			cloudName: Hetzner,
			osNames: []string{
				Ubuntu,
				RockyLinux,
			},
			cni:      Canal,
			ipFamily: util.DualStack,
		},
		{
			cloudName: DigitalOcean,
			osNames: []string{
				Ubuntu,
				RockyLinux,
			},
			cni:      Cilium,
			ipFamily: util.DualStack,
		},
		{
			cloudName: DigitalOcean,
			osNames: []string{
				Ubuntu,
				CentOS,
				RockyLinux,
			},
			cni:      Canal,
			ipFamily: util.DualStack,
		},
		{
			cloudName: Equinix,
			osNames: []string{
				Ubuntu,
				CentOS,
				Flatcar,
				RockyLinux,
			},
			cni:      Canal,
			ipFamily: util.DualStack,
		},
		{
			cloudName: Equinix,
			osNames: []string{
				Ubuntu,
				Flatcar,
				RockyLinux,
			},
			cni:      Cilium,
			ipFamily: util.DualStack,
		},
		{
			cloudName: VSphere,
			osNames: []string{
				Ubuntu,
			},
			cni:                    Canal,
			ipFamily:               util.DualStack,
			skipEgressConnectivity: true, // TODO: remove once public IPv6 is available in Kubermatic DC
		},
		{
			cloudName: VSphere,
			osNames: []string{
				Ubuntu,
			},
			cni:                    Cilium,
			ipFamily:               util.DualStack,
			skipEgressConnectivity: true, // TODO: remove once public IPv6 is available in Kubermatic DC
		},
	}

	retestBudget := 2
	ch := make(chan int, retestBudget)
	for i := 0; i < retestBudget; i++ {
		ch <- i
	}
	close(ch)
	var retested sync.Map
	var mu sync.Mutex

	for _, test := range tests {
		test := test
		name := fmt.Sprintf("c-%s-%s-%s", test.cloudName, test.cni, test.ipFamily)

		if cni != "" && !strings.Contains(cni, test.cni) {
			t.Logf("skipping %s due to cni (%s not in %s)...", name, test.cni, cni)
			continue
		}

		if provider != "all" && !strings.Contains(provider, test.cloudName) {
			t.Logf("skipping %s due to provider (%s not in %s)...", name, test.cloudName, provider)
			continue
		}

		var testOSNames []string
		if osNames == "all" {
			testOSNames = test.osNames
			t.Logf("testing all os in %q", testOSNames)
		} else {
			for _, osName := range test.osNames {
				if strings.Contains(osNames, osName) {
					testOSNames = append(testOSNames, osName)
				} else {
					t.Logf("skipping %q because it is not in %q", osName, osNames)
				}
			}
		}
		if len(testOSNames) == 0 {
			t.Logf("skipping because no OS specified to test (available %q)", strings.Join(test.osNames, ","))
			continue
		}

		cloud := cloudProviders[test.cloudName]
		cloudSpec := cloud.CloudSpec()
		cniSpec := cnis[test.cni]
		netConfig := defaultClusterNetworkingConfig()
		switch test.cni {
		case Canal:
			netConfig = netConfig.WithProxyMode("ipvs")
		case Cilium:
			netConfig = netConfig.WithProxyMode("ebpf")
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

		retest:
			clusterSpec := defaultClusterRequest().WithName(name).
				WithCloud(cloudSpec).
				WithCNI(cniSpec).
				WithNetworkConfig(models.ClusterNetworkingConfig(netConfig))
			spec := models.CreateClusterSpec(clusterSpec)

			mu.Lock()
			name := fmt.Sprintf("%s-%s", name, rand.String(4))
			config, projectID, clusterID, cleanup, err := createUsercluster(t, apicli, name, spec)
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

			nodeSpec := cloud.NodeSpec()

			for _, osName := range testOSNames {
				// This is needed because node spec for OpenStack requires image.
				operatingSystem := operatingSystems[osName]()

				if test.cloudName == OpenStack {
					img := openstack{}.getImage(osName)
					nodeSpec.Openstack.Image = &img
					if osName == RHEL {
						operatingSystem = addRHELSubscriptionInfo(operatingSystem)
					}
				}

				if test.cloudName == Azure {
					if osName == RHEL {
						operatingSystem = addRHELSubscriptionInfo(operatingSystem)
					}
				}

				err := createMachineDeployment(t, apicli, defaultCreateMachineDeploymentParams().
					WithName(fmt.Sprintf("md-%s", osName)).
					WithProjectID(projectID).
					WithClusterID(clusterID).
					WithOS(operatingSystem).
					WithNodeSpec(nodeSpec),
				)
				if err != nil {
					t.Fatalf("failed to create machine deployment: %v", err)
				}
			}

			userclusterClient, err := kubernetes.NewForConfig(config)
			if err != nil {
				t.Fatalf("failed to create usercluster client: %s", err)
			}

			t.Logf("waiting for nodes to come up")
			err = checkNodeReadiness(t, userclusterClient, len(testOSNames))
			if err != nil {
				go func() {
					mu.Lock()
					cleanup()
					mu.Unlock()
				}()

				_, ok := retested.Load(name)
				if !ok {
					retested.Store(name, true)
					_, ok := <-ch
					if !ok {
						t.Log("out of retest budget")
						t.Fatalf("nodes never became ready: %v", err)
					}
					t.Logf("retesting...")
					goto retest
				}
				t.Fatalf("nodes never became ready: %v", err)
			}

			t.Logf("nodes ready")
			t.Logf("sleeping for 4m...")
			time.Sleep(time.Minute * 4)

			err = waitForPods(t, userclusterClient, kubeSystem, "app", []string{
				"coredns", "konnectivity-agent", "kube-proxy", "metrics-server",
				"node-local-dns", "user-ssh-keys-agent",
			})

			if err != nil {
				t.Fatalf("pods never became ready: %v", err)
			}

			testUserCluster(t, userclusterClient, test.ipFamily, test.skipNodes, test.skipHostNetworkPods, test.skipEgressConnectivity)
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

func checkNodeReadiness(t *testing.T, userClient *kubernetes.Clientset, expectedNodes int) error {
	return wait.Poll(30*time.Second, 30*time.Minute, func() (bool, error) {
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

		return true, nil
	})
}

func createUsercluster(t *testing.T, apicli *utils.TestClient, projectName string, clusterSpec models.CreateClusterSpec) (*rest.Config, string, string, func(), error) {
	var teardowns []func() error
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			n := len(teardowns)
			for i := range teardowns {
				err := wait.Poll(10*time.Second, 10*time.Minute, func() (bool, error) {
					err := teardowns[n-1-i]()
					if err != nil {
						t.Log(err)
						return false, nil
					}
					return true, nil
				})
				if err != nil {
					t.Errorf("cleanup failed: %s", err)
				}
			}
		})
	}

	// create a project
	proj, err := apicli.CreateProject(projectName)
	if err != nil {
		return nil, "", "", nil, err
	}
	teardowns = append(teardowns, func() error {
		err := apicli.DeleteProject(proj.ID)
		if err != nil {
			return fmt.Errorf("failed to delete project %s: %w", proj.ID, err)
		}
		return nil
	})

	// create a usercluster on aws
	resp, err := apicli.GetKKPAPIClient().Project.CreateClusterV2(&project.CreateClusterV2Params{
		Body:       &clusterSpec,
		ProjectID:  proj.ID,
		Context:    context.Background(),
		HTTPClient: http.DefaultClient,
	}, apicli.GetBearerToken())
	if err != nil {
		return nil, "", "", nil, err
	}

	cluster := resp.Payload
	teardowns = append(teardowns, func() error {
		_, err := apicli.GetKKPAPIClient().Project.DeleteClusterV2(&project.DeleteClusterV2Params{
			DeleteLoadBalancers: pointer.Bool(true),
			DeleteVolumes:       pointer.Bool(true),
			ClusterID:           cluster.ID,
			ProjectID:           proj.ID,
			Context:             context.Background(),
			HTTPClient:          http.DefaultClient,
		}, apicli.GetBearerToken())
		if err != nil {
			return fmt.Errorf("failed to delete cluster %s/%s: %w", proj.ID, cluster.ID, err)
		}
		return nil
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
		return nil, "", "", nil, err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(userconfig))
	if err != nil {
		t.Fatalf("failed to build config: %s", err)
	}

	return config, proj.ID, cluster.ID, cleanup, nil
}

func createMachineDeployment(t *testing.T, apicli *utils.TestClient, params createMachineDeploymentParams) error {
	mdParams := project.CreateMachineDeploymentParams(params)
	return wait.Poll(30*time.Second, 10*time.Minute, func() (bool, error) {
		_, err := apicli.GetKKPAPIClient().Project.CreateMachineDeployment(
			&mdParams,
			apicli.GetBearerToken())
		if err != nil {
			respErr := new(project.CreateMachineDeploymentDefault)
			if errors.As(err, &respErr) {
				errData, err := respErr.GetPayload().MarshalBinary()
				if err != nil {
					t.Log("failed to marshal error response")
				}
				t.Log(string(errData))
			}
			t.Logf("failed to create machine deployment: %v", err)
			return false, nil
		}
		return true, nil
	})
}
