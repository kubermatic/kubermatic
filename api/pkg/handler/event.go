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
func GetMachineEvents(client kubernetes.Interface, machine clusterv1alpha1.Machine) (v1.EventList, error) {
	events, err := client.CoreV1().Events(metav1.NamespaceSystem).Search(runtime.NewScheme(), &machine)
	if err != nil {
		return v1.EventList{}, err
	}

	return createMachineEventList(events.Items, machine), nil
}

// FilterEventsByType filters kubernetes API event objects based on event type.
// Empty string will return all events.
func FilterEventsByType(eventList v1.EventList, eventType string) v1.EventList {
	if len(eventType) == 0 || len(eventList.Events) == 0 {
		return eventList
	}

	resultEvents := make([]v1.Event, 0)
	for _, event := range eventList.Events {
		if event.Type == eventType {
			resultEvents = append(resultEvents, event)
		}
	}
	return v1.EventList{
		Name:   eventList.Name,
		Events: resultEvents,
	}
}

// createMachineEventList converts array of api events to kubermatic EventList
func createMachineEventList(events []corev1.Event, machine clusterv1alpha1.Machine) v1.EventList {
	kubermaticEvents := make([]v1.Event, 0)

	for _, event := range events {
		kubermaticEvent := toEvent(event)
		kubermaticEvents = append(kubermaticEvents, kubermaticEvent)
	}

	return v1.EventList{
		Name:   machine.Name,
		Events: kubermaticEvents,
	}
}

// toEvent converts event api Event to Event model object.
func toEvent(event corev1.Event) v1.Event {
	result := v1.Event{
		Message: event.Message,
		Type:    event.Type,
	}
	return result
}
