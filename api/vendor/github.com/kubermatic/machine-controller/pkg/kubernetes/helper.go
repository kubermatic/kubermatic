/*
Copyright 2019 The Machine Controller Authors.

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

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// HasFinalizer tells if a object has the given finalizer
func HasFinalizer(o metav1.Object, name string) bool {
	return sets.NewString(o.GetFinalizers()...).Has(name)
}

// RemoveFinalizer removes the given finalizer and returns the cleaned list
func RemoveFinalizer(finalizers []string, toRemove string) []string {
	set := sets.NewString(finalizers...)
	set.Delete(toRemove)
	return set.List()
}
