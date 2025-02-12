//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resources

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ptr(s string) *string {
	return &s
}

// ServiceReconciler returns the function to create and update the Kyverno background controller service.
func ServiceReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return backgroundControllerName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
			}

			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
			}

			s.Spec.Ports = []corev1.ServicePort{
				{
					Name:        "https",
					Port:        443,
					TargetPort:  intstr.FromString("webhook"),
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: ptr("https"),
				},
			}

			return s, nil
		}
	}
}

// MetricsServiceReconciler returns the function to create and update the Kyverno background controller metrics service.
func MetricsServiceReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return backgroundControllerName + "-metrics", func(s *corev1.Service) (*corev1.Service, error) {
			s.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
			}

			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
			}

			s.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "metrics-port",
					Port:       8000,
					TargetPort: intstr.FromString("metrics"),
					Protocol:   corev1.ProtocolTCP,
				},
			}

			return s, nil
		}
	}
}
