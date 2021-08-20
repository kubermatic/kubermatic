/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metering

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PersistentVolumeClaimCreator creates a pvc for the metering tool where the processed data is bening saved before
// exporting it to the S3 bucket.
func PersistentVolumeClaimCreator(ctx context.Context, client ctrlruntimeclient.Client, namespace string) error {
	pvc := &corev1.PersistentVolumeClaim{}

	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: meteringDataName}, pvc); err != nil {
		if kerrors.IsNotFound(err) {
			pvc.ObjectMeta.Name = meteringDataName
			pvc.ObjectMeta.Namespace = namespace
			pvc.ObjectMeta.Labels = map[string]string{
				"app": meteringToolName,
			}

			pvc.Spec.StorageClassName = pointer.StringPtr("kubermatic-fast")
			pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			}

			pvcStorageSize, err := resource.ParseQuantity("100Gi")
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
