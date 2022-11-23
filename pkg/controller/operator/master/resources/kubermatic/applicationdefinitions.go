/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubermatic

import (
	"encoding/json"
	"fmt"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ciliumAppDefName    = "cilium"
	ciliumHelmChartName = "cilium"
)

func CiliumCNIApplicationDefinitionCreator() reconciling.NamedAppsKubermaticV1ApplicationDefinitionCreatorGetter {
	return func() (string, reconciling.AppsKubermaticV1ApplicationDefinitionCreator) {
		return ciliumAppDefName, func(app *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error) {
			app.Labels = map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
				appskubermaticv1.ApplicationTypeLabel:      appskubermaticv1.ApplicationTypeCNIValue,
			}

			app.Spec.Description = "Cilium CNI - eBPF-based Networking, Security, and Observability"
			app.Spec.Method = appskubermaticv1.HelmTemplateMethod
			app.Spec.Versions = []appskubermaticv1.ApplicationVersion{
				{
					Version: "1.13.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName: ciliumHelmChartName,
								// TODO (rastislavs): bump to the release version once it is out
								ChartVersion: "1.13.0-rc2",
								// TODO (rastislavs): Use Kubermatic OCI chart instead and allow overriding the registry
								URL: "https://helm.cilium.io/",
							},
						},
					},
				},
			}

			// we want to allow overriding the default values, so reconcile them only if nil
			if app.Spec.DefaultValues == nil {
				defaultValues := map[string]interface{}{
					"operator": map[string]interface{}{
						"replicas": "1",
					},
					"hubble": map[string]interface{}{
						"relay": map[string]interface{}{
							"enabled": "true",
						},
						"ui": map[string]interface{}{
							"enabled": "true",
						},
						// cronJob TLS cert gen method needs to be used for backward compatibility with older KKP
						"tls": map[string]interface{}{
							"auto": map[string]interface{}{
								"method": "cronJob",
							},
						},
					},
				}
				rawValues, err := json.Marshal(defaultValues)
				if err != nil {
					return app, fmt.Errorf("failed to marshall default CNI values: %w", err)
				}
				app.Spec.DefaultValues = &runtime.RawExtension{Raw: rawValues}
			}
			return app, nil
		}
	}
}
