// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Loodse GmbH

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

package metering

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// persistentVolumeClaimCreator creates a pvc for the metering tool where the processed data is being saved before
// exporting it to the S3 bucket.
func persistentVolumeClaimCreator(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed) error {
	pvc := &corev1.PersistentVolumeClaim{}

	if err := client.Get(ctx, types.NamespacedName{Namespace: seed.Namespace, Name: meteringDataName}, pvc); err != nil {
		if kerrors.IsNotFound(err) {
			pvc.ObjectMeta.Name = meteringDataName
			pvc.ObjectMeta.Namespace = seed.Namespace
			pvc.ObjectMeta.Labels = map[string]string{
				"app": meteringToolName,
			}

			pvc.Spec.StorageClassName = pointer.StringPtr(seed.Spec.Metering.StorageClassName)
			pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			}

			pvcStorageSize, err := resource.ParseQuantity(seed.Spec.Metering.StorageSize)
			if err != nil {
				return fmt.Errorf("failed to parse value of metering pvc storage size %q: %v", "100Gi", err)
			}

			pvc.Spec.Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": pvcStorageSize,
				},
			}

			if err := client.Create(ctx, pvc); err != nil {
				return fmt.Errorf("failed to create pvc %v for the metering tool: %v", meteringDataName, err)
			}
		}

		return err
	}

	return nil
}
