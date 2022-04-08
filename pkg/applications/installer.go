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

package applications

import (
	"context"

	"go.uber.org/zap"

	appkubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplicationInstaller handles the installation / uninstallation of an Application on the user cluster.
type ApplicationInstaller interface {
	// Apply function installs the application on the user-cluster and returns an error if the installation has failed; this is idempotent.
	Apply(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appkubermaticv1.ApplicationInstallation) error

	// Delete function uninstalls the application on the user-cluster and returns an error if the uninstallation has failed; this is idempotent.
	Delete(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appkubermaticv1.ApplicationInstallation) error
}
