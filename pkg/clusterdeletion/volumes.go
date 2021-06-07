/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	annotationKeyDescription = "description"

	// AnnDynamicallyProvisioned is added to a PV that is dynamically provisioned by kubernetes
	// Because the annotation is defined only at k8s.io/kubernetes, copying the content instead of vendoring
	// https://github.com/kubernetes/kubernetes/blob/v1.21.0/pkg/controller/volume/persistentvolume/util/util.go#L65
	AnnDynamicallyProvisioned = "pv.kubernetes.io/provisioned-by"
)

func (d *Deletion) cleanupVolumes(ctx context.Context, cluster *kubermaticv1.Cluster) (deletedSomeResource bool, err error) {
	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return false, err
	}

	// We disable the PV & PVC creation so nothing creates new PV's while we delete them
	if err := d.disablePVCreation(ctx, userClusterClient); err != nil {
		return false, fmt.Errorf("failed to disable future PV & PVC creation: %v", err)
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := userClusterClient.List(ctx, pvcList); err != nil {
		return false, fmt.Errorf("failed to list PVCs from user cluster: %v", err)
	}

	allPVList := &corev1.PersistentVolumeList{}
	if err := userClusterClient.List(ctx, allPVList); err != nil {
		return deletedSomeResource, fmt.Errorf("failed to list PVs from user cluster: %v", err)
	}
	pvList := &corev1.PersistentVolumeList{}
	for _, pv := range allPVList.Items {
		// Check only dynamically provisioned PVs with delete reclaim policy to verify provisioner has done the cleanup
		// this filters out everything else because we leave those be
		if pv.Annotations[AnnDynamicallyProvisioned] != "" && pv.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimDelete {
			pvList.Items = append(pvList.Items, pv)
		}
	}

	// Do not attempt to delete any pods when there are no PVs and PVCs
	if len(pvcList.Items) == 0 && len(pvList.Items) == 0 {
		return deletedSomeResource, nil
	}

	// Delete all Pods that use PVs. We must keep the remaining pods, otherwise
	// we end up in a deadlock when CSI is used
	if err := d.cleanupPVCUsingPods(ctx, userClusterClient); err != nil {
		return false, fmt.Errorf("failed to clean up PV using pod from user cluster: %v", err)
	}

	// Delete PVC's
	for _, pvc := range pvcList.Items {
		if err := userClusterClient.Delete(ctx, &pvc); err != nil && !kerrors.IsNotFound(err) {
			return deletedSomeResource, fmt.Errorf("failed to delete PVC '%s/%s' from user cluster: %v", pvc.Namespace, pvc.Name, err)
		}
		deletedSomeResource = true
	}

	if len(pvList.Items) > 0 {
		// We don't delete PVs but we want to wait for provisioners to cleanup dynamically provisioned PVs
		// pretend we need to requeue to avoid removing finalizer prematurely
		deletedSomeResource = true
	}

	return deletedSomeResource, nil
}

func (d *Deletion) disablePVCreation(ctx context.Context, userClusterClient ctrlruntimeclient.Client) error {
	// Prevent re-creation of PVs and PVCs by using an intentionally defunct admissionWebhook
	creatorGetters := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{
		creationPreventingWebhook("", []string{"persistentvolumes", "persistentvolumeclaims"}),
	}
	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, creatorGetters, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to create ValidatingWebhookConfiguration to prevent creation of PVs/PVCs: %v", err)
	}

	return nil
}

func (d *Deletion) cleanupPVCUsingPods(ctx context.Context, userClusterClient ctrlruntimeclient.Client) error {
	podList := &corev1.PodList{}
	if err := userClusterClient.List(ctx, podList); err != nil {
		return fmt.Errorf("failed to list Pods from user cluster: %v", err)
	}

	pvUsingPods := []*corev1.Pod{}
	for idx := range podList.Items {
		pod := &podList.Items[idx]
		if podUsesPV(pod) {
			pvUsingPods = append(pvUsingPods, pod)
		}
	}

	for _, pod := range pvUsingPods {
		if err := userClusterClient.Delete(ctx, pod); err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}
	}

	return nil
}

func podUsesPV(p *corev1.Pod) bool {
	for _, volume := range p.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			return true
		}
	}
	return false
}
