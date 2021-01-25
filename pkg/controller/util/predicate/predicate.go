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

package predicate

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Factory returns a predicate func that applies the given filter function
// on CREATE, UPDATE and DELETE events. For UPDATE events, the filter is applied
// to both the old and new object and OR's the result.
func Factory(filter func(o client.Object) bool) predicate.Funcs {
	if filter == nil {
		return predicate.Funcs{}
	}

	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return filter(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return filter(e.ObjectOld) || filter(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return filter(e.Object)
		},
	}
}

// ByNamespace returns a predicate func that only includes objects in the given namespace
func ByNamespace(namespace string) predicate.Funcs {
	return Factory(func(o client.Object) bool {
		return o.GetNamespace() == namespace
	})
}

// ByName returns a predicate func that only includes objects in the given names
func ByName(names ...string) predicate.Funcs {
	namesSet := sets.NewString(names...)
	return Factory(func(o client.Object) bool {
		return namesSet.Has(o.GetName())
	})
}

// ByLabel returns a predicate func that only includes objects with the given label
func ByLabel(key, value string) predicate.Funcs {
	return Factory(func(o client.Object) bool {
		labels := o.GetLabels()
		if labels != nil {
			if existingValue, ok := labels[key]; ok {
				if existingValue == value {
					return true
				}
			}
		}
		return false
	})
}
