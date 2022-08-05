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

package cilium

import (
	"bytes"
	"context"
	"errors"
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
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	userconfig      string
	accessKeyID     string
	secretAccessKey string
	logOptions      = log.NewDefaultOptions()
)

const (
	projectName  = "cilium-test-project"
	ciliumTestNs = "cilium-test"
)

func init() {
	flag.StringVar(&userconfig, "userconfig", "", "path to kubeconfig of usercluster")
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestInExistingCluster(t *testing.T) {
	if userconfig == "" {
		t.Logf("kubeconfig for usercluster not provided, test passes vacuously.")
		t.Logf("to run against an existing usercluster use following command:")
		t.Logf("go test ./pkg/test/e2e/cilium -v -tags e2e -timeout 30m -run TestInExistingCluster -userconfig <USERCLUSTER KUBECONFIG>")
		return
	}

	logger := log.NewFromOptions(logOptions).Sugar()

	config, err := clientcmd.BuildConfigFromFlags("", userconfig)
	if err != nil {
		t.Fatalf("failed to build config: %v", err)
	}

	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("failed to build ctrlruntime client: %v", err)
	}

	testUserCluster(context.Background(), t, logger, client)
}

func TestCiliumClusters(t *testing.T) {
	logger := log.NewFromOptions(logOptions).Sugar()

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
			client, cleanup, tLogger, err := createUserCluster(ctx, t, logger.With("proxymode", proxyMode), client, proxyMode)
			if cleanup != nil {
				defer cleanup()
			}

			if err != nil {
				t.Fatalf("failed to create user cluster: %v", err)
			}

			testUserCluster(ctx, t, tLogger, client)
		})
	}
}

//gocyclo:ignore
func testUserCluster(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	log.Info("Waiting for pods to get ready...")
	err := waitForPods(ctx, t, log, client, "kube-system", "k8s-app", []string{
		"cilium-operator",
		"cilium",
	})
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	log.Info("Running Cilium connectivity tests...")
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

	log = log.With("namespace", ciliumTestNs)
	log.Debug("Namespace created")

	installCiliumConnectivityTests(ctx, t, log, client)

	log.Info("Deploying hubble-relay-nodeport and hubble-ui-nodeport services...")
	cleanup := deployHubbleServices(ctx, t, log, client)
	defer cleanup()

	log.Info("Waiting for Cilium connectivity pods to get ready...")
	err = waitForPods(ctx, t, log, client, ciliumTestNs, "name", []string{
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

	log.Info("Checking for Hubble pods...")
	err = waitForPods(ctx, t, log, client, "kube-system", "k8s-app", []string{
		"hubble-relay",
		"hubble-ui",
	})
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	nodeIP, err := getAnyNodeIP(ctx, client)
	if err != nil {
		t.Fatalf("Nodes are ready, but could not get an IP: %v", err)
	}

	log.Info("Testing Hubble relay observe...")
	err = wait.PollLog(ctx, log, 2*time.Second, 5*time.Minute, func() (error, error) {
		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", nodeIP, 30077), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("failed to dial to Hubble relay: %w", err), nil
		}
		defer conn.Close()

		nFlows := 20
		flowsClient, err := observer.NewObserverClient(conn).
			GetFlows(ctx, &observer.GetFlowsRequest{Number: uint64(nFlows)})
		if err != nil {
			return fmt.Errorf("failed to get flow client: %w", err), nil
		}

		for c := 0; c < nFlows; c++ {
			_, err := flowsClient.Recv()
			if err != nil {
				return fmt.Errorf("failed to get flow: %w", err), nil
			}
			// fmt.Println(flow)
		}

		return nil, nil
	})
	if err != nil {
		t.Fatalf("Hubble relay observe test failed: %v", err)
	}

	log.Info("Testing Hubble UI observe...")
	err = wait.PollLog(ctx, log, 2*time.Second, 5*time.Minute, func() (error, error) {
		uiURL := fmt.Sprintf("http://%s", net.JoinHostPort(nodeIP, "30007"))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, uiURL, nil)
		if err != nil {
			return fmt.Errorf("failed to construct request to Hubble UI: %w", err), nil
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to get response from Hubble UI: %w", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected HTTP 200 OK, got HTTP %d", resp.StatusCode), nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err), nil
		}

		if !strings.Contains(string(body), "Hubble") {
			return errors.New("failed to find Hubble in the body"), nil
		}

		return nil, nil
	})
	if err != nil {
		t.Fatalf("Hubble UI observe test failed: %v", err)
	}
}

func waitForPods(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client, namespace string, key string, names []string) error {
	log = log.With("namespace", namespace)

	r, err := labels.NewRequirement(key, selection.In, names)
	if err != nil {
		return fmt.Errorf("failed to build requirement: %w", err)
	}
	l := labels.NewSelector().Add(*r)

	return wait.PollLog(ctx, log, 5*time.Second, 5*time.Minute, func() (error, error) {
		pods := corev1.PodList{}
		err = client.List(ctx, &pods, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabelsSelector{Selector: l})
		if err != nil {
			return fmt.Errorf("failed to list Pods: %w", err), nil
		}

		if len(pods.Items) == 0 {
			return errors.New("no Pods found"), nil
		}

		unready := sets.NewString()
		for _, pod := range pods.Items {
			ready := false
			for _, c := range pod.Status.Conditions {
				if c.Type == corev1.ContainersReady {
					ready = c.Status == corev1.ConditionTrue
				}
			}

			if !ready {
				unready.Insert(pod.Name)
			}
		}

		if unready.Len() > 0 {
			return fmt.Errorf("not all Pods are ready: %v", unready.List()), nil
		}

		return nil, nil
	})
}

func deployHubbleServices(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) func() {
	hubbleRelaySvc, err := resourcesFromYaml("./testdata/hubble-relay-svc.yaml")
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}

	hubbleUISvc, err := resourcesFromYaml("./testdata/hubble-ui-svc.yaml")
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}

	var cleanups []func()

	objects := []ctrlruntimeclient.Object{}
	objects = append(objects, hubbleRelaySvc...)
	objects = append(objects, hubbleUISvc...)

	cleanups = append(cleanups, func() {
		for _, object := range objects {
			err := client.Delete(ctx, object)
			if err != nil {
				log.Errorw("Failed to delete resource", zap.Error(err))
			}
		}
	})

	for _, object := range objects {
		err := client.Create(ctx, object)
		if err != nil {
			t.Fatalf("Failed to apply resource: %v", err)
		}

		log.Debugw("Created object", "kind", object.GetObjectKind(), "name", object.GetName())
	}

	return func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
}

func installCiliumConnectivityTests(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	objs, err := resourcesFromYaml("./testdata/connectivity-check.yaml")
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}

	for _, obj := range objs {
		obj.SetNamespace(ciliumTestNs)
		if err := client.Create(ctx, obj); err != nil {
			t.Fatalf("failed to apply resource: %v", err)
		}

		log.Debugw("Created object", "kind", obj.GetObjectKind(), "name", obj.GetName())
	}
}

func getAnyNodeIP(ctx context.Context, client ctrlruntimeclient.Client) (string, error) {
	nodeList := corev1.NodeList{}
	if err := client.List(ctx, &nodeList); err != nil {
		return "", fmt.Errorf("failed to get nodes list: %w", err)
	}

	if len(nodeList.Items) == 0 {
		return "", errors.New("cluster has no nodes")
	}

	for _, node := range nodeList.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeExternalIP {
				return addr.Address, nil
			}
		}
	}

	return "", errors.New("no node has an ExternalIP")
}

func resourcesFromYaml(filename string) ([]ctrlruntimeclient.Object, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	manifests, err := yamlutil.ParseMultipleDocuments(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var objs []ctrlruntimeclient.Object
	for _, m := range manifests {
		obj := &unstructured.Unstructured{}
		if err := kyaml.NewYAMLOrJSONDecoder(bytes.NewReader(m.Raw), 1024).Decode(obj); err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	}

	return objs, nil
}

// creates a usercluster on aws.
func createUserCluster(
	ctx context.Context,
	t *testing.T,
	log *zap.SugaredLogger,
	masterClient ctrlruntimeclient.Client,
	proxyMode string,
) (ctrlruntimeclient.Client, func(), *zap.SugaredLogger, error) {
	testJig := jig.NewAWSCluster(masterClient, log, accessKeyID, secretAccessKey, 2)
	testJig.ProjectJig.WithHumanReadableName(projectName)
	testJig.ClusterJig.
		WithTestName("cilium").
		WithAddons(jig.Addon{Name: "hubble"}).
		WithProxyMode(proxyMode).
		WithKonnectivity(true).
		WithCNIPlugin(&kubermaticv1.CNIPluginSettings{
			Type:    kubermaticv1.CNIPluginTypeCilium,
			Version: "v1.11",
		})

	cleanup := func() {
		testJig.Cleanup(ctx, t, true)
	}

	// let the magic happen
	if _, _, err := testJig.Setup(ctx, jig.WaitForReadyPods); err != nil {
		return nil, cleanup, log, fmt.Errorf("failed to setup test environment: %w", err)
	}

	clusterClient, err := testJig.ClusterClient(ctx)

	return clusterClient, cleanup, log, err
}
