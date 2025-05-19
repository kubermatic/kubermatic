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

package commonseedresources

const (
	// Kyverno version and registry.
	KyvernoVersion  = "v1.14.1"
	KyvernoRegistry = "reg.kyverno.io/kyverno"
)

const (
	// Kyverno Deployments.
	KyvernoAdmissionControllerDeploymentName  = "kyverno-admission-controller"
	KyvernoBackgroundControllerDeploymentName = "kyverno-background-controller"
	KyvernoCleanupControllerDeploymentName    = "kyverno-cleanup-controller"
	KyvernoReportsControllerDeploymentName    = "kyverno-reports-controller"

	// Kyverno Services.
	KyvernoAdmissionControllerServiceName         = "kyverno-svc"
	KyvernoAdmissionControllerMetricsServiceName  = "kyverno-svc-metrics"
	KyvernoCleanupControllerServiceName           = "kyverno-cleanup-controller"
	KyvernoCleanupControllerMetricsServiceName    = "kyverno-cleanup-controller-metrics"
	KyvernoBackgroundControllerMetricsServiceName = "kyverno-background-controller-metrics"
	KyvernoReportsControllerMetricsServiceName    = "kyverno-reports-controller-metrics"

	// Kyverno ServiceAccounts.
	KyvernoAdmissionControllerServiceAccountName  = "kyverno-service-account"
	KyvernoBackgroundControllerServiceAccountName = "kyverno-background-controller"
	KyvernoCleanupControllerServiceAccountName    = "kyverno-cleanup-controller"
	KyvernoReportsControllerServiceAccountName    = "kyverno-reports-controller"

	// Kyverno Roles.
	KyvernoAdmissionControllerRoleName  = "kyverno:admission-controller"
	KyvernoBackgroundControllerRoleName = "kyverno:background-controller"
	KyvernoCleanupControllerRoleName    = "kyverno:cleanup-controller"
	KyvernoReportsControllerRoleName    = "kyverno:reports-controller"

	// Kyverno RoleBindings.
	KyvernoAdmissionControllerRoleBindingName  = "kyverno:admission-controller"
	KyvernoBackgroundControllerRoleBindingName = "kyverno:background-controller"
	KyvernoCleanupControllerRoleBindingName    = "kyverno:cleanup-controller"
	KyvernoReportsControllerRoleBindingName    = "kyverno:reports-controller"

	// Kyverno Labels.
	AdmissionControllerComponentNameLabel  = "admission-controller"
	BackgroundControllerComponentNameLabel = "background-controller"
	CleanupControllerComponentNameLabel    = "cleanup-controller"
	ReportsControllerComponentNameLabel    = "reports-controller"

	// Kyverno ConfigMaps.
	KyvernoConfigMapName        = "kyverno"
	KyvernoMetricsConfigMapName = "kyverno-metrics"
)
