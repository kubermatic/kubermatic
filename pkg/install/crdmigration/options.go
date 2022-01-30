/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package crdmigration

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	KubermaticNamespace string

	// KubermaticConfiguration is the current configuration from the cluster,
	// but not defaulted because the defaulting code is rewritten for the new
	// API group already. Beware.
	KubermaticConfiguration *operatorv1alpha1.KubermaticConfiguration

	MasterClient    ctrlruntimeclient.Client
	Seeds           map[string]*kubermaticv1.Seed
	SeedClients     map[string]ctrlruntimeclient.Client
	ChartsDirectory string
}
