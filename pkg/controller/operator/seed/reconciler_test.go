/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package seed

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	imagePullSecret = `{"auths":{"your.private.registry.example.com":{"username":"janedoe","password":"xxxxxxxxxxx","email":"jdoe@example.com","auth":"c3R...zE2"}}}`
)

var (
	k8cConfig = kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "example.com",
			},
		},
	}
)

var testScheme = fake.NewScheme()

func init() {
	utilruntime.Must(apiextensionsv1.AddToScheme(testScheme))
}

func getSeeds(now metav1.Time) map[string]*kubermaticv1.Seed {
	return map[string]*kubermaticv1.Seed{
		"europe": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "europe",
				Namespace: "kubermatic",
			},
		},
		"asia": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "asia",
				Namespace: "kubermatic",
			},
		},
		"goner": {
			ObjectMeta: metav1.ObjectMeta{
				Name:              "goner",
				Namespace:         "kubermatic",
				DeletionTimestamp: &now,
				Finalizers:        []string{"dummy"},
			},
		},
		"other": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other",
				Namespace: "kube-system",
			},
		},
		"seed-with-nodeport-proxy-annotations": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "seed-with-nodeport-proxy-annotations",
				Namespace: "kubermatic",
			},
			Spec: kubermaticv1.SeedSpec{
				NodeportProxy: kubermaticv1.NodeportProxyConfig{
					Annotations: map[string]string{
						"foo.bar": "baz",
					},
				},
			},
		},
		"seed-with-metering-config": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "seed-with-metering-config",
				Namespace: "kubermatic",
			},
			Spec: kubermaticv1.SeedSpec{
				Metering: &kubermaticv1.MeteringConfiguration{
					Enabled:          true,
					StorageSize:      "10Gi",
					StorageClassName: "test",
					ReportConfigurations: map[string]kubermaticv1.MeteringReportConfiguration{
						"weekly-test": {
							Schedule: "0 1 * * 6",
							Interval: 7,
						},
					},
				},
			},
		},
	}
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
}

func TestBasicReconciling(t *testing.T) {
	now := metav1.NewTime(time.Now())
	allSeeds := getSeeds(now)

	type testcase struct {
		name            string
		seedToReconcile string
		configuration   *kubermaticv1.KubermaticConfiguration
		seedsOnMaster   []string
		syncedSeeds     sets.Set[string] // seeds where the seed-sync-controller copied the Seed CR over already
		assertion       func(test *testcase, reconciler *Reconciler) error
	}

	tests := []testcase{
		{
			name:            "finalizer is set on Seed copy",
			seedToReconcile: "europe",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"europe"},
			syncedSeeds:     sets.New("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seed := kubermaticv1.Seed{}
				if err := reconciler.seedClients["europe"].Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "europe",
				}, &seed); err != nil {
					return fmt.Errorf("failed to retrieve Seed: %w", err)
				}

				if !kubernetes.HasFinalizer(&seed, common.CleanupFinalizer) {
					return fmt.Errorf("Seed copy in seed cluster does not have cleanup finalizer %q", common.CleanupFinalizer)
				}

				return nil
			},
		},

		{
			name:            "finalizer triggers cleanup",
			seedToReconcile: "goner",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"goner"},
			syncedSeeds:     sets.New("goner"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seed := kubermaticv1.Seed{}
				if err := reconciler.seedClients["goner"].Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "goner",
				}, &seed); err != nil {
					return fmt.Errorf("failed to retrieve Seed: %w", err)
				}

				if kubernetes.HasFinalizer(&seed, common.CleanupFinalizer) {
					return fmt.Errorf("Seed copy in seed cluster should not have cleanup finalizer %q anymore", common.CleanupFinalizer)
				}

				return nil
			},
		},

		{
			name:            "all cluster-wide resources are cleaned up when deleting a seed",
			seedToReconcile: "europe",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"europe"},
			syncedSeeds:     sets.New("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seedClient := reconciler.seedClients["europe"]

				// assert that cluster-wide resources exist
				crbs := rbacv1.ClusterRoleBindingList{}
				must(t, seedClient.List(ctx, &crbs))
				if len(crbs.Items) == 0 {
					return errors.New("Seed should have ClusterRoleBindings, but has none")
				}

				hooks := admissionregistrationv1.ValidatingWebhookConfigurationList{}
				must(t, seedClient.List(ctx, &hooks))
				if len(hooks.Items) == 0 {
					return errors.New("Seed should have ValidatingWebhookConfigurations, but has none")
				}

				// and now delete the Seed on the seed cluster
				seedName := types.NamespacedName{Namespace: "kubermatic", Name: "europe"}

				seed := &kubermaticv1.Seed{}
				must(t, seedClient.Get(ctx, seedName, seed))
				must(t, seedClient.Delete(ctx, seed))

				// let the controller clean up
				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				// all global resources should be gone
				crbs = rbacv1.ClusterRoleBindingList{}
				must(t, seedClient.List(ctx, &crbs))
				if length := len(crbs.Items); length > 0 {
					return fmt.Errorf("Seed should have no ClusterRoleBindings left over, but has %d", length)
				}

				hooks = admissionregistrationv1.ValidatingWebhookConfigurationList{}
				must(t, seedClient.List(ctx, &hooks))
				if length := len(hooks.Items); length > 0 {
					return fmt.Errorf("Seed should have no ValidatingWebhookConfigurations left over, but has %d", length)
				}

				return nil
			},
		},

		{
			name:            "seeds in other namespaces are ignored",
			seedToReconcile: "other",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"other"},
			syncedSeeds:     sets.New("other"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				// The controller should never attempt to reconcile the Seed, so removing the
				// seed client should not hurt it.
				reconciler.seedClients = map[string]ctrlruntimeclient.Client{}

				if err := reconciler.reconcile(context.Background(), reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				return nil
			},
		},

		{
			name:            "nodeport-proxy annotations are carried over to the loadbalancer service",
			seedToReconcile: "seed-with-nodeport-proxy-annotations",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"seed-with-nodeport-proxy-annotations"},
			syncedSeeds:     sets.New("seed-with-nodeport-proxy-annotations"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seedClient := reconciler.seedClients["seed-with-nodeport-proxy-annotations"]

				svc := corev1.Service{}
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "nodeport-proxy",
				}, &svc); err != nil {
					return fmt.Errorf("failed to retrieve nodeport-proxy Service: %w", err)
				}

				if svc.Annotations == nil {
					return fmt.Errorf("Nodeport service in seed cluster does not have configured annotations: %q", allSeeds["seed-with-nodeport-proxy-annotations"].Spec.NodeportProxy.Annotations)
				}

				for k, v := range allSeeds["seed-with-nodeport-proxy-annotations"].Spec.NodeportProxy.Annotations {
					if svc.Annotations[k] != v {
						return fmt.Errorf("Nodeport service in seed cluster is missing configured annotation: %s: %s", k, v)
					}
				}

				return nil
			},
		},

		{
			name:            "when imagePullSecret is given secret should be provisioned",
			seedToReconcile: "europe",
			configuration: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					ImagePullSecret: imagePullSecret,
				},
			},
			seedsOnMaster: []string{"europe"},
			syncedSeeds:   sets.New("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seedClient := reconciler.seedClients["europe"]

				// check that secret with image pull secret has been created
				secret := corev1.Secret{}
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      common.DockercfgSecretName,
				}, &secret); err != nil {
					return fmt.Errorf("failed to retrieve dockercfg Secret: %w", err)
				}

				// secret data is not base64 encoded with fake client
				if i := string(secret.Data[corev1.DockerConfigJsonKey]); i != imagePullSecret {
					return fmt.Errorf("secret data expected %q but got %q", imagePullSecret, i)
				}

				// check that image pull secret has been inserted in the pod
				// spec of seed controller manager
				scm := appsv1.Deployment{}
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      common.SeedControllerManagerDeploymentName,
				}, &scm); err != nil {
					return fmt.Errorf("failed to retrieve seed controller manager deployment: %w", err)
				}

				var foundImagePullSecret bool
				for _, ips := range scm.Spec.Template.Spec.ImagePullSecrets {
					if ips.Name == common.DockercfgSecretName {
						foundImagePullSecret = true
					}
				}
				if !foundImagePullSecret {
					return fmt.Errorf("failed to find ImagePullSecret in seed-controller-manager pod spec")
				}

				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reconciler := createTestReconciler(allSeeds, test.configuration, test.seedsOnMaster, test.syncedSeeds)

			if err := test.assertion(&test, reconciler); err != nil {
				t.Fatalf("Failure: %v", err)
			}
		})
	}
}

func createTestReconciler(allSeeds map[string]*kubermaticv1.Seed, cfg *kubermaticv1.KubermaticConfiguration, seeds []string, syncedSeeds sets.Set[string]) *Reconciler {
	masterObjects := []ctrlruntimeclient.Object{}
	if cfg != nil {
		// CABundle is defaulted in reallife scenarios
		defaulted, err := defaulting.DefaultConfiguration(cfg, kubermaticlog.NewDefault().Sugar())
		if err != nil {
			panic(err)
		}

		caBundle := certificates.NewFakeCABundle()

		masterObjects = append(masterObjects, cfg, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaulted.Spec.CABundle.Name,
				Namespace: defaulted.Namespace,
			},
			Data: map[string]string{
				resources.CABundleConfigMapKey: caBundle.String(),
			},
		})
	}

	masterSeeds := map[string]*kubermaticv1.Seed{} // makes the seedsGetter implementation easier
	seedObjects := map[string][]ctrlruntimeclient.Object{}
	seedClients := map[string]ctrlruntimeclient.Client{}
	seedRecorders := map[string]record.EventRecorder{}

	for _, seedName := range seeds {
		masterSeed := allSeeds[seedName].DeepCopy()

		masterObjects = append(masterObjects, masterSeed)

		// the seedsGetter is only returning seeds in its given namespace, so we have to replicate the
		// behaviour here for the dummy seeds getter
		if masterSeed.Namespace == "kubermatic" {
			masterSeeds[seedName] = masterSeed
		}

		seedObjects[seedName] = []ctrlruntimeclient.Object{}
		if syncedSeeds.Has(seedName) {
			// make sure to put a copy of the Seed CR into the seed "cluster"
			seedObjects[seedName] = append(seedObjects[seedName], masterSeed.DeepCopy())
		}

		seedClients[seedName] = fake.
			NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(seedObjects[seedName]...).
			Build()

		seedRecorders[seedName] = record.NewFakeRecorder(999)
	}

	masterClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(masterObjects...).
		Build()

	masterRecorder := record.NewFakeRecorder(999)

	versions := kubermatic.GetVersions()
	versions.KubermaticContainerTag = "latest"
	versions.UIContainerTag = "latest"

	seedsGetter := func() (map[string]*kubermaticv1.Seed, error) {
		return masterSeeds, nil
	}

	return &Reconciler{
		log:                    zap.NewNop().Sugar(),
		scheme:                 scheme.Scheme,
		namespace:              "kubermatic",
		masterClient:           masterClient,
		masterRecorder:         masterRecorder,
		configGetter:           test.NewConfigGetter(cfg),
		seedClients:            seedClients,
		seedRecorders:          seedRecorders,
		initializedSeedsGetter: seedsGetter,
		versions:               versions,
	}
}
