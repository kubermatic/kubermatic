//go:build e2e

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
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cilium/cilium/api/v1/observer"
	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	ciliumclientset "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"google.golang.org/grpc"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
		t.Fatalf("failed to build config: %s", err)
	}

	testUserCluster(t, config)
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

	var mu sync.Mutex

	for _, test := range tests {
		proxyMode := test.proxyMode
		t.Run(test.name, func(t *testing.T) {
			// t.Parallel()
			mu.Lock()
			config, _, cleanup, err := createUsercluster(t, proxyMode)
			mu.Unlock()

			if err != nil {
				t.Fatalf("failed to create user cluster: %v", err)
			}

			defer func() {
				mu.Lock()
				cleanup()
				mu.Unlock()
			}()

			testUserCluster(t, config)
		})
	}
}

//gocyclo:ignore
func testUserCluster(t *testing.T, config *rest.Config) {
	userClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	t.Logf("waiting for nodes to come up")
	_, err = checkNodeReadiness(t, userClient)
	if err != nil {
		t.Fatalf("nodes never became ready: %v", err)
	}

	t.Logf("waiting for pods to get ready")
	err = waitForPods(t, userClient, "kube-system", "k8s-app", []string{
		"cilium-operator",
		"cilium",
	})
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	t.Logf("run Cilium connectivity tests")

	_, err = userClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ciliumTestNs}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create %q namespace: %v", ciliumTestNs, err)
	}
	defer func() {
		err := userClient.CoreV1().Namespaces().Delete(context.Background(), ciliumTestNs, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("failed to create %q namespace: %v", ciliumTestNs, err)
		}
	}()

	t.Logf("namespace %q created", ciliumTestNs)

	runCiliumConnectivityTests(t, userClient, config)

	t.Logf("deploy hubble-relay-nodeport and hubble-ui-nodeport services")
	cleanup := deployHubbleServices(t, userClient)
	defer cleanup()

	t.Logf("waiting for Cilium connectivity pods to get ready")
	err = waitForPods(t, userClient, ciliumTestNs, "name", []string{
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
	err = waitForPods(t, userClient, "kube-system", "k8s-app", []string{
		"hubble-relay",
		"hubble-ui",
	})
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	t.Logf("test hubble relay observe")
	err = wait.Poll(30*time.Second, 5*time.Minute, func() (bool, error) {
		nodeIP, err := checkNodeReadiness(t, userClient)
		if err != nil {
			t.Logf("nodes never became ready: %v", err)
			return false, nil
		}

		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", nodeIP, 30077), grpc.WithInsecure())
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
		nodeIP, err := checkNodeReadiness(t, userClient)
		if err != nil {
			t.Logf("nodes never became ready: %v", err)
			return false, nil
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
			fmt.Sprintf("http://%s:%d", nodeIP, 30007), nil)
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

		body, err := ioutil.ReadAll(resp.Body)
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

func waitForPods(t *testing.T, client *kubernetes.Clientset, namespace string, key string, names []string) error {
	t.Log("checking pod readiness...", namespace, key, names)

	return wait.Poll(30*time.Second, 5*time.Minute, func() (bool, error) {
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

func deployHubbleServices(t *testing.T, userClient *kubernetes.Clientset) func() {
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
	for _, o := range append(hubbleRelaySvc, hubbleUISvc...) {
		x := o.(*corev1.Service)
		_, err := userClient.CoreV1().Services("kube-system").Create(context.Background(), x, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("failed to apply resource: %v", err)
		}

		cleanups = append(cleanups, func() {
			err := userClient.CoreV1().Services("kube-system").Delete(context.Background(), x.Name, metav1.DeleteOptions{})
			if err != nil {
				t.Logf("failed to delete resource: %v", err)
			}
		})

		t.Logf("created %v", x.Name)
	}

	return func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
}

func runCiliumConnectivityTests(t *testing.T, userClient *kubernetes.Clientset, config *rest.Config) {
	s := makeScheme()

	objs, err := resourcesFromYaml("./testdata/connectivity-check.yaml", s)
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}

	for _, obj := range objs {
		switch x := obj.(type) {
		case *appsv1.Deployment:
			_, err := userClient.AppsV1().Deployments(ciliumTestNs).Create(context.Background(), x,
				metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to apply resource: %v", err)
			}
			t.Logf("created %v", x.Name)

		case *corev1.Service:
			_, err := userClient.CoreV1().Services(ciliumTestNs).Create(context.Background(), x,
				metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to apply resource: %v", err)
			}
			t.Logf("created %v", x.Name)

		case *ciliumv2.CiliumNetworkPolicy:
			crdConfig := *config
			crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{
				Group:   ciliumv2.CustomResourceDefinitionGroup,
				Version: ciliumv2.CustomResourceDefinitionVersion,
			}
			crdConfig.APIPath = "/apis"
			crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(s)
			crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()
			cs, err := ciliumclientset.NewForConfig(&crdConfig)
			if err != nil {
				t.Fatalf("failed to get clientset for config: %v", err)
			}
			_, err = cs.CiliumV2().CiliumNetworkPolicies(ciliumTestNs).Create(
				context.Background(), x, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create cilium network policy: %v", err)
			}
			t.Logf("created %v", x.Name)

		default:
			t.Fatalf("unknown resource type: %v", obj.GetObjectKind())
		}
	}
}

func checkNodeReadiness(t *testing.T, userClient *kubernetes.Clientset) (string, error) {
	expectedNodes := 2
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

func makeScheme() *runtime.Scheme {
	var s = runtime.NewScheme()
	_ = serializer.NewCodecFactory(s)
	_ = runtime.NewParameterCodec(s)
	utilruntime.Must(appsv1.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(ciliumv2.AddToScheme(s))
	return s
}

func resourcesFromYaml(filename string, s *runtime.Scheme) ([]runtime.Object, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	manifests, err := yamlutil.ParseMultipleDocuments(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	sr := kjson.NewSerializerWithOptions(&kjson.SimpleMetaFactory{}, s, s, kjson.SerializerOptions{})

	var objs []runtime.Object
	for _, m := range manifests {
		obj, err := runtime.Decode(sr, m.Raw)
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}

	return objs, nil
}

// creates a usercluster on aws.
func createUsercluster(t *testing.T, proxyMode string) (*rest.Config, string, func(), error) {
	var teardowns []func()
	cleanup := func() {
		n := len(teardowns)
		for i := range teardowns {
			teardowns[n-1-i]()
		}
	}

	// get kubermatic-api client
	token, err := utils.RetrieveMasterToken(context.Background())
	if err != nil {
		return nil, "", nil, err
	}

	apicli := utils.NewTestClient(token, t)

	// create a project
	project, err := apicli.CreateProject(projectName + "-" + proxyMode + "-" + rand.String(10))
	if err != nil {
		return nil, "", nil, err
	}
	teardowns = append(teardowns, func() {
		err := apicli.DeleteProject(project.ID)
		if err != nil {
			t.Errorf("failed to delete project %s: %s", project.ID, err)
		}
	})

	// create a usercluster on aws
	cluster, err := apicli.CreateAWSCluster(project.ID, seed, userclusterName+"-"+proxyMode,
		secretAccessKey, accessKeyID, utils.KubernetesVersion(),
		"aws-eu-central-1a", "eu-central-1a", proxyMode,
		2, true, &models.CNIPluginSettings{
			Version: "v1.11",
			Type:    "cilium",
		})
	if err != nil {
		return nil, "", nil, err
	}
	teardowns = append(teardowns, func() {
		err := apicli.DeleteCluster(project.ID, seed, cluster.ID) // TODO: this succeeds but cluster is not actually gone why?
		if err != nil {
			t.Errorf("failed to delete cluster %s/%s: %s", project.ID, cluster.ID, err)
		}
	})

	// try to get kubeconfig
	var userconfig string
	err = wait.Poll(30*time.Second, 30*time.Minute, func() (bool, error) {
		t.Logf("trying to get kubeconfig...")
		// construct clients
		userconfig, err = apicli.GetKubeconfig(seed, project.ID, cluster.ID)
		if err != nil {
			t.Logf("error trying to get kubeconfig: %s", err)
			return false, nil
		}

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
