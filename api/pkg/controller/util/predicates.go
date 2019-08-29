package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PredicateFactory returns a predicate func that applies the given filter function
// on CREATE, UPDATE and DELETE events. For UPDATE events, the the filter is applied
// to both the old and new object and OR's the result.
func PredicateFactory(filter func(m metav1.Object, r runtime.Object) bool) predicate.Funcs {
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

// NamespacePredicate returns a predicate func that only includes objects in the given namespace
func NamespacePredicate(namespace string) predicate.Funcs {
	return PredicateFactory(func(m metav1.Object, r runtime.Object) bool {
		return m.GetNamespace() == namespace
	})
}
