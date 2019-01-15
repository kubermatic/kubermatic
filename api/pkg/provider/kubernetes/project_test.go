package kubernetes_test

import (
	"testing"

	"github.com/go-test/deep"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestListProjects(t *testing.T) {
	// test data
	testcases := []struct {
		name             string
		existingProjects []*kubermaticv1.Project
		listOptions      *provider.ProjectListOptions
		expectedProjects []*kubermaticv1.Project
	}{
		{
			name:        "scenario 1: list bob's projects",
			listOptions: &provider.ProjectListOptions{OwnerUID: types.UID("bob")},
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
				// john's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id2",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("john")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n2",
					},
				},
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
			},
		},

		{
			name:        "scenario 2: list all projects",
			listOptions: nil,
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
				// john's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id2",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("john")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n2",
					},
				},
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
				// john's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id2",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("john")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n2",
					},
				},
			},
		},

		{
			name:        "scenario 3: list a project with a given name",
			listOptions: &provider.ProjectListOptions{ProjectName: "n1"},
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
				// john's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id2",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("john")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n2",
					},
				},
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
			},
		},

		{
			name:        "scenario 4: list a projects with a given name",
			listOptions: &provider.ProjectListOptions{ProjectName: "n1"},
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
				// john's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id2",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("john")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
				// john's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id2",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("john")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
			},
		},

		{
			name:        "scenario 4: list a bob's project with a given name",
			listOptions: &provider.ProjectListOptions{ProjectName: "n1", OwnerUID: types.UID("bob")},
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
				// john's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id2",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("john")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "id1",
						OwnerReferences: []metav1.OwnerReference{
							metav1.OwnerReference{UID: types.UID("bob")},
						},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "n1",
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			kubermaticObjects := []runtime.Object{}
			for _, binding := range tc.existingProjects {
				kubermaticObjects = append(kubermaticObjects, binding)
			}

			impersonationClient, _, indexer, err := createFakeClients(kubermaticObjects)
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}
			projectLister := kubermaticv1lister.NewProjectLister(indexer)

			// act
			target, err := kubernetes.NewProjectProvider(impersonationClient.CreateFakeImpersonatedClientSet, projectLister)
			if err != nil {
				t.Fatal(err)
			}
			result, err := target.List(tc.listOptions)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.expectedProjects) != len(result) {
				t.Fatalf("expected to get %d projects, but got %d", len(tc.expectedProjects), len(result))
			}
			for _, returnedProject := range result {
				bindingFound := false
				for _, expectedProject := range tc.expectedProjects {
					if diff := deep.Equal(returnedProject, expectedProject); diff == nil {
						bindingFound = true
						break
					}
				}
				if !bindingFound {
					t.Fatalf("returned project was not found on the list of expected ones, binding = %#v", returnedProject)
				}
			}
		})
	}
}
