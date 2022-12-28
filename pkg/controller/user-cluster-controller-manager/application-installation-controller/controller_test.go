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
	"errors"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications"
	"k8c.io/kubermatic/v2/pkg/applications/fake"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
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
					genApplicationInstallation("appInstallation-1", "app-def-1", "1.0.0", 0),
					genApplicationInstallation("appInstallation-2", "app-def-2", "1.0.0", 0),
					genApplicationInstallation("appInstallation-3", "app-def-1", "1.0.0", 0)).
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
					genApplicationInstallation("appInstallation-1", "app-def-2", "1.0.0", 0),
					genApplicationInstallation("appInstallation-2", "app-def-3", "1.0.0", 0),
					genApplicationInstallation("appInstallation-3", "app-def-4", "1.0.0", 0)).
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

func TestMaxRetriesOnInstallation(t *testing.T) {
	installError := fmt.Errorf("an install error")

	errorOnInstall := func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error) {
		return util.NoStatusUpdate, installError
	}

	testCases := []struct {
		name                     string
		applicationDefinition    *appskubermaticv1.ApplicationDefinition
		userClient               ctrlruntimeclient.Client
		appInstaller             applications.ApplicationInstaller
		installErr               error
		wantErr                  bool
		expectedFailure          int
		expectedInstallCondition appskubermaticv1.ApplicationInstallationCondition
	}{
		{
			name:                  "installation succeeds",
			applicationDefinition: genApplicationDefinition("app-def-1"),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", "app-def-1", "1.0.0", 0)).
				Build(),
			appInstaller:             fake.ApplicationInstallerLogger{},
			installErr:               nil,
			wantErr:                  false,
			expectedFailure:          0,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionTrue, Reason: "InstallationSuccessful", Message: "application successfully installed or upgraded"},
		},
		{
			name:                  "installation fails: app.Status.Failures should be increased and condition set to false",
			applicationDefinition: genApplicationDefinition("app-def-1"),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", "app-def-1", "1.0.0", 2)).
				Build(),
			appInstaller:             fake.CustomApplicationInstaller{ApplyFunc: errorOnInstall},
			installErr:               installError,
			wantErr:                  true,
			expectedFailure:          3,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionFalse, Reason: "InstallationFailed", Message: "an install error"},
		},
		{
			name:                  "installation succeeds after a failure: app.Status.Failures should be reset and condition set to true",
			applicationDefinition: genApplicationDefinition("app-def-1"),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", "app-def-1", "1.0.0", 2)).
				Build(),
			appInstaller:             fake.ApplicationInstallerLogger{},
			installErr:               nil,
			wantErr:                  false,
			expectedFailure:          0,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionTrue, Reason: "InstallationSuccessful", Message: "application successfully installed or upgraded"},
		},
		{
			name:                  "installation fails and reaches max retries: condition should be set to fails and no error should be returned (to not requeue object)",
			applicationDefinition: genApplicationDefinition("app-def-1"),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", "app-def-1", "1.0.0", maxRetries+1)).
				Build(),
			appInstaller:             fake.CustomApplicationInstaller{ApplyFunc: errorOnInstall},
			installErr:               nil,
			wantErr:                  false,
			expectedFailure:          maxRetries + 1,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionFalse, Reason: "InstallationFailedRetriesExceeded", Message: "Max number of retries was exceeded. Last error: previous error"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
			r := reconciler{log: kubermaticlog.Logger, seedClient: tc.userClient, userClient: tc.userClient, userRecorder: nil, clusterIsPaused: nil, appInstaller: tc.appInstaller}

			appInstall := &appskubermaticv1.ApplicationInstallation{}
			if err := tc.userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespace}, appInstall); err != nil {
				t.Fatalf("failed to get application installation")
			}
			err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1"), appInstall)

			// check the error
			if !tc.wantErr && err != nil {
				t.Fatalf("expect no error but error '%v' was raised'", err)
			}
			if tc.wantErr && !errors.Is(err, tc.installErr) {
				t.Fatalf("expected error '%v', got '%v'", tc.installErr, err)
			}

			appInstall = &appskubermaticv1.ApplicationInstallation{}
			if err := tc.userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespace}, appInstall); err != nil {
				t.Fatalf("failed to get application installation")
			}
			// Check number of failure.
			if appInstall.Status.Failures != tc.expectedFailure {
				t.Fatalf("expected appInstall.Status.Failures='%v' but got '%v'", tc.expectedFailure, appInstall.Status.Failures)
			}

			// Check condition has been correctly updated.
			condition := appInstall.Status.Conditions[appskubermaticv1.Ready]
			if tc.expectedInstallCondition.Status != condition.Status {
				t.Errorf("expected application ready condition status='%v' but got '%v'", tc.expectedInstallCondition.Status, condition.Status)
			}

			if tc.expectedInstallCondition.Reason != condition.Reason {
				t.Errorf("expected application ready condition reason='%v' but got '%v'", tc.expectedInstallCondition.Reason, condition.Reason)
			}
			if tc.expectedInstallCondition.Message != condition.Message {
				t.Errorf("expected application ready message='%v' but got '%v'", tc.expectedInstallCondition.Message, condition.Message)
			}
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
					},
				},
			},
		},
	}
}

func genApplicationInstallation(name string, applicationDefName string, appVersion string, failures int) *appskubermaticv1.ApplicationInstallation {
	message := ""
	if failures > maxRetries {
		message = "previous error"
	}
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: applicationNamespace,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.AppNamespaceSpec{
				Name:   "default",
				Create: false,
			},

			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    applicationDefName,
				Version: appVersion,
			},
		},

		Status: appskubermaticv1.ApplicationInstallationStatus{
			Failures: failures,
			Conditions: map[appskubermaticv1.ApplicationInstallationConditionType]appskubermaticv1.ApplicationInstallationCondition{
				appskubermaticv1.Ready: {Message: message},
			},
		},
	}
}
