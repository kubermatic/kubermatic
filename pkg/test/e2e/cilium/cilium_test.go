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

package cilium

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
	"go.uber.org/zap"
	"google.golang.org/grpc"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"google.golang.org/grpc/credentials/insecure"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
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
	logOptions      = log.NewDefaultOptions()
	namespace       = "kubermatic"
)

const (
	projectName  = "cilium-test-project"
	ciliumTestNs = "cilium-test"
)

func init() {
	flag.StringVar(&userconfig, "userconfig", "", "path to kubeconfig of usercluster")
	flag.StringVar(&namespace, "namespace", namespace, "namespace where KKP is installed into")
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
	log.Info("Waiting for nodes to come up...")
	_, err := checkNodeReadiness(ctx, t, log, client)
	if err != nil {
		t.Fatalf("nodes never became ready: %v", err)
	}

	log.Info("Waiting for pods to get ready...")
	err = waitForPods(ctx, t, log, client, "kube-system", "k8s-app", []string{
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

	log.Info("Testing Hubble relay observe...")
	err = wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		nodeIP, err := checkNodeReadiness(ctx, t, log, client)
		if err != nil {
			log.Errorw("Nodes never became ready", zap.Error(err))
			return false, nil
		}

		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", nodeIP, 30077), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Errorw("Failed to dial to Hubble relay", zap.Error(err))
			return false, nil
		}
		defer conn.Close()

		nFlows := 20
		flowsClient, err := observer.NewObserverClient(conn).
			GetFlows(ctx, &observer.GetFlowsRequest{Number: uint64(nFlows)})
		if err != nil {
			log.Errorw("Failed to get flow client", zap.Error(err))
			return false, nil
		}

		for c := 0; c < nFlows; c++ {
			_, err := flowsClient.Recv()
			if err != nil {
				log.Errorw("Failed to get flow", zap.Error(err))
				return false, nil
			}
			// fmt.Println(flow)
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Hubble relay observe test failed: %v", err)
	}

	log.Info("Testing Hubble UI observe...")
	err = wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		nodeIP, err := checkNodeReadiness(ctx, t, log, client)
		if err != nil {
			log.Debugw("Nodes never became ready", zap.Error(err))
			return false, nil
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("http://%s", net.JoinHostPort(nodeIP, "30007")), nil)
		if err != nil {
			log.Errorw("Failed to construct request to Hubble UI", zap.Error(err))
			return false, nil
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Debugw("Failed to get response from Hubble UI", zap.Error(err))
			return false, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Debugf("Expected: HTTP 200 OK, got: HTTP %d", resp.StatusCode)
			return false, nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorw("Failed to read response body", zap.Error(err))
			return false, nil
		}

		if !strings.Contains(string(body), "Hubble") {
			log.Debug("Failed to find Hubble in the body")
			return false, nil
		}

		return true, nil
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

	return wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		pods := corev1.PodList{}
		err = client.List(ctx, &pods, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabelsSelector{Selector: l})
		if err != nil {
			log.Errorw("Failed to list Pods", zap.Error(err))
			return false, nil
		}

		if len(pods.Items) == 0 {
			log.Debug("No Pods found")
			return false, nil
		}

		if !allPodsHealthy(t, log, &pods) {
			log.Debug("Not all Pods healthy yet...")
			return false, nil
		}

		log.Debug("All Pods healthy")

		return true, nil
	})
}

func allPodsHealthy(t *testing.T, log *zap.SugaredLogger, pods *corev1.PodList) bool {
	allHealthy := true

	for _, pod := range pods.Items {
		podLog := log.With("pod", pod.Name)

		if pod.Status.Phase != corev1.PodRunning {
			allHealthy = false
			podLog.Debugw("Pod is not running", "phase", pod.Status.Phase)
		} else {
			for _, c := range pod.Status.Conditions {
				switch c.Type {
				case corev1.PodReady:
					fallthrough
				case corev1.ContainersReady:
					if c.Status != corev1.ConditionTrue {
						allHealthy = false
						podLog.Debugw("Pod not ready")
					}
				}
			}
		}
	}

	return allHealthy
}

func deployHubbleServices(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) func() {
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
	s := makeScheme()

	objs, err := resourcesFromYaml("./testdata/connectivity-check.yaml", s)
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

func checkNodeReadiness(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) (string, error) {
	expectedNodes := 2
	var nodeIP string

	err := wait.Poll(5*time.Second, 15*time.Minute, func() (bool, error) {
		nodeList := corev1.NodeList{}
		err := client.List(ctx, &nodeList)
		if err != nil {
			log.Errorw("Failed to get nodes list", zap.Error(err))
			return false, nil
		}

		if len(nodeList.Items) != expectedNodes {
			log.Debugw("Cluster does not have expected number of nodes", "expected", expectedNodes, "current", len(nodeList.Items))
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
			log.Debugf("Not all nodes are ready yet", "expected", expectedNodes, "ready", readyNodeCount)
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
func createUserCluster(
	ctx context.Context,
	t *testing.T,
	log *zap.SugaredLogger,
	masterClient ctrlruntimeclient.Client,
	proxyMode string,
) (ctrlruntimeclient.Client, func(), *zap.SugaredLogger, error) {
	var teardowns []func()
	cleanup := func() {
		n := len(teardowns)
		for i := range teardowns {
			teardowns[n-1-i]()
		}
	}

	configGetter, err := provider.DynamicKubermaticConfigurationGetterFactory(masterClient, namespace)
	if err != nil {
		return nil, nil, log, fmt.Errorf("failed to create configGetter: %w", err)
	}

	// prepare helpers
	projectProvider, _ := kubernetes.NewProjectProvider(nil, masterClient)
	addonProvider := kubernetes.NewAddonProvider(masterClient, nil, configGetter)

	userClusterConnectionProvider, err := client.NewExternal(masterClient)
	if err != nil {
		return nil, nil, log, fmt.Errorf("failed to create userClusterConnectionProvider: %w", err)
	}

	clusterProvider := kubernetes.NewClusterProvider(
		nil,
		nil,
		userClusterConnectionProvider,
		"",
		nil,
		masterClient,
		nil,
		false,
		kubermatic.Versions{},
		nil,
	)

	log.Info("Creating project...")
	project, err := projectProvider.New(ctx, projectName, nil)
	if err != nil {
		return nil, nil, log, err
	}
	log = log.With("project", project.Name)
	teardowns = append(teardowns, func() {
		log.Info("Deleting project...")
		if err := masterClient.Delete(ctx, project); err != nil {
			t.Errorf("failed to delete project: %v", err)
		}
	})

	version := utils.KubernetesVersion()

	// create a usercluster on AWS
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("cilium-e2e-%s-", proxyMode),
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: project.Name,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: fmt.Sprintf("Cilium %s e2e test cluster", proxyMode),
			Version:           *semver.NewSemverOrDie(version),
			KubernetesDashboard: kubermaticv1.KubernetesDashboard{
				Enabled: false,
			},
			EnableUserSSHKeyAgent: pointer.Bool(false),
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

	log.Info("Creating cluster...")
	cluster, err = clusterProvider.NewUnsecured(ctx, project, cluster, "cilium@e2e.test")
	if err != nil {
		return nil, cleanup, log, fmt.Errorf("failed to create cluster: %w", err)
	}
	log = log.With("cluster", cluster.Name)
	teardowns = append(teardowns, func() {
		// This deletion will happen in the background, i.e. we are not waiting
		// for its completion. This is fine in e2e tests, where the surrounding
		// bash script will (as part of its normal cleanup) delete (and wait) all
		// userclusters anyway.
		log.Info("Deleting cluster...")
		if err := masterClient.Delete(ctx, cluster); err != nil {
			t.Errorf("failed to delete cluster: %v", err)
		}
	})

	// wait for cluster to be up and running
	log.Info("Waiting for cluster to become healthy...")
	err = wait.Poll(2*time.Second, 10*time.Minute, func() (bool, error) {
		curCluster := kubermaticv1.Cluster{}
		if err := masterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), &curCluster); err != nil {
			return false, fmt.Errorf("failed to retrieve cluster: %w", err)
		}

		return curCluster.Status.ExtendedHealth.AllHealthy(), nil
	})
	if err != nil {
		return nil, cleanup, log, fmt.Errorf("cluster did not become healthy: %w", err)
	}

	// create hubble addon
	log.Info("Installing hubble addon...")
	if _, err = addonProvider.NewUnsecured(ctx, cluster, "hubble", nil, nil); err != nil {
		return nil, cleanup, log, fmt.Errorf("failed to create addon: %w", err)
	}

	// update our local cluster variable with the newly reconciled address values
	if err := masterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster); err != nil {
		return nil, cleanup, log, fmt.Errorf("failed to retrieve cluster: %w", err)
	}

	// retrieve usercluster kubeconfig, this can fail a couple of times until
	// the exposing mechanism is ready
	log.Info("Retrieving cluster client...")
	var clusterClient ctrlruntimeclient.Client
	err = wait.Poll(1*time.Second, 30*time.Second, func() (bool, error) {
		var err error
		clusterClient, err = clusterProvider.GetAdminClientForCustomerCluster(ctx, cluster)
		return err == nil, nil
	})
	if err != nil {
		return nil, cleanup, log, fmt.Errorf("cluster did not become available: %w", err)
	}

	utilruntime.Must(clusterv1alpha1.AddToScheme(clusterClient.Scheme()))
	utilruntime.Must(ciliumv2.AddToScheme(clusterClient.Scheme()))

	// prepare MachineDeployment
	encodedOSSpec, err := json.Marshal(ubuntu.Config{})
	if err != nil {
		return nil, cleanup, log, fmt.Errorf("failed to encode osspec: %w", err)
	}

	encodedCloudProviderSpec, err := json.Marshal(awstypes.RawConfig{
		InstanceType:     providerconfig.ConfigVarString{Value: "t3.small"},
		DiskType:         providerconfig.ConfigVarString{Value: "standard"},
		DiskSize:         int64(25),
		VpcID:            providerconfig.ConfigVarString{Value: cluster.Spec.Cloud.AWS.VPCID},
		InstanceProfile:  providerconfig.ConfigVarString{Value: cluster.Spec.Cloud.AWS.InstanceProfileName},
		Region:           providerconfig.ConfigVarString{Value: "eu-central-1"},
		AvailabilityZone: providerconfig.ConfigVarString{Value: "eu-central-1a"},
		SecurityGroupIDs: []providerconfig.ConfigVarString{{
			Value: cluster.Spec.Cloud.AWS.SecurityGroupID,
		}},
		Tags: map[string]string{
			"kubernetes.io/cluster/" + cluster.Name: "",
			"system/cluster":                        cluster.Name,
		},
	})
	if err != nil {
		return nil, cleanup, log, fmt.Errorf("failed to encode providerspec: %w", err)
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
		return nil, cleanup, log, fmt.Errorf("failed to encode providerconfig: %w", err)
	}

	labels := map[string]string{
		"type": "worker",
	}

	md := clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "worker-nodes",
			Namespace: "kube-system",
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: pointer.Int32(2),
			Template: clusterv1alpha1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
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
	log.Info("Creating MachineDeployment...")
	err = wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
		return clusterClient.Create(ctx, &md) == nil, nil
	})
	if err != nil {
		return nil, cleanup, log, fmt.Errorf("failed to create MachineDeployment: %w", err)
	}

	return clusterClient, cleanup, log, nil
}
