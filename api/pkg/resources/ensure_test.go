package resources

import (
	"context"
	"testing"
	"time"

	"github.com/go-test/deep"

	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	controllerruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = "test-namespace"
	testRoleName  = "test-role"
)

func createTestRole(data RoleDataProvider, existing *rbacv1.Role) (*rbacv1.Role, error) {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testRoleName,
			Namespace:       testNamespace,
			OwnerReferences: []metav1.OwnerReference{data.GetClusterRef()},
		},
	}, nil
}

type testRoleDataProvider struct{}

func (p *testRoleDataProvider) GetClusterRef() metav1.OwnerReference {
	return metav1.OwnerReference{
		Name: "Foo",
	}
}

// BenchmarkEnsureRole benchmarks EnsureRole with an existing role, so the longest possible path gets benchmarked
func BenchmarkEnsureRole(b *testing.B) {
	b.StopTimer()

	existingRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRoleName,
			Namespace: testNamespace,
		},
	}

	client := kubefake.NewSimpleClientset(existingRole)
	informerFactory := informers.NewSharedInformerFactory(client, 5*time.Minute)

	roleLister := informerFactory.Rbac().V1().Roles().Lister()

	informerFactory.Start(wait.NeverStop)
	informerFactory.WaitForCacheSync(wait.NeverStop)

	data := &testRoleDataProvider{}

	b.StartTimer()
	for n := 0; n < b.N; n++ {
		if err := EnsureRole(data, createTestRole, roleLister.Roles(testNamespace), client.RbacV1().Roles(testNamespace)); err != nil {
			b.Fatalf("failed to ensure that the role exists: %v", err)
		}
	}
}

func TestEnsureObject(t *testing.T) {
	const testNamespace = "default"

	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name           string
		creator        ObjectCreator
		existingObject runtime.Object
		expectedObject runtime.Object
	}{
		{
			name: "Object gets created",
			creator: func(existing runtime.Object) (runtime.Object, error) {
				var sa *corev1.ServiceAccount
				if existing == nil {
					sa = &corev1.ServiceAccount{}
				} else {
					sa = existing.(*corev1.ServiceAccount)
				}
				sa.Name = "test"
				sa.Namespace = testNamespace
				sa.AutomountServiceAccountToken = boolPtr(true)
				return sa, nil
			},
			expectedObject: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: testNamespace,
				},
				AutomountServiceAccountToken: boolPtr(true),
			},
		},
		{
			name: "Object gets updated",
			existingObject: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: testNamespace,
				},
				AutomountServiceAccountToken: boolPtr(false),
			},
			creator: func(existing runtime.Object) (runtime.Object, error) {
				var sa *corev1.ServiceAccount
				if existing == nil {
					sa = &corev1.ServiceAccount{}
				} else {
					sa = existing.(*corev1.ServiceAccount)
				}
				sa.Name = "test"
				sa.Namespace = testNamespace
				sa.AutomountServiceAccountToken = boolPtr(true)
				return sa, nil
			},
			expectedObject: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: testNamespace,
				},
				AutomountServiceAccountToken: boolPtr(true),
			},
		},
		{
			name: "Object does not get updated",
			existingObject: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: testNamespace,
				},
				AutomountServiceAccountToken: boolPtr(true),
			},
			creator: func(existing runtime.Object) (runtime.Object, error) {
				var sa *corev1.ServiceAccount
				if existing == nil {
					sa = &corev1.ServiceAccount{}
				} else {
					sa = existing.(*corev1.ServiceAccount)
				}
				sa.Name = "test"
				sa.Namespace = testNamespace
				sa.AutomountServiceAccountToken = boolPtr(true)
				return sa, nil
			},
			expectedObject: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: testNamespace,
				},
				AutomountServiceAccountToken: boolPtr(true),
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

			if err := EnsureObject(testNamespace, test.creator, store, client); err != nil {
				t.Errorf("EnsureObject returned an error while none was expected: %v", err)
			}

			key, err := controllerruntimeclient.ObjectKeyFromObject(test.expectedObject)
			if err != nil {
				t.Fatalf("Failed to generate a ObjectKey for the expected object: %v", err)
			}

			gotSA := &corev1.ServiceAccount{}
			if err := client.Get(context.Background(), key, gotSA); err != nil {
				t.Fatalf("Failed to get the ServiceAccount from the client: %v", err)
			}

			if diff := deep.Equal(gotSA, test.expectedObject); diff != nil {
				t.Errorf("The ServiceAccount from the client does not match the expected ServiceAccount. Diff: \n%v", diff)
			}
		})
	}
}

type fakeSingleItemInformer struct {
	item   interface{}
	exists bool
	err    error
}

func (i *fakeSingleItemInformer) GetByKey(key string) (item interface{}, exists bool, err error) {
	return i.item, i.exists, i.err
}
