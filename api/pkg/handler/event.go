package handler

import (
	"github.com/kubermatic/kubermatic/api/pkg/api/v1"

	corev1 "k8s.io/api/core/v1"
)

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

// toEvent converts event api Event to Event model object.
func toEvent(event corev1.Event) v1.Event {
	result := v1.Event{
		Message: event.Message,
		Type:    event.Type,
	}
	return result
}
