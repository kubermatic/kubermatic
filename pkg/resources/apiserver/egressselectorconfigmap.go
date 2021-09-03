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

package apiserver

import (
	"sigs.k8s.io/yaml"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/apiserver"
)

// EgressSelectorConfigCreator returns function to create cm that contains egress selection configuration for apiserver
// to work with konnectivity proxy.
func EgressSelectorConfigCreator() reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.KonnectivityKubeApiserverEgress, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			egressConfig := apiserver.EgressSelectorConfiguration{
				TypeMeta: metav1.TypeMeta{
					Kind:       "EgressSelectorConfiguration",
					APIVersion: "apiserver.k8s.io/v1beta1",
				},
				EgressSelections: []apiserver.EgressSelection{
					{
						Name: "cluster",
						Connection: apiserver.Connection{
							ProxyProtocol: "GRPC",
							Transport: &apiserver.Transport{
								TCP: nil,
								UDS: &apiserver.UDSTransport{
									UDSName: "/etc/kubernetes/konnectivity-server/konnectivity-server.socket",
								},
							},
						},
					},
				},
			}

			data, err := yaml.Marshal(egressConfig)
			if err != nil {
				return nil, err
			}

			c.Data = map[string]string{
				"egress-selector-configuration.yaml": string(data),
			}

			return c, nil
		}
	}
}
