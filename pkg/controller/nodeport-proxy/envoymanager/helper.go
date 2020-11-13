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

package envoymanager

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func ServiceKey(service *corev1.Service) string {
	return fmt.Sprintf("%s/%s", service.Namespace, service.Name)
}

func getAnnotation(obj runtime.Object, annotation string) (string, bool) {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return "", false
	}
	return getAnnotationFromMeta(metaObj, annotation)
}

func getAnnotationFromMeta(obj metav1.Object, annotation string) (string, bool) {
	if obj.GetAnnotations() == nil {
		return "", false
	}
	val, ok := obj.GetAnnotations()[annotation]
	return val, ok
}
