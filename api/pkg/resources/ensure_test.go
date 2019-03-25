package resources

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	controllerruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type fakeSingleItemInformer struct {
	item   interface{}
	exists bool
	err    error
}

func (i *fakeSingleItemInformer) GetByKey(key string) (item interface{}, exists bool, err error) {
	return i.item, i.exists, i.err
}

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

			store := &fakeSingleItemInformer{
				item:   test.existingObject,
				exists: test.existingObject != nil,
				err:    nil,
			}

			if err := EnsureNamedObject(testResourceName, testNamespace, test.creator, store, client); err != nil {
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
