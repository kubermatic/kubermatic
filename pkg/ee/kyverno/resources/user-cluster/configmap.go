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

package userclusterresources

import (
	"fmt"
	"strings"

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

			// Set data
			cm.Data = map[string]string{
				"excludeGroups":          "system:nodes",
				"resourceFilters":        getResourceFilters(cluster.Status.NamespaceName),
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

func getResourceFilters(namespace string) string {
	filters := []string{
		"[Event,*,*]",
		"[*/*,kube-system,*]",
		"[*/*,kube-public,*]",
		"[*/*,kube-node-lease,*]",
		"[Node,*,*]",
		fmt.Sprintf("[ConfigMap,%s,kyverno]", namespace),
		fmt.Sprintf("[ConfigMap,%s,kyverno-metrics]", namespace),
		"[Node/*,*,*]",
		"[APIService,*,*]",
		"[APIService/*,*,*]",
		"[TokenReview,*,*]",
		"[SubjectAccessReview,*,*]",
		"[SelfSubjectAccessReview,*,*]",
		"[Binding,*,*]",
		"[Pod/binding,*,*]",
		"[ReplicaSet,*,*]",
		"[ReplicaSet/*,*,*]",
		"[EphemeralReport,*,*]",
		"[ClusterEphemeralReport,*,*]",
	}

	return strings.Join(filters, "\n")
}
