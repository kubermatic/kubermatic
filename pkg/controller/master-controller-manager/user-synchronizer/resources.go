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

package usersynchronizer

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

func userCreatorGetter(user *kubermaticv1.User) reconciling.NamedKubermaticV1UserCreatorGetter {
	return func() (string, reconciling.KubermaticV1UserCreator) {
		return user.Name, func(u *kubermaticv1.User) (*kubermaticv1.User, error) {
			u.Name = user.Name
			u.Spec = user.Spec
			return u, nil
		}
	}
}
