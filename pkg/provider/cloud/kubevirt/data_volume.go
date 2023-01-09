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

package kubevirt

import (
	"context"
	"fmt"

	"github.com/imdario/mergo"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	reconciling2 "k8c.io/reconciler/pkg/reconciling"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dataVolumeStandardImageAnnotationKey = "kubevirt-initialization.k8c.io/standard-image"
	// CDI: GC can be configured in [CDIConfig](cdi-config.md), so users cannot assume the DV exists after completion.
	// When the desired PVC exists, but its DV does not exist, it means that the PVC was successfully populated and the DV was garbage collected.
	// To prevent a DV from being garbage collected, it should be annotated with:
	// cdi.kubevirt.io/storage.deleteAfterCompletion: "false".
	// We need those DV to not be GC.
	dataVolumeDeleteAfterCompletionAnnotationKey = "cdi.kubevirt.io/storage.deleteAfterCompletion"
	dataVolumeOsAnnotationKeyForCustomDisk       = "cdi.kubevirt.io/os-type"
	dataVolumeStandarImageDefaultSize            = "11Gi"
	kubevirtImagesClusterRole                    = "datavolume-cloner"
	kubevirtImagesRoleBinding                    = "allow-datavolume-cloning"
)

type dataVolumeAnnotationFilter func(map[string]string) bool

func customDataVolumeAnnotations() map[string]string {
	return map[string]string{
		dataVolumeDeleteAfterCompletionAnnotationKey: "false",
	}
}
func customDataVolumeFilter(annotations map[string]string) bool {
	return annotations != nil && annotations[dataVolumeOsAnnotationKeyForCustomDisk] != ""
}

func standardDataVolumeAnnotations() map[string]string {
	return map[string]string{
		dataVolumeStandardImageAnnotationKey:         "true",
		dataVolumeDeleteAfterCompletionAnnotationKey: "false",
	}
}
func standardDataVolumeFilter(annotations map[string]string) bool {
	return annotations != nil && annotations[dataVolumeStandardImageAnnotationKey] != ""
}

func dataVolumeReconciler(datavolume *cdiv1beta1.DataVolume) reconciling.NamedDataVolumeReconcilerFactory {
	return func() (name string, create reconciling.DataVolumeReconciler) {
		return datavolume.Name, func(dv *cdiv1beta1.DataVolume) (*cdiv1beta1.DataVolume, error) {
			dv.Annotations = datavolume.Annotations
			dv.Labels = datavolume.Labels
			dv.Spec = datavolume.Spec
			return dv, nil
		}
	}
}

func reconcileDataVolume(ctx context.Context, client ctrlruntimeclient.Client, dataVolume *cdiv1beta1.DataVolume, namespace string) error {
	dvCreator := []reconciling.NamedDataVolumeReconcilerFactory{
		dataVolumeReconciler(dataVolume),
	}
	return reconciling.ReconcileDataVolumes(ctx, dvCreator, namespace, client)
}

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
	options                 options
}

type imageType string

const (
	// imageReconcilerCustom is the type for the reconcile for custom images.
	imageReconcilerCustom imageType = "custom"
	// imageReconcilerStandard is the type for the reconcile for standard images.
	imageReconcilerStandard imageType = "standard"
)

func newImageReconciler(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client,
	preAllocatedDataVolumes []kubermaticv1.PreAllocatedDataVolume,
	iType imageType, namespace string,
	annotationFilter dataVolumeAnnotationFilter, extraAnnotations map[string]string,
	options options) *imageReconciler {
	return &imageReconciler{
		ctx:                     ctx,
		imageType:               iType,
		logger:                  logger.With("type", iType),
		client:                  client,
		annotationFilter:        annotationFilter,
		extraAnnotations:        extraAnnotations,
		preAllocatedDataVolumes: preAllocatedDataVolumes,
		options:                 options,
	}
}

type options struct {
	// upgradeIfChanged: if it detects a change in the DV to reconcile, as a DV Spec cannot be updated,
	// the DV needs to be deleted and re-created.
	upgradeIfChanged bool
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
			if hasSpecChanged && ir.options.upgradeIfChanged {
				if err := ir.client.Delete(ir.ctx, &existing); ctrlruntimeclient.IgnoreNotFound(err) != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ir *imageReconciler) handleCleanup(existing map[string]cdiv1beta1.DataVolume) error {
	return nil
}

// reconcile is a generic mechanism to reconcile images.
// 1- Ensures that all DataVolumes in kdv are reconciled
func (ir *imageReconciler) reconcile() error {
	// dvToRemove first contains all the existing DV.
	// Then each dv reconciled is removed from this map.
	// When reconcile loop is over, it will contain the data volumes that are existing in the infra KubeVirt cluster,
	// But not in the list onf volumes to reconcile: they should be delete from the infra clusters.
	dvToRemove, err := getExistingDataVolumes(ir.ctx, ir.namespace, ir.client, ir.annotationFilter)
	if err != nil {
		return err
	}

	// DataVolume Spec can not be changed. If changed, delete the DataVolume and re-create a new one,
	// using the standard reconcile mechanism.
	ir.handleUpdatedImages(dvToRemove)

	// Reconcile what needs to be.
	for _, kdv := range ir.preAllocatedDataVolumes {
		// Fail at first error.
		// TODO: check if we really want to stop at first failure or best effort.
		if err = ir.reconcileImage(&kdv); err != nil {
			return err
		}
		// remove the created/updated dv from the removal map.
		delete(dvToRemove, kdv.Name)
	}

	// Cleanup extra DataVolumes (in infra but should not be)
	return ir.deleteDataVolumes(dvToRemove)
}

func (ir *imageReconciler) reconcileImage(kdv *kubermaticv1.PreAllocatedDataVolume) error {
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

// reconcileCustomImages reconciles the custom-disks from cluster.
func reconcileCustomImages(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	dvr := newImageReconciler(ctx, logger, client,
		customImages(cluster),
		imageReconcilerCustom, cluster.Status.NamespaceName, customDataVolumeFilter, customDataVolumeAnnotations(),
		options{upgradeIfChanged: true})
	return dvr.reconcile()
}

// reconcileStandardImagesCache reconciles the DataVolumes for standard VM images if cloning is enabled.
func reconcileStandardImagesCache(ctx context.Context, dc *kubermaticv1.DatacenterSpecKubevirt, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	dvr := newImageReconciler(ctx, logger, client,
		standardImages(dc),
		imageReconcilerCustom, KubeVirtImagesNamespace, standardDataVolumeFilter, standardDataVolumeAnnotations(),
		options{upgradeIfChanged: true})

	return dvr.reconcile()
}

// standardImages returns a list of PreAllocatedDataVolumes based on the list of standard images contained in the datacenter,
func standardImages(dc *kubermaticv1.DatacenterSpecKubevirt) []kubermaticv1.PreAllocatedDataVolume {
	dvs := make([]kubermaticv1.PreAllocatedDataVolume, 0)
	httpSource := dc.Images.HTTP

	// ImageSize : default
	imageSize := dataVolumeStandarImageDefaultSize
	if httpSource.ImageCloning.DataVolumeSize != "" {
		imageSize = httpSource.ImageCloning.DataVolumeSize
	}

	// For this version, we handle only HTTP sources
	for os, osVersion := range httpSource.OperatingSystems {
		for version, url := range osVersion {
			dv := kubermaticv1.PreAllocatedDataVolume{
				Name:         fmt.Sprintf("%s-%s", os, version),
				URL:          url,
				Size:         imageSize,
				StorageClass: httpSource.ImageCloning.StorageClass,
			}
			dvs = append(dvs, dv)
		}
	}
	return dvs
}

// customImages returns a list of PreAllocatedDataVolumes based on the list of standard images contained in the datacenter,
func customImages(cluster *kubermaticv1.Cluster) []kubermaticv1.PreAllocatedDataVolume {
	return cluster.Spec.Cloud.Kubevirt.PreAllocatedDataVolumes
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

func kubeVirtImagesClusterRoleCreator(name string) reconciling2.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling2.ClusterRoleReconciler) {
		return name, func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"cdi.kubevirt.io"},
					Resources: []string{"datavolumes/source"},
					Verbs:     []string{"*"},
				},
			}
			return r, nil
		}
	}
}

func kubeVirtImagesRoleBindingCreator(name, namespace string) reconciling2.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling2.RoleBindingReconciler) {
		return name, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "default",
					Namespace: namespace,
				},
			}

			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     kubevirtImagesClusterRole,
			}

			return rb, nil
		}
	}
}

func reconcileKubeVirtImagesRoleRoleBinding(ctx context.Context, sourceNamespace, destinationNamespace string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, client ctrlruntimeclient.Client) (*kubermaticv1.Cluster, error) {
	cluster, err := update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerClonerRoleBinding)
	})
	if err != nil {
		return cluster, err
	}
	clusterRoleCreator := []reconciling2.NamedClusterRoleReconcilerFactory{
		kubeVirtImagesClusterRoleCreator(kubevirtImagesClusterRole),
	}

	if err = reconciling2.ReconcileClusterRoles(ctx, clusterRoleCreator, "", client); err != nil {
		return cluster, err
	}

	roleBindingCreators := []reconciling2.NamedRoleBindingReconcilerFactory{
		kubeVirtImagesRoleBindingCreator(fmt.Sprintf("%s-%s", kubevirtImagesRoleBinding, destinationNamespace), destinationNamespace),
	}
	if err = reconciling2.ReconcileRoleBindings(ctx, roleBindingCreators, sourceNamespace, client); err != nil {
		return cluster, err
	}

	return cluster, nil
}

func deleteKubeVirtImagesRoleBinding(ctx context.Context, name string, client ctrlruntimeclient.Client) error {
	roleBinding := &rbacv1.RoleBinding{}
	if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: KubeVirtImagesNamespace}, roleBinding); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}
	return client.Delete(ctx, roleBinding)
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
		// only consider DV which are filtered
		if annotationFilter(dv.Annotations) {
			dvMap[dv.Name] = dv
		}
	}
	return dvMap, nil
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
