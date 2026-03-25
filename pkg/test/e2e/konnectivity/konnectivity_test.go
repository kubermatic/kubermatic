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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/cmd/cp"
	"k8s.io/kubectl/pkg/cmd/util"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

const (
	tailLines     = 5
	projectName   = "konne-test-project"
	alpineSleeper = "alpine-sleeper"
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestKonnectivity(t *testing.T) {
	ctx := context.Background()
	logger := log.NewFromOptions(logOptions).Sugar()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Fatal("Failed to build config", err)
	}

	seedClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("Failed to build ctrlruntime client: %v", err)
	}

	tests := []struct {
		name      string
		proxyMode string
	}{
		{
			name:      "ipvs proxy mode",
			proxyMode: resources.IPVSProxyMode,
		},
		{
			name:      "iptables proxy mode",
			proxyMode: resources.IPTablesProxyMode,
		},
		{
			name:      "nftables proxy mode",
			proxyMode: resources.NFTablesProxyMode,
		},
	}

	for _, test := range tests {
		proxyMode := test.proxyMode
		t.Run(test.name, func(t *testing.T) {
			cluster, userClusterClient, restConfig, cleanup, tLogger, err := createUserCluster(ctx, t, logger.With("proxymode", proxyMode), seedClient, proxyMode)
			if cleanup != nil {
				defer cleanup()
			}
			if err != nil {
				t.Fatalf("Failed to setup usercluster: %v", err)
			}

			testKonnectivityCluster(ctx, t, tLogger, seedClient, cluster, userClusterClient, restConfig)
		})
	}
}

func testKonnectivityCluster(ctx context.Context, t *testing.T, logger *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, restConfig *rest.Config) {
	userClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %s", err)
	}

	metricsClient, err := metrics.NewForConfig(restConfig)
	if err != nil {
		t.Fatalf("Failed to create metrics client: %s", err)
	}

	if err := verifyApiserverSidecar(ctx, seedClient, logger, cluster); err != nil {
		t.Fatalf("Failed to verify apiserver: %v", err)
	}

	sleeperPod, err := setupSleeperPod(ctx, userClusterClient, logger)
	if err != nil {
		t.Fatalf("Failed to create sleeper pod: %v", err)
	}
	defer func() {
		if err := userClusterClient.Delete(ctx, sleeperPod); err != nil {
			t.Fatalf("Failed to delete alpine-sleeper pod: %v", err)
		}
	}()

	logger.Info("Waiting for Deployments to get ready...")
	if err := utils.WaitForDeploymentReady(ctx, userClusterClient, logger, metav1.NamespaceSystem, "konnectivity-agent", 5*time.Minute); err != nil {
		t.Fatalf("konnectivity-agent Deployment did not get ready: %v", err)
	}
	if err := utils.WaitForDeploymentReady(ctx, userClusterClient, logger, metav1.NamespaceSystem, "metrics-server", 5*time.Minute); err != nil {
		t.Fatalf("metrics-server Deployment did not get ready: %v", err)
	}

	if err := verifyPodLogs(ctx, userClusterClient, logger, userClient); err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	if err := verifyCopyingFiles(ctx, logger, restConfig, userClient, sleeperPod); err != nil {
		t.Fatalf("Failed to verify copy operations: %v", err)
	}

	if err := verifyMetrics(ctx, metricsClient, logger); err != nil {
		t.Fatalf("Failed to verify metrics availability: %v", err)
	}
}

func verifyApiserverSidecar(ctx context.Context, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	log.Info("Verifying that apiserver has konnectivity-proxy sidecar...")

	deployment := appsv1.Deployment{}
	key := types.NamespacedName{Name: resources.ApiserverDeploymentName, Namespace: cluster.Status.NamespaceName}
	if err := seedClient.Get(ctx, key, &deployment); err != nil {
		return fmt.Errorf("failed to get apiserver Deployment: %w", err)
	}

	for _, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name == resources.KonnectivityServerContainer {
			return nil
		}
	}

	return fmt.Errorf("no %q sidecar container found in %q Deployment", resources.KonnectivityServerContainer, key.Name)
}

func setupSleeperPod(ctx context.Context, userClient ctrlruntimeclient.Client, log *zap.SugaredLogger) (*corev1.Pod, error) {
	log.Info("Creating alpine-sleeper pod for testing cp...")

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alpineSleeper,
			Namespace: metav1.NamespaceSystem,
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
	}

	if err := userClient.Create(ctx, pod); err != nil {
		return nil, err
	}

	if !utils.CheckPodsRunningReady(ctx, userClient, log, pod.Namespace, []string{pod.Name}, 2*time.Minute) {
		return nil, errors.New("Pod did not become ready")
	}

	return pod, nil
}

func verifyPodLogs(ctx context.Context, userClient ctrlruntimeclient.Client, log *zap.SugaredLogger, clientset *kubernetes.Clientset) error {
	log.Info("Verifying that we can get logs from Pods...")

	return wait.PollLog(ctx, log, 5*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		podList := corev1.PodList{}
		if err := userClient.List(ctx, &podList, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
			return fmt.Errorf("failed to list Pods: %w", err), nil
		}

		metricsPods := []corev1.Pod{}
		for i, pod := range podList.Items {
			if strings.HasPrefix(pod.Name, "metrics-server-") {
				metricsPods = append(metricsPods, podList.Items[i])
			}
		}

		if len(metricsPods) != 2 {
			return fmt.Errorf("expected 2 metrics-server pods but got %d", len(metricsPods)), nil
		}

		for _, pod := range metricsPods {
			s, err := getPodLogs(ctx, clientset, pod, tailLines)
			if err != nil {
				return fmt.Errorf("failed to get logs from Pod %s: %w", pod.Name, err), nil
			}

			lines := strings.TrimSpace(s)
			if n := len(strings.Split(lines, "\n")); n != tailLines {
				return fmt.Errorf("expected %d log lines but got %d", tailLines, n), nil
			}
		}

		return nil, nil
	})
}

func verifyCopyingFiles(ctx context.Context, log *zap.SugaredLogger, config *rest.Config, clientset *kubernetes.Clientset, sleeperPod *corev1.Pod) error {
	log.Info("Verifying that we can copy files to Pods...")

	podExec := NewPodExec(config, clientset)
	target := fmt.Sprintf("%s/%s:/", metav1.NamespaceSystem, sleeperPod.Name)

	if err := podExec.PodCopyFile("./testdata/copyMe.txt", target, sleeperPod.Spec.Containers[0].Name); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}

	return nil
}

func verifyMetrics(ctx context.Context, metricsClient *metrics.Clientset, log *zap.SugaredLogger) error {
	log.Info("Verifying that we can get metrics...")

	return wait.PollLog(ctx, log, 5*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		// TODO: check if metrics have sane values.
		nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to get node metrics: %w", err), nil
		}

		if len(nodeMetrics.Items) == 0 {
			return errors.New("no node metrics"), nil
		}

		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to get pod metrics: %w", err), nil
		}

		if len(podMetrics.Items) == 0 {
			return errors.New("no pod metrics"), nil
		}

		return nil, nil
	})
}

type PodExec struct {
	util.Factory

	RestConfig *rest.Config
	*kubernetes.Clientset
}

func NewPodExec(config *rest.Config, clientset *kubernetes.Clientset) *PodExec {
	config.APIPath = "/api"                                   // Make sure we target /api and not just /
	config.GroupVersion = &schema.GroupVersion{Version: "v1"} // this targets the core api groups so the url path will be /api/v1
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	return &PodExec{
		Factory:    util.NewFactory(genericclioptions.NewTestConfigFlags()),
		RestConfig: config,
		Clientset:  clientset,
	}
}

// KubernetesClientSet() implements part of util.Factory.
func (p *PodExec) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return &testClientConfig{}
}

// KubernetesClientSet() implements part of util.Factory.
func (p *PodExec) KubernetesClientSet() (*kubernetes.Clientset, error) {
	return p.Clientset, nil
}

// ToRESTConfig() implements part of util.Factory.
func (p *PodExec) ToRESTConfig() (*rest.Config, error) {
	return p.RestConfig, nil
}

func (p *PodExec) PodCopyFile(src string, dst string, containername string) error {
	ioStreams := genericclioptions.NewTestIOStreamsDiscard()
	copyOptions := cp.NewCopyOptions(ioStreams)

	// inject args into options
	if err := copyOptions.Complete(p, &cobra.Command{}, []string{src, dst}); err != nil {
		return fmt.Errorf("could not prepare copy operation: %w", err)
	}

	copyOptions.Container = containername
	copyOptions.NoPreserve = true

	if err := copyOptions.Run(); err != nil {
		return fmt.Errorf("could not run copy operation: %w", err)
	}

	return nil
}

func getPodLogs(ctx context.Context, cli *kubernetes.Clientset, pod corev1.Pod, lines int64) (string, error) {
	podLogOpts := corev1.PodLogOptions{
		TailLines: ptr.To[int64](lines),
	}

	req := cli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("error opening stream: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error while reading logs stream: %w", err)
	}
	str := buf.String()

	return str, nil
}

// creates a usercluster on aws.
func createUserCluster(
	ctx context.Context,
	t *testing.T,
	log *zap.SugaredLogger,
	masterClient ctrlruntimeclient.Client,
	proxyMode string,
) (*kubermaticv1.Cluster, ctrlruntimeclient.Client, *rest.Config, func(), *zap.SugaredLogger, error) {
	testJig := jig.NewAWSCluster(masterClient, log, credentials, 1, nil)
	testJig.ProjectJig.WithHumanReadableName(projectName)
	testJig.ClusterJig.
		WithKonnectivity(true).
		WithTestName("konnectivity").
		WithProxyMode(proxyMode)

	cleanup := func() {
		testJig.Cleanup(ctx, t, true)
	}

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	if err != nil {
		return nil, nil, nil, cleanup, log, fmt.Errorf("failed to setup test environment: %w", err)
	}

	clusterClient, err := testJig.ClusterClient(ctx)
	if err != nil {
		return nil, nil, nil, cleanup, log, fmt.Errorf("failed to create cluster client: %w", err)
	}

	clusterConfig, err := testJig.ClusterRESTConfig(ctx)

	return cluster, clusterClient, clusterConfig, cleanup, log, err
}

type testClientConfig struct{}

var _ clientcmd.ClientConfig = &testClientConfig{}

func (cc *testClientConfig) Namespace() (string, bool, error) {
	return "", false, nil
}

func (cc *testClientConfig) RawConfig() (clientcmdapi.Config, error) {
	return clientcmdapi.Config{}, nil
}

func (cc *testClientConfig) ClientConfig() (*rest.Config, error) {
	return nil, nil
}

func (cc *testClientConfig) ConfigAccess() clientcmd.ConfigAccess {
	return nil
}
