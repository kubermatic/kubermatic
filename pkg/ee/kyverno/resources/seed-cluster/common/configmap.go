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

package commonresources

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	kyvernoConfigMapName        = "kyverno"
	kyvernoMetricsConfigMapName = "kyverno-metrics"
)

// KyvernoConfigMapReconciler returns the function to create and update the Kyverno ConfigMap.
func KyvernoConfigMapReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return kyvernoConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			// Set labels
			cm.Labels = map[string]string{
				"app.kubernetes.io/component": "config",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			// Set annotations
			if cm.Annotations == nil {
				cm.Annotations = map[string]string{}
			}
			cm.Annotations["helm.sh/resource-policy"] = "keep"

			// Set data
			cm.Data = map[string]string{
				"enableDefaultRegistryMutation": "true",
				"defaultRegistry":               "docker.io",
				"generateSuccessEvents":         "false",
				"excludeGroups":                 "system:nodes",
				"resourceFilters": `[*/*,kyverno,*]
[Event,*,*]
[*/*,kube-system,*]
[*/*,kube-public,*]
[*/*,kube-node-lease,*]
[Node,*,*]
[Node/*,*,*]
[APIService,*,*]
[APIService/*,*,*]
[TokenReview,*,*]
[SubjectAccessReview,*,*]
[SelfSubjectAccessReview,*,*]
[Binding,*,*]
[Pod/binding,*,*]
[ReplicaSet,*,*]
[ReplicaSet/*,*,*]
[EphemeralReport,*,*]
[ClusterEphemeralReport,*,*]
[ClusterRole,*,kyverno:admission-controller]
[ClusterRole,*,kyverno:admission-controller:core]
[ClusterRole,*,kyverno:admission-controller:additional]
[ClusterRole,*,kyverno:background-controller]
[ClusterRole,*,kyverno:background-controller:core]
[ClusterRole,*,kyverno:background-controller:additional]
[ClusterRole,*,kyverno:cleanup-controller]
[ClusterRole,*,kyverno:cleanup-controller:core]
[ClusterRole,*,kyverno:cleanup-controller:additional]
[ClusterRole,*,kyverno:reports-controller]
[ClusterRole,*,kyverno:reports-controller:core]
[ClusterRole,*,kyverno:reports-controller:additional]
[ClusterRoleBinding,*,kyverno:admission-controller]
[ClusterRoleBinding,*,kyverno:background-controller]
[ClusterRoleBinding,*,kyverno:cleanup-controller]
[ClusterRoleBinding,*,kyverno:reports-controller]
[ServiceAccount,kyverno,kyverno-admission-controller]
[ServiceAccount/*,kyverno,kyverno-admission-controller]
[ServiceAccount,kyverno,kyverno-background-controller]
[ServiceAccount/*,kyverno,kyverno-background-controller]
[ServiceAccount,kyverno,kyverno-cleanup-controller]
[ServiceAccount/*,kyverno,kyverno-cleanup-controller]
[ServiceAccount,kyverno,kyverno-reports-controller]
[ServiceAccount/*,kyverno,kyverno-reports-controller]
[Role,kyverno,kyverno:admission-controller]
[Role,kyverno,kyverno:background-controller]
[Role,kyverno,kyverno:cleanup-controller]
[Role,kyverno,kyverno:reports-controller]
[RoleBinding,kyverno,kyverno:admission-controller]
[RoleBinding,kyverno,kyverno:background-controller]
[RoleBinding,kyverno,kyverno:cleanup-controller]
[RoleBinding,kyverno,kyverno:reports-controller]
[ConfigMap,kyverno,kyverno]
[ConfigMap,kyverno,kyverno-metrics]
[Deployment,kyverno,kyverno-admission-controller]
[Deployment/*,kyverno,kyverno-admission-controller]
[Deployment,kyverno,kyverno-background-controller]
[Deployment/*,kyverno,kyverno-background-controller]
[Deployment,kyverno,kyverno-cleanup-controller]
[Deployment/*,kyverno,kyverno-cleanup-controller]
[Deployment,kyverno,kyverno-reports-controller]
[Deployment/*,kyverno,kyverno-reports-controller]
[Pod,kyverno,kyverno-admission-controller-*]
[Pod/*,kyverno,kyverno-admission-controller-*]
[Pod,kyverno,kyverno-background-controller-*]
[Pod/*,kyverno,kyverno-background-controller-*]
[Pod,kyverno,kyverno-cleanup-controller-*]
[Pod/*,kyverno,kyverno-cleanup-controller-*]
[Pod,kyverno,kyverno-reports-controller-*]
[Pod/*,kyverno,kyverno-reports-controller-*]
[Job,kyverno,kyverno-hook-pre-delete]
[Job/*,kyverno,kyverno-hook-pre-delete]
[NetworkPolicy,kyverno,kyverno-admission-controller]
[NetworkPolicy/*,kyverno,kyverno-admission-controller]
[NetworkPolicy,kyverno,kyverno-background-controller]
[NetworkPolicy/*,kyverno,kyverno-background-controller]
[NetworkPolicy,kyverno,kyverno-cleanup-controller]
[NetworkPolicy/*,kyverno,kyverno-cleanup-controller]
[NetworkPolicy,kyverno,kyverno-reports-controller]
[NetworkPolicy/*,kyverno,kyverno-reports-controller]
[PodDisruptionBudget,kyverno,kyverno-admission-controller]
[PodDisruptionBudget/*,kyverno,kyverno-admission-controller]
[PodDisruptionBudget,kyverno,kyverno-background-controller]
[PodDisruptionBudget/*,kyverno,kyverno-background-controller]
[PodDisruptionBudget,kyverno,kyverno-cleanup-controller]
[PodDisruptionBudget/*,kyverno,kyverno-cleanup-controller]
[PodDisruptionBudget,kyverno,kyverno-reports-controller]
[PodDisruptionBudget/*,kyverno,kyverno-reports-controller]
[Service,kyverno,kyverno-svc]
[Service/*,kyverno,kyverno-svc]
[Service,kyverno,kyverno-svc-metrics]
[Service/*,kyverno,kyverno-svc-metrics]
[Service,kyverno,kyverno-background-controller-metrics]
[Service/*,kyverno,kyverno-background-controller-metrics]
[Service,kyverno,kyverno-cleanup-controller]
[Service/*,kyverno,kyverno-cleanup-controller]
[Service,kyverno,kyverno-cleanup-controller-metrics]
[Service/*,kyverno,kyverno-cleanup-controller-metrics]
[Service,kyverno,kyverno-reports-controller-metrics]
[Service/*,kyverno,kyverno-reports-controller-metrics]
[ServiceMonitor,kyverno,kyverno-admission-controller]
[ServiceMonitor,kyverno,kyverno-background-controller]
[ServiceMonitor,kyverno,kyverno-cleanup-controller]
[ServiceMonitor,kyverno,kyverno-reports-controller]
[Secret,kyverno,kyverno-svc.kyverno.svc.*]
[Secret,kyverno,kyverno-cleanup-controller.kyverno.svc.*]`,
				"updateRequestThreshold": "1000",
				"webhooks":               "{\"namespaceSelector\":{\"matchExpressions\":[{\"key\":\"kubernetes.io/metadata.name\",\"operator\":\"NotIn\",\"values\":[\"kube-system\"]},{\"key\":\"kubernetes.io/metadata.name\",\"operator\":\"NotIn\",\"values\":[\"kyverno\"]}],\"matchLabels\":null}}",
				"webhookAnnotations":     "{\"admissions.enforcer/disabled\":\"true\"}",
			}

			return cm, nil
		}
	}
}

// KyvernoMetricsConfigMapReconciler returns the function to create and update the Kyverno metrics ConfigMap.
func KyvernoMetricsConfigMapReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return kyvernoMetricsConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			// Set labels
			cm.Labels = map[string]string{
				"app.kubernetes.io/component": "config",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			// Set data
			cm.Data = map[string]string{
				"namespaces":       "{\"exclude\":[],\"include\":[]}",
				"metricsExposure":  "{\"kyverno_admission_requests_total\":{\"disabledLabelDimensions\":[\"resource_namespace\"]},\"kyverno_admission_review_duration_seconds\":{\"disabledLabelDimensions\":[\"resource_namespace\"]},\"kyverno_cleanup_controller_deletedobjects_total\":{\"disabledLabelDimensions\":[\"resource_namespace\",\"policy_namespace\"]},\"kyverno_policy_execution_duration_seconds\":{\"disabledLabelDimensions\":[\"resource_namespace\",\"resource_request_operation\"]},\"kyverno_policy_results_total\":{\"disabledLabelDimensions\":[\"resource_namespace\",\"policy_namespace\"]},\"kyverno_policy_rule_info_total\":{\"disabledLabelDimensions\":[\"resource_namespace\",\"policy_namespace\"]}}",
				"bucketBoundaries": "0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 15, 20, 25, 30",
			}

			return cm, nil
		}
	}
}
