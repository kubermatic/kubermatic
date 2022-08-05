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
	"context"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/onsi/gomega"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(appskubermaticv1.AddToScheme(scheme.Scheme))
}

const (
	applicationNamespace = "apps"
)

func TestEnqueueApplicationInstallation(t *testing.T) {
	testCases := []struct {
		name                      string
		applicationDefinition     *appskubermaticv1.ApplicationDefinition
		userClient                ctrlruntimeclient.Client
		expectedReconcileRequests []reconcile.Request
	}{
		{
			name:                  "scenario 1: only applications that reference ApplicationDef 'app-def-1' are enqueued",
			applicationDefinition: genApplicationDefinition("app-def-1"),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", "app-def-1", "1.0.0"),
					genApplicationInstallation("appInstallation-2", "app-def-2", "1.0.0"),
					genApplicationInstallation("appInstallation-3", "app-def-1", "1.0.0")).
				Build(),
			expectedReconcileRequests: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespace}},
				{NamespacedName: types.NamespacedName{Name: "appInstallation-3", Namespace: applicationNamespace}},
			},
		},
		{
			name:                  "scenario 2: when no application reference ApplicationDef 'app-def-1', nothing is enqueued",
			applicationDefinition: genApplicationDefinition("app-def-1"),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", "app-def-2", "1.0.0"),
					genApplicationInstallation("appInstallation-2", "app-def-3", "1.0.0"),
					genApplicationInstallation("appInstallation-3", "app-def-4", "1.0.0")).
				Build(),
			expectedReconcileRequests: []reconcile.Request{},
		},
		{
			name:                  "scenario 3: when no application in cluster, nothing is enqueued",
			applicationDefinition: genApplicationDefinition("app-def-1"),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				Build(),
			expectedReconcileRequests: []reconcile.Request{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			enqueueApplicationInstallationFunc := enqueueAppInstallationForAppDef(context.Background(), tc.userClient)
			actual := enqueueApplicationInstallationFunc(tc.applicationDefinition)

			g.Expect(actual).Should(gomega.ConsistOf(tc.expectedReconcileRequests))
		})
	}
}

func genApplicationDefinition(name string) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Method: appskubermaticv1.HelmTemplateMethod,
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "1.0.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "http://helmrepo.local",
								ChartName:    "someChartName",
								ChartVersion: "12",
								Credentials:  nil,
							},
						},
						FormSpec: nil,
					},
				},
				{
					Version: "2.0.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Git: &appskubermaticv1.GitSource{
								Remote:      "git@somerepo.local",
								Ref:         appskubermaticv1.GitReference{Tag: "v13"},
								Path:        "/",
								Credentials: nil,
							},
						},
						FormSpec: nil,
					},
				},
			},
		},
	}
}

func genApplicationInstallation(name string, applicationDefName string, appVersion string) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: applicationNamespace,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.NamespaceSpec{
				Name:   "default",
				Create: false,
			},

			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    applicationDefName,
				Version: appskubermaticv1.Version{Version: *semverlib.MustParse(appVersion)},
			},
		},
		Status: appskubermaticv1.ApplicationInstallationStatus{},
	}
}
