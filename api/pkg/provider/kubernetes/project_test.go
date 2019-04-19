package kubernetes_test

import (
	"testing"

	"github.com/go-test/deep"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/api/equality"
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
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
				// john's project
				genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("john")),
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
			},
		},

		{
			name:        "scenario 2: list all projects",
			listOptions: nil,
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
				// john's project
				genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("john")),
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
				// john's project
				genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("john")),
			},
		},

		{
			name:        "scenario 3: list a project with a given name",
			listOptions: &provider.ProjectListOptions{ProjectName: "n1"},
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
				// john's project
				genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("john")),
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
			},
		},

		{
			name:        "scenario 4: list a projects with a given name",
			listOptions: &provider.ProjectListOptions{ProjectName: "n1"},
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
				// john's project
				func() *kubermaticv1.Project {
					project := genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("john"))
					project.Spec.Name = "n1"
					return project
				}(),
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
				// john's project
				func() *kubermaticv1.Project {
					project := genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("john"))
					project.Spec.Name = "n1"
					return project
				}(),
			},
		},

		{
			name:        "scenario 4: list a bob's project with a given name",
			listOptions: &provider.ProjectListOptions{ProjectName: "n1", OwnerUID: types.UID("bob")},
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
				// john's project
				func() *kubermaticv1.Project {
					project := genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("john"))
					project.Spec.Name = "n1"
					return project
				}(),
			},
			expectedProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectActive, defaultCreationTimestamp(), genOwnerReference("bob")),
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			kubermaticObjects := []runtime.Object{}
			for _, binding := range tc.existingProjects {
				kubermaticObjects = append(kubermaticObjects, binding)
			}

			impersonationClient, _, indexer, err := createFakeKubermaticClients(kubermaticObjects)
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

func TestGetUnsecuredProjects(t *testing.T) {
	// test data
	testcases := []struct {
		name             string
		projectName      string
		existingProjects []*kubermaticv1.Project
		getOptions       *provider.ProjectGetOptions
		expectedProject  *kubermaticv1.Project
		expectedError    string
	}{
		{
			name:          "scenario 1: get inactive project",
			projectName:   "n1-ID",
			getOptions:    &provider.ProjectGetOptions{IncludeUninitialized: true},
			expectedError: "",
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectInactive, defaultCreationTimestamp(), metav1.OwnerReference{UID: types.UID("bob")}),
				// john's project
				genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), metav1.OwnerReference{UID: types.UID("john")}),
			},
			expectedProject: genProject("n1", kubermaticv1.ProjectInactive, defaultCreationTimestamp(), metav1.OwnerReference{UID: types.UID("bob")}),
		},
		{
			name:          "scenario 2: get only active project",
			projectName:   "n1-ID",
			getOptions:    &provider.ProjectGetOptions{IncludeUninitialized: false},
			expectedError: "Project is not initialized yet",
			existingProjects: []*kubermaticv1.Project{
				// bob's project
				genProject("n1", kubermaticv1.ProjectInactive, defaultCreationTimestamp(), metav1.OwnerReference{UID: types.UID("bob")}),
				// john's project
				genProject("n2", kubermaticv1.ProjectActive, defaultCreationTimestamp(), metav1.OwnerReference{UID: types.UID("john")}),
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			kubermaticObjects := []runtime.Object{}
			for _, binding := range tc.existingProjects {
				kubermaticObjects = append(kubermaticObjects, binding)
			}

			impersonationClient, _, _, err := createFakeKubermaticClients(kubermaticObjects)
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}

			// act
			target, err := kubernetes.NewPrivilegedProjectProvider(impersonationClient.CreateFakeImpersonatedClientSet)
			if err != nil {
				t.Fatal(err)
			}
			result, err := target.GetUnsecured(tc.projectName, tc.getOptions)

			if len(tc.expectedError) == 0 {
				// validate
				if err != nil {
					t.Fatal(err)
				}

				if !equality.Semantic.DeepEqual(result, tc.expectedProject) {
					t.Fatalf("expected project: %v got: %v", tc.expectedProject, result)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error message")
				}
				if err.Error() != tc.expectedError {
					t.Fatalf("expected error message: %s got: %s", tc.expectedError, err.Error())
				}
			}

		})
	}
}

func genOwnerReference(name string) metav1.OwnerReference {
	return metav1.OwnerReference{
		Name: name,
		UID:  types.UID(name),
	}
}
