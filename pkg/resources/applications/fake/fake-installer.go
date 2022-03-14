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

	"k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplicationInstallerRecorder is a fake ApplicationInstaller that record call to apply and delete for testing assertions.
type ApplicationInstallerRecorder struct {
	// ApplyEvents stores the call to apply function. Key is the name of the applicationInstallation.
	ApplyEvents sync.Map

	// DeleteEvents stores the call to delete function. Key is the name of the applicationInstallation.
	DeleteEvents sync.Map
}

func (a *ApplicationInstallerRecorder) Apply(ctx context.Context, log *zap.SugaredLogger, userClient client.Client, applicationInstallation *v1.ApplicationInstallation) error {
	a.ApplyEvents.Store(applicationInstallation.Name, *applicationInstallation.DeepCopy())
	return nil
}

func (a *ApplicationInstallerRecorder) Delete(ctx context.Context, log *zap.SugaredLogger, userClient client.Client, applicationInstallation *v1.ApplicationInstallation) error {
	a.DeleteEvents.Store(applicationInstallation.Name, *applicationInstallation.DeepCopy())
	return nil
}

// ApplicationInstallerLogger is a fake ApplicationInstaller that just log actions. it used for development of controller.
type ApplicationInstallerLogger struct {
}

func (a ApplicationInstallerLogger) Apply(ctx context.Context, log *zap.SugaredLogger, userClient client.Client, applicationInstallation *v1.ApplicationInstallation) error {
	log.Debugf("Install application %s. applicationVersion=%v", applicationInstallation.Name, applicationInstallation.Status.ApplicationVersion)
	return nil
}

func (a ApplicationInstallerLogger) Delete(ctx context.Context, log *zap.SugaredLogger, userClient client.Client, applicationInstallation *v1.ApplicationInstallation) error {
	log.Debugf("Uninstall application %s. applicationVersion=%v", applicationInstallation.Name, applicationInstallation.Status.ApplicationVersion)
	return nil
}
