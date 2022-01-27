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
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

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
	projectName     = "cilium-test-project"
	userclusterName = "cilium-test-usercluster"
	alpineSleeper   = "alpine-sleeper"
)

func init() {
	flag.StringVar(&seedconfig, "seedconfig", "", "path to kubeconfig of seedcluster")
	flag.StringVar(&userconfig, "userconfig", "", "path to kubeconfig of usercluster")
}

func TestCilium(t *testing.T) {
	var cleanup func()
	var err error
	var clusterID string
	if userconfig == "" {
		accessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
		if accessKeyID == "" {
			t.Fatalf("AWS_ACCESS_KEY_ID not set")
		}

		secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		if secretAccessKey == "" {
			t.Fatalf("AWS_SECRET_ACCESS_KEY not set")
		}

		userconfig, clusterID, cleanup, err = createUsercluster(t)
		defer cleanup()
		if err != nil {
			t.Fatalf("failed to setup usercluster: %s", err)
		}

	} else {
		t.Logf("running against ready usercluster")

		config, err := clientcmd.LoadFromFile(userconfig)
		if err != nil {
			t.Fatalf("failed to parse seedconfig: %s", err)
		}

		clusterID = config.Contexts[config.CurrentContext].Cluster
	}

	_ = clusterID

	config, err := clientcmd.BuildConfigFromFlags("", userconfig)
	if err != nil {
		t.Fatalf("failed to build config: %s", err)
	}

	userClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
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
			if len(nodes.Items) != 2 {
				t.Logf("node count: %d", len(nodes.Items))
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

	t.Logf("waiting for pods to get ready")
	{
		err := wait.Poll(30*time.Second, 30*time.Minute, func() (bool, error) {
			t.Logf("checking pod readiness...")

			for _, prefix := range []string{
				"cilium-operator",
				"cilium",
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
		"aws-eu-central-1a", "eu-central-1a", 1, true, &models.CNIPluginSettings{
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
