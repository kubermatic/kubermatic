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

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications"
	"k8c.io/kubermatic/v2/pkg/applications/fake"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	kubermaticfake "k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(appskubermaticv1.AddToScheme(scheme.Scheme))
}

const (
	applicationName          = "applicationName"
	applicationNamespaceName = "apps"
)

var (
	applicationNamespace = appskubermaticv1.AppNamespaceSpec{
		Name:   applicationNamespaceName,
		Create: true,
	}
	defaultApplicationNamespace = appskubermaticv1.AppNamespaceSpec{
		Name:   "default",
		Create: false,
	}
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
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", 0, 1, 0),
					genApplicationInstallation("appInstallation-2", &defaultApplicationNamespace, "app-def-2", "1.0.0", 0, 1, 0),
					genApplicationInstallation("appInstallation-3", &defaultApplicationNamespace, "app-def-1", "1.0.0", 0, 1, 0)).
				Build(),
			expectedReconcileRequests: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}},
				{NamespacedName: types.NamespacedName{Name: "appInstallation-3", Namespace: applicationNamespaceName}},
			},
		},
		{
			name:                  "scenario 2: when no application reference ApplicationDef 'app-def-1', nothing is enqueued",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-2", "1.0.0", 0, 1, 0),
					genApplicationInstallation("appInstallation-2", &defaultApplicationNamespace, "app-def-3", "1.0.0", 0, 1, 0),
					genApplicationInstallation("appInstallation-3", &defaultApplicationNamespace, "app-def-4", "1.0.0", 0, 1, 0)).
				Build(),
			expectedReconcileRequests: []reconcile.Request{},
		},
		{
			name:                  "scenario 3: when no application in cluster, nothing is enqueued",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				Build(),
			expectedReconcileRequests: []reconcile.Request{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			enqueueApplicationInstallationFunc := enqueueAppInstallationForAppDef(tc.userClient)
			actual := enqueueApplicationInstallationFunc(context.Background(), tc.applicationDefinition)

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
			name:                  "[atomic=true -> limited retries]installation succeeds",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", 0, 1, 0)).
				Build(),
			appInstaller:             fake.ApplicationInstallerLogger{},
			installErr:               nil,
			wantErr:                  false,
			expectedFailure:          0,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionTrue, Reason: "InstallationSuccessful", Message: "application successfully installed or upgraded"},
		},
		{
			name:                  "[atomic=false -> unlimited retries] installation succeeds",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					func() *appskubermaticv1.ApplicationInstallation {
						appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", 0, 1, 0)
						appInstall.Spec.DeployOptions.Helm.Atomic = false
						return appInstall
					}(),
				).
				Build(),
			appInstaller:             fake.ApplicationInstallerLogger{},
			installErr:               nil,
			wantErr:                  false,
			expectedFailure:          0,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionTrue, Reason: "InstallationSuccessful", Message: "application successfully installed or upgraded"},
		},
		{
			name:                  "[atomic=true -> limited retries] installation fails [atomic=true -> limited retries]: app.Status.Failures should be increased and condition set to false",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", 2, 1, 1)).
				Build(),
			appInstaller:             fake.CustomApplicationInstaller{ApplyFunc: errorOnInstall},
			installErr:               installError,
			wantErr:                  true,
			expectedFailure:          3,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionFalse, Reason: "InstallationFailed", Message: "an install error"},
		},
		{
			name:                  "[atomic=true -> limited retries] installation succeeds after a failure: app.Status.Failures should be reset and condition set to true",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", 2, 1, 0)).
				Build(),
			appInstaller:             fake.ApplicationInstallerLogger{},
			installErr:               nil,
			wantErr:                  false,
			expectedFailure:          0,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionTrue, Reason: "InstallationSuccessful", Message: "application successfully installed or upgraded"},
		},
		{
			name:                  "[atomic=true -> limited retries] installation fails after failure and spec changed app.Status.Failures should be set to 1 (reset + failure) and condition set to false",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", 2, 2, 1)).
				Build(),
			appInstaller:             fake.CustomApplicationInstaller{ApplyFunc: errorOnInstall},
			installErr:               installError,
			wantErr:                  true,
			expectedFailure:          1,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionFalse, Reason: "InstallationFailed", Message: "an install error"},
		},
		{
			name:                  "[atomic=true -> limited retries] installation fails and reaches max retries: condition should be set to fails and no error should be returned (to not requeue object)",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, 1, 1)).
				Build(),
			appInstaller:             fake.CustomApplicationInstaller{ApplyFunc: errorOnInstall},
			installErr:               nil,
			wantErr:                  false,
			expectedFailure:          maxRetries + 1,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionFalse, Reason: "InstallationFailedRetriesExceeded", Message: "Max number of retries was exceeded. Last error: previous error"},
		},

		{
			name:                  "[atomic=true -> limited retries] installation has reached max retries and then spec has changed (with working install): app.Status.Failures should be reset and condition set to true",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, 2, 1)).
				Build(),
			appInstaller:             fake.ApplicationInstallerLogger{},
			installErr:               nil,
			wantErr:                  false,
			expectedFailure:          0,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionTrue, Reason: "InstallationSuccessful", Message: "application successfully installed or upgraded"},
		},
		{
			name:                  "[atomic=true -> limited retries] installation has reached max retries and then spec has changed (with not working install): app.Status.Failures should be set to 1 ( reset + failure) and condition set to false",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, 2, 1)).
				Build(),
			appInstaller:             fake.CustomApplicationInstaller{ApplyFunc: errorOnInstall},
			installErr:               installError,
			wantErr:                  true,
			expectedFailure:          1,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionFalse, Reason: "InstallationFailed", Message: "an install error"},
		},
		{
			name:                  "[atomic=false -> unlimited retries] installation fails: retries should not be incremented",
			applicationDefinition: genApplicationDefinition("app-def-1", nil),
			userClient: kubermaticfake.
				NewClientBuilder().
				WithObjects(
					func() *appskubermaticv1.ApplicationInstallation {
						appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", 0, 1, 1)
						appInstall.Spec.DeployOptions.Helm.Atomic = false
						return appInstall
					}(),
				).
				Build(),
			appInstaller:             fake.CustomApplicationInstaller{ApplyFunc: errorOnInstall},
			installErr:               installError,
			wantErr:                  true,
			expectedFailure:          0,
			expectedInstallCondition: appskubermaticv1.ApplicationInstallationCondition{Status: corev1.ConditionFalse, Reason: "InstallationFailed", Message: "an install error"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
			r := reconciler{log: kubermaticlog.Logger, seedClient: tc.userClient, userClient: tc.userClient, userRecorder: nil, clusterIsPaused: nil, appInstaller: tc.appInstaller}

			appInstall := &appskubermaticv1.ApplicationInstallation{}
			if err := tc.userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, appInstall); err != nil {
				t.Fatalf("failed to get application installation")
			}
			err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1", nil), appInstall)

			// check the error
			if !tc.wantErr && err != nil {
				t.Fatalf("expect no error but error '%v' was raised'", err)
			}
			if tc.wantErr && !errors.Is(err, tc.installErr) {
				t.Fatalf("expected error '%v', got '%v'", tc.installErr, err)
			}

			appInstall = &appskubermaticv1.ApplicationInstallation{}
			if err := tc.userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, appInstall); err != nil {
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

func genApplicationDefinition(name string, defaultAppNamespace *appskubermaticv1.AppNamespaceSpec) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Method:           appskubermaticv1.HelmTemplateMethod,
			DefaultNamespace: defaultAppNamespace,
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

func genApplicationInstallation(name string, appNamespace *appskubermaticv1.AppNamespaceSpec, applicationDefName string, appVersion string, failures int, generation int64, observedGeneration int64) *appskubermaticv1.ApplicationInstallation {
	message := ""
	if failures > maxRetries {
		message = "previous error"
	}
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  applicationNamespace.Name,
			Generation: generation,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appNamespace,

			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    applicationDefName,
				Version: appVersion,
			},
			DeployOptions: &appskubermaticv1.DeployOptions{
				Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true},
			},
		},
		Status: appskubermaticv1.ApplicationInstallationStatus{
			Failures: failures,
			Conditions: map[appskubermaticv1.ApplicationInstallationConditionType]appskubermaticv1.ApplicationInstallationCondition{
				appskubermaticv1.Ready: {Message: message, ObservedGeneration: observedGeneration},
			},
		},
	}
}

func genNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func TestHasLimitedRetries(t *testing.T) {
	tests := []struct {
		name                 string
		defaultDeployOptions *appskubermaticv1.DeployOptions
		deployOptions        *appskubermaticv1.DeployOptions
		want                 bool
	}{
		// tests default value
		{
			name:                 "hasLimitedRetries should be false when defaultDeployOptions and deployOptions are nil",
			defaultDeployOptions: nil,
			deployOptions:        nil,
			want:                 false,
		},
		{
			name:                 "hasLimitedRetries should be false when defaultDeployOptions.helm=nil and  deployOptions.helm=nil",
			defaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: nil},
			deployOptions:        &appskubermaticv1.DeployOptions{Helm: nil},
			want:                 false,
		},
		{
			name:                 "hasLimitedRetries should be false when defaultDeployOptions.helm=nil and  deployOptions.helm=nil",
			defaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: nil},
			deployOptions:        &appskubermaticv1.DeployOptions{Helm: nil},
			want:                 false,
		},
		// test value defined at applicationInstall level only
		{
			name:                 "hasLimitedRetries should be false when defaultDeployOptions.helm=nil and  deployOptions.helm.atomic=false (defined at appInstall level)",
			defaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: nil},
			deployOptions:        &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false}},
			want:                 false,
		},
		{
			name:                 "hasLimitedRetries should be true when defaultDeployOptions.helm=nil and  deployOptions.helm.atomic=true (defined at appInstall level)",
			defaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: nil},
			deployOptions:        &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true}},
			want:                 true,
		},
		// test fallback to applicationDefinition
		{
			name:                 "hasLimitedRetries should be false when defaultDeployOptions.helm.atomic=false  and  deployOptions.helm=nil(use default at appDef level)",
			defaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false}},
			deployOptions:        &appskubermaticv1.DeployOptions{Helm: nil},
			want:                 false,
		},
		{
			name:                 "hasLimitedRetries should be true when defaultDeployOptions.helm.atomic=true and  deployOptions.helm.=nil (use default at appDef level)",
			defaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true}},
			deployOptions:        &appskubermaticv1.DeployOptions{Helm: nil},
			want:                 true,
		},
		// test applicationInstallation value has priority
		{
			name:                 "hasLimitedRetries should be false when defaultDeployOptions.helm.atomic=true  and  deployOptions.helm.atomic=false (priority to appInstall level)",
			defaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true}},
			deployOptions:        &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false}},
			want:                 false,
		},
		{
			name:                 "hasLimitedRetries should be true when defaultDeployOptions.helm.atomic=false  and  deployOptions.helm.atomic=true (priority to appInstall level)",
			defaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false}},
			deployOptions:        &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true}},
			want:                 true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appDefinition := &appskubermaticv1.ApplicationDefinition{Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: tt.defaultDeployOptions}}
			appInstallation := &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: tt.deployOptions}}

			if got := hasLimitedRetries(appDefinition, appInstallation); got != tt.want {
				t.Errorf("hasLimitedRetries() = %v, want %v", got, tt.want)
			}
		})
	}
}
