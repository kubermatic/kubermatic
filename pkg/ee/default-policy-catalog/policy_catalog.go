/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

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

package defaultpolicycatalog

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
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

func DeployDefaultPolicyTemplateCatalog(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	logger.Info("Deploying default Policy Template catalog")
	sublogger := log.Prefix(logger, "   ")

	if !opt.DeployDefaultPolicyTemplateCatalog {
		sublogger.Info("Skipping deployment of default Policy Template catalog, set --deploy-default-policy-catalog to enable it.")
		return nil
	}

	policyTemplateFiles, err := GetPolicyTemplates()
	if err != nil {
		return fmt.Errorf("failed to fetch PolicyTemplates: %w", err)
	}

	// Wait for webhook to be ready
	webhook := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.WebhookDeploymentName,
			Namespace: resources.KubermaticNamespace,
		},
	}
	if err := util.WaitForDeploymentRollout(ctx, kubeClient, webhook, opt.Versions.GitVersion, 5*time.Minute); err != nil {
		return fmt.Errorf("failed waiting for webhook: %w", err)
	}

	creators := []kkpreconciling.NamedPolicyTemplateReconcilerFactory{}
	for _, file := range policyTemplateFiles {
		b, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read PolicyTemplate: %w", err)
		}

		policyTemplate := &kubermaticv1.PolicyTemplate{}
		err = yaml.Unmarshal(b, policyTemplate)

		if err != nil {
			return fmt.Errorf("failed to parse PolicyTemplate: %w", err)
		}

		creators = append(creators, policyTemplateReconcilerFactory(policyTemplate))
	}

	if err = kkpreconciling.ReconcilePolicyTemplates(ctx, creators, "", kubeClient); err != nil {
		return fmt.Errorf("failed to apply PolicyTemplates: %w", err)
	}

	logger.Info("Successfully deployed default Policy Template catalog")

	return nil
}

func policyTemplateReconcilerFactory(policyTemplate *kubermaticv1.PolicyTemplate) kkpreconciling.NamedPolicyTemplateReconcilerFactory {
	return func() (string, kkpreconciling.PolicyTemplateReconciler) {
		return policyTemplate.Name, func(pt *kubermaticv1.PolicyTemplate) (*kubermaticv1.PolicyTemplate, error) {
			kubernetes.EnsureLabels(pt, policyTemplate.Labels)
			kubernetes.EnsureAnnotations(pt, policyTemplate.Annotations)

			if pt.Spec.Enforced {
				policyTemplate.Spec.Enforced = true
			}

			if pt.Spec.Default {
				policyTemplate.Spec.Default = true
			}

			if pt.Spec.Target != nil && pt.Spec.Target.ProjectSelector != nil {
				policyTemplate.Spec.Target.ProjectSelector = pt.Spec.Target.ProjectSelector
			}

			if pt.Spec.Target != nil && pt.Spec.Target.ClusterSelector != nil {
				policyTemplate.Spec.Target.ClusterSelector = pt.Spec.Target.ClusterSelector
			}

			pt.Spec = policyTemplate.Spec
			return pt, nil
		}
	}
}
