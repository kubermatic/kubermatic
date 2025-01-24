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

package k8sgpt

import (
	"encoding/json"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type K8sGpt struct {
	Namespace string
	Name      string
}

var DefaultK8sGpt = K8sGpt{
	Namespace: "k8sgpt",
	Name:      "k8sgpt",
}

func (kg *K8sGpt) GetApplication(version string) ([]byte, error) {
	valuesBlock := `serviceMonitor:
  enabled: false
  additionalLabels: {}
grafanaDashboard:
  enabled: false
  folder:
    annotation: grafana_folder
    name: ai
  label:
    key: grafana_dashboard
    value: "1"`

	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      kg.Name,
			Namespace: kg.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   kg.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    kg.Name,
				Version: version,
			},
			ValuesBlock: valuesBlock,
		},
	}
	applications := []apiv1.Application{app}
	data, err := json.Marshal(applications)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (kg *K8sGpt) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"k8sgpt-operator",
	}

	key = "app.kubernetes.io/name"
	return kg.Name, kg.Namespace, key, names
}
