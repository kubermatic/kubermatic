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

package cert_manager

import (
	"encoding/json"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type CertManager struct {
	Namespace string
	Name      string
}

var DefaultCertManager = CertManager{
	Namespace: "cert-manager",
	Name:      "cert-manager",
}

func (cm *CertManager) GetApplication(version string) ([]byte, error) {
	valuesBlock := `installCRDs: true`

	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      cm.Name,
			Namespace: cm.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   cm.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    cm.Name,
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

func (cm *CertManager) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"cert-manager",
		"cert-manager-cainjector",
		"cert-manager-startupapicheck",
		"cert-manager-webhook",
	}

	key = "app.kubernetes.io/name"
	return cm.Name, cm.Namespace, key, names
}
