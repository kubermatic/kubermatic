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

package kubeovn

import (
	"fmt"
	"strings"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

const (
	kubeOVNHelmChartName = "kube-ovn"
	kubeOVNImageRegistry = "docker.io/kubeovn/"
)

func toOciUrl(s string) string {
	if strings.Contains(s, "://") {
		return s
	}

	return "oci://" + s
}

// ApplicationDefinitionReconciler creates KubeOVN ApplicationDefinition managed by KKP to be used
// for installing KubeOVN CNI into KKP user clusters.
func ApplicationDefinitionReconciler(config *kubermaticv1.KubermaticConfiguration) reconciling.NamedApplicationDefinitionReconcilerFactory {
	return func() (string, reconciling.ApplicationDefinitionReconciler) {
		return kubermaticv1.CNIPluginTypeKubeOVN.String(), func(app *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error) {
			app.Labels = map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
				appskubermaticv1.ApplicationTypeLabel:      appskubermaticv1.ApplicationTypeCNIValue,
			}

			app.Spec.Description = "KubeOVN CNI - Kubernetes native OVN-based networking"
			app.Spec.Method = appskubermaticv1.HelmTemplateMethod

			var credentials *appskubermaticv1.HelmCredentials
			if config.Spec.UserCluster.SystemApplications.HelmRegistryConfigFile != nil {
				credentials = &appskubermaticv1.HelmCredentials{
					RegistryConfigFile: config.Spec.UserCluster.SystemApplications.HelmRegistryConfigFile,
				}
			}

			app.Spec.Versions = []appskubermaticv1.ApplicationVersion{
				// NOTE: When introducing a new version, make sure it is:
				//  - introduced in pkg/cni/version.go with the version string exactly matching the Spec.Versions.Version here
				//  - Helm chart is mirrored in Kubermatic OCI registry, use the script kubeovn-mirror-chart.sh
				{
					Version: "1.13.3",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    kubeOVNHelmChartName,
								ChartVersion: "1.13.3",
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
								Credentials:  credentials,
							},
						},
					},
				},
			}

			// to make it compatible with older kube-ovn controller versions, convert any DefaultValues into DefaultValuesBlock
			if err := convertDefaultValuesToDefaultValuesBlock(app); err != nil {
				return app, fmt.Errorf("failed to convert DefaultValues into DefaultValuesBlock: %w", err)
			}

			// we want to allow overriding the default values, so reconcile them only if nil
			if app.Spec.DefaultValuesBlock == "" || app.Spec.DefaultValuesBlock == "{}" {
				defaultValues := map[string]any{}
				rawValues, err := yaml.Marshal(defaultValues)
				if err != nil {
					return app, fmt.Errorf("failed to marshall default CNI values: %w", err)
				}
				app.Spec.DefaultValuesBlock = string(rawValues)
			}
			return app, nil
		}
	}
}

// GetAppInstallOverrideValues returns Helm values to be enforced on the cluster's ApplicationInstallation
// of the KubeOVN CNI managed by KKP.
func GetAppInstallOverrideValues(cluster *kubermaticv1.Cluster, overwriteRegistry string) map[string]any {
	return nil
}

// ValidateValuesUpdate validates the update operation on provided KubeOVN Helm values.
func ValidateValuesUpdate(newValues, oldValues map[string]any, fieldPath *field.Path) field.ErrorList {
	return nil
}

func convertDefaultValuesToDefaultValuesBlock(app *appskubermaticv1.ApplicationDefinition) error {
	if app.Spec.DefaultValues != nil {
		oldDefVals, err := yaml.JSONToYAML(app.Spec.DefaultValues.Raw)
		if err != nil {
			return err
		}
		app.Spec.DefaultValuesBlock = string(oldDefVals)
		app.Spec.DefaultValues = nil
	}
	return nil
}
