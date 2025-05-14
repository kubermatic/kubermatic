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

package cleanupcontrollerresources

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ServiceName        = "kyverno-cleanup-controller"
	MetricsServiceName = "kyverno-cleanup-controller-metrics"
)

// ServiceReconciler returns the function to create and update the Kyverno cleanup controller service.
func ServiceReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return ServiceName, func(svc *corev1.Service) (*corev1.Service, error) {
			svc.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			svc.Spec.Type = corev1.ServiceTypeClusterIP

			svc.Spec.Ports = []corev1.ServicePort{
				{
					Name:        "https",
					Port:        443,
					Protocol:    corev1.ProtocolTCP,
					TargetPort:  intstr.FromString("https"),
					AppProtocol: stringPtr("https"),
				},
			}

			svc.Spec.Selector = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
			}

			return svc, nil
		}
	}
}

// MetricsServiceReconciler returns the function to create and update the Kyverno cleanup controller metrics service.
func MetricsServiceReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return MetricsServiceName, func(svc *corev1.Service) (*corev1.Service, error) {
			svc.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			svc.Spec.Type = corev1.ServiceTypeClusterIP

			svc.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "metrics-port",
					Port:       8000,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8000),
				},
			}

			svc.Spec.Selector = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
			}

			return svc, nil
		}
	}
}

func stringPtr(s string) *string {
	return &s
}
