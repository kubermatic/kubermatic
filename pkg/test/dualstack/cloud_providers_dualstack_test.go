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
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/util"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/operating-system-manager/pkg/providerconfig/centos"
	"k8c.io/operating-system-manager/pkg/providerconfig/flatcar"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"
	"k8c.io/operating-system-manager/pkg/providerconfig/rockylinux"
	"k8c.io/operating-system-manager/pkg/providerconfig/sles"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CanalCNI  string = "canal"
	CiliumCNI string = "cilium"
)

type testCase struct {
	cloudProvider          kubermaticv1.ProviderType
	operatingSystems       []providerconfig.OperatingSystem
	cni                    string
	ipFamily               util.IPFamily
	skipNodes              bool
	skipHostNetworkPods    bool
	skipEgressConnectivity bool
}

var (
	osSpecs = map[providerconfig.OperatingSystem]interface{}{
		providerconfig.OperatingSystemCentOS:     centos.Config{},
		providerconfig.OperatingSystemFlatcar:    flatcar.Config{},
		providerconfig.OperatingSystemRHEL:       rhel.Config{},
		providerconfig.OperatingSystemSLES:       sles.Config{},
		providerconfig.OperatingSystemUbuntu:     ubuntu.Config{},
		providerconfig.OperatingSystemRockyLinux: rockylinux.Config{},
	}

	cloudProviderJiggers = map[kubermaticv1.ProviderType]CreateJigFunc{
		kubermaticv1.AzureCloudProvider:        newAzureTestJig,
		kubermaticv1.GCPCloudProvider:          newGCPTestJig,
		kubermaticv1.AWSCloudProvider:          newAWSTestJig,
		kubermaticv1.OpenstackCloudProvider:    newOpenstackTestJig,
		kubermaticv1.HetznerCloudProvider:      newHetznerTestJig,
		kubermaticv1.DigitaloceanCloudProvider: newDigitaloceanTestJig,
		kubermaticv1.PacketCloudProvider:       newEquinixMetalTestJig,
		kubermaticv1.VSphereCloudProvider:      newVSphereTestJig,
	}

	cnis = map[string]*kubermaticv1.CNIPluginSettings{
		CiliumCNI: {
			Type: "cilium",
		},
		CanalCNI: {
			Type: "canal",
		},
	}

	tests = []testCase{
		{
			cloudProvider: kubermaticv1.AzureCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
				providerconfig.OperatingSystemRockyLinux,
				providerconfig.OperatingSystemUbuntu,
			},
			cni:      CiliumCNI,
			ipFamily: util.DualStack,
		},
		{
			cloudProvider: kubermaticv1.AzureCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemCentOS,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
				providerconfig.OperatingSystemRockyLinux,
				providerconfig.OperatingSystemUbuntu,
			},
			cni:      CanalCNI,
			ipFamily: util.DualStack,
		},
		{
			cloudProvider: kubermaticv1.AWSCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemRHEL,
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:                 CiliumCNI,
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.AWSCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemRHEL,
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:                 CanalCNI,
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.GCPCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                 CiliumCNI,
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.GCPCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                 CanalCNI,
			ipFamily:            util.DualStack,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.OpenstackCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
			},
			cni:      CiliumCNI,
			ipFamily: util.DualStack,
		},
		{
			cloudProvider: kubermaticv1.OpenstackCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
			},
			cni:      CanalCNI,
			ipFamily: util.DualStack,
		},
		{
			cloudProvider: kubermaticv1.HetznerCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:      CiliumCNI,
			ipFamily: util.DualStack,
		},
		{
			cloudProvider: kubermaticv1.HetznerCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:      CanalCNI,
			ipFamily: util.DualStack,
		},
		{
			cloudProvider: kubermaticv1.DigitaloceanCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:      CiliumCNI,
			ipFamily: util.DualStack,
		},
		{
			cloudProvider: kubermaticv1.DigitaloceanCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemCentOS,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:      CanalCNI,
			ipFamily: util.DualStack,
		},
		{
			cloudProvider: kubermaticv1.PacketCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemCentOS,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:       CanalCNI,
			ipFamily:  util.DualStack,
			skipNodes: true,
		},
		{
			cloudProvider: kubermaticv1.PacketCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:       CiliumCNI,
			ipFamily:  util.DualStack,
			skipNodes: true,
		},
		{
			cloudProvider: kubermaticv1.VSphereCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                    CanalCNI,
			ipFamily:               util.DualStack,
			skipEgressConnectivity: true, // TODO: remove once public IPv6 is available in Kubermatic DC
		},
		{
			cloudProvider: kubermaticv1.VSphereCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                    CiliumCNI,
			ipFamily:               util.DualStack,
			skipEgressConnectivity: true, // TODO: remove once public IPv6 is available in Kubermatic DC
		},
	}
)

func isAll(s sets.String) bool {
	return s.Len() == 0 || (s.Len() == 1 && s.Has("all"))
}

func osToStringSet(os []providerconfig.OperatingSystem) sets.String {
	result := sets.NewString()
	for _, o := range os {
		result.Insert(string(o))
	}
	return result
}

func stringToOSSet(os sets.String) []providerconfig.OperatingSystem {
	result := []providerconfig.OperatingSystem{}
	for _, o := range os.List() {
		result = append(result, providerconfig.OperatingSystem(o))
	}
	return result
}

// TestNewClusters creates clusters and runs dualstack tests against them.
func TestNewClusters(t *testing.T) {
	ctx := context.Background()
	log := log.NewFromOptions(logOptions).Sugar()

	seedClient, _, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to get client for seed cluster: %v", err)
	}

	for i := range tests {
		test := tests[i] // use a loop-local variable because parts of this loop are run concurrently
		name := fmt.Sprintf("c-%s-%s-%s", test.cloudProvider, test.cni, test.ipFamily)
		testLog := log.With("test", name)

		if !enabledCNIs.Has(test.cni) {
			testLog.Info("Skipping test because CNI is not enabled.")
			continue
		}

		if !isAll(enabledProviders) && !enabledProviders.Has(string(test.cloudProvider)) {
			testLog.Info("Skipping test because cloud provider is not enabled.")
			continue
		}

		var operatingSystems []providerconfig.OperatingSystem
		if isAll(enabledOperatingSystems) {
			operatingSystems = test.operatingSystems
		} else {
			testOperatingSystems := osToStringSet(test.operatingSystems)
			operatingSystems = stringToOSSet(testOperatingSystems.Intersection(enabledOperatingSystems))
		}

		if len(operatingSystems) == 0 {
			testLog.Info("Skipping test because no OS specified to test")
			continue
		}

		testLog.Infow("Testing operating systems", "os", osToStringSet(operatingSystems).List())

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			jigCreator := cloudProviderJiggers[test.cloudProvider]
			clusterName := fmt.Sprintf("%s-%s", name, rand.String(4))

			// setup a test jig for the given provider, using the default e2e test settings
			testJig := jigCreator(seedClient, testLog)

			// customize the default cluster config to be suitable for dualstack networking
			testJig.ClusterJig.
				WithName(clusterName).
				WithKonnectivity(true).
				WithCNIPlugin(cnis[test.cni]).
				WithPatch(func(c *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
					c.ClusterNetwork.IPFamily = "IPv4+IPv6"
					c.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16", "fd01::/48"}
					c.ClusterNetwork.Services.CIDRBlocks = []string{"10.240.16.0/20", "fd02::/120"}
					return c
				})

			// change proxy mode depending on the CNI
			switch test.cni {
			case CanalCNI:
				testJig.ClusterJig.WithProxyMode("ipvs")
			case CiliumCNI:
				testJig.ClusterJig.WithProxyMode("ebpf")
			}

			// we create the machines later ourselves, so for now we do not want the jig to create them
			machineJig := testJig.MachineJig
			testJig.MachineJig = nil

			// create the project and cluster (waits until the control plane is healthy, the WaitForNothing
			// has no effect since no machines are being created)
			_, cluster, err := testJig.Setup(ctx, jig.WaitForNothing)
			defer testJig.Cleanup(ctx, t, true)

			// create a MachineDeployment with 1 replica each per operating system, do not yet wait for anything
			for _, osName := range operatingSystems {
				osSpec := osSpecs[osName]

				if osName == providerconfig.OperatingSystemRHEL {
					osSpec = addRHELSubscriptionInfo(osSpec)
				}

				osMachineJig := machineJig.Clone()
				osMachineJig.WithName(fmt.Sprintf("md-%s", osName)).WithOSSpec(osSpec)

				if err := osMachineJig.Create(ctx, jig.WaitForNothing, cluster.Spec.Cloud.DatacenterName); err != nil {
					t.Fatalf("Failed to create machine deployment: %v", err)
				}
			}

			userclusterClient, err := testJig.ClusterClient(ctx)
			if err != nil {
				t.Fatalf("Failed to create usercluster client: %v", err)
			}

			// now we wait for all nodes to become ready; we cheat a bit and abuse a MachineJig
			testLog.Info("Waiting for all nodes to be ready...")
			if err := machineJig.WithReplicas(len(operatingSystems)).WaitForReadyNodes(ctx, userclusterClient); err != nil {
				t.Fatalf("Not all nodes did get ready: %v", err)
			}
			testLog.Info("All nodes are ready.")

			// give things time to settle
			duration := 4 * time.Minute
			testLog.Infow("Letting things settle down...", "wait", duration)
			time.Sleep(duration)

			log.Infow("Checking pod readiness...", "namespace", metav1.NamespaceSystem)
			err = waitForPods(t, ctx, log, userclusterClient, metav1.NamespaceSystem, "app", []string{
				"coredns", "konnectivity-agent", "kube-proxy", "metrics-server",
				"node-local-dns", "user-ssh-keys-agent",
			})
			if err != nil {
				t.Fatalf("Pods never became ready: %v", err)
			}

			// and now run the actual test logic, which is shared between this test and the one for
			// existing clusters
			testUserCluster(t, ctx, testLog, userclusterClient, test.ipFamily, test.skipNodes, test.skipHostNetworkPods, test.skipEgressConnectivity)
		})
	}
}

func waitForPods(t *testing.T, ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, namespace string, key string, names []string) error {
	r, err := labels.NewRequirement(key, selection.In, names)
	if err != nil {
		return fmt.Errorf("failed to build requirement: %v", err)
	}
	l := labels.NewSelector().Add(*r)

	return wait.PollLog(ctx, log, 30*time.Second, 15*time.Minute, func() (error, error) {
		pods := corev1.PodList{}
		err := client.List(ctx, &pods, &ctrlruntimeclient.ListOptions{
			Namespace:     namespace,
			LabelSelector: l,
		})
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err), nil
		}

		if len(pods.Items) == 0 {
			return errors.New("no pods founds"), nil
		}

		return allPodsHealthy(t, &pods), nil
	})
}

func allPodsHealthy(t *testing.T, pods *corev1.PodList) error {
	unhealthy := sets.NewString()

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			unhealthy.Insert(pod.Name)
			continue
		}

		healthy := true
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodReady || c.Type == corev1.ContainersReady {
				if c.Status != corev1.ConditionTrue {
					healthy = false
				}
			}
		}

		if !healthy {
			unhealthy.Insert(pod.Name)
		}
	}

	if unhealthy.Len() > 0 {
		return fmt.Errorf("not all pods are ready: %v", unhealthy.List())
	}

	return nil
}
