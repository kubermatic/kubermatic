/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package defaulting

import (
	"time"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
)

// DefaultHelmTimeout is the default time to wait for any individual Kubernetes operation.
const DefaultHelmTimeout = 5 * time.Minute

func DefaultApplicationInstallation(appInstall *appskubermaticv1.ApplicationInstallation) error {
	DefaultDeployOpts(appInstall.Spec.DeployOptions)
	DefaultOmittedValues(&appInstall.Spec)
	return nil
}

func DefaultDeployOpts(deployOpts *appskubermaticv1.DeployOptions) {
	if deployOpts != nil && deployOpts.Helm != nil {
		// atomic implies wait = true
		if deployOpts.Helm.Atomic {
			deployOpts.Helm.Wait = true
		}
		// wait implies timeout > 0
		if deployOpts.Helm.Wait && deployOpts.Helm.Timeout.Duration == 0 {
			deployOpts.Helm.Timeout.Duration = DefaultHelmTimeout
		}
	}
}

// DefaultOmittedValues defaults the values field to "{}" if it was not explicitly set.
// Defaulting the field on our end allows users to omit the field in case they are not
// using it or making use of the newer ValuesBlock field.
// Without this defaulting, the k8s-apiserver complains that the field cannot be null.
// This is most likely due to the fact that Values is of type runtime.RawExtension.
func DefaultOmittedValues(spec *appskubermaticv1.ApplicationInstallationSpec) {
	if len(spec.Values.Raw) == 0 {
		spec.Values.Raw = []byte("{}")
	}
}
