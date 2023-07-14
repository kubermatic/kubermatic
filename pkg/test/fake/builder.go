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

package fake

import (
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// NewClientBuilder returns a client builder pre-configured to
// treat KKP CRDs with enabled status subresource behaviour.
// See https://github.com/kubernetes-sigs/controller-runtime/pull/2259
// for more information.
func NewClientBuilder() *fakectrlruntimeclient.ClientBuilder {
	return fakectrlruntimeclient.NewClientBuilder().WithStatusSubresource(
		&appskubermaticv1.ApplicationInstallation{},
		&kubermaticv1.Addon{},
		&kubermaticv1.Alertmanager{},
		&kubermaticv1.Cluster{},
		&kubermaticv1.Seed{},
		&kubermaticv1.EtcdBackupConfig{},
		&kubermaticv1.EtcdRestore{},
		&kubermaticv1.Project{},
		&kubermaticv1.ResourceQuota{},
		&kubermaticv1.User{},
	)
}
