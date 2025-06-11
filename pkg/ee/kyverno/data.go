//go:build ee

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

package kyverno

import (
	admissioncontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/admission-controller"
	backgroundcontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/background-controller"
	cleanupcontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/cleanup-controller"
	reportscontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/reports-controller"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"
)

func GetDeploymentReconcilers(data *resources.TemplateData) []reconciling.NamedDeploymentReconcilerFactory {
	deployments := []reconciling.NamedDeploymentReconcilerFactory{
		admissioncontrollerresources.DeploymentReconciler(data.Cluster()),
		backgroundcontrollerresources.DeploymentReconciler(data.Cluster()),
		cleanupcontrollerresources.DeploymentReconciler(data.Cluster()),
		reportscontrollerresources.DeploymentReconciler(data.Cluster()),
	}
	return deployments
}
