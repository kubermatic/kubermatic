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
	"time"

	"github.com/sirupsen/logrus"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
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
		sublogger.Info("Skipping deployment of default Application catalog, set --deploy-default-app-catalog to enable it.")
		return nil
	}

	appDefFiles, err := GetAppDefFiles()
	if err != nil {
		return fmt.Errorf("failed to fetch ApplicationDefinitions: %w", err)
	}

	sublogger.Infof("Waiting for webhook %q with version %q to become ready", common.WebhookDeploymentName, opt.Versions.Kubermatic)
	operatorDeploySelector := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.WebhookDeploymentName,
			Namespace: resources.KubermaticNamespace,
		},
	}
	if err := util.WaitForUpdatedDeployment(ctx, operatorDeploySelector, opt.Versions.Kubermatic, kubeClient, 5*time.Minute); err != nil {
		return fmt.Errorf("failed waiting for webhook to come ready %w", err)
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

		creators = append(creators, applicationDefinitionReconcilerFactory(appDef))
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
			a.Labels = appDef.Labels
			a.Annotations = appDef.Annotations
			a.Spec = appDef.Spec
			return a, nil
		}
	}
}
