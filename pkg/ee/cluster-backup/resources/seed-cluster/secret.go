//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

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

package seedclusterresources

import (
	"context"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretReconciler returns a function to create the Secret containing the backup destination credentials.
func SecretReconciler(ctx context.Context, client ctrlruntimeclient.Client, data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return cloudCredentialsSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			refName := data.ClusterBackupConfig().Destination.Credentials.Name
			refNamespace := data.ClusterBackupConfig().Destination.Credentials.Namespace

			secret := &corev1.Secret{}
			if err := client.Get(ctx, types.NamespacedName{Name: refName, Namespace: refNamespace}, secret); err != nil {
				return nil, fmt.Errorf("failed to get backup destination credentials secret: %w", err)
			}
			cm.Data = secret.Data
			return cm, nil
		}
	}
}
