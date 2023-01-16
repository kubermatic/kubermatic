/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package image

import (
	"context"
	"fmt"

	"github.com/imdario/mergo"
	"go.uber.org/zap"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ImagesNamespace namespace contains globally available custom images and cached standard images.
	ImagesNamespace = "kubevirt-images"
	// imageReconcilerCustom is the type for to reconcile for custom images.
	imageReconcilerCustom imageType = "custom"
	// imageReconcilerStandard is the type for to reconcile for standard images.
	imageReconcilerStandard imageType = "standard"
	// dataVolumeDeleteAfterCompletionAnnotationKey
	dataVolumeDeleteAfterCompletionAnnotationKey = "cdi.kubevirt.io/storage.deleteAfterCompletion"
)

type (
	imageType                  string
	dataVolumeAnnotationFilter func(map[string]string) bool
)

type imageReconciler struct {
	ctx    context.Context
	logger *zap.SugaredLogger
	client ctrlruntimeclient.Client
	// Standard / Custom
	imageType imageType
	// namespace in the infra KubeVirt cluster where DataVolume will be reconciled.
	namespace string
	// filter to apply when getting the existing DataVolume on the infra KubeVirt cluster.
	annotationFilter dataVolumeAnnotationFilter
	// extra annotations to add to a DataVolume (not in PreAllocatedDataVolume).
	extraAnnotations map[string]string
	// list of PreAllocatedDataVolumes to reconcile
	preAllocatedDataVolumes []kubermaticv1.PreAllocatedDataVolume
}

func newImageReconciler(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client,
	preAllocatedDataVolumes []kubermaticv1.PreAllocatedDataVolume, iType imageType, annotationFilter dataVolumeAnnotationFilter,
	extraAnnotations map[string]string) *imageReconciler {
	return &imageReconciler{
		ctx:                     ctx,
		imageType:               iType,
		logger:                  logger.With("type", iType),
		client:                  client,
		annotationFilter:        annotationFilter,
		extraAnnotations:        extraAnnotations,
		preAllocatedDataVolumes: preAllocatedDataVolumes,
	}
}

func (ir *imageReconciler) newDataVolume(kdv kubermaticv1.PreAllocatedDataVolume) (*cdiv1beta1.DataVolume, error) {
	dvSize, err := resource.ParseQuantity(kdv.Size)
	if err != nil {
		return nil, err
	}
	mergo.Merge(kdv.Annotations, ir.extraAnnotations, mergo.WithOverride)
	if err != nil {
		return nil, err
	}

	return &cdiv1beta1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kdv.Name,
			Namespace: ir.namespace,
			// Keep any annotation from PreAllocatedDataVolume and append the extra annotations.
			Annotations: kdv.Annotations,
		},
		Spec: cdiv1beta1.DataVolumeSpec{
			Source: &cdiv1beta1.DataVolumeSource{
				HTTP: &cdiv1beta1.DataVolumeSourceHTTP{
					URL: kdv.URL,
				},
			},
			PVC: &corev1.PersistentVolumeClaimSpec{
				StorageClassName: utilpointer.String(kdv.StorageClass),
				AccessModes: []corev1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: dvSize},
				},
			},
		},
	}, nil
}

// hasSpecChanged checks if the Spec has changed between the expected PreAllocationDataVolume and the existing DataVolume
func (ir *imageReconciler) hasSpecChanged(expected kubermaticv1.PreAllocatedDataVolume, existing cdiv1beta1.DataVolume) (bool, error) {
	expectedDataVolume, err := ir.newDataVolume(expected)
	if err != nil {
		return false, err
	}

	return (!equality.Semantic.DeepEqual(expectedDataVolume.Spec.Source, existing.Spec.Source) ||
		!equality.Semantic.DeepEqual(expectedDataVolume.Spec.PVC, existing.Spec.PVC)), nil
}

func (ir *imageReconciler) handleUpdatedImages(existings map[string]cdiv1beta1.DataVolume) error {
	// Parse expected PreAllocation DataVolumes
	// If spec is updated -> needs to first delete the DataVolume as Spec update is not possible.
	for _, kdv := range ir.preAllocatedDataVolumes {
		existing, exist := existings[kdv.Name]
		if exist {
			hasSpecChanged, err := ir.hasSpecChanged(kdv, existing)
			if err != nil {
				return err
			}
			if hasSpecChanged {
				if err := ir.client.Delete(ir.ctx, &existing); ctrlruntimeclient.IgnoreNotFound(err) != nil {
					return err
				}
			}
		}
	}
	return nil
}

// reconcile is a generic mechanism to reconcile images.
// 1- Ensures that all DataVolumes in kdv are reconciled
func (ir *imageReconciler) reconcile() error {
	// dvToRemove first contains all the existing DV.
	// Then each dv reconciled is removed from this map.
	// When reconcile loop is over, it will contain the data volumes that are existing in the infra KubeVirt cluster,
	// But not in the list onf volumes to reconcile: they should be deleted from the infra clusters.
	dvToRemove, err := getExistingDataVolumes(ir.ctx, ir.namespace, ir.client, ir.annotationFilter)
	if err != nil {
		return err
	}

	// DataVolume Spec can not be changed. If changed, delete the DataVolume and re-create a new one,
	// using the standard reconcile mechanism.
	ir.handleUpdatedImages(dvToRemove)

	// Reconcile what needs to be.
	for _, kdv := range ir.preAllocatedDataVolumes {
		if err = ir.reconcileDataVolumeImage(&kdv); err != nil {
			return err
		}
		// remove the created/updated dv from the removal map.
		delete(dvToRemove, kdv.Name)
	}

	// Cleanup extra DataVolumes
	return ir.deleteDataVolumes(dvToRemove)
}

func (ir *imageReconciler) reconcileDataVolumeImage(kdv *kubermaticv1.PreAllocatedDataVolume) error {
	if kdv.Annotations == nil {
		kdv.Annotations = make(map[string]string)
	}

	dv, err := ir.newDataVolume(*kdv)
	if err != nil {
		return fmt.Errorf("error generating image Data Volume: %w", err)
	}

	if err = reconcileDataVolume(ir.ctx, ir.client, dv, ir.namespace); err != nil {
		return fmt.Errorf("failed to reconcile DataVolume: %w", err)
	}

	return nil
}

// deleteDataVolumes specified in dvToBeRemoved.
func (ir *imageReconciler) deleteDataVolumes(dvToBeRemoved map[string]cdiv1beta1.DataVolume) error {
	for _, dv := range dvToBeRemoved {
		if err := ir.client.Delete(ir.ctx, &dv); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

func dataVolumeReconciler(datavolume *cdiv1beta1.DataVolume) kkpreconciling.NamedDataVolumeReconcilerFactory {
	return func() (name string, create kkpreconciling.DataVolumeReconciler) {
		return datavolume.Name, func(dv *cdiv1beta1.DataVolume) (*cdiv1beta1.DataVolume, error) {
			dv.Annotations = datavolume.Annotations
			dv.Labels = datavolume.Labels
			dv.Spec = datavolume.Spec
			return dv, nil
		}
	}
}

func reconcileDataVolume(ctx context.Context, client ctrlruntimeclient.Client, dataVolume *cdiv1beta1.DataVolume, namespace string) error {
	dvCreator := []kkpreconciling.NamedDataVolumeReconcilerFactory{
		dataVolumeReconciler(dataVolume),
	}
	return kkpreconciling.ReconcileDataVolumes(ctx, dvCreator, namespace, client)
}

// getExistingDataVolumes returns a map of DataVolumes based on annotation filter.
func getExistingDataVolumes(ctx context.Context, namespace string, client ctrlruntimeclient.Client, annotationFilter dataVolumeAnnotationFilter) (map[string]cdiv1beta1.DataVolume, error) {
	existingDiskList := cdiv1beta1.DataVolumeList{}
	listOption := ctrlruntimeclient.ListOptions{
		Namespace: namespace,
	}
	if err := client.List(ctx, &existingDiskList, &listOption); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return nil, err
	}

	dvMap := make(map[string]cdiv1beta1.DataVolume)
	for _, dv := range existingDiskList.Items {
		if annotationFilter(dv.Annotations) {
			dvMap[dv.Name] = dv
		}
	}
	return dvMap, nil
}
