package predicate

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Factory returns a predicate func that applies the given filter function
// on CREATE, UPDATE and DELETE events. For UPDATE events, the filter is applied
// to both the old and new object and OR's the result.
func Factory(filter func(m metav1.Object, r runtime.Object) bool) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return filter(e.Meta, e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return filter(e.MetaOld, e.ObjectOld) || filter(e.MetaNew, e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return filter(e.Meta, e.Object)
		},
	}
}

// ByNamespace returns a predicate func that only includes objects in the given namespace
func ByNamespace(namespace string) predicate.Funcs {
	return Factory(func(m metav1.Object, r runtime.Object) bool {
		return m.GetNamespace() == namespace
	})
}

// ByName returns a predicate func that only includes objects in the given names
func ByName(names ...string) predicate.Funcs {
	namesSet := sets.NewString(names...)
	return Factory(func(m metav1.Object, r runtime.Object) bool {
		return namesSet.Has(m.GetName())
	})
}

// ByLabel returns a predicate func that only includes objects with the given label
func ByLabel(key, value string) predicate.Funcs {
	return Factory(func(m metav1.Object, r runtime.Object) bool {
		labels := m.GetLabels()
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
