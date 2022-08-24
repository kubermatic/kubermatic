//go:build integration

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
package applicationinstallationcontroller

import (
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1
)

var _ = Describe("application Installation controller", func() {
	Context("when an application is created", func() {
		It("should update application.Status with applicationVersion and install application", func() {
			appDefName := "app-def-1"
			appInstallName := "app-1"

			def := createApplicationDef(appDefName)
			Expect(userClient.Create(ctx, genApplicationInstallation(appInstallName, appDefName, "1.0.0"))).To(Succeed())

			app := &appskubermaticv1.ApplicationInstallation{}
			Eventually(func() bool {
				if err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, app); err != nil {
					return false
				}
				return app.Status.ApplicationVersion != nil
			}, timeout, interval).Should(BeTrue())

			By("check status is updated with applicationVersion")
			Expect(*app.Status.ApplicationVersion).To(Equal(def.Spec.Versions[0]))

			expectApplicationInstalledWithVersion(app.Name, def.Spec.Versions[0])
			expectStatusHasConditions(app.Name)
		})
	})

	Context("when creating an application that references an ApplicationDefinton that does not exist", func() {
		It("nothing should happen", func() {
			appDefName := "app-def-2"
			appInstallName := "app-2"

			createApplicationDef(appDefName)
			Expect(userClient.Create(ctx, genApplicationInstallation(appInstallName, "app-def-not-exist", "1.0.0"))).To(Succeed())

			By("wait for application to be created")
			app := appskubermaticv1.ApplicationInstallation{}
			Eventually(func() bool {
				err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app)
				return err == nil
			}, 3*time.Second, interval).Should(BeTrue())

			By("ensure application is not deleted")
			Consistently(func() bool {
				err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("application has not been installed")
			_, found := applicationInstallerRecorder.ApplyEvents.Load(appInstallName)
			Expect(found).To(BeFalse())
		})
	})

	Context("when creating an application that references an applicationVersion that does not exist", func() {
		It("nothing should happen", func() {
			appDefName := "app-def-3"
			appInstallName := "app-3"

			createApplicationDef(appDefName)
			Expect(userClient.Create(ctx, genApplicationInstallation(appInstallName, appDefName, "1.0.0-not-exist"))).To(Succeed())

			By("wait for application to be created")
			app := appskubermaticv1.ApplicationInstallation{}
			Eventually(func() bool {
				err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app)
				return err == nil
			}, 3*time.Second, interval).Should(BeTrue())

			By("ensure application is not deleted")
			Consistently(func() bool {
				err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &app)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("application has not been installed")
			_, found := applicationInstallerRecorder.ApplyEvents.Load(appInstallName)
			Expect(found).To(BeFalse())
		})
	})

	Context("when an applicationDefinition is removed", func() {
		It("should remove the application using this ApplicationDefinition", func() {
			appDefName := "app-def-5"
			appInstallName := "app-5"

			def := createApplicationDef(appDefName)
			Expect(userClient.Create(ctx, genApplicationInstallation(appInstallName, appDefName, "1.0.0"))).To(Succeed())

			By("wait for application to be created")
			app := &appskubermaticv1.ApplicationInstallation{}
			Eventually(func() bool {
				if err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, app); err != nil {
					return false
				}
				return app.Status.ApplicationVersion != nil
			}, timeout, interval).Should(BeTrue())

			expectApplicationInstalledWithVersion(appInstallName, def.Spec.Versions[0])

			By("removing applicationDefinition")
			Expect(userClient.Delete(ctx, def)).To(Succeed())

			By("Checking application Installation CR is removed")
			Eventually(func() bool {
				err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &appskubermaticv1.ApplicationInstallation{})

				return err != nil && apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			expectApplicationUninstalledWithVersion(appInstallName, def.Spec.Versions[0])
		})
	})

	Context("when an applicationVersion is removed", func() {
		It("should remove the application using this appVersion", func() {
			appDefName := "app-def-4"
			appInstallName := "app-4"

			def := createApplicationDef(appDefName)
			Expect(userClient.Create(ctx, genApplicationInstallation(appInstallName, appDefName, "1.0.0"))).To(Succeed())

			By("wait for application to be created")
			app := &appskubermaticv1.ApplicationInstallation{}
			Eventually(func() bool {
				if err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, app); err != nil {
					return false
				}
				return app.Status.ApplicationVersion != nil
			}, timeout, interval).Should(BeTrue())

			expectApplicationInstalledWithVersion(appInstallName, def.Spec.Versions[0])

			previousVersion := def.Spec.Versions[0]

			By("removing applicationVersion from applicationDefinition")
			def.Spec.Versions = []appskubermaticv1.ApplicationVersion{
				{
					Version: appskubermaticv1.Version{Version: *semverlib.MustParse("3.0.0")},
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "http://helmrepo.local",
								ChartName:    "someChartName",
								ChartVersion: "12",
								Credentials:  nil,
							},
						},
					},
				}}
			Expect(userClient.Update(ctx, def)).To(Succeed())

			By("Checking application Installation CR is removed")
			Eventually(func() bool {
				err := userClient.Get(ctx, types.NamespacedName{Name: appInstallName, Namespace: applicationNamespace}, &appskubermaticv1.ApplicationInstallation{})

				return err != nil && apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			expectApplicationUninstalledWithVersion(appInstallName, previousVersion)
		})
	})

})

func createApplicationDef(appDefName string) *appskubermaticv1.ApplicationDefinition {
	Expect(userClient.Create(ctx, genApplicationDefinition(appDefName))).To(Succeed())

	def := &appskubermaticv1.ApplicationDefinition{Spec: appskubermaticv1.ApplicationDefinitionSpec{Method: appskubermaticv1.HelmTemplateMethod}}
	EventuallyWithOffset(1, func(g Gomega) {
		g.Expect(userClient.Get(ctx, types.NamespacedName{Name: appDefName}, def)).To(Succeed())
	}, timeout, interval).Should(Succeed())

	return def
}

func expectApplicationInstalledWithVersion(appName string, expectedVersion appskubermaticv1.ApplicationVersion) {
	By("check application has been installed")
	EventuallyWithOffset(1, func(g Gomega) {
		_, found := applicationInstallerRecorder.DownloadEvents.Load(appName)
		g.Expect(found).To(BeTrue(), "Application "+appName+"'s sources have not been installed")

		result, found := applicationInstallerRecorder.ApplyEvents.Load(appName)
		g.Expect(found).To(BeTrue(), "Application "+appName+" has not been installed")

		currentVersion := result.(appskubermaticv1.ApplicationInstallation)
		g.Expect(*currentVersion.Status.ApplicationVersion).To(Equal(expectedVersion))
	}, timeout, interval).Should(Succeed())
}

func expectApplicationUninstalledWithVersion(appName string, expectedVersion appskubermaticv1.ApplicationVersion) {
	By("Checking application Installation has been uninstalled")
	EventuallyWithOffset(1, func(g Gomega) {
		result, found := applicationInstallerRecorder.DeleteEvents.Load(appName)
		g.Expect(found).To(BeTrue(), "Application "+appName+" has not been uninstalled")

		currentVersion := result.(appskubermaticv1.ApplicationInstallation)
		g.Expect(*currentVersion.Status.ApplicationVersion).To(Equal(expectedVersion))
	}, timeout, interval).Should(Succeed())
}

func expectStatusHasConditions(appName string) {
	app := &appskubermaticv1.ApplicationInstallation{}
	EventuallyWithOffset(1, func(g Gomega) {
		g.Expect(userClient.Get(ctx, types.NamespacedName{Name: appName, Namespace: applicationNamespace}, app)).To(Succeed())

		g.Expect(app.Status.Conditions).To(HaveKey(appskubermaticv1.ManifestsRetrieved))
		g.Expect(app.Status.Conditions[appskubermaticv1.ManifestsRetrieved].Status).To(Equal(corev1.ConditionTrue))

		g.Expect(app.Status.Conditions).To(HaveKey(appskubermaticv1.Ready))
		g.Expect(app.Status.Conditions[appskubermaticv1.Ready].Status).To(Equal(corev1.ConditionTrue))
	}, timeout, interval).Should(Succeed())
}
