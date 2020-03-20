package common

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/label"

	corev1 "k8s.io/api/core/v1"
)

func ConvertInternalSSHKeysToExternal(internalKeys []*kubermaticapiv1.UserSSHKey) []*apiv1.SSHKey {
	apiKeys := make([]*apiv1.SSHKey, len(internalKeys))
	for index, key := range internalKeys {
		apiKey := &apiv1.SSHKey{
			ObjectMeta: apiv1.ObjectMeta{
				ID:                key.Name,
				Name:              key.Spec.Name,
				CreationTimestamp: apiv1.NewTime(key.CreationTimestamp.Time),
			},
			Spec: apiv1.SSHKeySpec{
				Fingerprint: key.Spec.Fingerprint,
				PublicKey:   key.Spec.PublicKey,
			},
		}
		apiKeys[index] = apiKey
	}
	return apiKeys
}

// ConvertInternalEventToExternal converts Kubernetes Events to Kubermatic ones (used in the API).
func ConvertInternalEventToExternal(event corev1.Event) apiv1.Event {
	switch event.InvolvedObject.Kind {
	case "Machine":
		event.InvolvedObject.Kind = "Node"
	case "MachineSet":
		event.InvolvedObject.Kind = "Node Set"
	case "MachineDeployment":
		event.InvolvedObject.Kind = "Node Deployment"
	}

	return apiv1.Event{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                string(event.ObjectMeta.UID),
			Name:              event.ObjectMeta.Name,
			CreationTimestamp: apiv1.NewTime(event.ObjectMeta.CreationTimestamp.Time),
		},
		Message: event.Message,
		Type:    event.Type,
		InvolvedObject: apiv1.ObjectReferenceResource{
			Name:      event.InvolvedObject.Name,
			Namespace: event.InvolvedObject.Namespace,
			Type:      event.InvolvedObject.Kind,
		},
		LastTimestamp: apiv1.NewTime(event.LastTimestamp.Time),
		Count:         event.Count,
	}
}

func ConvertInternalProjectToExternal(kubermaticProject *kubermaticapiv1.Project, projectOwners []apiv1.User, clustersNumber int) *apiv1.Project {
	return &apiv1.Project{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                kubermaticProject.Name,
			Name:              kubermaticProject.Spec.Name,
			CreationTimestamp: apiv1.NewTime(kubermaticProject.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if kubermaticProject.DeletionTimestamp != nil {
					dt := apiv1.NewTime(kubermaticProject.DeletionTimestamp.Time)
					return &dt
				}
				return nil
			}(),
		},
		Labels:         label.FilterLabels(label.ProjectResourceType, kubermaticProject.Labels),
		Status:         kubermaticProject.Status.Phase,
		Owners:         projectOwners,
		ClustersNumber: clustersNumber,
	}
}
