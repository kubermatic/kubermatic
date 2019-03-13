package serviceaccount

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestCreateBinding(t *testing.T) {

	tests := []struct {
		name                      string
		saName                    string
		existingKubermaticObjects []runtime.Object
		expectedBindingName       string
		expectedBinding           *kubermaticv1.UserProjectBinding
	}{
		{
			name:   "scenario 1: this test creates binding for service account",
			saName: "serviceaccount-abcd",
			existingKubermaticObjects: []runtime.Object{
				test.GenDefaultProject(),
				test.GenServiceAccount("abcd", "test", "editors", "my-first-project-ID"),
			},
			expectedBindingName: "sa-my-first-project-ID-abcd",
			expectedBinding: &kubermaticv1.UserProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.UserKindName,
							UID:        "",
							Name:       "serviceaccount-abcd",
						},
					},
					Name:   "sa-my-first-project-ID-abcd",
					Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "my-first-project-ID"},
				},
				Spec: kubermaticv1.UserProjectBindingSpec{
					ProjectID: "my-first-project-ID",
					UserEmail: "serviceaccount-abcd@sa.kubermatic.io",
					Group:     "editors",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario

			scheme := scheme.Scheme
			if err := kubermaticv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			kubermaticFakeClient := fake.NewFakeClientWithScheme(scheme, test.existingKubermaticObjects...)

			// act
			target := reconcileServiceAccountProjectBinding{ctx: context.TODO(), Client: kubermaticFakeClient}

			_, err := target.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: test.saName}})

			// validate
			if err != nil {
				t.Fatal(err)
			}
			binding := &kubermaticv1.UserProjectBinding{}
			err = kubermaticFakeClient.Get(target.ctx, controllerclient.ObjectKey{Name: test.expectedBindingName}, binding)
			if err != nil {
				t.Fatal(err)
			}

			if !equality.Semantic.DeepEqual(binding, test.expectedBinding) {
				t.Fatalf("%v", diff.ObjectDiff(binding, test.expectedBinding))
			}

		})
	}
}
