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

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	imagePullSecret = `{"auths":{"your.private.registry.example.com":{"username":"janedoe","password":"xxxxxxxxxxx","email":"jdoe@example.com","auth":"c3R...zE2"}}}`
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(certmanagerv1alpha2.AddToScheme(scheme.Scheme))
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
}

func TestBasicReconciling(t *testing.T) {
	now := metav1.NewTime(time.Now())

	allSeeds := map[string]*kubermaticv1.Seed{
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
	}

	type testcase struct {
		name            string
		seedToReconcile string
		configuration   *operatorv1alpha1.KubermaticConfiguration
		seedsOnMaster   []string
		syncedSeeds     sets.String // seeds where the seed-sync-controller copied the Seed CR over already
		assertion       func(test *testcase, reconciler *Reconciler) error
	}

	tests := []testcase{
		{
			name:            "finalizer is set on Seed copy",
			seedToReconcile: "europe",
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"europe"},
			syncedSeeds:   sets.NewString("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				seed := kubermaticv1.Seed{}
				if err := reconciler.seedClients["europe"].Get(reconciler.ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "europe",
				}, &seed); err != nil {
					return fmt.Errorf("failed to retrieve Seed: %v", err)
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
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"goner"},
			syncedSeeds:   sets.NewString("goner"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				seed := kubermaticv1.Seed{}
				if err := reconciler.seedClients["goner"].Get(reconciler.ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "goner",
				}, &seed); err != nil {
					return fmt.Errorf("failed to retrieve Seed: %v", err)
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
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"europe"},
			syncedSeeds:   sets.NewString("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				seedClient := reconciler.seedClients["europe"]

				// assert that cluster-wide resources exist
				crbs := rbacv1.ClusterRoleBindingList{}
				must(t, seedClient.List(reconciler.ctx, &crbs))
				if len(crbs.Items) == 0 {
					return errors.New("Seed should have ClusterRoleBindings, but has none")
				}

				hooks := admissionregistrationv1.ValidatingWebhookConfigurationList{}
				must(t, seedClient.List(reconciler.ctx, &hooks))
				if len(hooks.Items) == 0 {
					return errors.New("Seed should have ValidatingWebhookConfigurations, but has none")
				}

				// and now delete the Seed on the seed cluster
				seedName := types.NamespacedName{Namespace: "kubermatic", Name: "europe"}

				seed := &kubermaticv1.Seed{}
				must(t, seedClient.Get(reconciler.ctx, seedName, seed))
				seed.DeletionTimestamp = &now
				must(t, seedClient.Update(reconciler.ctx, seed))

				// let the controller clean up
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				// all global resources should be gone
				crbs = rbacv1.ClusterRoleBindingList{}
				must(t, seedClient.List(reconciler.ctx, &crbs))
				if length := len(crbs.Items); length > 0 {
					return fmt.Errorf("Seed should have no ClusterRoleBindings left over, but has %d", length)
				}

				hooks = admissionregistrationv1.ValidatingWebhookConfigurationList{}
				must(t, seedClient.List(reconciler.ctx, &hooks))
				if length := len(hooks.Items); length > 0 {
					return fmt.Errorf("Seed should have no ValidatingWebhookConfigurations left over, but has %d", length)
				}

				return nil
			},
		},

		{
			name:            "seeds in other namespaces are ignored",
			seedToReconcile: "other",
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"other"},
			syncedSeeds:   sets.NewString("other"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				// The controller should never attempt to reconcile the Seed, so removing the
				// seed client should not hurt it.
				reconciler.seedClients = map[string]ctrlruntimeclient.Client{}

				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				return nil
			},
		},

		{
			name:            "seeds without KubermaticConfiguration are ignored",
			seedToReconcile: "europe",
			configuration:   nil,
			seedsOnMaster:   []string{"europe"},
			syncedSeeds:     sets.NewString("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				return nil
			},
		},

		{
			name:            "nodeport-proxy annotations are carried over to the loadbalancer service",
			seedToReconcile: "seed-with-nodeport-proxy-annotations",
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"seed-with-nodeport-proxy-annotations"},
			syncedSeeds:   sets.NewString("seed-with-nodeport-proxy-annotations"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				seedClient := reconciler.seedClients["seed-with-nodeport-proxy-annotations"]

				svc := corev1.Service{}
				if err := seedClient.Get(reconciler.ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "nodeport-proxy",
				}, &svc); err != nil {
					return fmt.Errorf("failed to retrieve nodeport-proxy Service: %v", err)
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
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					ImagePullSecret: imagePullSecret,
				},
			},
			seedsOnMaster: []string{"europe"},
			syncedSeeds:   sets.NewString("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				seedClient := reconciler.seedClients["europe"]

				// check that secret with image pull secret has been created
				secret := corev1.Secret{}
				if err := seedClient.Get(reconciler.ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      common.DockercfgSecretName,
				}, &secret); err != nil {
					return fmt.Errorf("failed to retrieve dockercfg Secret: %v", err)
				}

				// secret data is not base64 encoded with fake client
				if i := string(secret.Data[corev1.DockerConfigJsonKey]); i != imagePullSecret {
					return fmt.Errorf("secret data expected %q but got %q", imagePullSecret, i)
				}

				// check that image pull secret has been inserted in the pod
				// spec of seed controller manager
				scm := appsv1.Deployment{}
				if err := seedClient.Get(reconciler.ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      common.SeedControllerManagerDeploymentName,
				}, &scm); err != nil {
					return fmt.Errorf("failed to retrieve seed controller manager deployment: %v", err)
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

func createTestReconciler(allSeeds map[string]*kubermaticv1.Seed, cfg *operatorv1alpha1.KubermaticConfiguration, seeds []string, syncedSeeds sets.String) *Reconciler {
	masterObjects := []runtime.Object{}
	if cfg != nil {
		masterObjects = append(masterObjects, cfg)
	}

	masterSeeds := map[string]*kubermaticv1.Seed{} // makes the seedsGetter implementation easier
	seedObjects := map[string][]runtime.Object{}
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

		seedObjects[seedName] = []runtime.Object{}
		if syncedSeeds.Has(seedName) {
			// make sure to put a copy of the Seed CR into the seed "cluster"
			seedObjects[seedName] = append(seedObjects[seedName], masterSeed.DeepCopy())
		}

		seedClients[seedName] = ctrlruntimefakeclient.NewFakeClient(seedObjects[seedName]...)
		seedRecorders[seedName] = record.NewFakeRecorder(999)
	}

	masterClient := ctrlruntimefakeclient.NewFakeClient(masterObjects...)
	masterRecorder := record.NewFakeRecorder(999)

	versions := kubermatic.NewDefaultVersions()
	versions.Kubermatic = "latest"
	versions.UI = "latest"

	seedsGetter := func() (map[string]*kubermaticv1.Seed, error) {
		return masterSeeds, nil
	}

	return &Reconciler{
		ctx:            context.Background(),
		log:            kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
		scheme:         scheme.Scheme,
		namespace:      "kubermatic",
		masterClient:   masterClient,
		masterRecorder: masterRecorder,
		seedClients:    seedClients,
		seedRecorders:  seedRecorders,
		seedsGetter:    seedsGetter,
		versions:       versions,
	}
}
