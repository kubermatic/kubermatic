//go:build integration

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

package template

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	. "k8c.io/kubermatic/v2/pkg/test/gomegautil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1
)

var _ = Describe("helm template", func() {
	const chartLoc = "../../helmclient/testdata/examplechart"

	// Name of the config map deployed by the chart.
	const configmapName = "testcm"

	// Key in the values.yaml that holds custom configmap data.
	const cmDataKey = "cmData"

	// Key in the values.yaml that holds custom version label value. it's also the name of the label in the configmap.
	const versionLabelKey = "versionLabel"

	defaultCmData := map[string]string{"foo": "bar"}
	defaultVersionLabel := "1.0"

	var helmCacheDir string
	var testNs *corev1.Namespace
	var app *appskubermaticv1.ApplicationInstallation

	BeforeEach(func() {
		var err error
		helmCacheDir, err = os.MkdirTemp("", "helm-template-test")
		Expect(err).NotTo(HaveOccurred())

		testNs, err = createNamespace(ctx, userClient, "testns")
		Expect(err).ToNot(HaveOccurred())

		app = &appskubermaticv1.ApplicationInstallation{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "app1",
			},
			Spec: appskubermaticv1.ApplicationInstallationSpec{
				Namespace: appskubermaticv1.NamespaceSpec{
					Name: testNs.Name,
				},
				Values: runtime.RawExtension{},
			},
			Status: appskubermaticv1.ApplicationInstallationStatus{
				Method: appskubermaticv1.HelmTemplateMethod,
				ApplicationVersion: &appskubermaticv1.ApplicationVersion{
					Version: "0.1.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "localhost",
								ChartName:    "example",
								ChartVersion: "0.1.0",
							},
						},
					},
				},
			},
		}
	})

	AfterEach(func() {
		os.Remove(helmCacheDir)
		Expect(userClient.Delete(ctx, testNs)).To(Succeed())
	})

	Context("when an application is created with no values", func() {
		It("should install application into user cluster", func() {
			By("installing chart")
			template := HelmTemplate{
				Ctx:                     context.Background(),
				Kubeconfig:              testEnvKubeConfig,
				CacheDir:                helmCacheDir,
				Log:                     kubermaticlog.Logger,
				ApplicationInstallation: app,
				SecretNamespace:         "abc",
				SeedClient:              userClient,
			}

			statusUpdater, err := template.InstallOrUpgrade(chartLoc, app)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: testNs.Name, Name: configmapName}, cm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("creating the config map with default data defined in values.yaml")
			Expect(cm.Data).To(SemanticallyEqual(defaultCmData))

			By("creating the config map with default version label defined in values.yaml")
			Expect(cm.Labels[versionLabelKey]).To(Equal(defaultVersionLabel))

			By("status should be updated")
			assertStatusIsUpdated(app, statusUpdater, 1)
		})
	})

	Context("when an application is created with customCmData", func() {
		It("should install application into user cluster", func() {
			By("installing chart")
			customCmData := map[string]string{"hello": "world", "a": "b"}
			rawValues := toHelmRawValues(cmDataKey, customCmData)
			app.Spec.Values.Raw = rawValues

			template := HelmTemplate{
				Ctx:                     context.Background(),
				Kubeconfig:              testEnvKubeConfig,
				CacheDir:                helmCacheDir,
				Log:                     kubermaticlog.Logger,
				ApplicationInstallation: app,
				SecretNamespace:         "abc",
				SeedClient:              userClient,
			}

			statusUpdater, err := template.InstallOrUpgrade(chartLoc, app)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: testNs.Name, Name: configmapName}, cm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("creating the configmap with data merged from customCmData and default data defined in values.yaml")
			appendDefaultValues(customCmData, defaultCmData)
			Expect(cm.Data).To(SemanticallyEqual(customCmData))

			By("creating the configmap with default version label defined in values.yaml")
			Expect(cm.Labels[versionLabelKey]).To(Equal(defaultVersionLabel))

			By("status should be updated")
			assertStatusIsUpdated(app, statusUpdater, 1)
		})
	})

	Context("when an application is created with custom versionLabel", func() {
		It("should install application into user cluster", func() {
			By("installing chart")
			customVersionLabel := "1.2.3"
			rawValues := toHelmRawValues(versionLabelKey, customVersionLabel)
			app.Spec.Values.Raw = rawValues

			template := HelmTemplate{
				Ctx:                     context.Background(),
				Kubeconfig:              testEnvKubeConfig,
				CacheDir:                helmCacheDir,
				Log:                     kubermaticlog.Logger,
				ApplicationInstallation: app,
				SecretNamespace:         "abc",
				SeedClient:              userClient,
			}

			statusUpdater, err := template.InstallOrUpgrade(chartLoc, app)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: testNs.Name, Name: configmapName}, cm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("creating the configmap with default data defined in values.yaml")
			Expect(cm.Data).To(SemanticallyEqual(defaultCmData))

			By("creating the configmap with label versionLabel equal to custom versionLabel")
			Expect(cm.Labels[versionLabelKey]).To(Equal(customVersionLabel))

			By("status should be updated")
			assertStatusIsUpdated(app, statusUpdater, 1)
		})
	})

	Context("when an application is updated with customCmData", func() {
		It("should update application into user cluster with new data", func() {
			By("installing chart")
			customCmData := map[string]string{"hello": "world", "a": "b"}
			app.Spec.Values.Raw = toHelmRawValues(cmDataKey, customCmData)

			template := HelmTemplate{
				Ctx:                     context.Background(),
				Kubeconfig:              testEnvKubeConfig,
				CacheDir:                helmCacheDir,
				Log:                     kubermaticlog.Logger,
				ApplicationInstallation: app,
				SecretNamespace:         "abc",
				SeedClient:              userClient,
			}

			statusUpdater, err := template.InstallOrUpgrade(chartLoc, app)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: testNs.Name, Name: configmapName}, cm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("creating the configmap with data merged from customCmData and default data defined in values.yaml")
			appendDefaultValues(customCmData, defaultCmData)
			Expect(cm.Data).To(SemanticallyEqual(customCmData))

			By("creating the configmap with default version label defined in values.yaml")
			Expect(cm.Labels[versionLabelKey]).To(Equal(defaultVersionLabel))

			By("status should be updated")
			assertStatusIsUpdated(app, statusUpdater, 1)

			By("updating application")
			newCustomCmData := map[string]string{"c": "d", "e": "f"}
			app.Spec.Values.Raw = toHelmRawValues(cmDataKey, newCustomCmData)

			template = HelmTemplate{
				Ctx:                     context.Background(),
				Kubeconfig:              testEnvKubeConfig,
				CacheDir:                helmCacheDir,
				Log:                     kubermaticlog.Logger,
				ApplicationInstallation: app,
				SecretNamespace:         "abc",
				SeedClient:              userClient,
			}

			statusUpdater, err = template.InstallOrUpgrade(chartLoc, app)
			Expect(err).NotTo(HaveOccurred())

			By("configmap should be updated with new data")
			appendDefaultValues(newCustomCmData, defaultCmData)
			Eventually(func() map[string]string {
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: testNs.Name, Name: configmapName}, cm)
				if err != nil {
					return nil
				}
				return cm.Data
			}, timeout, interval).Should(SemanticallyEqual(newCustomCmData))

			By("status should be update with new version")
			assertStatusIsUpdated(app, statusUpdater, 2)
		})
	})

	Context("when an application is removed", func() {
		It("should uninstall application of user cluster", func() {
			By("installing chart")
			customCmData := map[string]string{"hello": "world", "a": "b"}
			app.Spec.Values.Raw = toHelmRawValues(cmDataKey, customCmData)

			template := HelmTemplate{
				Ctx:                     context.Background(),
				Kubeconfig:              testEnvKubeConfig,
				CacheDir:                helmCacheDir,
				Log:                     kubermaticlog.Logger,
				ApplicationInstallation: app,
				SecretNamespace:         "abc",
				SeedClient:              userClient,
			}

			statusUpdater, err := template.InstallOrUpgrade(chartLoc, app)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: testNs.Name, Name: configmapName}, cm)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("creating the configmap with data merged from customCmData and default data defined in values.yaml")
			appendDefaultValues(customCmData, defaultCmData)
			Expect(cm.Data).To(SemanticallyEqual(customCmData))

			By("creating the configmap with default version label defined in values.yaml")
			Expect(cm.Labels[versionLabelKey]).To(Equal(defaultVersionLabel))

			By("status should be updated")
			assertStatusIsUpdated(app, statusUpdater, 1)

			By("unsintalling chart")

			template = HelmTemplate{
				Ctx:                     context.Background(),
				Kubeconfig:              testEnvKubeConfig,
				CacheDir:                helmCacheDir,
				Log:                     kubermaticlog.Logger,
				ApplicationInstallation: app,
				SecretNamespace:         "abc",
				SeedClient:              userClient,
			}

			statusUpdater, err = template.Uninstall(app)
			Expect(err).NotTo(HaveOccurred())

			By("configmap should be removed")
			Eventually(func() bool {
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: testNs.Name, Name: configmapName}, cm)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			By("status should be updated")
			assertStatusIsUpdated(app, statusUpdater, 1)
		})
	})
})

func assertStatusIsUpdated(app *appskubermaticv1.ApplicationInstallation, statusUpdater util.StatusUpdater, expectedVersion int) {
	statusUpdater(&app.Status)
	ExpectWithOffset(1, app.Status.HelmRelease).NotTo(BeNil())
	ExpectWithOffset(1, app.Status.HelmRelease.Name).To(Equal(getReleaseName(app)), "app.Status.HelmRelease.Name is invalid")
	ExpectWithOffset(1, app.Status.HelmRelease.Version).To(Equal(expectedVersion), "app.Status.HelmRelease.Version is invalid")
	ExpectWithOffset(1, app.Status.HelmRelease.Info).NotTo(BeNil())
}

// appendDefaultValues merges the source with the defaultValues by simply copy key, values of defaultValues into source.
func appendDefaultValues(source map[string]string, defaultValues map[string]string) {
	for k, v := range defaultValues {
		source[k] = v
	}
}

// toHelmRawValues build the helm value map and transforms it to runtime.RawExtension.Raw.
// Key is the key (i.e. name) in the value.yaml and values it's corresponding value.
// example:
// toHelmRawValues("cmData", map[string]string{"hello": "world", "a": "b"}) produces this helm value file
// cmData:
//    hello: world
//    a: b
func toHelmRawValues(key string, values any) []byte {
	helmValues := map[string]any{key: values}
	rawValues, err := json.Marshal(helmValues)
	Expect(err).ShouldNot(HaveOccurred())
	return rawValues
}

// createNamespace creates a namespace with a generated name and returns it.
func createNamespace(ctx context.Context, k8sClient ctrlruntimeclient.Client, generateName string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", generateName),
		},
	}
	if err := k8sClient.Create(ctx, ns); err != nil {
		return nil, err
	}

	return ns, nil
}
