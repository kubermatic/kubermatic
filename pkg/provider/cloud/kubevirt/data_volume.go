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

	"go.uber.org/zap"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	reconciling2 "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dataVolumeStandardImageAnnotation = "kubevirt-initialization.k8c.io/standard-image"
	dataVolumeRetainAnnotation        = "cdi.kubevirt.io/storage.deleteAfterCompletion"
	dataVolumeStandardImageSize       = "11Gi"
	kubevirtImagesClusterRole         = "datavolume-cloner"
	kubevirtImagesRoleBinding         = "allow-datavolume-cloning"
)

func dataVolumeReconciler(datavolume *cdiv1beta1.DataVolume) reconciling.NamedDataVolumeReconcilerFactory {
	return func() (name string, create reconciling.DataVolumeReconciler) {
		return datavolume.Name, func(dv *cdiv1beta1.DataVolume) (*cdiv1beta1.DataVolume, error) {
			dv.Annotations = datavolume.Annotations
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

// reconcileCustomImages reconciles the custom-disks from cluster.
func reconcileCustomImages(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, isCustomImagesEnabled bool) error {
	if isCustomImagesEnabled {
		for _, d := range cluster.Spec.Cloud.Kubevirt.PreAllocatedDataVolumes {
			if d.Annotations == nil {
				d.Annotations = make(map[string]string)
			}
			d.Annotations[dataVolumeRetainAnnotation] = "false"
			dv, err := newDataVolume(d, cluster.Status.NamespaceName)
			if err != nil {
				logger.Error("error generating custom image Data Volume: %s", err)
				continue
			}
			if err = reconcileDataVolume(ctx, client, dv, cluster.Status.NamespaceName); err != nil {
				logger.Error("failed to reconcile DataVolume: %s", err)
				continue
			}
		}
	}
	return nil
}

// reconcileStandardImagesCache reconciles the DataVolumes for standard VM images if cloning is enabled.
func reconcileStandardImagesCache(ctx context.Context, dc *kubermaticv1.DatacenterSpecKubevirt, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	existingDiskList := cdiv1beta1.DataVolumeList{}
	listOption := ctrlruntimeclient.ListOptions{
		Namespace: KubeVirtImagesNamespace,
	}
	if err := client.List(ctx, &existingDiskList, &listOption); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return err
	}

	// pdvToBeRemoved contains info about pdv that are going to be removed.
	pdvToBeRemoved := make(map[string]cdiv1beta1.DataVolume)
	for _, dv := range existingDiskList.Items {
		// only consider preAllocated DV and ignore custom-disks.
		if dv.Annotations[dataVolumeStandardImageAnnotation] == "true" {
			pdvToBeRemoved[dv.Name] = dv
		}
	}

	for os, osVersion := range dc.Images.HTTP.OperatingSystems {
		for version, url := range osVersion {
			pdvCreateOrUpdate := kubermaticv1.PreAllocatedDataVolume{
				Name: fmt.Sprintf("%s-%s", os, version),
				Annotations: map[string]string{
					dataVolumeRetainAnnotation:        "false",
					dataVolumeStandardImageAnnotation: "true",
				},
				URL:          url,
				Size:         dataVolumeStandardImageSize,
				StorageClass: dc.Images.HTTP.ImageCloning.StorageClass,
			}

			// remove the created/updated pdv from the removal map.
			delete(pdvToBeRemoved, pdvCreateOrUpdate.Name)

			dv, err := newDataVolume(pdvCreateOrUpdate, KubeVirtImagesNamespace)
			if err != nil {
				logger.Error("error generating new Data Volume: %s", err)
				continue
			}
			if err = reconcileDataVolume(ctx, client, dv, KubeVirtImagesNamespace); err != nil {
				logger.Error("failed to reconcile DataVolume: %s", err)
				continue
			}
		}
	}

	for _, dv := range pdvToBeRemoved {
		if err := client.Delete(ctx, &dv); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			logger.Error("failed to remove Allocated DataVolume: %s", err)
			continue
		}
	}

	return nil
}

func newDataVolume(dv kubermaticv1.PreAllocatedDataVolume, namespace string) (*cdiv1beta1.DataVolume, error) {
	dvSize, err := resource.ParseQuantity(dv.Size)
	if err != nil {
		return nil, err
	}
	return &cdiv1beta1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        dv.Name,
			Namespace:   namespace,
			Annotations: dv.Annotations,
		},
		Spec: cdiv1beta1.DataVolumeSpec{
			Source: &cdiv1beta1.DataVolumeSource{
				HTTP: &cdiv1beta1.DataVolumeSourceHTTP{
					URL: dv.URL,
				},
			},
			PVC: &corev1.PersistentVolumeClaimSpec{
				StorageClassName: utilpointer.String(dv.StorageClass),
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
