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
	"k8s.io/utils/strings/slices"
	"time"

	"github.com/sirupsen/logrus"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func DeployDefaultApplicationCatalog(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying default Application catalog…")
	sublogger := log.Prefix(logger, "   ")

	if !opt.DeployDefaultAppCatalog {
		sublogger.Info("Skipping deployment of default Application catalog. To enable it, set the 'enabled' field in the KubermaticConfiguration file's DefaultAppCatalogConfig section to true.")
		sublogger.Info("If you want to limit the applications, specify the 'limitApps' field with a list of allowed apps.")
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

		if opt.LimitApps == nil || len(opt.LimitApps) == 0 {
			creators = append(creators, applicationDefinitionReconcilerFactory(appDef))
		}

		if opt.LimitApps != nil && len(opt.LimitApps) >= 0 && slices.Contains(opt.LimitApps, appDef.Spec.DisplayName) {
			creators = append(creators, applicationDefinitionReconcilerFactory(appDef))
		}
	}

	if err = kkpreconciling.ReconcileApplicationDefinitions(ctx, creators, "", kubeClient); err != nil {
		return fmt.Errorf("failed to apply ApplicationDefinitions: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func applicationDefinitionReconcilerFactory(appDef *appskubermaticv1.ApplicationDefinition) kkpreconciling.NamedApplicationDefinitionReconcilerFactory {
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

			a.Spec = appDef.Spec
			return a, nil
		}
	}
}
