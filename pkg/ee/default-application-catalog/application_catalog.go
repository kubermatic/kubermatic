/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2023 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package applicationcatalog

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

func DefaultApplicationCatalogReconcilerFactories(
	logger *zap.SugaredLogger,
	config *kubermaticv1.KubermaticConfiguration,
	mirror bool,
) ([]kkpreconciling.NamedApplicationDefinitionReconcilerFactory, error) {
	if !config.Spec.Applications.DefaultApplicationCatalog.Enable {
		logger.Info("Default application catalog is disabled, skipping deployment of default application definitions.")
		return nil, nil
	}

	appDefFiles, err := GetAppDefFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ApplicationDefinitions: %w", err)
	}

	filterApps := len(config.Spec.Applications.DefaultApplicationCatalog.Applications) > 0
	requestedApps := make(map[string]struct{})
	if filterApps {
		for _, appName := range config.Spec.Applications.DefaultApplicationCatalog.Applications {
			requestedApps[appName] = struct{}{}
		}

		logger.Debugf("Installing only specified system applications: %+v", config.Spec.Applications.DefaultApplicationCatalog.Applications)
	}

	creators := make([]kkpreconciling.NamedApplicationDefinitionReconcilerFactory, 0, len(appDefFiles))
	for _, file := range appDefFiles {
		b, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read ApplicationDefinition: %w", err)
		}

		appDef := &appskubermaticv1.ApplicationDefinition{}
		err = yaml.Unmarshal(b, appDef)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ApplicationDefinition: %w", err)
		}

		if filterApps {
			if _, ok := requestedApps[appDef.Name]; !ok {
				logger.Debugf("Skipping application %q as it's not in the requested list", appDef.Name)
				continue
			}
		}
		creators = append(creators, applicationDefinitionReconcilerFactory(appDef, config, mirror))
	}
	return creators, nil
}

func applicationDefinitionReconcilerFactory(appDef *appskubermaticv1.ApplicationDefinition, config *kubermaticv1.KubermaticConfiguration, mirror bool) kkpreconciling.NamedApplicationDefinitionReconcilerFactory {
	return func() (string, kkpreconciling.ApplicationDefinitionReconciler) {
		return appDef.Name, func(a *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error) {
			// Labels and annotations specified in the ApplicationDefinition installed on the cluster are merged with the ones specified in the ApplicationDefinition
			// that is generated from the default application catalog.
			kubernetes.EnsureLabels(a, appDef.Labels)
			kubernetes.EnsureAnnotations(a, appDef.Annotations)

			// State of the following fields in the cluster has a higher precedence than the one coming from the default application catalog.
			if a.Spec.Enforced {
				appDef.Spec.Enforced = true
			}

			if a.Spec.Default {
				appDef.Spec.Default = true
			}

			if a.Spec.Selector.Datacenters != nil {
				appDef.Spec.Selector.Datacenters = a.Spec.Selector.Datacenters
			}

			// Update the application definition (fileAppDef) based on the KubermaticConfiguration.
			// If the KubermaticConfiguration includes HelmRegistryConfigFile, update the application
			// definition to incorporate the Helm credentials provided by the user in the cluster.
			//
			// When running mirror-images, leave the application definition unchanged. This ensures
			// that charts are downloaded from the default upstream repositories used by KKP,
			// preserving the original image references for discovery.
			if !mirror {
				updateApplicationDefinition(appDef, config)
			}

			a.Spec = appDef.Spec
			return a, nil
		}
	}
}

func updateApplicationDefinition(appDef *appskubermaticv1.ApplicationDefinition, config *kubermaticv1.KubermaticConfiguration) {
	if config == nil || appDef == nil {
		return
	}

	var credentials *appskubermaticv1.HelmCredentials
	appConfig := config.Spec.Applications.DefaultApplicationCatalog

	// Use HelmCredentials if provided, otherwise fall back to HelmRegistryConfigFile for backward compatibility
	if appConfig.HelmCredentials != nil {
		credentials = appConfig.HelmCredentials
	} else if appConfig.HelmRegistryConfigFile != nil {
		credentials = &appskubermaticv1.HelmCredentials{
			RegistryConfigFile: appConfig.HelmRegistryConfigFile,
		}
	}

	for i := range appDef.Spec.Versions {
		if appConfig.HelmRepository != "" {
			appDef.Spec.Versions[i].Template.Source.Helm.URL = registry.ToOCIURL(appConfig.HelmRepository)
		}

		if credentials != nil {
			appDef.Spec.Versions[i].Template.Source.Helm.Credentials = credentials
		}

		// Apply Insecure setting if provided
		if appConfig.Insecure != nil {
			appDef.Spec.Versions[i].Template.Source.Helm.Insecure = appConfig.Insecure
		}

		// Apply PlainHTTP setting if provided
		if appConfig.PlainHTTP != nil {
			appDef.Spec.Versions[i].Template.Source.Helm.PlainHTTP = appConfig.PlainHTTP
		}
	}
}
