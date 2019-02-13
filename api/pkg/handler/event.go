package handler

import (
	"github.com/kubermatic/kubermatic/api/pkg/api/v1"

	corev1 "k8s.io/api/core/v1"
)

// FilterEventsByType filters Kubernetes Events based on their type. Empty type string will return all of them.
func FilterEventsByType(events []v1.Event, eventType string) []v1.Event {
	if len(eventType) == 0 || len(events) == 0 {
		return events
	}

	resultEvents := make([]v1.Event, 0)
	for _, event := range events {
		if event.Type == eventType {
			resultEvents = append(resultEvents, event)
		}
	}
	return resultEvents
}

// toEvent converts Kubernetes Events to Kubermatic ones (used in the API).
func toEvent(event corev1.Event) v1.Event {
	result := v1.Event{
		ObjectMeta: v1.ObjectMeta{
			ID:                string(event.ObjectMeta.UID),
			Name:              event.ObjectMeta.Name,
			CreationTimestamp: v1.NewTime(event.ObjectMeta.CreationTimestamp.Time),
		},
		Message:            event.Message,
		Type:               event.Type,
		InvolvedObjectName: event.InvolvedObject.Name,
		LastTimestamp:      v1.NewTime(event.LastTimestamp.Time),
		Count:              event.Count,
	}
	return result
}
