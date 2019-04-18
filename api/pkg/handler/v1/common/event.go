package common

import (
	"context"

	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
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

// GetEvents returns events related to an object in a given namespace.
func GetEvents(ctx context.Context, client ctrlruntimeclient.Client, obj v12.Object, objNamespace string) ([]v1.Event, error) {
	events := &v13.EventList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     objNamespace,
		FieldSelector: fields.OneTermEqualSelector("involvedObject.uid", string(obj.GetUID())),
	}
	if err := client.List(ctx, listOpts, events); err != nil {
		return nil, err
	}

	kubermaticEvents := make([]v1.Event, 0)
	for _, event := range events.Items {
		kubermaticEvent := ConvertInternalEventToExternal(event)
		kubermaticEvents = append(kubermaticEvents, kubermaticEvent)
	}

	return kubermaticEvents, nil
}
