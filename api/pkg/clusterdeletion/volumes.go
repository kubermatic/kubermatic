package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	annotationKeyDescription = "description"
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

	pvList := &corev1.PersistentVolumeList{}
	if err := userClusterClient.List(ctx, pvList); err != nil {
		return deletedSomeResource, fmt.Errorf("failed to list PVs from user cluster: %v", err)
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

	// Delete PV's
	for _, pv := range pvList.Items {
		if err := userClusterClient.Delete(ctx, &pv); err != nil && !kerrors.IsNotFound(err) {
			return deletedSomeResource, fmt.Errorf("failed to delete PV '%s' from user cluster: %v", pv.Name, err)
		}
		deletedSomeResource = true
	}

	return deletedSomeResource, nil
}

func (d *Deletion) disablePVCreation(ctx context.Context, userClusterClient controllerruntimeclient.Client) error {
	// Prevent re-creation of PVs and PVCs by using an intentionally defunct admissionWebhook
	creatorGetters := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{
		creationPreventingWebhook("", []string{"persistentvolumes", "persistentvolumeclaims"}),
	}
	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, creatorGetters, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to create ValidatingWebhookConfiguration to prevent creation of PVs/PVCs: %v", err)
	}

	return nil
}

func (d *Deletion) cleanupPVCUsingPods(ctx context.Context, userClusterClient controllerruntimeclient.Client) error {
	podList := &corev1.PodList{}
	if err := userClusterClient.List(ctx, podList); err != nil {
		return fmt.Errorf("failed to list Pods from user cluster: %v", err)
	}

	pvUsingPods := []*corev1.Pod{}
	for _, pod := range podList.Items {
		if podUsesPV(&pod) {
			pvUsingPods = append(pvUsingPods, &pod)
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
