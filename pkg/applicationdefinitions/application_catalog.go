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

package applicationdefinitions

import (
	"fmt"
	"io"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	"sigs.k8s.io/yaml"
)

func SystemApplicationDefinitionReconcilerFactories(
	logger *zap.SugaredLogger,
	config *kubermaticv1.KubermaticConfiguration,
) ([]kkpreconciling.NamedApplicationDefinitionReconcilerFactory, error) {
	sysAppDefFiles, err := GetSysAppDefFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get system application definition files: %w", err)
	}

	creators := make([]kkpreconciling.NamedApplicationDefinitionReconcilerFactory, 0, len(sysAppDefFiles))
	for _, file := range sysAppDefFiles {
		b, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read ApplicationDefinition: %w", err)
		}

		appDef := &appskubermaticv1.ApplicationDefinition{}

		err = yaml.Unmarshal(b, appDef)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ApplicationDefinition: %w", err)
		}

		creators = append(creators, systemApplicationDefinitionReconcilerFactory(appDef, config))
	}

	return creators, nil
}

// systemApplicationDefinitionReconcilerFactory creates a reconciler that handles system ApplicationDefinitions.
func systemApplicationDefinitionReconcilerFactory(
	fileAppDef *appskubermaticv1.ApplicationDefinition,
	config *kubermaticv1.KubermaticConfiguration,
) kkpreconciling.NamedApplicationDefinitionReconcilerFactory {
	return func() (string, kkpreconciling.ApplicationDefinitionReconciler) {
		return fileAppDef.Name, func(clusterAppDef *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error) {
			l := fileAppDef.GetLabels()
			if l == nil {
				l = make(map[string]string)
			}
			l[appskubermaticv1.ApplicationManagedByLabel] = appskubermaticv1.ApplicationManagedByKKPValue
			fileAppDef.SetLabels(l)

			// Labels and annotations specified in the ApplicationDefinition installed on the cluster are merged with the ones specified in the ApplicationDefinition
			// that is generated from the system applications.
			kubernetes.EnsureLabels(clusterAppDef, fileAppDef.Labels)
			kubernetes.EnsureAnnotations(clusterAppDef, fileAppDef.Annotations)

			// State of the following fields in the cluster has a higher precedence than the one coming from the default application catalog.
			if clusterAppDef.Spec.Enforced {
				fileAppDef.Spec.Enforced = true
			}

			if clusterAppDef.Spec.Default {
				fileAppDef.Spec.Default = true
			}

			if clusterAppDef.Spec.Selector.Datacenters != nil {
				fileAppDef.Spec.Selector.Datacenters = clusterAppDef.Spec.Selector.Datacenters
			}

			// Update the application definition (fileAppDef) according to the KubermaticConfiguration.
			// When KubermaticConfiguration includes HelmRegistryConfigFile, we need to update the application definition
			// to include the Helm credentials provided by the user in the cluster.
			updateApplicationDefinition(fileAppDef, config)

			// Also, we need to ensure that the default values are set correctly. To do this:
			// 1. Get the default values from the currently being reconciled application definition (clusterAppDef)
			// 2. If it's empty, use `fileAppDef` default values as a source of truth - ensuring the values are never empty
			// 3. If it's not empty, use the current values from the reconciled object, allowing users to override the default values
			if clusterAppDef.Spec.DefaultValuesBlock != "" && clusterAppDef.Spec.DefaultValuesBlock != "{}" {
				fileAppDef.Spec.DefaultValuesBlock = clusterAppDef.Spec.DefaultValuesBlock
			}

			clusterAppDef.Spec = fileAppDef.Spec
			return clusterAppDef, nil
		}
	}
}

func updateApplicationDefinition(appDef *appskubermaticv1.ApplicationDefinition, config *kubermaticv1.KubermaticConfiguration) {
	if config == nil || appDef == nil {
		return
	}

	var credentials *appskubermaticv1.HelmCredentials
	sysAppConfig := config.Spec.UserCluster.SystemApplications
	if sysAppConfig.HelmRegistryConfigFile != nil {
		credentials = &appskubermaticv1.HelmCredentials{
			RegistryConfigFile: sysAppConfig.HelmRegistryConfigFile,
		}
	}

	for i := range appDef.Spec.Versions {
		appDef.Spec.Versions[i].Template.Source.Helm.URL = registry.ToOCIURL(sysAppConfig.HelmRepository)
		appDef.Spec.Versions[i].Template.Source.Helm.Credentials = credentials
	}
}
