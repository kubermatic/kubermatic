package common

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
)

// FilterEventsByType filters Kubernetes Events based on their type. Empty type string will return all of them.
func FilterEventsByType(events []kubermaticapiv1.Event, eventType string) []kubermaticapiv1.Event {
	if len(eventType) == 0 || len(events) == 0 {
		return events
	}

	resultEvents := make([]kubermaticapiv1.Event, 0)
	for _, event := range events {
		if event.Type == eventType {
			resultEvents = append(resultEvents, event)
		}
	}
	return resultEvents
}

// GetEvents returns events related to an object in a given namespace.
func GetEvents(ctx context.Context, client ctrlruntimeclient.Client, obj metav1.Object, objNamespace string) ([]kubermaticapiv1.Event, error) {
	events := &corev1.EventList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     objNamespace,
		FieldSelector: fields.OneTermEqualSelector("involvedObject.uid", string(obj.GetUID())),
	}
	if err := client.List(ctx, events, listOpts); err != nil {
		return nil, err
	}

	kubermaticEvents := make([]kubermaticapiv1.Event, 0)
	for _, event := range events.Items {
		kubermaticEvent := ConvertInternalEventToExternal(event)
		kubermaticEvents = append(kubermaticEvents, kubermaticEvent)
	}

	return kubermaticEvents, nil
}
