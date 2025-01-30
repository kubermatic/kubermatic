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

package nvidia_gpu_operator

import (
	"encoding/json"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type NvidiaGpuOperator struct {
	Namespace string
	Name      string
}

var DefaultNvidiaGpuOperator = NvidiaGpuOperator{
	Namespace: "nvidia-gpu-operator",
	Name:      "gpu-operator",
}

func (ngo *NvidiaGpuOperator) GetApplication(version string) ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      ngo.Name,
			Namespace: ngo.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   ngo.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    ngo.Name,
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

func (ngo *NvidiaGpuOperator) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"gpu-operator",
		"node-feature-discovery",
	}

	key = "app.kubernetes.io/name"
	return ngo.Name, ngo.Namespace, key, names
}
