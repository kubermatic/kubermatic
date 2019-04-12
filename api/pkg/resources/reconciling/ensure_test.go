package reconciling

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	controllerruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureObjectByAnnotation(t *testing.T) {
	const (
		testNamespace    = "default"
		testResourceName = "test"
	)

	tests := []struct {
		name           string
		creator        ObjectCreator
		existingObject runtime.Object
		expectedObject runtime.Object
	}{
		{
			name: "Object gets created",
			expectedObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
			creator: func(existing runtime.Object) (runtime.Object, error) {
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
			creator: func(existing runtime.Object) (runtime.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				// Required as we wait for the resource version to change in EnsureNamedObject & the fake client does not set it
				sa.ResourceVersion = "jf82762lh7"
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
			expectedObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
					// Required as we wait for the resource version to change in EnsureNamedObject & the fake client does not set it
					ResourceVersion: "jf82762lh7",
				},
				Data: map[string]string{
					"foo": "bar",
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
			creator: func(existing runtime.Object) (runtime.Object, error) {
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
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var client controllerruntimeclient.Client
			if test.existingObject != nil {
				client = controllerruntimefake.NewFakeClient(test.existingObject)
			} else {
				client = controllerruntimefake.NewFakeClient()
			}

			name := types.NamespacedName{Namespace: testNamespace, Name: testResourceName}
			if err := EnsureNamedObject(context.Background(), name, test.creator, client, &corev1.ConfigMap{}); err != nil {
				t.Errorf("EnsureObject returned an error while none was expected: %v", err)
			}

			key, err := controllerruntimeclient.ObjectKeyFromObject(test.expectedObject)
			if err != nil {
				t.Fatalf("Failed to generate a ObjectKey for the expected object: %v", err)
			}

			gotConfigMap := &corev1.ConfigMap{}
			if err := client.Get(context.Background(), key, gotConfigMap); err != nil {
				t.Fatalf("Failed to get the ServiceAccount from the client: %v", err)
			}

			if diff := deep.Equal(gotConfigMap, test.expectedObject); diff != nil {
				t.Errorf("The ConfigMap from the client does not match the expected ConfigMap. Diff: \n%v", diff)
			}
		})
	}
}
