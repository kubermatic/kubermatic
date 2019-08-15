package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilpointer "k8s.io/utils/pointer"
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
	if err := userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, pvcList); err != nil {
		return false, fmt.Errorf("failed to list PVCs from user cluster: %v", err)
	}

	pvList := &corev1.PersistentVolumeList{}
	if err := userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, pvList); err != nil {
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

func terminatingAdmissionWebhook() (string, *admissionregistrationv1beta1.ValidatingWebhookConfiguration) {
	name := "kubernetes-cluster-cleanup"
	failurePolicy := admissionregistrationv1beta1.Fail
	return name, &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				annotationKeyDescription: "This webhook configuration exists to prevent creation of any new stateful resources in a cluster that is currently being terminated",
			},
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				// Must be a domain with at least three segments separated by dots
				Name: "kubernetes.cluster.cleanup",
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					URL: utilpointer.StringPtr("https://127.0.0.1:1"),
				},
				Rules: []admissionregistrationv1beta1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"*"},
							Resources:   []string{"persistentvolumes", "persistentvolumeclaims"},
						},
					},
					{
						Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{"policy"},
							APIVersions: []string{"*"},
							Resources:   []string{"poddisruptionbudgets"},
						},
					},
				},
				FailurePolicy: &failurePolicy,
			},
		},
	}
}

func (d *Deletion) disablePVCreation(ctx context.Context, userClusterClient controllerruntimeclient.Client) error {
	// Prevent re-creation of PVs, PVCs and PDBs by using an intentionally defunct admissionWebhook
	admissionWebhookName, admissionWebhook := terminatingAdmissionWebhook()
	err := userClusterClient.Get(ctx, types.NamespacedName{Name: admissionWebhookName}, &admissionregistrationv1beta1.ValidatingWebhookConfiguration{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("error checking if %q webhook configuration already exists: %v", admissionWebhookName, err)
	}
	if kerrors.IsNotFound(err) {
		if err := userClusterClient.Create(ctx, admissionWebhook); err != nil {
			return fmt.Errorf("failed to create %q webhook configuration: %v", admissionWebhookName, err)
		}
	}

	return nil
}

func (d *Deletion) cleanupPVCUsingPods(ctx context.Context, userClusterClient controllerruntimeclient.Client) error {
	podList := &corev1.PodList{}
	if err := userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, podList); err != nil {
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
