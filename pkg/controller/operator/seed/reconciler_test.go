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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	operatorresources "k8c.io/kubermatic/v3/pkg/controller/operator/seed/resources"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v3/pkg/log"
	kubernetesprovider "k8c.io/kubermatic/v3/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/resources/certificates"
	"k8c.io/kubermatic/v3/pkg/util/edition"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme.Scheme))
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
}

func createTestConfiguration(modifier func(config *kubermaticv1.KubermaticConfiguration)) *kubermaticv1.KubermaticConfiguration {
	config := &kubermaticv1.KubermaticConfiguration{
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

	if modifier != nil {
		modifier(config)
	}

	return config
}

func TestBasicReconciling(t *testing.T) {
	now := metav1.Now()

	type testcase struct {
		name          string
		configuration *kubermaticv1.KubermaticConfiguration
		assertion     func(ctx context.Context, test *testcase, reconciler *Reconciler) error
	}

	tests := []testcase{
		{
			name:          "finalizer is set",
			configuration: createTestConfiguration(nil),
			assertion: func(ctx context.Context, test *testcase, reconciler *Reconciler) error {
				newConfig, err := reconciler.configGetter(context.Background())
				if err != nil {
					t.Fatalf("Failed to get Kubermatic config: %v", err)
				}

				if !kubernetes.HasFinalizer(newConfig, operatorresources.CleanupFinalizer) {
					return fmt.Errorf("Configuration does not have cleanup finalizer %q", operatorresources.CleanupFinalizer)
				}

				return nil
			},
		},

		{
			name: "finalizer triggers cleanup",
			configuration: createTestConfiguration(func(config *kubermaticv1.KubermaticConfiguration) {
				config.DeletionTimestamp = &now
				config.Finalizers = []string{operatorresources.CleanupFinalizer}
			}),
			assertion: func(ctx context.Context, test *testcase, reconciler *Reconciler) error {
				if _, err := reconciler.configGetter(context.Background()); err == nil {
					t.Fatal("Succeeded to get config, but it should have been gone now.")
				}

				return nil
			},
		},

		{
			name:          "all cluster-wide resources are cleaned up when deleting a configuration",
			configuration: createTestConfiguration(nil),
			assertion: func(ctx context.Context, test *testcase, reconciler *Reconciler) error {
				seedClient := reconciler.seedClient

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

				newConfig, err := reconciler.configGetter(context.Background())
				if err != nil {
					t.Fatalf("Failed to get Kubermatic config: %v", err)
				}

				if !kubernetes.HasFinalizer(newConfig, operatorresources.CleanupFinalizer) {
					return fmt.Errorf("Configuration does not have cleanup finalizer %q", operatorresources.CleanupFinalizer)
				}

				// and now delete the configuration
				if err := reconciler.seedClient.Delete(ctx, newConfig); err != nil {
					return fmt.Errorf("failed to delete config: %w", err)
				}

				// let the controller clean up
				if err := reconciler.reconcile(ctx, reconciler.log); err != nil {
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
			name: "nodeport-proxy annotations are carried over to the loadbalancer service",
			configuration: createTestConfiguration(func(config *kubermaticv1.KubermaticConfiguration) {
				config.Spec.NodeportProxy = &kubermaticv1.NodeportProxyConfig{
					Annotations: map[string]string{
						"foo.bar": "baz",
					},
				}
			}),
			assertion: func(ctx context.Context, test *testcase, reconciler *Reconciler) error {
				seedClient := reconciler.seedClient

				svc := corev1.Service{}
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: test.configuration.Namespace,
					Name:      "nodeport-proxy",
				}, &svc); err != nil {
					return fmt.Errorf("failed to retrieve nodeport-proxy Service: %w", err)
				}

				if svc.Annotations == nil {
					return fmt.Errorf("Nodeport service does not have configured annotations: %q", test.configuration.Spec.NodeportProxy.Annotations)
				}

				for k, v := range test.configuration.Spec.NodeportProxy.Annotations {
					if svc.Annotations[k] != v {
						return fmt.Errorf("Nodeport service is missing configured annotation: %s: %s", k, v)
					}
				}

				return nil
			},
		},

		{
			name: "when imagePullSecret is given secret should be provisioned",
			configuration: createTestConfiguration(func(config *kubermaticv1.KubermaticConfiguration) {
				config.Spec.ImagePullSecret = imagePullSecret
			}),
			assertion: func(ctx context.Context, test *testcase, reconciler *Reconciler) error {
				seedClient := reconciler.seedClient

				// check that secret with image pull secret has been created
				secret := corev1.Secret{}
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      operatorresources.DockercfgSecretName,
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
					Name:      operatorresources.SeedControllerManagerDeploymentName,
				}, &scm); err != nil {
					return fmt.Errorf("failed to retrieve seed controller manager deployment: %w", err)
				}

				var foundImagePullSecret bool
				for _, ips := range scm.Spec.Template.Spec.ImagePullSecrets {
					if ips.Name == operatorresources.DockercfgSecretName {
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
			reconciler := createTestReconciler(test.configuration)

			if err := reconciler.reconcile(context.Background(), reconciler.log); err != nil {
				t.Fatalf("Reconciliation failed: %v", err)
			}

			if err := test.assertion(context.Background(), &test, reconciler); err != nil {
				t.Fatalf("Failure: %v", err)
			}
		})
	}
}

func createTestReconciler(config *kubermaticv1.KubermaticConfiguration) *Reconciler {
	seedObjects := []ctrlruntimeclient.Object{}

	// CABundle is defaulted in reallife scenarios
	defaulted, err := defaulting.DefaultConfiguration(config, kubermaticlog.NewDefault().Sugar())
	if err != nil {
		panic(err)
	}

	caBundle := certificates.NewFakeCABundle()

	seedObjects = append(seedObjects, config, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaulted.Spec.CABundle.Name,
			Namespace: defaulted.Namespace,
		},
		Data: map[string]string{
			resources.CABundleConfigMapKey: caBundle.String(),
		},
	})

	seedClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(seedObjects...).
		Build()

	versions := kubermatic.NewDefaultVersions(edition.CommunityEdition)
	versions.Kubermatic = "latest"
	versions.UI = "latest"

	// Do not use a test getter, as we will do assertions on the reconciled state of the
	// config and the test getter would just return the in-memory object.
	configGetter, err := kubernetesprovider.DynamicKubermaticConfigurationGetterFactory(seedClient, config.Namespace)
	if err != nil {
		panic(err)
	}

	return &Reconciler{
		log:          zap.NewNop().Sugar(),
		scheme:       scheme.Scheme,
		namespace:    "kubermatic",
		seedClient:   seedClient,
		seedRecorder: record.NewFakeRecorder(999),
		configGetter: configGetter,
		versions:     versions,
	}
}
