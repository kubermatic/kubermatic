/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"context"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
		FieldSelector: fields.OneTermEqualSelector("involvedObject.name", obj.GetName()),
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
