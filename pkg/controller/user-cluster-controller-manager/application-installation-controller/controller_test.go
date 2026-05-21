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
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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

func TestStuckReleaseRecoveryRunsBeforeMaxRetries(t *testing.T) {
	ctx := context.Background()
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, 1, 1)
	userClient := kubermaticfake.NewClientBuilder().WithObjects(appInstall).Build()

	rollbackCalled := false
	applyCalled := false
	isDeployedCalled := false
	appInstaller := fake.CustomApplicationInstaller{
		IsStuckFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
			return true, nil
		},
		RollbackFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) error {
			rollbackCalled = true
			if applicationInstallation.Status.Failures != 0 {
				t.Fatalf("expected failures to be reset before rollback, got %d", applicationInstallation.Status.Failures)
			}
			return nil
		},
		ApplyFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error) {
			applyCalled = true
			return util.NoStatusUpdate, nil
		},
		IsDeployedFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
			isDeployedCalled = true
			return false, nil
		},
	}

	r := reconciler{log: kubermaticlog.Logger, seedClient: userClient, userClient: userClient, appInstaller: appInstaller}
	if err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1", nil), appInstall); err != nil {
		t.Fatalf("expected stuck release recovery to succeed, got %v", err)
	}
	if !rollbackCalled {
		t.Fatal("expected rollback/recovery to be called before enforcing max retries")
	}
	if !applyCalled {
		t.Fatal("expected apply to run after stuck release recovery reset failures")
	}
	if isDeployedCalled {
		t.Fatal("expected deployed check not to run after stuck release recovery reset failures")
	}

	updatedAppInstall := &appskubermaticv1.ApplicationInstallation{}
	if err := userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, updatedAppInstall); err != nil {
		t.Fatalf("failed to get application installation: %v", err)
	}
	if updatedAppInstall.Status.Failures != 0 {
		t.Fatalf("expected failures to be reset after recovery, got %d", updatedAppInstall.Status.Failures)
	}
}

func TestStuckReleaseRecoveryDoesNotRollbackWhenFailureResetCannotBePersisted(t *testing.T) {
	ctx := context.Background()
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, 1, 1)
	statusPatchErr := errors.New("status patch failed")
	userClient := kubermaticfake.NewClientBuilder().
		WithObjects(appInstall).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourcePatch: func(ctx context.Context, c ctrlruntimeclient.Client, subResourceName string, obj ctrlruntimeclient.Object, patch ctrlruntimeclient.Patch, opts ...ctrlruntimeclient.SubResourcePatchOption) error {
				if subResourceName == "status" {
					return statusPatchErr
				}
				return c.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
			},
		}).
		Build()

	rollbackCalled := false
	appInstaller := fake.CustomApplicationInstaller{
		IsStuckFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
			return true, nil
		},
		RollbackFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) error {
			rollbackCalled = true
			return nil
		},
	}

	r := reconciler{log: kubermaticlog.Logger, seedClient: userClient, userClient: userClient, appInstaller: appInstaller}
	err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1", nil), appInstall)
	if !errors.Is(err, statusPatchErr) {
		t.Fatalf("expected failure reset error, got %v", err)
	}
	if rollbackCalled {
		t.Fatal("expected rollback not to run when failure reset cannot be persisted")
	}
}

func TestStuckReleaseCheckErrorStopsReconcile(t *testing.T) {
	ctx := context.Background()
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", 0, 1, 1)
	expectedReadyCondition := appInstall.Status.Conditions[appskubermaticv1.Ready]
	expectedFailures := appInstall.Status.Failures
	userClient := kubermaticfake.NewClientBuilder().WithObjects(appInstall).Build()

	stuckCheckErr := errors.New("helm metadata lookup failed")
	applyCalled := false
	appInstaller := fake.CustomApplicationInstaller{
		IsStuckFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
			return false, stuckCheckErr
		},
		ApplyFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error) {
			applyCalled = true
			return util.NoStatusUpdate, nil
		},
	}

	r := reconciler{log: kubermaticlog.Logger, seedClient: userClient, userClient: userClient, appInstaller: appInstaller}
	err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1", nil), appInstall)
	if !errors.Is(err, stuckCheckErr) {
		t.Fatalf("expected stuck release check error to be returned, got %v", err)
	}
	if applyCalled {
		t.Fatal("expected apply not to run when stuck release check fails")
	}

	updatedAppInstall := &appskubermaticv1.ApplicationInstallation{}
	if err := userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, updatedAppInstall); err != nil {
		t.Fatalf("failed to get application installation: %v", err)
	}
	if updatedAppInstall.Status.Conditions[appskubermaticv1.Ready] != expectedReadyCondition {
		t.Fatalf("expected ready condition not to change on stuck release check error, got %+v", updatedAppInstall.Status.Conditions[appskubermaticv1.Ready])
	}
	if updatedAppInstall.Status.Failures != expectedFailures {
		t.Fatalf("expected failures not to change on stuck release check error, got %d", updatedAppInstall.Status.Failures)
	}
}

func TestDeployedReleaseClearsStaleMaxRetries(t *testing.T) {
	ctx := context.Background()
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, 1, 1)
	appInstall.Status.Conditions[appskubermaticv1.Ready] = appskubermaticv1.ApplicationInstallationCondition{
		Status:             corev1.ConditionTrue,
		Reason:             "InstallationSuccessful",
		Message:            "application successfully installed or upgraded",
		ObservedGeneration: appInstall.Generation,
	}
	userClient := kubermaticfake.NewClientBuilder().WithObjects(appInstall).Build()

	applyCalled := false
	isDeployedCalled := false
	appInstaller := fake.CustomApplicationInstaller{
		IsDeployedFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
			isDeployedCalled = true
			return true, nil
		},
		ApplyFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error) {
			applyCalled = true
			return util.NoStatusUpdate, nil
		},
	}

	r := reconciler{log: kubermaticlog.Logger, seedClient: userClient, userClient: userClient, appInstaller: appInstaller}
	if err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1", nil), appInstall); err != nil {
		t.Fatalf("expected stale max retries on deployed release to recover, got %v", err)
	}
	if !applyCalled {
		t.Fatal("expected apply to run after clearing stale max retries")
	}
	if !isDeployedCalled {
		t.Fatal("expected deployed release check to run before enforcing max retries")
	}

	updatedAppInstall := &appskubermaticv1.ApplicationInstallation{}
	if err := userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, updatedAppInstall); err != nil {
		t.Fatalf("failed to get application installation: %v", err)
	}
	if updatedAppInstall.Status.Failures != 0 {
		t.Fatalf("expected failures to be reset after successful apply, got %d", updatedAppInstall.Status.Failures)
	}

	readyCondition := updatedAppInstall.Status.Conditions[appskubermaticv1.Ready]
	if readyCondition.Reason != "InstallationSuccessful" {
		t.Fatalf("expected ready condition reason InstallationSuccessful, got %q", readyCondition.Reason)
	}
}

func TestDeployedRollbackAfterFailedAtomicUpgradePreservesMaxRetries(t *testing.T) {
	ctx := context.Background()
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, 2, 2)
	appInstall.Status.Conditions[appskubermaticv1.Ready] = appskubermaticv1.ApplicationInstallationCondition{
		Status:             corev1.ConditionFalse,
		Reason:             "InstallationFailed",
		Message:            "upgrade failed",
		ObservedGeneration: appInstall.Generation,
	}
	userClient := kubermaticfake.NewClientBuilder().WithObjects(appInstall).Build()

	applyCalled := false
	isDeployedCalled := false
	appInstaller := fake.CustomApplicationInstaller{
		IsDeployedFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
			isDeployedCalled = true
			return true, nil
		},
		ApplyFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error) {
			applyCalled = true
			return util.NoStatusUpdate, nil
		},
	}

	r := reconciler{log: kubermaticlog.Logger, seedClient: userClient, userClient: userClient, appInstaller: appInstaller}
	if err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1", nil), appInstall); err != nil {
		t.Fatalf("expected failed atomic upgrade to remain blocked by max retries, got %v", err)
	}
	if applyCalled {
		t.Fatal("expected apply to remain blocked after max retries when only the rolled-back Helm release is deployed")
	}
	if isDeployedCalled {
		t.Fatal("expected deployed release check not to run when Ready condition is not current and true")
	}

	updatedAppInstall := &appskubermaticv1.ApplicationInstallation{}
	if err := userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, updatedAppInstall); err != nil {
		t.Fatalf("failed to get application installation: %v", err)
	}
	if updatedAppInstall.Status.Failures != maxRetries+1 {
		t.Fatalf("expected failures to remain above max retries, got %d", updatedAppInstall.Status.Failures)
	}

	readyCondition := updatedAppInstall.Status.Conditions[appskubermaticv1.Ready]
	if readyCondition.Reason != installationFailedRetriesExceededReason {
		t.Fatalf("expected ready condition reason %q, got %q", installationFailedRetriesExceededReason, readyCondition.Reason)
	}
	expectedMessage := installationFailedRetriesExceededMessagePrefix + "upgrade failed"
	if readyCondition.Message != expectedMessage {
		t.Fatalf("expected ready condition message %q, got %q", expectedMessage, readyCondition.Message)
	}
}

func TestSpecChangeResetsFailuresBeforeDeployedReleaseCheck(t *testing.T) {
	ctx := context.Background()
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	const (
		generation         int64 = 2
		observedGeneration int64 = 1
	)
	appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, generation, observedGeneration)
	userClient := kubermaticfake.NewClientBuilder().WithObjects(appInstall).Build()

	applyCalled := false
	isDeployedCalled := false
	appInstaller := fake.CustomApplicationInstaller{
		IsDeployedFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
			isDeployedCalled = true
			return true, nil
		},
		ApplyFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error) {
			applyCalled = true
			return util.NoStatusUpdate, nil
		},
	}

	r := reconciler{log: kubermaticlog.Logger, seedClient: userClient, userClient: userClient, appInstaller: appInstaller}
	if err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1", nil), appInstall); err != nil {
		t.Fatalf("expected spec change to clear stale failures and reconcile, got %v", err)
	}
	if !applyCalled {
		t.Fatal("expected apply to run after spec change reset failures")
	}
	if isDeployedCalled {
		t.Fatal("expected deployed release check not to run after spec change reset failures")
	}

	updatedAppInstall := &appskubermaticv1.ApplicationInstallation{}
	if err := userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, updatedAppInstall); err != nil {
		t.Fatalf("failed to get application installation: %v", err)
	}
	if updatedAppInstall.Status.Failures != 0 {
		t.Fatalf("expected failures to be reset after spec change reconciliation, got %d", updatedAppInstall.Status.Failures)
	}
}

func TestResetFailuresIfSpecHasChanged(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name               string
		failures           int
		generation         int64
		observedGeneration int64
		wantFailures       int
	}{
		{
			name:               "spec generation changed",
			failures:           maxRetries + 1,
			generation:         2,
			observedGeneration: 1,
			wantFailures:       0,
		},
		{
			name:               "spec generation unchanged",
			failures:           maxRetries + 1,
			generation:         2,
			observedGeneration: 2,
			wantFailures:       maxRetries + 1,
		},
		{
			name:               "spec generation changed with no failures",
			generation:         2,
			observedGeneration: 1,
			wantFailures:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", tt.failures, tt.generation, tt.observedGeneration)
			userClient := kubermaticfake.NewClientBuilder().WithObjects(appInstall).Build()

			r := reconciler{userClient: userClient}
			if err := r.resetFailuresIfSpecHasChanged(ctx, appInstall); err != nil {
				t.Fatalf("expected resetFailuresIfSpecHasChanged to succeed, got %v", err)
			}

			updatedAppInstall := &appskubermaticv1.ApplicationInstallation{}
			if err := userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, updatedAppInstall); err != nil {
				t.Fatalf("failed to get application installation: %v", err)
			}
			if updatedAppInstall.Status.Failures != tt.wantFailures {
				t.Fatalf("expected failures %d, got %d", tt.wantFailures, updatedAppInstall.Status.Failures)
			}
		})
	}
}

func TestMaxRetriesMessageDoesNotGrow(t *testing.T) {
	ctx := context.Background()
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	appInstall := genApplicationInstallation("appInstallation-1", &defaultApplicationNamespace, "app-def-1", "1.0.0", maxRetries+1, 1, 1)
	previousHeartbeat := metav1.NewTime(time.Now().Add(-time.Hour))
	appInstall.Status.Conditions[appskubermaticv1.Ready] = appskubermaticv1.ApplicationInstallationCondition{
		Status:             corev1.ConditionFalse,
		LastHeartbeatTime:  previousHeartbeat,
		Reason:             installationFailedRetriesExceededReason,
		Message:            installationFailedRetriesExceededMessagePrefix + installationFailedRetriesExceededMessagePrefix + "previous error",
		ObservedGeneration: appInstall.Generation,
	}
	userClient := kubermaticfake.NewClientBuilder().WithObjects(appInstall).Build()

	applyCalled := false
	appInstaller := fake.CustomApplicationInstaller{
		ApplyFunc: func(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error) {
			applyCalled = true
			return util.NoStatusUpdate, nil
		},
	}

	r := reconciler{log: kubermaticlog.Logger, seedClient: userClient, userClient: userClient, appInstaller: appInstaller}
	if err := r.handleInstallation(ctx, kubermaticlog.Logger, genApplicationDefinition("app-def-1", nil), appInstall); err != nil {
		t.Fatalf("expected max retries handling to succeed, got %v", err)
	}
	if applyCalled {
		t.Fatal("expected apply to remain blocked while max retries are exceeded and release is not deployed")
	}

	updatedAppInstall := &appskubermaticv1.ApplicationInstallation{}
	if err := userClient.Get(ctx, types.NamespacedName{Name: "appInstallation-1", Namespace: applicationNamespaceName}, updatedAppInstall); err != nil {
		t.Fatalf("failed to get application installation: %v", err)
	}
	expectedMessage := installationFailedRetriesExceededMessagePrefix + "previous error"
	readyCondition := updatedAppInstall.Status.Conditions[appskubermaticv1.Ready]
	if readyCondition.Message != expectedMessage {
		t.Fatalf("expected max retries message %q, got %q", expectedMessage, readyCondition.Message)
	}
	if !readyCondition.LastHeartbeatTime.After(previousHeartbeat.Time) {
		t.Fatalf("expected max retries handling to update ready condition heartbeat, got %v", readyCondition.LastHeartbeatTime)
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
