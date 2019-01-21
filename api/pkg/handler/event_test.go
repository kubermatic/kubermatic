package handler_test

import (
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler"

	corev1 "k8s.io/api/core/v1"
)

func TestFilterEventsByType(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name           string
		Filter         string
		ExpectedEvents []v1.Event
		InputEvents    v1.EventList
	}{
		{
			Name:   "scenario 1, filter out warning event types",
			Filter: corev1.EventTypeWarning,
			ExpectedEvents: []v1.Event{
				genEvent("test1", corev1.EventTypeWarning),
				genEvent("test2", corev1.EventTypeWarning),
			},
			InputEvents: v1.EventList{
				Name: "machine",
				Events: []v1.Event{
					genEvent("test1", corev1.EventTypeWarning),
					genEvent("test2", corev1.EventTypeWarning),
					genEvent("test3", corev1.EventTypeNormal),
					genEvent("test4", corev1.EventTypeNormal),
				},
			},
		},
		{
			Name:   "scenario 2, filter out normal event types",
			Filter: corev1.EventTypeNormal,
			ExpectedEvents: []v1.Event{
				genEvent("test3", corev1.EventTypeNormal),
				genEvent("test4", corev1.EventTypeNormal),
			},
			InputEvents: v1.EventList{
				Name: "machine",
				Events: []v1.Event{
					genEvent("test1", corev1.EventTypeWarning),
					genEvent("test2", corev1.EventTypeWarning),
					genEvent("test3", corev1.EventTypeNormal),
					genEvent("test4", corev1.EventTypeNormal),
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			result := handler.FilterEventsByType(tc.InputEvents, tc.Filter)
			if !equal(result.Events, tc.ExpectedEvents) {
				t.Fatalf("event list %v is not the same as expected %v", result, tc.ExpectedEvents)
			}

		})
	}

}

// equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func equal(a, b []v1.Event) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func genEvent(message, eventType string) v1.Event {
	return v1.Event{
		Type:    eventType,
		Message: message,
	}
}
