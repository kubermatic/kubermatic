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

package kubermaticmaster

import (
	"context"
	"fmt"
	"io"

	defaultappdefs "github.com/kubermatic-labs/appdefs-test"
	"github.com/sirupsen/logrus"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/log"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func deployDefaultApplicationCatalog(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	logger.Infof("📦 Deploying default ApplicationCatalog")
	sublogger := log.Prefix(logger, "   ")

	if !opt.DeployDefaultAppCatalog {
		sublogger.Info("⏭️ Skipping deployment of default ApplicationCatalog")
		return nil
	}

	appDefFiles, err := defaultappdefs.GetAppDefFiles()
	if err != nil {
		return fmt.Errorf("Failed to fetch ApplicationDefinitions: %w", err)
	}

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

		err = kubeClient.Create(ctx, appDef)
		if err != nil {
			return fmt.Errorf("failed to create ApplicationDefinition: %w", err)
		}
	}

	logger.Info("✅ Success.")

	return nil
}
