package handler

import (
	"github.com/kubermatic/kubermatic/api/pkg/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// GetMachineEvents returns kubernetes API event objects assigned to the machine.
func GetMachineEvents(client kubernetes.Interface, machine clusterv1alpha1.Machine) ([]v1.Event, error) {
	events, err := client.CoreV1().Events(metav1.NamespaceSystem).Search(runtime.NewScheme(), &machine)
	if err != nil {
		return nil, err
	}

	return createEventList(events.Items), nil
}

// FilterEventsByType filters kubernetes API event objects based on event type.
// Empty string will return all events.
func FilterEventsByType(events []v1.Event, eventType string) []v1.Event {
	if len(eventType) == 0 || len(events) == 0 {
		return events
	}

	result := make([]v1.Event, 0)
	for _, event := range events {
		if event.Type == eventType {
			result = append(result, event)
		}
	}

	return result
}

// createEventList converts array of api events to kubermatic array events
func createEventList(events []corev1.Event) []v1.Event {
	kubermaticEvents := make([]v1.Event, 0)

	for _, event := range events {
		kubermaticEvent := toEvent(event)
		kubermaticEvents = append(kubermaticEvents, kubermaticEvent)
	}
	return kubermaticEvents
}

// toEvent converts event api Event to Event model object.
func toEvent(event corev1.Event) v1.Event {
	result := v1.Event{
		Name:           event.ObjectMeta.Name,
		Namespace:      event.ObjectMeta.Namespace,
		FirstTimestamp: v1.NewTime(event.FirstTimestamp.Time),
		LastTimestamp:  v1.NewTime(event.LastTimestamp.Time),
		InvolvedObject: v1.ObjectReference{
			Name:      event.InvolvedObject.Name,
			Namespace: event.InvolvedObject.Namespace,
			Kind:      event.InvolvedObject.Kind,
		},
		Source:  event.Source.Component,
		Message: event.Message,
		Count:   event.Count,
		Reason:  event.Reason,
		Type:    event.Type,
	}

	return result
}
