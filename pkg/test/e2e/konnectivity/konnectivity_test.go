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

package konnectivity_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"
	
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/cmd/cp"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	"k8s.io/utils/pointer"
)

var (
	userconfig      string
	seedconfig      string
	accessKeyID     string
	secretAccessKey string
)

const (
	tailLines       = 5
	seed            = "kubermatic"
	projectName     = "konne-test-project"
	userclusterName = "konne-test-usercluster"
	alpineSleeper   = "alpine-sleeper"
)

func init() {
	flag.StringVar(&seedconfig, "seedconfig", "", "path to kubeconfig of seedcluster")
	flag.StringVar(&userconfig, "userconfig", "", "path to kubeconfig of usercluster")
	
	if userconfig != "" {
		log.Println("running agaist ready usercluster")
		return
	}
	
	accessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKeyID == "" {
		panic("AWS_ACCESS_KEY_ID not set")
	}
	
	secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		panic("AWS_SECRET_ACCESS_KEY not set")
	}
}

func TestKonnectivity(t *testing.T) {
	var cleanup func()
	var err error
	var clusterID string
	if userconfig == "" {
		userconfig, clusterID, cleanup, err = createUsercluster(t)
		defer cleanup()
		if err != nil {
			t.Fatalf("failed to setup usercluster: %s", err)
		}
		
	} else {
		config, err := clientcmd.LoadFromFile(userconfig)
		if err != nil {
			t.Fatalf("failed to parse seedconfig: %s", err)
		}
		
		clusterID = config.Contexts[config.CurrentContext].Cluster
	}
	
	config, err := clientcmd.BuildConfigFromFlags("", userconfig)
	if err != nil {
		t.Fatalf("failed to build config: %s", err)
	}
	
	userClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	
	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		t.Fatalf("failed to create metrics client: %s", err)
	}
	
	ctx := context.Background()
	
	t.Logf("waiting for nodes to come up")
	{
		err := wait.Poll(30*time.Second, 10*time.Minute, func() (bool, error) {
			nodes, err := userClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Logf("failed to get nodes list: %s", err)
				return false, nil
			}
			if len(nodes.Items) != 1 {
				t.Logf(fmt.Sprintf("node count: %d", len(nodes.Items)))
				return false, nil
			}
			
			for _, c := range nodes.Items[0].Status.Conditions {
				if c.Type == corev1.NodeReady {
					t.Logf("node is ready")
					return true, nil
				}
			}
			t.Logf("no nodes are ready")
			return false, nil
		})
		if err != nil {
			t.Fatalf("nodes never became ready: %v", err)
		}
	}
	
	t.Logf("checking if apiserver has konnectivity-proxy in sidecar")
	{
		config, err := clientcmd.BuildConfigFromFlags("", seedconfig)
		if err != nil {
			t.Fatal("failed to build config", err)
		}
		
		seedClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			t.Fatal("failed to build kubeclient", err)
		}
		
		ns := "cluster-" + clusterID
		
		pods, err := seedClient.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Println("failed to list pods", ns, err)
		}
		
		var apiserverPod corev1.Pod
		for _, p := range pods.Items {
			if strings.HasPrefix(p.Name, "apiserver") {
				apiserverPod = p
				break
			}
		}
		
		found := false
		for _, c := range apiserverPod.Spec.Containers {
			if c.Name == resources.KonnectivityServerContainer {
				found = true
				break
			}
		}
		
		if !found {
			t.Fatalf("no konnectivity-proxy container was found in apiserver sidecar")
		}
	}
	
	t.Logf("creating alpine-sleeper pod for testing cp")
	{
		_, err = userClient.CoreV1().Pods("kube-system").Create(context.Background(), &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alpineSleeper,
				Namespace: "kube-system",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "alpine:latest",
						Name:  alpineSleeper,
						Command: []string{
							"/bin/sh", "-c", "--",
						},
						Args: []string{
							"sleep 1001d",
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("failed to create alpine-sleeper pod")
		}
		
		defer func() {
			t.Logf("deleting alpine-sleeper pod")
			err := userClient.CoreV1().Pods("kube-system").Delete(context.Background(), alpineSleeper, metav1.DeleteOptions{})
			if err != nil {
				t.Fatalf("failed to delete alpine-sleeper pod")
			}
		}()
	}
	
	t.Logf("waiting for pods to get ready")
	{
		err := wait.Poll(30*time.Second, 30*time.Minute, func() (bool, error) {
			t.Logf("checking pod readiness...")
			
			for _, prefix := range []string{
				"konnectivity-agent",
				"metrics-server",
				alpineSleeper,
			} {
				pods, err := getPods(ctx, userClient, prefix)
				if err != nil {
					t.Logf("failed to get pod list: %s", err)
					return false, nil
				}
				
				if len(pods) == 0 {
					t.Logf("no %s pods found", prefix)
					return false, nil
				}
				
				allRunning := true
				for _, pod := range pods {
					if pod.Status.Phase != corev1.PodRunning {
						allRunning = false
					}
					t.Log(pod.Name, pod.Status.Phase)
				}
				
				if !allRunning {
					t.Logf("not all pods running yet...")
					return false, nil
				}
			}
			
			return true, nil
		})
		if err != nil {
			t.Fatalf("pods never became ready: %v", err)
		}
	}
	
	t.Log("check if konnectivity-agents are deployed")
	{
		pods, err := getPods(ctx, userClient, "konnectivity-agent")
		if err != nil {
			t.Errorf("failed to get konnectivity-agent pods: %s", err)
		}
		
		if len(pods) != 2 {
			t.Errorf("expected 2 konnectivity-agent pods got: %d", len(pods))
		}
	}
	
	t.Log("check if we can get logs from pods")
	{
		pods, err := getPods(ctx, userClient, "metrics-server")
		if err != nil {
			t.Errorf("failed to get metrics-server pods: %s", err)
		}
		
		if len(pods) != 2 {
			t.Errorf("expected 2 metrics-server pods got: %d", len(pods))
		}
		
		for _, pod := range pods {
			lines := strings.TrimSpace(getPodLogs(ctx, userClient, pod))
			if n := len(strings.Split(lines, "\n")); n != tailLines {
				t.Fatalf("expected 5 lines got: %d", n)
			}
		}
	}
	
	t.Log("check if it is possible to copy")
	{
		pods, err := getPods(ctx, userClient, alpineSleeper)
		if err != nil {
			t.Fatalf("failed to get alpine-sleeper pods: %s", err)
		}
		
		if len(pods) == 0 {
			t.Fatalf("no envoy-agent pods")
		}
		
		podExec := NewPodExec(*config, userClient)
		err = podExec.PodCopyFile(
			"./testdata/copyMe.txt",
			fmt.Sprintf("%s/%s:/", "kube-system", pods[0].Name),
			alpineSleeper,
		)
		
		if err != nil {
			t.Fatalf("failed to copy: %s", err)
		}
	}
	
	t.Log("check if it is possible to get metrics")
	{
		// TODO: check if metrics have sane values.
		nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to get node metrics: %s", err)
		}
		
		if len(nodeMetrics.Items) == 0 {
			t.Fatalf("no node metrics")
		}
		
		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Error("failed to get pod metrics: ", err)
		}
		
		if len(podMetrics.Items) == 0 {
			t.Fatalf("no podmetrics")
		}
	}
}

func getPods(ctx context.Context, kubeclient *kubernetes.Clientset, prefix string) ([]corev1.Pod, error) {
	pods, err := kubeclient.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	
	var matchingPods []corev1.Pod
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, prefix) {
			matchingPods = append(matchingPods, pod)
		}
	}
	return matchingPods, nil
}

type PodExec struct {
	RestConfig *rest.Config
	*kubernetes.Clientset
}

func NewPodExec(config rest.Config, clientset *kubernetes.Clientset) *PodExec {
	config.APIPath = "/api"                                   // Make sure we target /api and not just /
	config.GroupVersion = &schema.GroupVersion{Version: "v1"} // this targets the core api groups so the url path will be /api/v1
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	return &PodExec{
		RestConfig: &config,
		Clientset:  clientset,
	}
}

func (p *PodExec) PodCopyFile(src string, dst string, containername string) error {
	ioStreams := genericclioptions.NewTestIOStreamsDiscard()
	copyOptions := cp.NewCopyOptions(ioStreams)
	copyOptions.Clientset = p.Clientset
	copyOptions.ClientConfig = p.RestConfig
	copyOptions.Container = containername
	copyOptions.NoPreserve = true
	err := copyOptions.Run([]string{src, dst})
	if err != nil {
		return fmt.Errorf("Could not run copy operation: %v", err)
	}
	return nil
}

// gets logs from pod
func getPodLogs(ctx context.Context, cli *kubernetes.Clientset, pod corev1.Pod) string {
	podLogOpts := corev1.PodLogOptions{
		TailLines: pointer.Int64(tailLines),
	}
	
	req := cli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "error in opening stream"
	}
	defer podLogs.Close()
	
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "error in copy information from podLogs to buf"
	}
	str := buf.String()
	
	return str
}

// creates a usercluster on aws
func createUsercluster(t *testing.T) (string, string, func(), error) {
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
	project, err := apicli.CreateProject(projectName)
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
	cluster, err := apicli.CreateAWSCluster(project.ID, seed, userclusterName,
		secretAccessKey, accessKeyID, utils.KubernetesVersion(),
		"aws-eu-central-1a", "eu-central-1a", 1, true)
	if err != nil {
		return "", "", nil, err
	}
	teardowns = append(teardowns, func() {
		err := apicli.DeleteCluster(project.ID, seed, cluster.ID) //TODO: this succeeds but cluster is not actually gone why?
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
