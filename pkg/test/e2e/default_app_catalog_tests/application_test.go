//go:build e2e

/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package default_app_catalog_applications_tests

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/argocd"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/cert_manager"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/falco"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/flux"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/k8sgpt"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/kube-vip"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/kubevirt"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/metallb"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/nginx_ingress_controller"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/nvidia_gpu_operator"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/trivy"
	"k8c.io/kubermatic/v2/pkg/test/e2e/default_app_catalog_tests/trivy_operator"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	applicationName    string
	applicationVersion string
	credentials        jig.AWSCredentials
	logOptions         = utils.DefaultLogOptions
)

const (
	projectName = "def-app-catalog-test-project"
)

func init() {
	flag.StringVar(&applicationName, "application-name", "", "name of an application from the default app catalog")
	flag.StringVar(&applicationVersion, "application-version", "", "version of an application from the default app catalog")
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func getChosenApplication() ApplicationInterface {
	// Parse the flags
	flag.Parse()

	var applicationStruct ApplicationInterface
	switch applicationName {
	case "argocd":
		applicationStruct = &argocd.DefaultArgoCD
	case "cert-manager":
		applicationStruct = &cert_manager.DefaultCertManager
	case "falco":
		applicationStruct = &falco.DefaultFalco
	case "flux":
		applicationStruct = &flux.DefaultFlux
	case "k8sgpt":
		applicationStruct = &k8sgpt.DefaultK8sGpt
	case "kube-vip":
		applicationStruct = &kube_vip.DefaultKubeVip
	case "kubevirt":
		applicationStruct = &kubevirt.DefaultKubeVirt
	case "metallb":
		applicationStruct = &metallb.DefaultMetalLB
	case "nginx_ingress_controller":
		applicationStruct = &nginx_ingress_controller.DefaultNginxIngressController
	case "nvidia_gpu_operator":
		applicationStruct = &nvidia_gpu_operator.DefaultNvidiaGpuOperator
	case "trivy":
		applicationStruct = &trivy.DefaultTrivy
	case "trivy-operator":
		applicationStruct = &trivy_operator.DefaultTrivyOperator
	default:
		// Handle unknown applicationName if necessary
		applicationStruct = nil
	}

	return applicationStruct
}

func TestClusters(t *testing.T) {
	if applicationName == "" || applicationVersion == "" {
		return
	}

	rawLog := log.NewFromOptions(logOptions)
	logger := rawLog.Sugar()
	ctx := context.Background()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Fatalf("failed to build config: %v", err)
	}

	seedClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("failed to build ctrlruntime client: %v", err)
	}

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	client, cleanup, tLogger, err := createUserCluster(ctx, t, logger, seedClient)
	if cleanup != nil {
		defer cleanup()
	}

	if err != nil {
		t.Fatalf("failed to create user cluster: %v", err)
	}

	testUserCluster(ctx, t, tLogger, client)
}

//gocyclo:ignore
func testUserCluster(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	application := getChosenApplication()
	name, namespace, key, names := application.FetchData()
	log.Info("Waiting for pods to get ready...")
	err := waitForPods(ctx, t, log, client, namespace, key, names)
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	log.Info("Running tests...")

	log.Info("Checking for ApplicationInstallation...")
	err = wait.PollLog(ctx, log, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		app := &appskubermaticv1.ApplicationInstallation{}
		if err := client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, app); err != nil {
			return fmt.Errorf("failed to get ApplicationInstallation in user cluster: %w", err), nil
		}
		if app.Status.ApplicationVersion == nil {
			return fmt.Errorf("application not yet installed: %v", app.Status), nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Application observe test failed: %v", err)
	}

	err = isHelmReleaseDeployed(ctx, t, log, client, name, namespace)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func waitForPods(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client, namespace string, key string, names []string) error {
	log = log.With("namespace", namespace)

	r, err := labels.NewRequirement(key, selection.In, names)
	if err != nil {
		return fmt.Errorf("failed to build requirement: %w", err)
	}
	l := labels.NewSelector().Add(*r)

	return wait.PollLog(ctx, log, 5*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		pods := corev1.PodList{}
		err = client.List(ctx, &pods, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabelsSelector{Selector: l})
		if err != nil {
			return fmt.Errorf("failed to list Pods: %w", err), nil
		}

		if len(pods.Items) == 0 {
			return errors.New("no Pods found"), nil
		}

		unready := sets.New[string]()
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
			return fmt.Errorf("not all Pods are ready: %v", sets.List(unready)), nil
		}

		return nil, nil
	})
}

func isHelmReleaseDeployed(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client, appName, namespace string) error {
	secrets := corev1.SecretList{}
	err := client.List(ctx, &secrets, ctrlruntimeclient.InNamespace(namespace))
	if err != nil {
		log.Fatalf("failed to list secrets: %v", err)
		return err
	}

	for _, secret := range secrets.Items {
		if containsString(secret.Name, appName) && secret.Type == "helm.sh/release.v1" {
			if status, exists := secret.Labels["status"]; exists && status == "deployed" {
				log.Infof("secret %s in namespace %s is deployed\n", secret.Name, secret.Namespace)
			} else {
				return fmt.Errorf("secret %s in namespace %s is not deployed", secret.Name, secret.Namespace)
			}
		}
	}

	return fmt.Errorf("no helm release deployed")
}

func containsString(name, search string) bool {
	return strings.Contains(name, search)
}

// creates a usercluster on aws.
func createUserCluster(
	ctx context.Context,
	t *testing.T,
	log *zap.SugaredLogger,
	masterClient ctrlruntimeclient.Client,
) (ctrlruntimeclient.Client, func(), *zap.SugaredLogger, error) {
	application := getChosenApplication()
	appAnnotation, err := application.GetApplication(applicationVersion)
	if err != nil {
		return nil, nil, log, fmt.Errorf("failed to prepare test application: %w", err)
	}

	testJig := jig.NewAWSCluster(masterClient, log, credentials, 2, nil)
	testJig.ProjectJig.WithHumanReadableName(projectName)
	testJig.ClusterJig.
		WithTestName("default application catalog test").
		WithKonnectivity(true).
		WithAnnotations(map[string]string{
			kubermaticv1.InitialApplicationInstallationsRequestAnnotation: string(appAnnotation),
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
