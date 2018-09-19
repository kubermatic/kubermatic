package resources

import (
	"testing"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
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

	client := fake.NewSimpleClientset(existingRole)
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
