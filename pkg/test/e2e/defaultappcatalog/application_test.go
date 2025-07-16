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

package defaultappcatalog

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
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	applicationInstallationName string
	applicationName             string
	applicationNamespace        string
	applicationVersion          string
	key                         string
	names                       string
	defaultValuesBlock          string
	credentials                 jig.AWSCredentials
	logOptions                  = utils.DefaultLogOptions
)

const (
	projectName = "def-app-catalog-test-project"
)

func init() {
	flag.StringVar(&applicationInstallationName, "application-installation-name", "", "name of the ApplicationInstallation object")
	flag.StringVar(&applicationName, "application-name", "", "name of an application from the default app catalog")
	flag.StringVar(&applicationNamespace, "application-namespace", "", "namespace of an application from the default app catalog")
	flag.StringVar(&applicationVersion, "application-version", "", "version of an application from the default app catalog")
	flag.StringVar(&key, "app-label-key", "", "a Kubernetes recommended label used for identifying the name of an application")
	flag.StringVar(&names, "names", "", "names of the pods of an application from the default app catalog")
	flag.StringVar(&defaultValuesBlock, "default-values-block", "", "default values block of an application from the default app catalog")
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestClusters(t *testing.T) {
	if applicationName == "" || applicationVersion == "" || applicationInstallationName == "" || applicationNamespace == "" || key == "" || names == "" {
		t.Fatal("All values must be set.")
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

func testUserCluster(ctx context.Context, t *testing.T, tLogger *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	logger := log.NewLogrus()
	sublogger := log.Prefix(logrus.NewEntry(logger), "   ")

	// Create the namespace in Kubernetes
	if err := util.EnsureNamespace(ctx, sublogger, client, applicationNamespace); err != nil {
		t.Fatalf("%v", err)
	}

	application := appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      applicationInstallationName,
			Namespace: applicationNamespace,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: &appskubermaticv1.AppNamespaceSpec{
				Name:   applicationNamespace,
				Create: true,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    applicationName,
				Version: applicationVersion,
			},
			ValuesBlock: defaultValuesBlock,
		},
	}

	tLogger.Infof("Creating an ApplicationInstallation")

	err := client.Create(ctx, &application)
	if err != nil {
		t.Fatalf("%v", err)
	}

	tLogger.Info("Running tests...")

	tLogger.Info("Check if ApplicationInstallation exists")
	err = wait.PollLog(ctx, tLogger, 2*time.Second, 10*time.Minute, func(ctx context.Context) (error, error) {
		app := &appskubermaticv1.ApplicationInstallation{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: applicationNamespace, Name: applicationInstallationName}, app); err != nil {
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

	tLogger.Info("Checking if the helm release is deployed")
	err = wait.PollLog(ctx, tLogger, 2*time.Second, 8*time.Minute, func(ctx context.Context) (error, error) {
		err = isHelmReleaseDeployed(ctx, tLogger, client, applicationInstallationName, applicationNamespace)
		if err != nil {
			return fmt.Errorf("failed to verify that helm release is deployed: %w", err), nil
		}

		return nil, nil
	})
	if err != nil {
		t.Fatalf("Application observe test failed: %v", err)
	}

	tLogger.Info("Checking if all conditions are ok")
	err = wait.PollLog(ctx, tLogger, 2*time.Second, 8*time.Minute, func(ctx context.Context) (error, error) {
		err = checkApplicationInstallationConditions(ctx, tLogger, client)
		if err != nil {
			return fmt.Errorf("failed to verify that all conditions are in healthy state: %w", err), nil
		}

		return nil, nil
	})
	if err != nil {
		t.Fatalf("Application observe test failed: %v", err)
	}

	namesStrArray := strings.Split(names, ",")
	tLogger.Info("Waiting for pods to get ready...")
	err = wait.PollLog(ctx, tLogger, 2*time.Second, 7*time.Minute, func(ctx context.Context) (error, error) {
		err = waitForPods(ctx, tLogger, client, applicationNamespace, key, namesStrArray)
		if err != nil {
			return fmt.Errorf("failed to verify that all pods are ready: %w", err), nil
		}

		return nil, nil
	})
	if err != nil {
		t.Fatalf("Application observe test failed: %v", err)
	}
}

func waitForPods(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, namespace string, key string, names []string) error {
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
			ready := podIsReadyOrCompleted(&pod)

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

func podIsReadyOrCompleted(pod *corev1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if !containerIsReadyOrCompleted(cs) {
			return false
		}
	}

	for _, cs := range pod.Status.InitContainerStatuses {
		if !containerIsReadyOrCompleted(cs) {
			return false
		}
	}

	return true
}

func containerIsReadyOrCompleted(cs corev1.ContainerStatus) bool {
	if cs.Ready {
		return true
	}

	if cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 0 {
		return true
	}

	return false
}

func isHelmReleaseDeployed(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, appName, namespace string) error {
	secrets := corev1.SecretList{}
	err := client.List(ctx, &secrets, ctrlruntimeclient.InNamespace(namespace))
	if err != nil {
		log.Fatalf("failed to list secrets: %v", err)
		return err
	}

	for _, secret := range secrets.Items {
		if strings.Contains(secret.Name, appName) && secret.Type == "helm.sh/release.v1" {
			if secret.Labels["status"] == "deployed" {
				log.Infof("secret %s in namespace %s, helm release is deployed\n", secret.Name, secret.Namespace)
			} else {
				return fmt.Errorf("secret %s in namespace %s, helm release is not deployed", secret.Name, secret.Namespace)
			}
		}
	}

	return nil
}

func checkApplicationInstallationConditions(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	applicationInstallation := &appskubermaticv1.ApplicationInstallation{}
	err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: applicationInstallationName, Namespace: applicationNamespace}, applicationInstallation)
	if err != nil {
		log.Fatalf("failed to get ApplicationInstallation: %v", err)
		return err
	}

	// Check if all conditions are "Ready"
	allConditionsReady := true
	for _, condition := range applicationInstallation.Status.Conditions {
		if condition.Status != "True" {
			allConditionsReady = false
			break
		}
	}

	if !allConditionsReady {
		return fmt.Errorf("ApplicationInstallation %s in namespace %s, conditions are not ready", applicationName, applicationNamespace)
	}

	log.Info("all conditions ready")

	// Check if Helm release status is "deployed"
	helmReleaseDeployed := applicationInstallation.Status.HelmRelease.Info.Status == "deployed"

	if !helmReleaseDeployed {
		return fmt.Errorf("ApplicationInstallation %s in namespace %s, helm release is not deployed", applicationName, applicationNamespace)
	}

	log.Info("helm release deployed")

	log.Infof("ApplicationInstallation %s in namespace %s is deployed and ready\n", applicationName, applicationNamespace)

	return nil
}

// creates a usercluster on aws.
func createUserCluster(
	ctx context.Context,
	t *testing.T,
	log *zap.SugaredLogger,
	masterClient ctrlruntimeclient.Client,
) (ctrlruntimeclient.Client, func(), *zap.SugaredLogger, error) {
	testJig := jig.NewAWSCluster(masterClient, log, credentials, 2, nil)
	testJig.ProjectJig.WithHumanReadableName(projectName)
	testJig.ClusterJig.
		WithTestName("application-test").
		WithKonnectivity(true).
		WithAnnotations(map[string]string{
			"env": "dev",
		}).
		WithProxyMode("ebpf")

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
