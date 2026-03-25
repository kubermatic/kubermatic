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
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/machine-controller/sdk/net"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/providerconfig/flatcar"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"
	"k8c.io/operating-system-manager/pkg/providerconfig/rockylinux"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	CanalCNI  string = "canal"
	CiliumCNI string = "cilium"
)

type testCase struct {
	cloudProvider          kubermaticv1.ProviderType
	operatingSystems       []providerconfig.OperatingSystem
	cni                    string
	ipFamily               net.IPFamily
	proxyMode              string
	skipNodes              bool
	skipHostNetworkPods    bool
	skipEgressConnectivity bool
}

func (t *testCase) Log(log *zap.SugaredLogger) *zap.SugaredLogger {
	return log.With("provider", t.cloudProvider, "cni", t.cni, "ipfamily", t.ipFamily)
}

var (
	osSpecs = map[providerconfig.OperatingSystem]interface{}{
		providerconfig.OperatingSystemFlatcar:    flatcar.Config{},
		providerconfig.OperatingSystemRHEL:       rhel.Config{},
		providerconfig.OperatingSystemRockyLinux: rockylinux.Config{},
		providerconfig.OperatingSystemUbuntu:     ubuntu.Config{},
	}

	cloudProviderJiggers = map[kubermaticv1.ProviderType]CreateJigFunc{
		kubermaticv1.AlibabaCloudProvider:      newAlibabaTestJig,
		kubermaticv1.AWSCloudProvider:          newAWSTestJig,
		kubermaticv1.AzureCloudProvider:        newAzureTestJig,
		kubermaticv1.DigitaloceanCloudProvider: newDigitaloceanTestJig,
		kubermaticv1.GCPCloudProvider:          newGCPTestJig,
		kubermaticv1.HetznerCloudProvider:      newHetznerTestJig,
		kubermaticv1.OpenstackCloudProvider:    newOpenstackTestJig,
		kubermaticv1.VSphereCloudProvider:      newVSphereTestJig,
	}

	cnis = map[string]*kubermaticv1.CNIPluginSettings{
		CanalCNI: {
			Type: "canal",
		},
		CiliumCNI: {
			Type: "cilium",
		},
	}

	tests = []testCase{
		{
			cloudProvider: kubermaticv1.AlibabaCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:      CiliumCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.AlibabaCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:      CanalCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.AWSCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				// providerconfig.OperatingSystemRHEL, // TODO: disabled due to "BPF host reachable services for UDP needs kernel 4.19.57, 5.1.16, 5.2.0 or newer"
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:                 CiliumCNI,
			ipFamily:            net.IPFamilyIPv4IPv6,
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
			ipFamily:            net.IPFamilyIPv4IPv6,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.AWSCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:                 CanalCNI,
			ipFamily:            net.IPFamilyIPv4IPv6,
			proxyMode:           resources.NFTablesProxyMode,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.AWSCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:                 CiliumCNI,
			ipFamily:            net.IPFamilyIPv4IPv6,
			proxyMode:           resources.NFTablesProxyMode,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.AzureCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
				providerconfig.OperatingSystemRockyLinux,
				providerconfig.OperatingSystemUbuntu,
			},
			cni:      CiliumCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.AzureCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
				providerconfig.OperatingSystemRockyLinux,
				providerconfig.OperatingSystemUbuntu,
			},
			cni:       CiliumCNI,
			proxyMode: resources.NFTablesProxyMode,
			ipFamily:  net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.AzureCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
				providerconfig.OperatingSystemRockyLinux,
				providerconfig.OperatingSystemUbuntu,
			},
			cni:      CanalCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.AzureCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
				providerconfig.OperatingSystemRockyLinux,
				providerconfig.OperatingSystemUbuntu,
			},
			cni:       CanalCNI,
			proxyMode: resources.NFTablesProxyMode,
			ipFamily:  net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.GCPCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                 CiliumCNI,
			ipFamily:            net.IPFamilyIPv4IPv6,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.GCPCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                 CiliumCNI,
			proxyMode:           resources.NFTablesProxyMode,
			ipFamily:            net.IPFamilyIPv4IPv6,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.GCPCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                 CanalCNI,
			ipFamily:            net.IPFamilyIPv4IPv6,
			skipNodes:           true,
			skipHostNetworkPods: true,
		},
		{
			cloudProvider: kubermaticv1.GCPCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                 CanalCNI,
			proxyMode:           resources.NFTablesProxyMode,
			ipFamily:            net.IPFamilyIPv4IPv6,
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
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.OpenstackCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
			},
			cni:       CiliumCNI,
			proxyMode: resources.NFTablesProxyMode,
			ipFamily:  net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.OpenstackCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
			},
			cni:      CanalCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.OpenstackCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemFlatcar,
				providerconfig.OperatingSystemRHEL,
			},
			cni:       CanalCNI,
			proxyMode: resources.NFTablesProxyMode,
			ipFamily:  net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.HetznerCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:      CiliumCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.HetznerCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:       CiliumCNI,
			proxyMode: resources.NFTablesProxyMode,
			ipFamily:  net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.HetznerCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:      CanalCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.HetznerCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:       CanalCNI,
			proxyMode: resources.NFTablesProxyMode,
			ipFamily:  net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.DigitaloceanCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:      CiliumCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.DigitaloceanCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:       CiliumCNI,
			proxyMode: resources.NFTablesProxyMode,
			ipFamily:  net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.DigitaloceanCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:      CanalCNI,
			ipFamily: net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.DigitaloceanCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
				providerconfig.OperatingSystemRockyLinux,
			},
			cni:       CanalCNI,
			proxyMode: resources.NFTablesProxyMode,
			ipFamily:  net.IPFamilyIPv4IPv6,
		},
		{
			cloudProvider: kubermaticv1.VSphereCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                    CanalCNI,
			ipFamily:               net.IPFamilyIPv4IPv6,
			skipEgressConnectivity: true, // TODO: remove once public IPv6 is available in Kubermatic DC
		},
		{
			cloudProvider: kubermaticv1.VSphereCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                    CanalCNI,
			proxyMode:              resources.NFTablesProxyMode,
			ipFamily:               net.IPFamilyIPv4IPv6,
			skipEgressConnectivity: true, // TODO: remove once public IPv6 is available in Kubermatic DC
		},
		{
			cloudProvider: kubermaticv1.VSphereCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                    CiliumCNI,
			ipFamily:               net.IPFamilyIPv4IPv6,
			skipEgressConnectivity: true, // TODO: remove once public IPv6 is available in Kubermatic DC
		},
		{
			cloudProvider: kubermaticv1.VSphereCloudProvider,
			operatingSystems: []providerconfig.OperatingSystem{
				providerconfig.OperatingSystemUbuntu,
			},
			cni:                    CiliumCNI,
			proxyMode:              resources.NFTablesProxyMode,
			ipFamily:               net.IPFamilyIPv4IPv6,
			skipEgressConnectivity: true, // TODO: remove once public IPv6 is available in Kubermatic DC
		},
	}
)

func isAll(s sets.Set[string]) bool {
	return s.Len() == 0 || (s.Len() == 1 && s.Has("all"))
}

func osToStringSet(os []providerconfig.OperatingSystem) sets.Set[string] {
	result := sets.New[string]()
	for _, o := range os {
		result.Insert(string(o))
	}
	return result
}

func stringToOSSet(os sets.Set[string]) []providerconfig.OperatingSystem {
	result := []providerconfig.OperatingSystem{}
	for _, o := range sets.List(os) {
		result = append(result, providerconfig.OperatingSystem(o))
	}
	return result
}

// removeDisabledTests removes tests with criteria that are not enabled via
// CLI flags and also adjusts the list of OS's per test to match what is
// enabled via CLI.
func removeDisabledTests(allTests []testCase, log *zap.SugaredLogger) []testCase {
	result := []testCase{}

	for _, test := range allTests {
		testLog := test.Log(log)

		if !isAll(enabledCNIs) && !enabledCNIs.Has(test.cni) {
			testLog.Info("Skipping scenario because CNI is not enabled.")
			continue
		}

		if !isAll(enabledProviders) && !enabledProviders.Has(string(test.cloudProvider)) {
			testLog.Info("Skipping scenario because cloud provider is not enabled.")
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
			testLog.Info("Skipping scenario because no OS specified to test.")
			continue
		}

		result = append(result, testCase{
			cloudProvider:          test.cloudProvider,
			operatingSystems:       operatingSystems, // NB: here we override the OS list
			cni:                    test.cni,
			ipFamily:               test.ipFamily,
			proxyMode:              test.proxyMode,
			skipNodes:              test.skipNodes,
			skipHostNetworkPods:    test.skipHostNetworkPods,
			skipEgressConnectivity: test.skipEgressConnectivity,
		})
	}

	for _, test := range result {
		test.Log(log).Info("Enabled scenario")
	}

	return result
}

// TestNewClusters creates clusters and runs dualstack tests against them.
func TestNewClusters(t *testing.T) {
	ctx := signals.SetupSignalHandler()
	log := log.NewFromOptions(logOptions).Sugar()

	parseProviderCredentials(t)

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to get client for seed cluster: %v", err)
	}

	filteredTests := removeDisabledTests(tests, log)

	for _, test := range filteredTests {
		testName := fmt.Sprintf("%s-%s-%s", test.cloudProvider, test.cni, test.ipFamily)
		if test.proxyMode != "" {
			testName = fmt.Sprintf("%s-%s", testName, test.proxyMode)
		}

		t.Run(testName, func(t *testing.T) {
			testLog := log.With("provider", test.cloudProvider, "cni", test.cni, "ipfamily", test.ipFamily)
			if test.proxyMode != "" {
				testLog = testLog.With("proxymode", test.proxyMode)
			}

			jigCreator := cloudProviderJiggers[test.cloudProvider]
			clusterName := fmt.Sprintf("dualstack-e2e-%s", rand.String(5))
			clusterName = strings.ReplaceAll(strings.ToLower(clusterName), "+", "-")

			// setup a test jig for the given provider, using the default e2e test settings
			testLog = testLog.With("cluster", clusterName)
			testJig := jigCreator(seedClient, testLog)

			// customize the default cluster config to be suitable for dualstack networking
			testJig.ClusterJig.
				WithName(clusterName).
				WithHumanReadableName(fmt.Sprintf("Dualstack E2E (%s, %s, %s)", test.cloudProvider, test.cni, test.ipFamily)).
				WithKonnectivity(true).
				WithCNIPlugin(cnis[test.cni]).
				WithPatch(func(c *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
					c.ClusterNetwork.IPFamily = kubermaticv1.IPFamilyDualStack
					c.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16", "fd01::/48"}
					c.ClusterNetwork.Services.CIDRBlocks = []string{"10.240.16.0/20", "fd02::/108"}
					return c
				})

			// TODO: Update to nftables, when updating to default for canal CNI
			// change proxy mode depending on the CNI
			switch test.cni {
			case CanalCNI:
				testJig.ClusterJig.WithProxyMode(resources.IPVSProxyMode)
			case CiliumCNI:
				testJig.ClusterJig.WithProxyMode(resources.EBPFProxyMode)
			}

			// Override proxy mode if specified in test case
			if test.proxyMode != "" {
				testJig.ClusterJig.WithProxyMode(test.proxyMode)
			}

			// we create the machines later ourselves, so for now we do not want the jig to create them
			machineJig := testJig.MachineJig
			testJig.MachineJig = nil

			// create the project and cluster (waits until the control plane is healthy, the WaitForNothing
			// has no effect since no machines are being created)
			_, cluster, err := testJig.Setup(ctx, jig.WaitForNothing)
			if err != nil {
				t.Fatalf("Failed to create cluster: %v", err)
			}

			// The cleanup uses a background context so that when the tests are cancelled,
			// the cleanup is *not* interrupted, otherwise on CI we leak lots of cloud resources.
			defer testJig.Cleanup(context.Background(), t, true)

			// create a MachineDeployment with 1 replica each per operating system, do not yet wait for anything
			for _, osName := range test.operatingSystems {
				osSpec := osSpecs[osName]

				if osName == providerconfig.OperatingSystemRHEL {
					osSpec = addRHELSubscriptionInfo(osSpec)
				}

				osMachineJig := machineJig.Clone()
				osMachineJig.
					WithName(fmt.Sprintf("md-%s", osName)).
					WithOSSpec(osSpec).
					WithNetworkConfig(&providerconfig.NetworkConfig{
						IPFamily: net.IPFamilyIPv4IPv6,
					})

				// no need to keep track of machine cleanups, as KKP will delete all machines in the
				// cluster when the cluster itself is being deleted
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
			if err := machineJig.WithReplicas(len(test.operatingSystems)).WaitForReadyNodes(ctx, userclusterClient); err != nil {
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
		return fmt.Errorf("failed to build requirement: %w", err)
	}
	l := labels.NewSelector().Add(*r)

	return wait.PollImmediateLog(ctx, log, 10*time.Second, 15*time.Minute, func(ctx context.Context) (error, error) {
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
	unhealthy := sets.New[string]()

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
		return fmt.Errorf("not all pods are ready: %v", sets.List(unhealthy))
	}

	return nil
}

func parseProviderCredentials(t *testing.T) {
	if isAll(enabledProviders) || enabledProviders.Has(string(kubermaticv1.AlibabaCloudProvider)) {
		if err := alibabaCredentials.Parse(); err != nil {
			t.Fatalf("Failed to get alibaba credentials: %v", err)
		}
	}

	if isAll(enabledProviders) || enabledProviders.Has(string(kubermaticv1.AWSCloudProvider)) {
		if err := awsCredentials.Parse(); err != nil {
			t.Fatalf("Failed to get aws credentials: %v", err)
		}
	}

	if isAll(enabledProviders) || enabledProviders.Has(string(kubermaticv1.AzureCloudProvider)) {
		if err := azureCredentials.Parse(); err != nil {
			t.Fatalf("Failed to get azure credentials: %v", err)
		}
	}

	if isAll(enabledProviders) || enabledProviders.Has(string(kubermaticv1.DigitaloceanCloudProvider)) {
		if err := digitaloceanCredentials.Parse(); err != nil {
			t.Fatalf("Failed to get digitalocean credentials: %v", err)
		}
	}

	if isAll(enabledProviders) || enabledProviders.Has(string(kubermaticv1.GCPCloudProvider)) {
		if err := gcpCredentials.Parse(); err != nil {
			t.Fatalf("Failed to get gcp credentials: %v", err)
		}
	}

	if isAll(enabledProviders) || enabledProviders.Has(string(kubermaticv1.HetznerCloudProvider)) {
		if err := hetznerCredentials.Parse(); err != nil {
			t.Fatalf("Failed to get hetzner credentials: %v", err)
		}
	}

	if isAll(enabledProviders) || enabledProviders.Has(string(kubermaticv1.OpenstackCloudProvider)) {
		if err := openstackCredentials.Parse(); err != nil {
			t.Fatalf("Failed to get openstack credentials: %v", err)
		}
	}

	if isAll(enabledProviders) || enabledProviders.Has(string(kubermaticv1.VSphereCloudProvider)) {
		if err := vsphereCredentials.Parse(); err != nil {
			t.Fatalf("Failed to get vsphere credentials: %v", err)
		}
	}
}
