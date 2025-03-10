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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
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
	applicationName      string
	applicationNamespace string
	applicationVersion   string
	key                  string
	names                string
	credentials          jig.AWSCredentials
	logOptions           = utils.DefaultLogOptions
)

const (
	projectName = "def-app-catalog-test-project"
)

func init() {
	flag.StringVar(&applicationName, "application-name", "", "name of an application from the default app catalog")
	flag.StringVar(&applicationNamespace, "application-namespace", "", "namespace of an application from the default app catalog")
	flag.StringVar(&applicationVersion, "application-version", "", "version of an application from the default app catalog")
	flag.StringVar(&key, "app-label-key", "", "a Kubernetes recommended label used for identifying the name of an application")
	flag.StringVar(&names, "names", "", "names of the pods of an application from the default app catalog")
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
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
	//namespace := &corev1.Namespace{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: applicationNamespace,
	//	},
	//}
	//
	//// Create the namespace in Kubernetes
	//err := client.Create(ctx, namespace)
	//if err != nil {
	//	t.Fatalf("%v", err)
	//} else {
	//	log.Infof("Namespace %s created", namespace.Name)
	//}
	//
	//applicationRefName := applicationName
	//if applicationName == "gpu-operator" {
	//	applicationRefName = "nvidia-gpu-operator"
	//}
	//
	//application := appskubermaticv1.ApplicationInstallation{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      applicationName,
	//		Namespace: applicationNamespace,
	//	},
	//	Spec: appskubermaticv1.ApplicationInstallationSpec{
	//		Namespace: &appskubermaticv1.AppNamespaceSpec{
	//			Name:   applicationNamespace,
	//			Create: true,
	//		},
	//		ApplicationRef: appskubermaticv1.ApplicationRef{
	//			Name:    applicationRefName,
	//			Version: applicationVersion,
	//		},
	//	},
	//}
	//
	//log.Infof("Creating an ApplicationInstallation")
	//
	//err = client.Create(ctx, &application)
	//if err != nil {
	//	t.Fatalf("%v", err)
	//}
	//
	//// wait for the ApplicationInstallation to be installed
	//time.Sleep(720 * time.Second)

	log.Info("Running tests...")

	log.Info("Checking for ApplicationInstallation...")
	err := wait.PollLog(ctx, log, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		app := &appskubermaticv1.ApplicationInstallation{}
		if err := client.Get(context.Background(), types.NamespacedName{Namespace: applicationNamespace, Name: applicationName}, app); err != nil {
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

	// this sleep is specifically needed for the cert-manager
	if applicationName == "cert-manager" {
		time.Sleep(720 * time.Second)
	}

	log.Info("Checking if the helm release is deployed")
	err = isHelmReleaseDeployed(ctx, log, client, applicationName, applicationNamespace)
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Info("Checking if all conditions are ok")
	err = checkApplicationInstallationConditions(ctx, log, client)
	if err != nil {
		t.Fatalf("%v", err)
	}

	namesStrArray := strings.Split(names, ",")
	log.Info("Waiting for pods to get ready...")
	err = waitForPods(ctx, log, client, applicationNamespace, key, namesStrArray)
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
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

func isHelmReleaseDeployed(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, appName, namespace string) error {
	secrets := corev1.SecretList{}
	err := client.List(ctx, &secrets, ctrlruntimeclient.InNamespace(namespace))
	if err != nil {
		log.Fatalf("failed to list secrets: %v", err)
		return err
	}

	for _, secret := range secrets.Items {
		if containsString(secret.Name, appName) && secret.Type == "helm.sh/release.v1" {
			if status, exists := secret.Labels["status"]; exists && status == "deployed" {
				log.Infof("Exists: %s and status: %s", secret.Name, status)
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
	err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: applicationName, Namespace: applicationNamespace}, applicationInstallation)
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
	helmReleaseDeployed := false
	if applicationInstallation.Status.HelmRelease.Info.Status == "deployed" {
		helmReleaseDeployed = true
	}

	if !helmReleaseDeployed {
		return fmt.Errorf("ApplicationInstallation %s in namespace %s, helm release is not deployed", applicationName, applicationNamespace)
	}

	log.Info("helm release deployed")

	log.Infof("ApplicationInstallation %s in namespace %s is deployed and ready\n", applicationName, applicationNamespace)

	return nil
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
	//application := appskubermaticv1.ApplicationInstallation{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      applicationName,
	//		Namespace: applicationNamespace,
	//	},
	//	Spec: appskubermaticv1.ApplicationInstallationSpec{
	//		Namespace: &appskubermaticv1.AppNamespaceSpec{
	//			Name:   applicationNamespace,
	//			Create: true,
	//		},
	//		ApplicationRef: appskubermaticv1.ApplicationRef{
	//			Name:    applicationName,
	//			Version: applicationVersion,
	//		},
	//	},
	//}

	//appAnnotation, err := json.Marshal(application)
	//if err != nil {
	//	return nil, nil, log, fmt.Errorf("failed to setup an application: %w", err)
	//}

	// -----
	applicationRefName := applicationName
	if applicationName == "gpu-operator" {
		applicationRefName = "nvidia-gpu-operator"
	}
	application := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      applicationName,
			Namespace: applicationNamespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   applicationNamespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    applicationRefName,
				Version: applicationVersion,
			},
		},
	}

	if applicationName == "cluster-autoscaler" {
		valuesBlock := `
cloudProvider: clusterapi
clusterAPIMode: incluster-incluster
autoDiscovery:
  namespace: kube-system
image:
  # 'Cluster.AutoscalerVersion' is injected by KKP based on the Kubernetes version of the cluster.
  tag: '{{ .Cluster.AutoscalerVersion }}'
extraEnv:
  CAPI_GROUP: cluster.k8s.io
rbac:
  create: true
  pspEnabled: false
  clusterScoped: true
  serviceAccount:
    annotations: {}
    create: true
    name: "cluster-autoscaler-clusterapi-cluster-autoscaler"
    automountServiceAccountToken: true
extraObjects:
- apiVersion: rbac.authorization.k8s.io/v1
  kind: ClusterRole
  metadata:
    name: cluster-autoscaler-management
  rules:
  - apiGroups:
    - cluster.k8s.io
    resources:
    - machinedeployments
    - machinedeployments/scale
    - machines
    - machinesets
    verbs:
    - get
    - list
    - update
    - watch
- apiVersion: rbac.authorization.k8s.io/v1
  kind: ClusterRoleBinding
  metadata:
    name: cluster-autoscaler-clusterapi-cluser-autoscaler
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: cluster-autoscaler-management
  subjects:
  - kind: ServiceAccount
    name: cluster-autoscaler-clusterapi-cluster-autoscaler
    # 'Release.Namespace' is injected by Helm.
    namespace: '{{ "{{.Release.Namespace}}" }}'
`
		application.Spec.ValuesBlock = valuesBlock
	}

	applications := []apiv1.Application{application}
	appAnnotation, err := json.Marshal(applications)
	if err != nil {
		return nil, nil, log, fmt.Errorf("failed to setup an application: %w", err)
	}

	testJig := jig.NewAWSCluster(masterClient, log, credentials, 2, nil)
	testJig.ProjectJig.WithHumanReadableName(projectName)
	testJig.ClusterJig.
		WithTestName("application-test").
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
