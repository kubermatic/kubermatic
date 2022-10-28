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

package reconciling

import (
	"context"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/diff"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureObjectByAnnotation(t *testing.T) {
	const (
		testNamespace    = "default"
		testResourceName = "test"
	)

	tests := []struct {
		name           string
		creator        ObjectCreator
		existingObject ctrlruntimeclient.Object
		expectedObject ctrlruntimeclient.Object
		recreate       bool
	}{
		{
			name: "Object gets created",
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "1",
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
			creator: func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
		},
		{
			name: "Object gets updated",
			existingObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"foo": "hopefully-gets-overwritten",
				},
			},
			creator: func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "1",
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
		},
		{
			name: "Object update stopped by annotation",
			existingObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						ResourceReconciliationPausedAnnotation: "true",
					},
				},
				Data: map[string]string{
					"foo": "hopefully-does-not-get-overwritten",
				},
			},
			creator: func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						ResourceReconciliationPausedAnnotation: "true",
					},
				},
				Data: map[string]string{
					"foo": "hopefully-does-not-get-overwritten",
				},
			},
		},
		{
			name: "Object does not get updated",
			existingObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
			creator: func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
		},
		{
			name:     "Recreating objects works",
			recreate: true,
			existingObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "123",
					UID:             "abcd-1234",
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
			creator: func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar-new",
				}
				return sa, nil
			},
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"foo": "bar-new",
				},
			},
		},
		{
			name:     "Object recreation stopped by annotation",
			recreate: true,
			existingObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "123",
					UID:             "abcd-1234",
					Annotations: map[string]string{
						ResourceReconciliationPausedAnnotation: "true",
					},
				},
				Data: map[string]string{
					"foo": "hopefully-does-not-get-overwritten",
				},
			},
			creator: func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "123",
					UID:             "abcd-1234",
					Annotations: map[string]string{
						ResourceReconciliationPausedAnnotation: "true",
					},
				},
				Data: map[string]string{
					"foo": "hopefully-does-not-get-overwritten",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientBuilder := fakectrlruntimeclient.NewClientBuilder()
			if test.existingObject != nil {
				clientBuilder.WithObjects(test.existingObject)
			}

			client := clientBuilder.Build()
			ctx := context.Background()
			name := types.NamespacedName{Namespace: testNamespace, Name: testResourceName}
			if err := EnsureNamedObject(ctx, name, test.creator, client, &corev1.ConfigMap{}, test.recreate); err != nil {
				t.Errorf("EnsureObject returned an error while none was expected: %v", err)
			}

			key := ctrlruntimeclient.ObjectKeyFromObject(test.expectedObject)

			gotConfigMap := &corev1.ConfigMap{}
			if err := client.Get(ctx, key, gotConfigMap); err != nil {
				t.Fatalf("Failed to get the ConfigMap from the client: %v", err)
			}

			test.expectedObject.SetResourceVersion(gotConfigMap.ResourceVersion)
			test.expectedObject.SetUID(gotConfigMap.UID)
			test.expectedObject.SetGeneration(gotConfigMap.Generation)

			if !diff.SemanticallyEqual(test.expectedObject, gotConfigMap) {
				t.Fatalf("The ConfigMap from the client does not match the expected ConfigMap:\n%v", diff.ObjectDiff(test.expectedObject, gotConfigMap))
			}
		})
	}
}
