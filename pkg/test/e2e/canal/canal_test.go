//go:build e2e

/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package canal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	userconfig  string
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

const (
	projectName  = "canal-test-project"
	canalTestNs  = "canal-test"
	nginxAppName = "nginx-canal-test"
)

var allProxyModes = []string{
	resources.IPVSProxyMode,
	resources.IPTablesProxyMode,
	resources.NFTablesProxyMode,
}

func init() {
	flag.StringVar(&userconfig, "userconfig", "", "path to kubeconfig of usercluster")
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestInExistingCluster(t *testing.T) {
	if userconfig == "" {
		t.Logf("kubeconfig for usercluster not provided, test passes vacuously.")
		t.Logf("to run against an existing usercluster use following command:")
		t.Logf("go test ./pkg/test/e2e/canal -v -tags e2e -timeout 30m -run TestInExistingCluster -userconfig <USERCLUSTER KUBECONFIG>")
		return
	}

	rawLog := log.NewFromOptions(logOptions)
	logger := rawLog.Sugar()

	config, err := clientcmd.BuildConfigFromFlags("", userconfig)
	if err != nil {
		t.Fatalf("failed to build config: %v", err)
	}

	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("failed to build ctrlruntime client: %v", err)
	}

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	testUserCluster(context.Background(), t, logger, client)
}

func TestCanalClusters(t *testing.T) {
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

	testAppDefs, err := resourcesFromYaml("./testdata/test-app-def.yaml")
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}
	for _, testAppDef := range testAppDefs {
		if err := seedClient.Create(ctx, testAppDef); err != nil {
			t.Fatalf("failed to apply resource: %v", err)
		}

		logger.Infow("Created object", "kind", testAppDef.GetObjectKind(), "name", testAppDef.GetName())
	}

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	tests := []struct {
		name      string
		proxyMode string
	}{
		{
			name:      "ipvs proxy mode test",
			proxyMode: resources.IPVSProxyMode,
		},
		{
			name:      "iptables proxy mode test",
			proxyMode: resources.IPTablesProxyMode,
		},
		{
			name:      "nftables proxy mode test",
			proxyMode: resources.NFTablesProxyMode,
		},
	}

	for _, test := range tests {
		proxyMode := test.proxyMode
		t.Run(test.name, func(t *testing.T) {
			testJig, client, cleanup, tLogger, err := createUserCluster(ctx, t, logger.With("proxymode", proxyMode), seedClient, proxyMode)
			if cleanup != nil {
				defer cleanup()
			}

			if err != nil {
				t.Fatalf("failed to create user cluster: %v", err)
			}

			testUserCluster(ctx, t, tLogger, client)

			// Test proxy mode upgrades: switch to each of the other proxy modes
			// on the same cluster and re-run connectivity tests.
			migrateProxyModes := func(mode string) []string {
				var modes []string
				for _, upgradeMode := range allProxyModes {
					if strings.EqualFold(upgradeMode, mode) {
						continue
					}
					modes = append(modes, upgradeMode)
				}
				return modes
			}(proxyMode)

			currentMode := proxyMode
			for _, upgradeMode := range migrateProxyModes {
				upgradeLogger := tLogger.With("upgrade-to", upgradeMode)
				upgradeLogger.Infof("Upgrading proxy mode from %s to %s...", currentMode, upgradeMode)

				err := changeProxyMode(ctx, upgradeLogger, seedClient, testJig, upgradeMode)
				if err != nil {
					t.Fatalf("failed to change proxy mode to %s: %v", upgradeMode, err)
				}

				upgradeLogger.Info("Waiting for kube-proxy rollout after proxy mode change...")
				err = waitForKubeProxyRollout(ctx, upgradeLogger, client)
				if err != nil {
					t.Fatalf("kube-proxy rollout failed after proxy mode change to %s: %v", upgradeMode, err)
				}

				testUserCluster(ctx, t, upgradeLogger, client)
				currentMode = upgradeMode
			}
		})
	}
}

func testUserCluster(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	log.Info("Waiting for Canal and kube-proxy pods to get ready...")
	err := waitForPods(ctx, t, log, client, "kube-system", "k8s-app", []string{
		"calico-kube-controllers",
		"canal",
		"kube-proxy",
	})
	if err != nil {
		t.Fatalf("Canal/kube-proxy pods never became ready: %v", err)
	}

	log.Info("Verifying kube-proxy DaemonSet has config hash annotation...")
	err = verifyKubeProxyConfigHash(ctx, log, client)
	if err != nil {
		t.Fatalf("kube-proxy config hash verification failed: %v", err)
	}

	log.Info("Verifying kube-proxy ConfigMap exists...")
	cm := &corev1.ConfigMap{}
	if err := client.Get(ctx, types.NamespacedName{Name: "kube-proxy", Namespace: "kube-system"}, cm); err != nil {
		t.Fatalf("failed to get kube-proxy ConfigMap: %v", err)
	}

	if _, ok := cm.Data["config.conf"]; !ok {
		t.Fatal("kube-proxy ConfigMap is missing config.conf key")
	}

	// --- Connectivity checks ---
	log.Info("Running Canal connectivity tests...")

	// Clean up any leftover namespace from a previous run (e.g. proxy mode upgrade).
	log.Info("Ensuring canal-test namespace is clean...")
	if err := cleanupConnectivityNamespace(ctx, log, client, canalTestNs); err != nil {
		t.Fatalf("failed to clean up %q namespace: %v", canalTestNs, err)
	}

	ns := corev1.Namespace{}
	ns.Name = canalTestNs
	if err := client.Create(ctx, &ns); err != nil {
		t.Fatalf("failed to create %q namespace: %v", canalTestNs, err)
	}
	defer func() {
		if err := cleanupConnectivityNamespace(ctx, log, client, canalTestNs); err != nil {
			log.Errorw("failed to clean up connectivity-test namespace", zap.Error(err))
		}
	}()

	log = log.With("namespace", canalTestNs)
	log.Debug("Namespace created")

	installConnectivityCheck(ctx, t, log, client)

	log.Info("Waiting for connectivity-check pods to get ready...")
	err = waitForPods(ctx, t, log, client, canalTestNs, "name", []string{
		"echo-a",
		"echo-b",
		"echo-b-host",
		"host-to-b-multi-node-clusterip",
		"host-to-b-multi-node-headless",
		"pod-to-a",
		"pod-to-a-allowed-knp",
		"pod-to-a-denied-knp",
		"pod-to-b-multi-node-clusterip",
		"pod-to-b-multi-node-headless",
		"pod-to-b-multi-node-nodeport",
		"pod-to-b-intra-node-nodeport",
		"pod-to-external-1111",
		"pod-to-external-fqdn-google",
	})
	if err != nil {
		t.Fatalf("connectivity-check pods never became ready: %v", err)
	}

	log.Info("Checking for the test nginx ApplicationInstallation...")
	err = wait.PollLog(ctx, log, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		app := &appskubermaticv1.ApplicationInstallation{}
		if err := client.Get(context.Background(), types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: nginxAppName}, app); err != nil {
			return fmt.Errorf("failed to get nginx ApplicationInstallation in user cluster: %w", err), nil
		}
		if app.Status.ApplicationVersion == nil {
			return fmt.Errorf("application not yet installed: %v", app.Status), nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Application observe test failed: %v", err)
	}

	log.Info("Canal cluster tests passed.")
}

// installConnectivityCheck deploys echo servers, client pods, and services
// from the testdata manifests into the canal-test namespace.
func installConnectivityCheck(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	objs, err := resourcesFromYaml("./testdata/connectivity-check.yaml")
	if err != nil {
		t.Fatalf("failed to read connectivity-check manifests: %v", err)
	}

	for _, obj := range objs {
		obj.SetNamespace(canalTestNs)
		if err := client.Create(ctx, obj); err != nil {
			t.Fatalf("failed to apply resource: %v", err)
		}

		log.Debugw("Created object", "kind", obj.GetObjectKind(), "name", obj.GetName())
	}
}

// verifyKubeProxyConfigHash checks that the kube-proxy DaemonSet pod template
// has the checksum/config annotation set (non-empty), which ensures pods will
// be restarted when the kube-proxy ConfigMap changes.
func verifyKubeProxyConfigHash(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	return wait.PollLog(ctx, log, 5*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		ds := &appsv1.DaemonSet{}
		if err := client.Get(ctx, types.NamespacedName{Name: "kube-proxy", Namespace: "kube-system"}, ds); err != nil {
			return fmt.Errorf("failed to get kube-proxy DaemonSet: %w", err), nil
		}

		annotations := ds.Spec.Template.Annotations
		if annotations == nil {
			return errors.New("kube-proxy DaemonSet pod template has no annotations"), nil
		}

		hash, ok := annotations["checksum/config"]
		if !ok {
			return errors.New("kube-proxy DaemonSet pod template is missing checksum/config annotation"), nil
		}

		if hash == "" {
			return errors.New("kube-proxy DaemonSet checksum/config annotation is empty"), nil
		}

		log.Infow("kube-proxy config hash verified", "hash", hash)
		return nil, nil
	})
}

// changeProxyMode updates the proxy mode on the cluster object in the seed,
// using the unsafe-cni-migration label to bypass the immutability check.
func changeProxyMode(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, testJig *jig.TestJig, newMode string) error {
	return wait.PollLog(ctx, log, 2*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		cluster, err := testJig.ClusterJig.Cluster(ctx)
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err), nil
		}

		oldCluster := cluster.DeepCopy()

		// Set the unsafe-cni-migration label to allow proxy mode change.
		if cluster.Labels == nil {
			cluster.Labels = map[string]string{}
		}
		cluster.Labels["unsafe-cni-migration"] = "true"
		cluster.Spec.ClusterNetwork.ProxyMode = newMode

		if err := seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("failed to patch cluster proxy mode to %s: %w", newMode, err), nil
		}

		log.Infow("Proxy mode changed", "newMode", newMode)

		// Remove the unsafe label after the change.
		patched := cluster.DeepCopy()
		delete(patched.Labels, "unsafe-cni-migration")
		if err := seedClient.Patch(ctx, patched, ctrlruntimeclient.MergeFrom(cluster)); err != nil {
			return fmt.Errorf("failed to remove unsafe-cni-migration label: %w", err), nil
		}

		return nil, nil
	})
}

// waitForKubeProxyRollout waits for the kube-proxy DaemonSet to complete its
// rollout after a proxy mode change. It checks that all pods are updated and ready.
func waitForKubeProxyRollout(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	return wait.PollLog(ctx, log, 5*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		ds := &appsv1.DaemonSet{}
		if err := client.Get(ctx, types.NamespacedName{Name: "kube-proxy", Namespace: "kube-system"}, ds); err != nil {
			return fmt.Errorf("failed to get kube-proxy DaemonSet: %w", err), nil
		}

		if ds.Status.DesiredNumberScheduled == 0 {
			return errors.New("kube-proxy DaemonSet has 0 desired pods"), nil
		}

		if ds.Status.UpdatedNumberScheduled != ds.Status.DesiredNumberScheduled {
			return fmt.Errorf("kube-proxy rollout in progress: %d/%d updated",
				ds.Status.UpdatedNumberScheduled, ds.Status.DesiredNumberScheduled), nil
		}

		if ds.Status.NumberReady != ds.Status.DesiredNumberScheduled {
			return fmt.Errorf("kube-proxy pods not all ready: %d/%d ready",
				ds.Status.NumberReady, ds.Status.DesiredNumberScheduled), nil
		}

		if ds.Status.ObservedGeneration < ds.Generation {
			return fmt.Errorf("kube-proxy DaemonSet generation not yet observed: %d < %d",
				ds.Status.ObservedGeneration, ds.Generation), nil
		}

		log.Infow("kube-proxy rollout complete",
			"ready", ds.Status.NumberReady,
			"desired", ds.Status.DesiredNumberScheduled)
		return nil, nil
	})
}

// cleanupConnectivityNamespace deletes the given namespace if it exists and
// waits for it to be fully removed. Safe to call when the namespace does not exist.
func cleanupConnectivityNamespace(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, namespace string) error {
	ns := &corev1.Namespace{}
	if err := client.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			return nil // nothing to clean up
		}
		return fmt.Errorf("failed to check namespace existence: %w", err)
	}

	// Namespace exists — delete it if not already terminating.
	if ns.Status.Phase != corev1.NamespaceTerminating {
		if err := client.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete namespace %s: %w", namespace, err)
		}
	}

	// Wait for the namespace to be fully removed.
	return waitForNamespaceDeletion(ctx, log, client, namespace)
}

// waitForNamespaceDeletion waits until the given namespace no longer exists.
// Returns immediately if the namespace is already gone.
func waitForNamespaceDeletion(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, namespace string) error {
	return wait.PollLog(ctx, log, 2*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		ns := &corev1.Namespace{}
		err := client.Get(ctx, types.NamespacedName{Name: namespace}, ns)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil // namespace is gone
			}
			return fmt.Errorf("failed to get namespace %s: %w", namespace, err), nil
		}

		return fmt.Errorf("namespace %s still exists (phase: %s)", namespace, ns.Status.Phase), nil
	})
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

func getTestApplicationAnnotation(appName string) ([]byte, error) {
	var values json.RawMessage
	err := json.Unmarshal([]byte(`{"controller":{"ingressClass":"test-nginx"}}`), &values)
	if err != nil {
		return nil, err
	}

	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name: appName,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name: metav1.NamespaceSystem,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    appName,
				Version: "1.8.1",
			},
			Values: values,
		},
	}
	applications := []apiv1.Application{app}
	data, err := json.Marshal(applications)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// createUserCluster creates a user cluster on AWS with Canal CNI.
func createUserCluster(
	ctx context.Context,
	t *testing.T,
	log *zap.SugaredLogger,
	masterClient ctrlruntimeclient.Client,
	proxyMode string,
) (*jig.TestJig, ctrlruntimeclient.Client, func(), *zap.SugaredLogger, error) {
	testAppAnnotation, err := getTestApplicationAnnotation(nginxAppName)
	if err != nil {
		return nil, nil, nil, log, fmt.Errorf("failed to prepare test application: %w", err)
	}

	testJig := jig.NewAWSCluster(masterClient, log, credentials, 2, nil)
	testJig.ProjectJig.WithHumanReadableName(projectName)
	testJig.ClusterJig.
		WithTestName("canal").
		WithProxyMode(proxyMode).
		WithKonnectivity(true).
		WithCNIPlugin(&kubermaticv1.CNIPluginSettings{
			Type:    kubermaticv1.CNIPluginTypeCanal,
			Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
		}).
		WithAnnotations(map[string]string{
			kubermaticv1.InitialApplicationInstallationsRequestAnnotation: string(testAppAnnotation),
		})

	cleanup := func() {
		testJig.Cleanup(ctx, t, true)
	}

	// let the magic happen
	if _, _, err := testJig.Setup(ctx, jig.WaitForReadyPods); err != nil {
		return nil, nil, cleanup, log, fmt.Errorf("failed to setup test environment: %w", err)
	}

	clusterClient, err := testJig.ClusterClient(ctx)

	return testJig, clusterClient, cleanup, log, err
}
