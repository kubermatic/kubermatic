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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
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
	testUserCluster(t, userconfig)
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
			t.Parallel()
			mu.Lock()
			uc, _, cleanup, err := createUsercluster(t, proxyMode)
			mu.Unlock()

			if err != nil {
				t.Fatalf("failed to create user cluster: %v", err)
			}

			defer func() {
				mu.Lock()
				cleanup()
				mu.Unlock()
			}()

			testUserCluster(t, uc)
		})
	}
}

//gocyclo:ignore
func testUserCluster(t *testing.T, userconfig string) {
	config, err := clientcmd.BuildConfigFromFlags("", userconfig)
	if err != nil {
		t.Fatalf("failed to build config: %s", err)
	}

	userClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	var nodeIP string

	t.Logf("waiting for nodes to come up")
	{
		expectedNodes := 2
		err := wait.Poll(30*time.Second, 10*time.Minute, func() (bool, error) {
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
		if err != nil {
			t.Fatalf("nodes never became ready: %v", err)
		}
	}

	t.Logf("waiting for pods to get ready")
	{
		err := wait.Poll(30*time.Second, 30*time.Minute, func() (bool, error) {
			t.Logf("checking pod readiness...")

			pods, err := userClient.CoreV1().Pods("kube-system").List(
				context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Logf("failed to get pod list: %s", err)
				return false, nil
			}

			names := []string{
				"cilium-operator",
				"cilium",
			}

			pods.Items = filterByLabel(pods.Items, "k8s-app", names...)

			if len(pods.Items) == 0 {
				t.Logf("no cilium pods found")
				return false, nil
			}

			allRunning := true
			for _, pod := range pods.Items {
				if pod.Status.Phase != corev1.PodRunning {
					allRunning = false
				}
				t.Log(pod.Name, pod.Status.Phase)
			}

			if !allRunning {
				t.Logf("not all pods running yet...")
				return false, nil
			}

			return true, nil
		})
		if err != nil {
			t.Fatalf("pods never became ready: %v", err)
		}
	}

	t.Logf("run Cilium connectivity tests")
	{
		_, err := userClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ciliumTestNs,
			},
		}, metav1.CreateOptions{})
		defer func() {
			err := userClient.CoreV1().Namespaces().Delete(context.Background(), ciliumTestNs,
				metav1.DeleteOptions{})
			if err != nil {
				t.Fatalf("failed to create %q namespace: %v", ciliumTestNs, err)
			}
		}()
		if err != nil {
			t.Fatalf("failed to create %q namespace: %v", ciliumTestNs, err)
		}

		t.Logf("namespace %q created", ciliumTestNs)

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

	t.Logf("deploy hubble-relay-nodeport and hubble-ui-nodeport services")
	{
		s := makeScheme()

		objs1, err := resourcesFromYaml("./testdata/hubble-relay-svc.yaml", s)
		if err != nil {
			t.Fatalf("failed to read objects from yaml: %v", err)
		}

		objs2, err := resourcesFromYaml("./testdata/hubble-ui-svc.yaml", s)
		if err != nil {
			t.Fatalf("failed to read objects from yaml: %v", err)
		}

		for _, o := range append(objs1, objs2...) {
			x := o.(*corev1.Service)
			_, err := userClient.CoreV1().Services("kube-system").Create(context.Background(), x,
				metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to apply resource: %v", err)
			}
			t.Logf("created %v", x.Name)
		}
	}

	t.Logf("waiting for Cilium connectivity pods to get ready")
	{
		err := wait.Poll(30*time.Second, 30*time.Minute, func() (bool, error) {
			t.Logf("checking pod readiness...")

			names := []string{
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
			}

			pods, err := userClient.CoreV1().Pods(ciliumTestNs).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Logf("failed to get pod list: %s", err)
				return false, nil
			}

			pods.Items = filterByLabel(pods.Items, "name", names...)

			if len(pods.Items) == 0 {
				t.Logf("no connectivity test pods found")
				return false, nil
			}

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

			if !allHealthy {
				t.Logf("not all pods healthy yet...")
				return false, nil
			}

			t.Logf("all pods healthy")

			return true, nil
		})
		if err != nil {
			t.Fatalf("pods never became ready: %v", err)
		}
	}

	t.Logf("checking for Hubble pods")
	{
		err := wait.Poll(30*time.Second, 30*time.Minute, func() (bool, error) {
			t.Logf("checking pod readiness...")

			names := []string{
				"hubble-relay",
				"hubble-ui",
			}

			pods, err := userClient.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Logf("failed to get pod list: %s", err)
				return false, nil
			}

			pods.Items = filterByLabel(pods.Items, "k8s-app", names...)

			if len(pods.Items) == 0 {
				t.Logf("no hubble pods found")
				return false, nil
			}

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

			if !allHealthy {
				t.Logf("not all pods healthy yet...")
				return false, nil
			}

			t.Logf("all hubble pods healthy")

			return true, nil
		})
		if err != nil {
			t.Fatalf("pods never became ready: %v", err)
		}
	}

	t.Logf("test hubble relay observe")
	{
		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", nodeIP, 30077), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("failed to dial to hubble relay: %v", err)
		}
		defer conn.Close()

		nFlows := 20
		flowsClient, err := observer.NewObserverClient(conn).
			GetFlows(context.Background(), &observer.GetFlowsRequest{Number: uint64(nFlows)})
		if err != nil {
			t.Fatalf("failed to get flow client:%v", err)
		}

		for c := 0; c < nFlows; c++ {
			flow, err := flowsClient.Recv()
			if err != nil {
				t.Fatalf("failed to get flow:%v", err)
			}
			fmt.Println(flow)
		}
	}

	t.Logf("test hubble ui observe")
	{
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
			fmt.Sprintf("http://%s:%d", nodeIP, 30007), nil)
		if err != nil {
			t.Fatalf("failed to construct request to hubble ui: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to get response from hubble ui: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected: 200 OK, got: %d", resp.StatusCode)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body:%v", err)
		}

		if !strings.Contains(string(body), "Hubble") {
			t.Fatalf("failed to find Hubble in the body")
		}
	}
}

func filterByLabel(pods []corev1.Pod, key string, values ...string) []corev1.Pod {
	c := 0
	for _, p := range pods {
		for _, v := range values {
			vv, ok := p.Labels[key]
			if ok && vv == v {
				c++
				pods = append(pods, p)
			}
		}
	}
	return pods[len(pods)-c:]
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
func createUsercluster(t *testing.T, proxyMode string) (string, string, func(), error) {
	var teardowns []func()
	cleanup := func() {
		n := len(teardowns)
		for i := range teardowns {
			teardowns[n-1-i]()
		}
	} // get kubermatic-api client
	token, err := utils.RetrieveMasterToken(context.Background())
	if err != nil {
		return "", "", nil, err
	}

	apicli := utils.NewTestClient(token, t)

	// create a project
	project, err := apicli.CreateProject(projectName + "-" + proxyMode + "-" + rand.String(10))
	if err != nil {
		return "", "", nil, err
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
		return "", "", nil, err
	}
	teardowns = append(teardowns, func() {
		err := apicli.DeleteCluster(project.ID, seed, cluster.ID) // TODO: this succeeds but cluster is not actually gone why?
		if err != nil {
			t.Errorf("failed to delete cluster %s/%s: %s", project.ID, cluster.ID, err)
		}
	})

	// try to get kubeconfig
	var data string
	err = wait.Poll(30*time.Second, 10*time.Minute, func() (bool, error) {
		t.Logf("trying to get kubeconfig...")
		// construct clients
		data, err = apicli.GetKubeconfig(seed, project.ID, cluster.ID)
		if err != nil {
			t.Logf("error trying to get kubeconfig: %s", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", "", nil, err
	}

	file, err := ioutil.TempFile("/tmp", "kubeconfig-")
	if err != nil {
		return "", "", nil, err
	}

	err = os.WriteFile(file.Name(), []byte(data), 0664)
	if err != nil {
		return "", "", nil, err
	}
	teardowns = append(teardowns, func() {
		err = os.Remove(file.Name())
		if err != nil {
			t.Errorf("failed to delete kubeconfig %s: %s", file.Name(), err)
		}
	})

	return file.Name(), cluster.ID, cleanup, nil
}
