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
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Factory returns a predicate func that applies the given filter function
// on CREATE, UPDATE and DELETE events. For UPDATE events, the filter is applied
// to both the old and new object and OR's the result.
func Factory(filter func(o ctrlruntimeclient.Object) bool) predicate.Funcs {
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

// MultiFactory returns a predicate func that applies the given filter functions
// to the respective events for CREATE, UPDATE and DELETE. For UPDATE events, the
// filter is applied to both the old and new object and OR's the result.
func MultiFactory(createFilter func(o ctrlruntimeclient.Object) bool, updateFilter func(o ctrlruntimeclient.Object) bool, deleteFilter func(o ctrlruntimeclient.Object) bool) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return createFilter(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return updateFilter(e.ObjectOld) || updateFilter(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return deleteFilter(e.Object)
		},
	}
}

// ByNamespace returns a predicate func that only includes objects in the given namespace.
func ByNamespace(namespace string) predicate.Funcs {
	return Factory(func(o ctrlruntimeclient.Object) bool {
		return o.GetNamespace() == namespace
	})
}

// ByName returns a predicate func that only includes objects in the given names.
func ByName(names ...string) predicate.Funcs {
	namesSet := sets.NewString(names...)
	return Factory(func(o ctrlruntimeclient.Object) bool {
		return namesSet.Has(o.GetName())
	})
}

// ByLabel returns a predicate func that only includes objects with the given label.
func ByLabel(key, value string) predicate.Funcs {
	return Factory(func(o ctrlruntimeclient.Object) bool {
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

// ByLabel returns a predicate func that only includes objects that have a specific label key (value is ignored).
func ByLabelExists(key string) predicate.Funcs {
	return Factory(func(o ctrlruntimeclient.Object) bool {
		labels := o.GetLabels()
		if labels != nil {
			if _, ok := labels[key]; ok {
				return true
			}
		}
		return false
	})
}

// ByAnnotation returns a predicate func that only includes objects with the given annotation.
func ByAnnotation(key, value string, checkValue bool) predicate.Funcs {
	return Factory(func(o ctrlruntimeclient.Object) bool {
		annotations := o.GetAnnotations()
		if annotations != nil {
			if existingValue, ok := annotations[key]; ok {
				if !checkValue {
					return true
				}
				if existingValue == value {
					return true
				}
			}
		}
		return false
	})
}

// TrueFilter is a helper filter implementation that always returns true, e.g. for use with MultiFactory.
func TrueFilter(_ ctrlruntimeclient.Object) bool {
	return true
}
