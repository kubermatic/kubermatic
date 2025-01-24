/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package trivy

import (
	"encoding/json"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type Trivy struct {
	Namespace string
	Name      string
}

var DefaultTrivy = Trivy{
	Namespace: "trivy",
	Name:      "trivy",
}

func (t *Trivy) GetApplication(version string) ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      t.Name,
			Namespace: t.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   t.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    t.Name,
				Version: version,
			},
		},
	}
	applications := []apiv1.Application{app}
	data, err := json.Marshal(applications)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *Trivy) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"trivy",
	}

	key = "app.kubernetes.io/name"
	return t.Name, t.Namespace, key, names
}
