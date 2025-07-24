//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2024 Kubermatic GmbH

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

package synccontroller

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func cbslReconcilerFactory(cbsl *kubermaticv1.ClusterBackupStorageLocation) kkpreconciling.NamedClusterBackupStorageLocationReconcilerFactory {
	return func() (string, kkpreconciling.ClusterBackupStorageLocationReconciler) {
		return cbsl.Name, func(existing *kubermaticv1.ClusterBackupStorageLocation) (*kubermaticv1.ClusterBackupStorageLocation, error) {
			if existing.ObjectMeta.Labels == nil {
				existing.ObjectMeta.Labels = map[string]string{}
			}
			for k, v := range cbsl.ObjectMeta.Labels {
				existing.ObjectMeta.Labels[k] = v
			}
			existing.Spec = cbsl.Spec
			return existing, nil
		}
	}
}

func secretReconcilerFactory(s *corev1.Secret) reconciling.NamedSecretReconcilerFactory {
	return func() (name string, create reconciling.SecretReconciler) {
		return s.Name, func(existing *corev1.Secret) (*corev1.Secret, error) {
			if existing.UID != "" && s.UID == existing.UID {
				return nil, nil
			}
			existing.Labels = s.Labels
			existing.Annotations = s.Annotations
			existing.Data = s.Data
			return existing, nil
		}
	}
}
