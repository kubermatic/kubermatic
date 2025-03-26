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

package userprojectbindingsynchronizer

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

func userProjectBindingReconcilerFactory(userProjectBinding *kubermaticv1.UserProjectBinding) reconciling.NamedUserProjectBindingReconcilerFactory {
	return func() (string, reconciling.UserProjectBindingReconciler) {
		return userProjectBinding.Name, func(p *kubermaticv1.UserProjectBinding) (*kubermaticv1.UserProjectBinding, error) {
			p.Name = userProjectBinding.Name
			p.Labels = userProjectBinding.Labels
			p.Spec = userProjectBinding.Spec
			return p, nil
		}
	}
}
