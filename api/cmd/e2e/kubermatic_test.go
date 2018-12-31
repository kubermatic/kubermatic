// +build e2e

package main

import (
	"flag"
	goos "os"
	"path"
	"strings"
	"testing"
	"time"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	clustercontroller "github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfigPath          string
	dcFile                  string
	nodeCount               int
	workerName              string
	controlPlaneWaitTimeout time.Duration
	deleteClustersWhenDone  bool
	workingDir              string
	testBinRoot             string
)

func init() {
	flag.StringVar(&kubeconfigPath, "kubeconfig", "/config/kubeconfig", "path to kubeconfig file")
	flag.StringVar(&dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&workerName, "worker-name", "", "Worker name to set on the cluster object")
	flag.StringVar(&workingDir, "working-dir", "", "Working directory. Used to store test specific files like Kubeconfig and Ginkgo reports.")
	flag.StringVar(&testBinRoot, "test-bin-dir", "", "Root containing the test binaries for all different Kubernetes versions. The folder must contain sub folder for each Kubernetes minor version.")
	flag.DurationVar(&controlPlaneWaitTimeout, "control-plane-wait-timeout", 30*time.Minute, "Time to wait until the control plane of the cluster comes up")
	flag.IntVar(&nodeCount, "node-count", 3, "The number of nodes to add to a cluster")
	flag.BoolVar(&deleteClustersWhenDone, "delete-clusters-when-done", true, "Delete the cluster after all tests are done")
	flag.Parse()
}

func TestE2E(t *testing.T) {
	dcs, err := provider.LoadDatacentersMeta(dcFile)
	if err != nil {
		t.Fatalf("failed to load datacenter yaml %q: %v", dcFile, err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		t.Fatal(err)
	}

	kubermaticClient, err := kubermaticclientset.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	kubermaticInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, informer.DefaultInformerResyncPeriod)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, informer.DefaultInformerResyncPeriod)
	clusterLister := kubermaticInformerFactory.Kubermatic().V1().Clusters().Lister()
	clusterClientProvider := clusterclient.New(kubeInformerFactory.Core().V1().Secrets().Lister())

	kubermaticInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.Start(wait.NeverStop)
	kubermaticInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	versions := []*semver.Semver{
		semver.NewSemverOrDie("v1.10.12"),
		semver.NewSemverOrDie("v1.11.6"),
		semver.NewSemverOrDie("v1.12.5"),
		semver.NewSemverOrDie("v1.13.2"),
	}

	for _, version := range versions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			t.Parallel()

			providers := []string{
				provider.AWSCloudProvider,
				provider.DigitaloceanCloudProvider,
				provider.OpenstackCloudProvider,
				provider.AzureCloudProvider,
				provider.VSphereCloudProvider,
				provider.HetznerCloudProvider,
			}

			for _, prov := range providers {
				prov := prov
				t.Run(prov, func(t *testing.T) {
					t.Parallel()

					operatingSystems := []string{
						"coreos",
						"ubuntu",
						"centos",
					}

					for _, os := range operatingSystems {
						os := os
						t.Run(os, func(t *testing.T) {
							t.Parallel()
							if prov == provider.HetznerCloudProvider && os == "coreos" {
								t.Skip("Hetzner has no native support for CoreOS")
							}
							if os == "centos" {
								t.Skip("TODO: Fix CentOS for conformance tests")
							}

							cluster := getCluster(prov, version, workerName, t)
							node := getNode(prov, os, version, t)

							dir := path.Join(workingDir, strings.Replace(t.Name(), "/", "_", -1))
							if err := goos.MkdirAll(dir, 0755); err != nil {
								t.Fatalf("failed to create test directory: %v", err)
							}

							executeClusterTests(&TestContext{
								node:    node,
								cluster: cluster,

								nodeCount:               nodeCount,
								clusterClientProvider:   clusterClientProvider,
								clusterLister:           clusterLister,
								dcs:                     dcs,
								kubeClient:              kubeClient,
								kubermaticClient:        kubermaticClient,
								workerName:              workerName,
								controlPlaneWaitTimeout: controlPlaneWaitTimeout,
								deleteClustersWhenDone:  deleteClustersWhenDone,
								workingDir:              dir,
								testBinRoot:             testBinRoot,
							}, t)
						})
					}
				})
			}
		})
	}
}

func executeClusterTests(ctx *TestContext, t *testing.T) {
	// Create the cluster object
	setupCluster(ctx, t)

	defer func() {
		if ctx.deleteClustersWhenDone {
			if err := ctx.kubermaticClient.KubermaticV1().Clusters().Delete(ctx.cluster.Name, nil); err != nil {
				t.Logf("failed to delete cluster '%s': %v", ctx.cluster.Name, err)
			}
		}
	}()

	// Wait until the control plane is up and running
	waitForControlPlane(ctx, t)
	// Fills ClusterContext inside the test context. Not pretty but avoids having to pass another variable around
	setupClusterContext(ctx, t)
	// Creates the instances
	setupNodes(ctx, t)
	// Wait for nodes
	waitForNodes(ctx, t)
	// Set cleanup finalizers - those should only be set if we reached a healthy cluster
	setFinalizers(ctx, t)
	// Wait for all kube-system pods to be ready
	waitForAllSystemPods(ctx, t)

	// We can run all tests in paralle, except for the ginkgo serial run
	t.Run("Parallel tests", func(t *testing.T) {
		if supportsStorage(ctx.cluster) {
			t.Run("[CloudProvider] Test PVC support with the existing StorageClass", func(t *testing.T) {
				t.Parallel()
				Retry(t, 3, 30*time.Second, func(t *R) {
					testStorageSupport(ctx, t)
				})
			})
		}

		if supportsLBs(ctx.cluster) {
			t.Run("[CloudProvider] Test LB support", func(t *testing.T) {
				t.Parallel()
				Retry(t, 3, 30*time.Second, func(t *R) {
					testLBSupport(ctx, t)
				})
			})
		}

		t.Run("[Conformance] Parallel", func(t *testing.T) {
			t.Parallel()
			testConformanceParallel(ctx, t)
		})
	})

	t.Run("[Conformance] Serial", func(t *testing.T) {
		testConformanceSerial(ctx, t)
	})
}

func getClusterCloudSpec(prov string, t *testing.T) *kubermaticv1.CloudSpec {
	if prov == provider.AWSCloudProvider {
		return awsClusterCloudSpec.DeepCopy()
	}

	t.Fatalf("No cluster cloud spec found for provider %s", prov)
	return nil
}

func getNodeCloudSpec(prov string, t *testing.T) *kubermaticapiv1.NodeCloudSpec {
	if prov == provider.AWSCloudProvider {
		// Create a local copy
		nspec := awsNodeCloudSpec
		return &nspec
	}

	t.Fatalf("No node cloud spec found for provider %s", prov)
	return nil
}

func getNodeOSSpec(os string, t *testing.T) *kubermaticapiv1.OperatingSystemSpec {
	if os == "coreos" {
		return &kubermaticapiv1.OperatingSystemSpec{
			ContainerLinux: &kubermaticapiv1.ContainerLinuxSpec{
				// Otherwise the nodes restart directly after creation - bad for tests
				DisableAutoUpdate: true,
			},
		}
	}
	if os == "ubuntu" {
		return &kubermaticapiv1.OperatingSystemSpec{
			Ubuntu: &kubermaticapiv1.UbuntuSpec{},
		}
	}
	if os == "centos" {
		return &kubermaticapiv1.OperatingSystemSpec{
			CentOS: &kubermaticapiv1.CentOSSpec{},
		}
	}

	t.Fatalf("No node os spec found for os %s", os)
	return nil
}

func getCluster(prov string, version *semver.Semver, workerName string, t *testing.T) *kubermaticv1.Cluster {
	cloudSpec := getClusterCloudSpec(prov, t)
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "e2e-test-",
			Labels:       map[string]string{},
			Finalizers: []string{
				clustercontroller.InClusterPVCleanupFinalizer,
				clustercontroller.InClusterLBCleanupFinalizer,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Version:           *version,
			HumanReadableName: t.Name(),
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"10.10.10.0/24"},
				},
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"172.25.0.0/16"},
				},
				DNSDomain: "cluster.local",
			},
			Cloud: *cloudSpec,
		},
	}

	if workerName != "" {
		cluster.Labels[kubermaticv1.WorkerNameLabelKey] = workerName
	}

	var replicas int32 = 2
	cluster.Spec.ComponentsOverride.Apiserver.Replicas = &replicas
	cluster.Spec.ComponentsOverride.ControllerManager.Replicas = &replicas
	cluster.Spec.ComponentsOverride.Scheduler.Replicas = &replicas

	return cluster
}

func getNode(prov, os string, version *semver.Semver, t *testing.T) *kubermaticapiv1.Node {
	cloudSpec := getNodeCloudSpec(prov, t)
	osSpec := getNodeOSSpec(os, t)
	return &kubermaticapiv1.Node{
		ObjectMeta: kubermaticapiv1.ObjectMeta{},
		Spec: kubermaticapiv1.NodeSpec{
			Cloud: *cloudSpec,
			Versions: kubermaticapiv1.NodeVersionInfo{
				Kubelet: version.String(),
			},
			OperatingSystem: *osSpec,
		},
	}
}

func supportsStorage(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Openstack != nil ||
		cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil ||
		cluster.Spec.Cloud.VSphere != nil
}

func supportsLBs(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil
}
