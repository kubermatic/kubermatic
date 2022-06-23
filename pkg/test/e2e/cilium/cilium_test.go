//--- go:build e2e

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

package cilium_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cilium/cilium/api/v1/observer"
	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"google.golang.org/grpc"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"google.golang.org/grpc/credentials/insecure"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	userconfig      string
	accessKeyID     string
	secretAccessKey string
)

const (
	seed            = "kubermatic"
	projectName     = "cilium-test-project"
	userclusterName = "cilium-test-usercluster"
	ciliumTestNs    = "cilium-test"
)

func init() {
	flag.StringVar(&userconfig, "userconfig", "", "path to kubeconfig of usercluster")
}

func TestReadyCluster(t *testing.T) {
	if userconfig == "" {
		t.Logf("kubeconfig for usercluster not provided, test passes vacuously.")
		t.Logf("to run against ready usercluster use following command")
		t.Logf("go test ./pkg/test/e2e/cilium -v -race -tags e2e -timeout 30m -run TestReadyCluster -args --userconfig <USERCLUSTER KUBECONFIG>")
		return
	}

	config, err := clientcmd.BuildConfigFromFlags("", userconfig)
	if err != nil {
		t.Fatalf("failed to build config: %v", err)
	}

	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("failed to build ctrlruntime client: %v", err)
	}

	testUserCluster(context.Background(), t, client)
}

func TestCiliumClusters(t *testing.T) {
	accessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKeyID == "" {
		t.Fatalf("AWS_ACCESS_KEY_ID not set")
	}

	secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		t.Fatalf("AWS_SECRET_ACCESS_KEY not set")
	}

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Fatalf("failed to build config: %v", err)
	}

	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("failed to build ctrlruntime client: %v", err)
	}

	tests := []struct {
		name      string
		proxyMode string
	}{
		{
			name:      "ebpf proxy mode test",
			proxyMode: resources.EBPFProxyMode,
		},
		{
			name:      "ipvs proxy mode test",
			proxyMode: resources.IPVSProxyMode,
		},
		{
			name:      "iptables proxy mode test",
			proxyMode: resources.IPTablesProxyMode,
		},
	}

	for _, test := range tests {
		proxyMode := test.proxyMode
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			client, cleanup, err := createUsercluster(ctx, t, client, proxyMode)
			defer cleanup()

			if err != nil {
				t.Fatalf("failed to create user cluster: %v", err)
			}

			testUserCluster(ctx, t, client)
		})
	}
}

//gocyclo:ignore
func testUserCluster(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client) {
	t.Log("waiting for nodes to come up")
	_, err := checkNodeReadiness(ctx, t, client)
	if err != nil {
		t.Fatalf("nodes never became ready: %v", err)
	}

	t.Log("waiting for pods to get ready")
	err = waitForPods(ctx, t, client, "kube-system", "k8s-app", []string{
		"cilium-operator",
		"cilium",
	})
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	t.Log("run Cilium connectivity tests")
	ns := corev1.Namespace{}
	ns.Name = ciliumTestNs
	err = client.Create(ctx, &ns)
	if err != nil {
		t.Fatalf("failed to create %q namespace: %v", ciliumTestNs, err)
	}
	defer func() {
		err := client.Delete(ctx, &ns)
		if err != nil {
			t.Fatalf("failed to delete %q namespace: %v", ciliumTestNs, err)
		}
	}()

	t.Logf("namespace %q created", ciliumTestNs)

	installCiliumConnectivityTests(ctx, t, client)

	t.Logf("deploy hubble-relay-nodeport and hubble-ui-nodeport services")
	cleanup := deployHubbleServices(ctx, t, client)
	defer cleanup()

	t.Logf("waiting for Cilium connectivity pods to get ready")
	err = waitForPods(ctx, t, client, ciliumTestNs, "name", []string{
		"echo-a",
		"echo-b",
		"echo-b-headless",
		"echo-b-host",
		"echo-b-host-headless",
		"host-to-b-multi-node-clusterip",
		"host-to-b-multi-node-headless",
		"pod-to-a",
		"pod-to-a-allowed-cnp",
		"pod-to-a-denied-cnp",
		"pod-to-b-intra-node-nodeport",
		"pod-to-b-multi-node-clusterip",
		"pod-to-b-multi-node-headless",
		"pod-to-b-multi-node-nodeport",
		"pod-to-external-1111",
		"pod-to-external-fqdn-allow-google-cnp",
	})
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	t.Logf("checking for Hubble pods")
	err = waitForPods(ctx, t, client, "kube-system", "k8s-app", []string{
		"hubble-relay",
		"hubble-ui",
	})
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	t.Logf("test hubble relay observe")
	err = wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		nodeIP, err := checkNodeReadiness(ctx, t, client)
		if err != nil {
			t.Logf("nodes never became ready: %v", err)
			return false, nil
		}

		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", nodeIP, 30077), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Logf("failed to dial to hubble relay: %v", err)
			return false, nil
		}
		defer conn.Close()

		nFlows := 20
		flowsClient, err := observer.NewObserverClient(conn).
			GetFlows(context.Background(), &observer.GetFlowsRequest{Number: uint64(nFlows)})
		if err != nil {
			t.Logf("failed to get flow client:%v", err)
			return false, nil
		}

		for c := 0; c < nFlows; c++ {
			flow, err := flowsClient.Recv()
			if err != nil {
				t.Logf("failed to get flow:%v", err)
				return false, nil
			}
			fmt.Println(flow)
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("hubble relay observe test failed: %v", err)
	}

	t.Logf("test hubble ui observe")
	err = wait.Poll(30*time.Second, 5*time.Minute, func() (bool, error) {
		nodeIP, err := checkNodeReadiness(ctx, t, client)
		if err != nil {
			t.Logf("nodes never became ready: %v", err)
			return false, nil
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
			fmt.Sprintf("http://%s", net.JoinHostPort(nodeIP, "30007")), nil)
		if err != nil {
			t.Logf("failed to construct request to hubble ui: %v", err)
			return false, nil
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Logf("failed to get response from hubble ui: %v", err)
			return false, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Logf("expected: 200 OK, got: %d", resp.StatusCode)
			return false, nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Logf("failed to read response body:%v", err)
			return false, nil
		}

		if !strings.Contains(string(body), "Hubble") {
			t.Logf("failed to find Hubble in the body")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("hubble ui observe test failed: %v", err)
	}
}

func waitForPods(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, key string, names []string) error {
	t.Log("checking pod readiness...", namespace, key, names)

	return wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		r, err := labels.NewRequirement(key, selection.In, names)
		if err != nil {
			t.Logf("failed to build requirement: %v", err)
			return false, nil
		}

		l := labels.NewSelector().Add(*r)
		pods := corev1.PodList{}
		err = client.List(ctx, &pods, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabelsSelector{Selector: l})
		if err != nil {
			t.Logf("failed to get pod list: %v", err)
			return false, nil
		}

		if len(pods.Items) == 0 {
			t.Logf("no pods found")
			return false, nil
		}

		if !allPodsHealthy(t, &pods) {
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

func deployHubbleServices(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client) func() {
	s := makeScheme()

	hubbleRelaySvc, err := resourcesFromYaml("./testdata/hubble-relay-svc.yaml", s)
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}

	hubbleUISvc, err := resourcesFromYaml("./testdata/hubble-ui-svc.yaml", s)
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}

	var cleanups []func()

	objects := []ctrlruntimeclient.Object{}
	objects = append(objects, hubbleRelaySvc...)
	objects = append(objects, hubbleUISvc...)

	for i, object := range objects {
		err := client.Create(ctx, object)
		if err != nil {
			t.Fatalf("failed to apply resource: %v", err)
		}

		cleanups = append(cleanups, func() {
			err := client.Delete(ctx, objects[i])
			if err != nil {
				t.Logf("failed to delete resource: %v", err)
			}
		})

		t.Logf("created %v", object.GetName())
	}

	return func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
}

func installCiliumConnectivityTests(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client) {
	s := makeScheme()

	objs, err := resourcesFromYaml("./testdata/connectivity-check.yaml", s)
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}

	for _, obj := range objs {
		if err := client.Create(ctx, obj); err != nil {
			t.Fatalf("failed to apply resource: %v", err)
		}
		t.Logf("created %v", obj.GetName())

		// switch x := obj.(type) {
		// case *appsv1.Deployment:
		// 	_, err := userClient.AppsV1().Deployments(ciliumTestNs).Create(ctx, x,
		// 		metav1.CreateOptions{})
		// 	if err != nil {
		// 		t.Fatalf("failed to apply resource: %v", err)
		// 	}
		// 	t.Logf("created %v", x.Name)

		// case *corev1.Service:
		// 	_, err := userClient.CoreV1().Services(ciliumTestNs).Create(ctx, x,
		// 		metav1.CreateOptions{})
		// 	if err != nil {
		// 		t.Fatalf("failed to apply resource: %v", err)
		// 	}
		// 	t.Logf("created %v", x.Name)

		// case *ciliumv2.CiliumNetworkPolicy:
		// 	crdConfig := *config
		// 	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{
		// 		Group:   ciliumv2.CustomResourceDefinitionGroup,
		// 		Version: ciliumv2.CustomResourceDefinitionVersion,
		// 	}
		// 	crdConfig.APIPath = "/apis"
		// 	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(s)
		// 	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()
		// 	cs, err := ciliumclientset.NewForConfig(&crdConfig)
		// 	if err != nil {
		// 		t.Fatalf("failed to get clientset for config: %v", err)
		// 	}
		// 	_, err = cs.CiliumV2().CiliumNetworkPolicies(ciliumTestNs).Create(
		// 		ctx, x, metav1.CreateOptions{})
		// 	if err != nil {
		// 		t.Fatalf("failed to create cilium network policy: %v", err)
		// 	}
		// 	t.Logf("created %v", x.Name)

		// default:
		// 	t.Fatalf("unknown resource type: %v", obj.GetObjectKind())
		// }
	}
}

func checkNodeReadiness(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client) (string, error) {
	expectedNodes := 2
	var nodeIP string

	err := wait.Poll(2*time.Second, 15*time.Minute, func() (bool, error) {
		nodeList := corev1.NodeList{}
		err := client.List(ctx, &nodeList)
		if err != nil {
			t.Logf("failed to get nodes list: %v", err)
			return false, nil
		}

		if len(nodeList.Items) != expectedNodes {
			t.Logf("node count: %d, expected: %d", len(nodeList.Items), expectedNodes)
			return false, nil
		}

		readyNodeCount := 0

		for _, node := range nodeList.Items {
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

		for _, addr := range nodeList.Items[0].Status.Addresses {
			if addr.Type == corev1.NodeExternalIP {
				nodeIP = addr.Address
				break
			}
		}

		return true, nil
	})

	return nodeIP, err
}

func makeScheme() *runtime.Scheme {
	var s = runtime.NewScheme()
	_ = serializer.NewCodecFactory(s)
	_ = runtime.NewParameterCodec(s)
	utilruntime.Must(appsv1.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(ciliumv2.AddToScheme(s))
	return s
}

func resourcesFromYaml(filename string, s *runtime.Scheme) ([]ctrlruntimeclient.Object, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	manifests, err := yamlutil.ParseMultipleDocuments(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	sr := kjson.NewSerializerWithOptions(&kjson.SimpleMetaFactory{}, s, s, kjson.SerializerOptions{})

	var objs []ctrlruntimeclient.Object
	for _, m := range manifests {
		obj, err := runtime.Decode(sr, m.Raw)
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj.(ctrlruntimeclient.Object))
	}

	return objs, nil
}

// creates a usercluster on aws.
func createUsercluster(ctx context.Context, t *testing.T, masterClient ctrlruntimeclient.Client, proxyMode string) (ctrlruntimeclient.Client, func(), error) {
	var teardowns []func()
	cleanup := func() {
		n := len(teardowns)
		for i := range teardowns {
			teardowns[n-1-i]()
		}
	}

	// prepare helpers
	projectProvider, _ := kubernetes.NewProjectProvider(nil, masterClient)
	clusterProvider := kubernetes.NewClusterProvider(
		nil,
		nil,
		nil,
		"",
		nil,
		masterClient,
		nil,
		false,
		kubermatic.Versions{},
		nil,
	)

	project, err := projectProvider.New(ctx, projectName, nil)
	if err != nil {
		return nil, nil, err
	}
	teardowns = append(teardowns, func() {
		if err := masterClient.Delete(ctx, project); err != nil {
			t.Errorf("failed to delete project: %v", err)
		}
	})

	version := utils.KubernetesVersion()

	// create a usercluster on AWS
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cilium-e2e-",
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: project.Name,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: "Cilium e2e test cluster",
			Version:           *semver.NewSemverOrDie(version),
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "aws-eu-central-1a",
				AWS: &kubermaticv1.AWSCloudSpec{
					SecretAccessKey: secretAccessKey,
					AccessKeyID:     accessKeyID,
				},
			},
			CNIPlugin: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "v1.11",
			},
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				ProxyMode:           proxyMode,
				KonnectivityEnabled: pointer.Bool(true),
			},
		},
	}

	cluster, err = clusterProvider.NewUnsecured(ctx, project, cluster, "cilium@e2e.test")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// wait for cluster to be up and running
	err = wait.Poll(2*time.Second, 2*time.Minute, func() (bool, error) {
		curCluster := kubermaticv1.Cluster{}
		if err := masterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), &curCluster); err != nil {
			return false, fmt.Errorf("failed to retrieve cluster: %w", err)
		}

		return curCluster.Status.ExtendedHealth.AllHealthy(), nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("cluster did not become healthy: %w", err)
	}

	// retrieve usercluster kubeconfig
	clusterClient, err := clusterProvider.GetAdminClientForCustomerCluster(ctx, cluster)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create usercluster client: %w", err)
	}

	// prepare MachineDeployment
	encodedOSSpec, err := json.Marshal(ubuntu.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode osspec: %w", err)
	}

	encodedCloudProviderSpec, err := json.Marshal(awstypes.RawConfig{
		InstanceType:     providerconfig.ConfigVarString{Value: "t3.small"},
		DiskType:         providerconfig.ConfigVarString{Value: "standard"},
		DiskSize:         int64(25),
		AvailabilityZone: providerconfig.ConfigVarString{Value: "eu-central-1a"},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode providerspec: %w", err)
	}

	cfg := providerconfig.Config{
		CloudProvider: providerconfig.CloudProviderAWS,
		CloudProviderSpec: runtime.RawExtension{
			Raw: encodedCloudProviderSpec,
		},
		OperatingSystem: providerconfig.OperatingSystemUbuntu,
		OperatingSystemSpec: runtime.RawExtension{
			Raw: encodedOSSpec,
		},
	}

	encodedConfig, err := json.Marshal(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode providerconfig: %w", err)
	}

	md := clusterv1alpha1.MachineDeployment{
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Replicas: pointer.Int32(2),
			Template: clusterv1alpha1.MachineTemplateSpec{
				Spec: clusterv1alpha1.MachineSpec{
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: version,
					},
					ProviderSpec: clusterv1alpha1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: encodedConfig,
						},
					},
				},
			},
		},
	}

	// create MachineDeployment
	if err := clusterClient.Create(ctx, &md); err != nil {
		return nil, nil, fmt.Errorf("failed to create MachineDeployment: %w", err)
	}
	teardowns = append(teardowns, func() {
		if err := masterClient.Delete(ctx, cluster); err != nil {
			t.Errorf("failed to delete cluster: %v", err)
		}
	})

	return clusterClient, cleanup, nil
}
