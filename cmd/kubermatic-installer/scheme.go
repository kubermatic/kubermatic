/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// setupKubermaticInstallerScheme adds all required API types to the manager's scheme.
// This includes KKP types, cert-manager and apiextensions.
func setupKubermaticInstallerScheme(mgr manager.Manager) error {
	if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("failed to add Kubermatic v1 scheme: %w", err)
	}

	if err := certmanagerv1.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("failed to add cert-manager v1 scheme: %w", err)
	}

	if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("failed to add apiextensions v1 scheme: %w", err)
	}

	return nil
}
