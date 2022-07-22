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

package fake

import (
	"context"
	"sync"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplicationInstallerRecorder is a fake ApplicationInstaller that records calls to apply and delete for testing assertions.
type ApplicationInstallerRecorder struct {
	// ApplyEvents stores the call to apply function. Key is the name of the applicationInstallation.
	ApplyEvents sync.Map

	// DeleteEvents stores the call to delete function. Key is the name of the applicationInstallation.
	DeleteEvents sync.Map
}

func (a *ApplicationInstallerRecorder) Apply(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	a.ApplyEvents.Store(applicationInstallation.Name, *applicationInstallation.DeepCopy())
	return nil, nil
}

func (a *ApplicationInstallerRecorder) Delete(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	a.DeleteEvents.Store(applicationInstallation.Name, *applicationInstallation.DeepCopy())
	return nil, nil
}

// ApplicationInstallerLogger is a fake ApplicationInstaller that just logs actions. it's used for the development of the controller.
type ApplicationInstallerLogger struct {
}

func (a ApplicationInstallerLogger) Apply(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	log.Debugf("Install application %s. applicationVersion=%v", applicationInstallation.Name, applicationInstallation.Status.ApplicationVersion)
	return nil, nil
}

func (a ApplicationInstallerLogger) Delete(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	log.Debugf("Uninstall application %s. applicationVersion=%v", applicationInstallation.Name, applicationInstallation.Status.ApplicationVersion)
	return nil, nil
}
