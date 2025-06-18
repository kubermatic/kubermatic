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
	"context"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func DeployDefaultApplicationCatalog(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying default Application catalog…")
	sublogger := log.Prefix(logger, "   ")

	if !opt.DeployDefaultAppCatalog {
		sublogger.Info("Skipping deployment of default Application catalog, set --deploy-default-app-catalog to enable it.")
		return nil
	}

	appDefFiles, err := GetAppDefFiles()
	if err != nil {
		return fmt.Errorf("failed to fetch ApplicationDefinitions: %w", err)
	}

	sublogger.Info("Waiting for KKP webhook to become ready…")
	webhook := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.WebhookDeploymentName,
			Namespace: resources.KubermaticNamespace,
		},
	}
	if err := util.WaitForDeploymentRollout(ctx, kubeClient, webhook, opt.Versions.GitVersion, 5*time.Minute); err != nil {
		return fmt.Errorf("failed waiting for webhook: %w", err)
	}

	creators := []kkpreconciling.NamedApplicationDefinitionReconcilerFactory{}
	for _, file := range appDefFiles {
		b, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read ApplicationDefinition: %w", err)
		}

		appDef := &appskubermaticv1.ApplicationDefinition{}
		err = yaml.Unmarshal(b, appDef)

		if err != nil {
			return fmt.Errorf("failed to parse ApplicationDefinition: %w", err)
		}

		if len(opt.LimitApps) == 0 || (len(opt.LimitApps) > 0 && (slices.Contains(opt.LimitApps, appDef.Spec.DisplayName) || slices.Contains(opt.LimitApps, appDef.Name))) {
			creators = append(creators, applicationDefinitionReconcilerFactory(appDef, opt.KubermaticConfiguration, false))
		}
	}

	if err = kkpreconciling.ReconcileApplicationDefinitions(ctx, creators, "", kubeClient); err != nil {
		return fmt.Errorf("failed to apply ApplicationDefinitions: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func ApplicationDefinitionReconcilerFactories(
	logger *zap.SugaredLogger,
	config *kubermaticv1.KubermaticConfiguration,
	mirror bool,
) ([]kkpreconciling.NamedApplicationDefinitionReconcilerFactory, error) {
	appDefFiles, err := GetAppDefFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get default application definition files: %w", err)
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

		creators = append(creators, applicationDefinitionReconcilerFactory(appDef, config, mirror))
	}

	return creators, nil
}

func applicationDefinitionReconcilerFactory(appDef *appskubermaticv1.ApplicationDefinition, config *kubermaticv1.KubermaticConfiguration,
	mirror bool) kkpreconciling.NamedApplicationDefinitionReconcilerFactory {
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
	appConfig := config.Spec.UserCluster.DefaultApplicationCatalog
	if appConfig.HelmRegistryConfigFile != nil {
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
	}
}
