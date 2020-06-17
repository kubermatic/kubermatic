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

package rbac

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac/test"
	fakeInformerProvider "github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac/test/fake"
	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	k8scorev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestEnsureProjectIsInActivePhase(t *testing.T) {
	tests := []struct {
		name            string
		projectToSync   *kubermaticv1.Project
		expectedProject *kubermaticv1.Project
	}{
		{
			name:          "scenario 1: a project's phase is set to Active",
			projectToSync: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedProject: func() *kubermaticv1.Project {
				project := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				project.Status.Phase = "Active"
				return project
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			objs = append(objs, test.expectedProject)
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)
			fakeMasterClusterProvider := &ClusterProvider{
				kubermaticClient: kubermaticFakeClient,
			}

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			err := target.ensureProjectIsInActivePhase(test.projectToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if test.expectedProject == nil {
				if len(kubermaticFakeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
				}
				return
			}
			if len(kubermaticFakeClient.Actions()) != 1 {
				t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
			}

			action := kubermaticFakeClient.Actions()[0]
			if !action.Matches("update", "projects") {
				t.Fatalf("unexpected action %#v", action)
			}
			updateAction, ok := action.(clienttesting.UpdateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			if !equality.Semantic.DeepEqual(updateAction.GetObject().(*kubermaticv1.Project), test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(test.expectedProject, updateAction.GetObject().(*kubermaticv1.Project)))
			}
		})
	}
}

func TestEnsureProjectInitialized(t *testing.T) {
	tests := []struct {
		name            string
		projectToSync   *kubermaticv1.Project
		expectedProject *kubermaticv1.Project
	}{
		{
			name:          "scenario 1: cleanup finializer is added to a project",
			projectToSync: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedProject: func() *kubermaticv1.Project {
				project := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				project.Finalizers = []string{"kubermatic.io/controller-manager-rbac-cleanup"}
				return project
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			objs = append(objs, test.expectedProject)
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)
			fakeMasterClusterProvider := &ClusterProvider{
				kubermaticClient: kubermaticFakeClient,
			}

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			err := target.ensureCleanupFinalizerExists(test.projectToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if test.expectedProject == nil {
				if len(kubermaticFakeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
				}
				return
			}
			if len(kubermaticFakeClient.Actions()) != 1 {
				t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
			}

			action := kubermaticFakeClient.Actions()[0]
			if !action.Matches("update", "projects") {
				t.Fatalf("unexpected action %#v", action)
			}
			updateAction, ok := action.(clienttesting.UpdateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			if !equality.Semantic.DeepEqual(updateAction.GetObject().(*kubermaticv1.Project), test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(test.expectedProject, updateAction.GetObject().(*kubermaticv1.Project)))
			}
		})
	}
}

func TestEnsureProjectClusterRBACRoleBindingForResources(t *testing.T) {
	tests := []struct {
		name                                 string
		projectResourcesToSync               []projectResource
		projectToSync                        string
		expectedClusterRoleBindingsForMaster []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindingsForMaster []*rbacv1.ClusterRoleBinding
		expectedActionsForMaster             []string
		seedClusters                         int
		expectedActionsForSeeds              []string
		expectedClusterRoleBindingsForSeeds  []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindingsForSeeds  []*rbacv1.ClusterRoleBinding
	}{
		// scenario 1
		{
			name:                     "Scenario 1: Proper set of RBAC Bindings for project's resources are created on master and seed clusters",
			projectToSync:            "thunderball",
			expectedActionsForMaster: []string{"create", "create"},
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.ClusterResourceName,
					},
					kind:        kubermaticv1.ClusterKindName,
					destination: destinationSeed,
				},

				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.SSHKeyResourceName,
					},
					kind: kubermaticv1.SSHKeyKind,
				},
			},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:editors",
					},
				},
			},
			seedClusters:            2,
			expectedActionsForSeeds: []string{"create", "create"},
			expectedClusterRoleBindingsForSeeds: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:editors",
					},
				},
			},
		},

		// scenario 2
		{
			name:                     "Scenario 2: Existing RBAC Bindings are properly updated when a new project is added",
			projectToSync:            "thunderball",
			expectedActionsForMaster: []string{"update", "update"},
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.ClusterResourceName,
					},
					kind:        kubermaticv1.ClusterKindName,
					destination: destinationSeed,
				},
				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.SSHKeyResourceName,
					},
					kind: kubermaticv1.SSHKeyKind,
				},
			},
			existingClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-existing-project-1",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:editors",
					},
				},
			},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
						{

							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-existing-project-1",
						},
						{

							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:editors",
					},
				},
			},
			seedClusters:            2,
			expectedActionsForSeeds: []string{"update", "update"},
			existingClusterRoleBindingsForSeeds: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:editors",
					},
				},
			},
			expectedClusterRoleBindingsForSeeds: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:editors",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForMaster {
				objs = append(objs, existingClusterRoleBinding)
				err := roleBindingsIndexer.Add(existingClusterRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
			}
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			// manually set lister as we don't want to start informers in the tests
			fakeKubeInformerProviderForMaster := NewInformerProvider(fakeKubeClient, time.Minute*5)
			fakeInformerFactoryForClusterRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, metav1.NamespaceAll)
			fakeInformerFactoryForClusterRole.AddFakeClusterRoleBindingInformer(roleBindingsIndexer)
			fakeKubeInformerProviderForMaster.kubeInformers[metav1.NamespaceAll] = fakeInformerFactoryForClusterRole

			fakeMasterClusterProvider := &ClusterProvider{
				kubeClient:           fakeKubeClient,
				kubeInformerProvider: fakeKubeInformerProviderForMaster,
			}

			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
				for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForSeeds {
					objs = append(objs, existingClusterRoleBinding)
					err := roleBindingsIndexer.Add(existingClusterRoleBinding)
					if err != nil {
						t.Fatal(err)
					}
				}
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerProviderForSeed := NewInformerProvider(fakeSeedKubeClient, time.Minute*5)

				// manually set lister as we don't want to start informers in the tests
				fakeInformerFactoryForClusterRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeSeedKubeClient, metav1.NamespaceAll)
				fakeInformerFactoryForClusterRole.AddFakeClusterRoleBindingInformer(roleBindingsIndexer)

				fakeKubeInformerProviderForSeed.kubeInformers[metav1.NamespaceAll] = fakeInformerFactoryForClusterRole
				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerProviderForSeed, nil, nil)
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.projectResources = test.projectResourcesToSync
			target.seedClusterProviders = seedClusterProviders
			err := target.ensureClusterRBACRoleBindingForResources(test.projectToSync)

			// validate master cluster
			{
				if err != nil {
					t.Fatal(err)
				}

				if len(test.expectedClusterRoleBindingsForMaster) == 0 {
					if len(fakeKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
					}
					return
				}

				if len(fakeKubeClient.Actions()) != len(test.expectedActionsForMaster) {
					t.Fatalf("unexpected number of actions, expected %d, got %d, actions %v", len(test.expectedActionsForMaster), len(fakeKubeClient.Actions()), fakeKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range fakeKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForMaster[index], "clusterrolebindings") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRoleBinding), test.expectedClusterRoleBindingsForMaster[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoleBindingsForMaster[createActionIndex], createAction.GetObject().(*rbacv1.ClusterRoleBinding)))
					}
					createActionIndex++
				}
			}

			// validate seed clusters
			for i := 0; i < test.seedClusters; i++ {

				seedKubeClient, ok := seedClusterProviders[i].kubeClient.(*fake.Clientset)
				if !ok {
					t.Fatal("expected thatt seedClusterRESTClient will hold *fake.Clientset")
				}
				if len(test.expectedClusterRoleBindingsForSeeds) == 0 {
					if len(seedKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", seedKubeClient.Actions())
					}
					return
				}

				if len(seedKubeClient.Actions()) != len(test.expectedActionsForSeeds) {
					t.Fatalf("unexpected number of actions, expected %d, got %d, actions %v", len(test.expectedActionsForSeeds), len(seedKubeClient.Actions()), seedKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range seedKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForSeeds[index], "clusterrolebindings") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRoleBinding), test.expectedClusterRoleBindingsForSeeds[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoleBindingsForSeeds[createActionIndex], createAction.GetObject().(*rbacv1.ClusterRoleBinding)))
					}
					createActionIndex++
				}
			}
		})
	}
}

// TestEnsureClusterResourcesCleanup test if cluster resources for the given
// project were removed from all physical locations
func TestEnsureClusterResourcesCleanup(t *testing.T) {
	tests := []struct {
		name               string
		projectToSync      *kubermaticv1.Project
		existingClustersOn map[string][]*kubermaticv1.Cluster
		deletedClustersOn  map[string][]string
	}{
		// scenario 1
		{
			name:          "scenario 1: when a project is removed all cluster resources from all clusters (physical location) are also removed",
			projectToSync: test.CreateProject("plan9", test.CreateUser("bob")),
			existingClustersOn: map[string][]*kubermaticv1.Cluster{

				// cluster resources that are on "a" physical location
				"a": {

					// cluster "abcd" that belongs to "thunderball" project
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "abcd",
							UID:  types.UID("abcdID"),
							Labels: map[string]string{
								kubermaticv1.ProjectIDLabelKey: "thunderball",
							},
						},
						Spec:    kubermaticv1.ClusterSpec{},
						Address: kubermaticv1.ClusterAddress{},
						Status: kubermaticv1.ClusterStatus{
							NamespaceName: "cluster-abcd",
						},
					},

					// cluster "ab" that belongs to "plan9" project
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "ab",
							UID:  types.UID("abID"),
							Labels: map[string]string{
								kubermaticv1.ProjectIDLabelKey: "plan9",
							},
						},
						Spec:    kubermaticv1.ClusterSpec{},
						Address: kubermaticv1.ClusterAddress{},
						Status: kubermaticv1.ClusterStatus{
							NamespaceName: "cluster-ab",
						},
					},
				},

				// cluster resources that are on "b" physical location
				"b": {

					// cluster "xyz" that belongs to "plan9" project
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "xyz",
							UID:  types.UID("xyzID"),
							Labels: map[string]string{
								kubermaticv1.ProjectIDLabelKey: "plan9",
							},
						},
						Spec:    kubermaticv1.ClusterSpec{},
						Address: kubermaticv1.ClusterAddress{},
						Status: kubermaticv1.ClusterStatus{
							NamespaceName: "cluster-xyz",
						},
					},

					// cluster "zzz" that belongs to "plan9" project
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "zzz",
							UID:  types.UID("zzzID"),
							Labels: map[string]string{
								kubermaticv1.ProjectIDLabelKey: "plan9",
							},
						},
						Spec:    kubermaticv1.ClusterSpec{},
						Address: kubermaticv1.ClusterAddress{},
						Status: kubermaticv1.ClusterStatus{
							NamespaceName: "cluster-zzz",
						},
					},
				},

				// cluster resources that are on "c" physical location
				"c": {

					// cluster "cat" that belongs to "acme" project
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cat",
							UID:  types.UID("catID"),
							Labels: map[string]string{
								kubermaticv1.ProjectIDLabelKey: "acme",
							},
						},
						Spec:    kubermaticv1.ClusterSpec{},
						Address: kubermaticv1.ClusterAddress{},
						Status: kubermaticv1.ClusterStatus{
							NamespaceName: "cluster-cat",
						},
					},

					// cluster "bat" that belongs to "acme" project
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "bat",
							UID:  types.UID("batID"),
							Labels: map[string]string{
								kubermaticv1.ProjectIDLabelKey: "acme",
							},
						},
						Spec:    kubermaticv1.ClusterSpec{},
						Address: kubermaticv1.ClusterAddress{},
						Status: kubermaticv1.ClusterStatus{
							NamespaceName: "cluster-bat",
						},
					},
				},
			},
			deletedClustersOn: map[string][]string{
				"a": {"ab"},
				"b": {"xyz", "zzz"},
				"c": {},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// prepare test data
			getClusterProviderByName := func(name string, providers []*ClusterProvider) (*ClusterProvider, error) {
				for _, provider := range providers {
					if provider.providerName == name {
						return provider, nil
					}
				}
				return nil, fmt.Errorf("provider %s not found", name)
			}
			seedClusterProviders := make([]*ClusterProvider, len(test.existingClustersOn))
			{
				index := 0
				for providerName, clusterResources := range test.existingClustersOn {
					kubermaticObjs := []runtime.Object{}
					kubeObjs := []runtime.Object{}
					clusterResourcesIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
					for _, clusterResource := range clusterResources {
						err := clusterResourcesIndexer.Add(clusterResource)
						if err != nil {
							t.Fatal(err)
						}
						kubermaticObjs = append(kubermaticObjs, clusterResource)
					}

					fakeKubeClient := fake.NewSimpleClientset(kubeObjs...)
					fakeKubeInformerProvider := NewInformerProvider(fakeKubeClient, time.Minute*5)
					fakeKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)
					fakeProvider := NewClusterProvider(providerName, fakeKubeClient, fakeKubeInformerProvider, fakeKubermaticClient, nil)
					fakeProvider.AddIndexerFor(clusterResourcesIndexer, schema.GroupVersionResource{Resource: kubermaticv1.ClusterResourceName})
					seedClusterProviders[index] = fakeProvider
					index++
				}
			}
			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			userLister := kubermaticv1lister.NewUserLister(userIndexer)
			fakeKubermaticMasterClient := kubermaticfakeclientset.NewSimpleClientset(test.projectToSync)
			fakeMasterClusterProvider := &ClusterProvider{
				kubermaticClient: fakeKubermaticMasterClient,
			}

			// act
			target := projectController{}
			target.seedClusterProviders = seedClusterProviders
			target.userLister = userLister
			target.masterClusterProvider = fakeMasterClusterProvider
			err := target.ensureProjectCleanup(test.projectToSync)
			if err != nil {
				t.Fatal(err)
			}

			// validate
			if len(test.deletedClustersOn) != len(test.existingClustersOn) {
				t.Fatalf("deletedClustersOn field is different than existingClusterOn in length, did you forget to update deletedClusterOn ?")
			}
			for providerName, deletedClusterResources := range test.deletedClustersOn {
				provider, err := getClusterProviderByName(providerName, seedClusterProviders)
				if err != nil {
					t.Fatalf("unable to validate deleted cluster resources because didn't find the provider %s", providerName)
				}
				fakeKubermaticClient, ok := provider.kubermaticClient.(*kubermaticfakeclientset.Clientset)
				if !ok {
					t.Fatalf("cannot cast kubermaticClient for provider %s", providerName)
				}

				if len(fakeKubermaticClient.Actions()) != len(deletedClusterResources) {
					t.Fatalf("unexpected number of clusters were deleted, expected only %d, but got %d, for provider %s", len(deletedClusterResources), len(fakeKubermaticClient.Actions()), providerName)
				}

				for _, action := range fakeKubermaticClient.Actions() {
					if !action.Matches("delete", "clusters") {
						t.Fatalf("unexpected action %#v", action)
					}
					deleteAction, ok := action.(clienttesting.DeleteAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}

					foundDeletedResourceOnTheList := false
					for _, deletedClusterResource := range deletedClusterResources {
						if deleteAction.GetName() == deletedClusterResource {
							foundDeletedResourceOnTheList = true
							break
						}

					}
					if !foundDeletedResourceOnTheList {
						t.Fatalf("wrong cluster has been deleted %s, the cluster is not on the list  %v", deleteAction.GetName(), deletedClusterResources)
					}
				}
			}
		})
	}
}

func TestEnsureProjectCleanup(t *testing.T) {
	tests := []struct {
		name                                 string
		projectResourcesToSync               []projectResource
		projectToSync                        *kubermaticv1.Project
		existingUser                         *kubermaticv1.User
		expectedClusterRoleBindingsForMaster []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindingsForMaster []*rbacv1.ClusterRoleBinding
		expectedActionsForMaster             []string
		seedClusters                         int
		expectedActionsForSeeds              []string
		expectedClusterRoleBindingsForSeeds  []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindingsForSeeds  []*rbacv1.ClusterRoleBinding
	}{
		// scenario 1
		{

			name:          "Scenario 1: When a project is removed corresponding Subject from the Cluster RBAC Binding are removed",
			projectToSync: test.CreateProject("plan9", test.CreateUser("James Bond")),
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.ClusterResourceName,
					},
					kind:        kubermaticv1.ClusterKindName,
					destination: destinationSeed,
				},
				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.SSHKeyResourceName,
					},
					kind: kubermaticv1.SSHKeyKind,
				},
			},
			expectedActionsForMaster: []string{"get", "update", "get", "update"},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:owners",
					},
					Subjects: []rbacv1.Subject{},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:editors",
					},
					Subjects: []rbacv1.Subject{},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:editors",
					},
				},
			},
			existingClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-plan9",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-plan9",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:editors",
					},
				},
			},
			seedClusters:            2,
			expectedActionsForSeeds: []string{"get", "update", "get", "update"},
			expectedClusterRoleBindingsForSeeds: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:owners",
					},
					Subjects: []rbacv1.Subject{},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:editors",
					},
					Subjects: []rbacv1.Subject{},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:editors",
					},
				},
			},
			existingClusterRoleBindingsForSeeds: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-plan9",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-plan9",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:editors",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			kubermaticObjs := []runtime.Object{}
			projectIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.projectToSync != nil {
				err := projectIndexer.Add(test.projectToSync)
				if err != nil {
					t.Fatal(err)
				}
				kubermaticObjs = append(kubermaticObjs, test.projectToSync)
			}
			projectLister := kubermaticv1lister.NewProjectLister(projectIndexer)

			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.existingUser != nil {
				err := userIndexer.Add(test.existingUser)
				if err != nil {
					t.Fatal(err)
				}
				kubermaticObjs = append(kubermaticObjs, test.projectToSync)
			}
			userLister := kubermaticv1lister.NewUserLister(userIndexer)

			for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForMaster {
				objs = append(objs, existingClusterRoleBinding)
			}

			fakeMasterKubeClient := fake.NewSimpleClientset(objs...)
			fakeMasterKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)
			fakeMasterClusterProvider := &ClusterProvider{
				kubeClient:       fakeMasterKubeClient,
				kubermaticClient: fakeMasterKubermaticClient,
			}
			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForSeeds {
					objs = append(objs, existingClusterRoleBinding)
				}
				clusterResourcesIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerProvider := NewInformerProvider(fakeSeedKubeClient, time.Minute*5)
				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerProvider, nil, nil)
				fakeProvider.AddIndexerFor(clusterResourcesIndexer, schema.GroupVersionResource{Resource: kubermaticv1.ClusterResourceName})
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.projectResources = test.projectResourcesToSync
			target.seedClusterProviders = seedClusterProviders
			target.projectLister = projectLister
			target.userLister = userLister
			err := target.ensureProjectCleanup(test.projectToSync)

			// validate master cluster
			{
				if err != nil {
					t.Fatal(err)
				}

				fakeKubeClient, ok := fakeMasterClusterProvider.kubeClient.(*fake.Clientset)
				if !ok {
					t.Fatal("unable to cast fakeMasterClusterProvider.kubeCLient to *fake.Clientset")
				}

				if len(test.expectedClusterRoleBindingsForMaster) == 0 {
					if len(fakeKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
					}
					return
				}

				if len(fakeKubeClient.Actions()) != len(test.expectedActionsForMaster) {
					t.Fatalf("unexpected actions %v", fakeKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range fakeKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForMaster[index], "clusterrolebindings") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRoleBinding), test.expectedClusterRoleBindingsForMaster[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoleBindingsForMaster[createActionIndex], createAction.GetObject().(*rbacv1.ClusterRoleBinding)))
					}
					createActionIndex++
				}
			}

			// validate seed clusters
			for i := 0; i < test.seedClusters; i++ {

				seedKubeClient, ok := seedClusterProviders[i].kubeClient.(*fake.Clientset)
				if !ok {
					t.Fatal("expected thatt seedClusterRESTClient will hold *fake.Clientset")
				}
				if len(test.expectedClusterRoleBindingsForSeeds) == 0 {
					if len(seedKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", seedKubeClient.Actions())
					}
					return
				}

				if len(seedKubeClient.Actions()) != len(test.expectedActionsForSeeds) {
					t.Fatalf("unexpected actions %#v", seedKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range seedKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForSeeds[index], "clusterrolebindings") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRoleBinding), test.expectedClusterRoleBindingsForSeeds[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoleBindingsForSeeds[createActionIndex], createAction.GetObject().(*rbacv1.ClusterRoleBinding)))
					}
					createActionIndex++
				}
			}
		})
	}
}

func TestEnsureProjectClusterRBACRoleForResources(t *testing.T) {
	tests := []struct {
		name                          string
		projectResourcesToSync        []projectResource
		expectedClusterRolesForMaster []*rbacv1.ClusterRole
		expectedActionsForMaster      []string
		seedClusters                  int
		expectedActionsForSeeds       []string
		expectedClusterRolesForSeeds  []*rbacv1.ClusterRole
	}{
		// scenario 1
		{
			name:                     "Scenario 1: Proper set of RBAC Roles for project's resources are created on \"master\" and seed clusters",
			expectedActionsForMaster: []string{"create", "create"},
			expectedActionsForSeeds:  []string{"create", "create"},
			seedClusters:             2,
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.ClusterResourceName,
					},
					kind:        kubermaticv1.ClusterKindName,
					destination: destinationSeed,
				},

				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.SSHKeyResourceName,
					},
					kind: kubermaticv1.SSHKeyKind,
				},
			},

			expectedClusterRolesForSeeds: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:owners",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"clusters"},
							Verbs:     []string{"create"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:editors",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"clusters"},
							Verbs:     []string{"create"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:viewers",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"clusters"},
							Verbs:     []string{},
						},
					},
				},
			},

			expectedClusterRolesForMaster: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:owners",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"usersshkeies"},
							Verbs:     []string{"create"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeies:editors",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"usersshkeies"},
							Verbs:     []string{"create"},
						},
					},
				},
			},
		},

		// scenario 2
		{
			name:                     "Scenario 2: Proper set of RBAC Roles for UserProjectBinding resource are created on \"master\" and seed clusters",
			expectedActionsForMaster: []string{"create"},
			// UserProjectBinding is a resource that is only on master cluster
			expectedActionsForSeeds: []string{},
			seedClusters:            2,
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    kubermaticv1.GroupName,
						Version:  kubermaticv1.GroupVersion,
						Resource: kubermaticv1.UserProjectBindingResourceName,
					},
					kind: kubermaticv1.UserProjectBindingKind,
				},
			},

			expectedClusterRolesForMaster: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:userprojectbindings:owners",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"userprojectbindings"},
							Verbs:     []string{"create"},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			roleIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})

			// manually set lister as we don't want to start informers in the tests
			fakeKubeInformerProviderForMaster := NewInformerProvider(fakeKubeClient, time.Minute*5)
			fakeInformerFactoryForClusterRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, metav1.NamespaceAll)
			fakeInformerFactoryForClusterRole.AddFakeClusterRoleInformer(roleIndexer)
			fakeKubeInformerProviderForMaster.kubeInformers[metav1.NamespaceAll] = fakeInformerFactoryForClusterRole

			fakeMasterClusterProvider := &ClusterProvider{
				kubeClient:           fakeKubeClient,
				kubeInformerProvider: fakeKubeInformerProviderForMaster,
			}

			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerProvider := NewInformerProvider(fakeSeedKubeClient, time.Minute*5)
				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerProvider, nil, nil)
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.projectResources = test.projectResourcesToSync
			target.seedClusterProviders = seedClusterProviders
			err := target.ensureClusterRBACRoleForResources()

			// validate master cluster
			{
				if err != nil {
					t.Fatal(err)
				}

				if len(test.expectedClusterRolesForMaster) == 0 {
					if len(fakeKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
					}
					return
				}

				if len(fakeKubeClient.Actions()) != len(test.expectedActionsForMaster) {
					t.Fatalf("unexpected number of actions, expected to get %d but got %d, actions %v", len(test.expectedActionsForMaster), len(fakeKubeClient.Actions()), fakeKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range fakeKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForMaster[index], "clusterroles") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRole), test.expectedClusterRolesForMaster[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRolesForMaster[createActionIndex], createAction.GetObject().(*rbacv1.ClusterRole)))
					}
					createActionIndex++
				}
			}

			// validate seed clusters
			for i := 0; i < test.seedClusters; i++ {

				seedKubeClient, ok := seedClusterProviders[i].kubeClient.(*fake.Clientset)
				if !ok {
					t.Fatal("expected thatt seedClusterRESTClient will hold *fake.Clientset")
				}
				if len(test.expectedClusterRolesForSeeds) == 0 {
					if len(seedKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", seedKubeClient.Actions())
					}
					return
				}

				if len(seedKubeClient.Actions()) != len(test.expectedActionsForSeeds) {
					t.Fatalf("unexpected number of actions, got %d, but expected to get %d, actions %v", len(seedKubeClient.Actions()), len(test.expectedActionsForSeeds), seedKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range seedKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForSeeds[index], "clusterroles") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRole), test.expectedClusterRolesForSeeds[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRolesForSeeds[createActionIndex], createAction.GetObject().(*rbacv1.ClusterRole)))
					}
					createActionIndex++
				}
			}
		})
	}
}

func TestEnsureProjectOwner(t *testing.T) {
	tests := []struct {
		name            string
		projectToSync   *kubermaticv1.Project
		existingUser    *kubermaticv1.User
		expectedBinding *kubermaticv1.UserProjectBinding
		existingBinding *kubermaticv1.UserProjectBinding
	}{
		{
			name:          "scenario 1: make sure, that the owner of the newly created project is set properly.",
			projectToSync: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingUser:  test.CreateUser("James Bond"),
			expectedBinding: func() *kubermaticv1.UserProjectBinding {
				binding := test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond")))
				binding.Finalizers = []string{"kubermatic.io/controller-manager-rbac-cleanup"}
				return binding
			}(),
		},
		{
			name:            "scenario 2: no op when the owner of the project was set.",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingUser:    test.CreateUser("James Bond"),
			existingBinding: test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond"))),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.existingUser != nil {
				err := userIndexer.Add(test.existingUser)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, test.existingUser)
			}
			bindingIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.existingBinding != nil {
				err := bindingIndexer.Add(test.existingBinding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, test.existingBinding)
			}
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)
			fakeMasterClusterProvider := &ClusterProvider{
				kubermaticClient: kubermaticFakeClient,
			}
			userLister := kubermaticv1lister.NewUserLister(userIndexer)
			bindingLister := kubermaticv1lister.NewUserProjectBindingLister(bindingIndexer)

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.userLister = userLister
			target.userProjectBindingLister = bindingLister
			err := target.ensureProjectOwner(test.projectToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if test.expectedBinding == nil {
				if len(kubermaticFakeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
				}
				return
			}
			if len(kubermaticFakeClient.Actions()) != 1 {
				t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
			}

			action := kubermaticFakeClient.Actions()[0]
			if !action.Matches("create", "userprojectbindings") {
				t.Fatalf("unexpected action %#v", action)
			}
			createAction, ok := action.(clienttesting.CreateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			createdBinding := createAction.GetObject().(*kubermaticv1.UserProjectBinding)
			// name was generated by the test framework just update it
			test.expectedBinding.Name = createdBinding.Name
			if !equality.Semantic.DeepEqual(createdBinding, test.expectedBinding) {
				t.Fatalf("%v", diff.ObjectDiff(test.expectedBinding, createdBinding))
			}
		})
	}
}

func TestEnsureProjectRBACRoleForResources(t *testing.T) {
	tests := []struct {
		name                     string
		projectResourcesToSync   []projectResource
		expectedRolesForMaster   []*rbacv1.Role
		expectedActionsForMaster []string
		seedClusters             int
		expectedActionsForSeeds  []string
		expectedRolesForSeeds    []*rbacv1.Role
		existingRoles            []*rbacv1.Role
	}{
		// scenario 1
		{
			name:                     "Scenario 1: Proper set of RBAC Roles for secrets in kubermatic namespace are created on master seed clusters",
			expectedActionsForMaster: []string{"create"},
			expectedActionsForSeeds:  []string{"create"},
			seedClusters:             1,
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:      "Secret",
					namespace: "kubermatic",
				},
				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:        "Secret",
					destination: destinationSeed,
					namespace:   "kubermatic",
				},
			},

			expectedRolesForSeeds: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{k8scorev1.SchemeGroupVersion.Group},
							Resources: []string{"secrets"},
							Verbs:     []string{"create"},
						},
					},
				},
			},

			expectedRolesForMaster: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{k8scorev1.SchemeGroupVersion.Group},
							Resources: []string{"secrets"},
							Verbs:     []string{"create"},
						},
					},
				},
			},
		},

		// scenario 2
		{
			name:                     "Scenario 2: No-op if proper set of RBAC Roles for secrets in kubermatic namespace already exist on master and seed clusters",
			expectedActionsForMaster: []string{},
			expectedActionsForSeeds:  []string{},
			seedClusters:             1,
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:      "Secret",
					namespace: "kubermatic",
				},
				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:        "Secret",
					destination: destinationSeed,
					namespace:   "kubermatic",
				},
			},

			expectedRolesForSeeds: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{k8scorev1.SchemeGroupVersion.Group},
							Resources: []string{"secrets"},
							Verbs:     []string{"create"},
						},
					},
				},
			},

			existingRoles: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{k8scorev1.SchemeGroupVersion.Group},
							Resources: []string{"secrets"},
							Verbs:     []string{"create"},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			roleIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, res := range test.existingRoles {
				if err := roleIndexer.Add(res); err != nil {
					t.Fatal(err)
				}
			}

			// manually set lister as we don't want to start informers in the tests
			fakeKubeInformerProviderForMaster := NewInformerProvider(fakeKubeClient, time.Minute*5)
			for _, res := range test.projectResourcesToSync {
				fakeInformerFactoryForRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, res.namespace)
				fakeInformerFactoryForRole.AddFakeRoleInformer(roleIndexer)
				fakeKubeInformerProviderForMaster.kubeInformers[res.namespace] = fakeInformerFactoryForRole
			}
			fakeKubeInformerProviderForMaster.started = true

			fakeMasterClusterProvider := &ClusterProvider{
				kubeClient:           fakeKubeClient,
				kubeInformerProvider: fakeKubeInformerProviderForMaster,
			}

			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerProvider := NewInformerProvider(fakeSeedKubeClient, time.Minute*5)
				for _, res := range test.projectResourcesToSync {
					fakeInformerFactoryForRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, res.namespace)
					fakeInformerFactoryForRole.AddFakeRoleInformer(roleIndexer)
					fakeKubeInformerProvider.kubeInformers[res.namespace] = fakeInformerFactoryForRole
				}
				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerProvider, nil, nil)
				fakeKubeInformerProvider.started = true
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.projectResources = test.projectResourcesToSync
			target.seedClusterProviders = seedClusterProviders
			err := target.ensureRBACRoleForResources()

			// validate master cluster
			{
				if err != nil {
					t.Fatal(err)
				}

				if len(test.expectedRolesForMaster) == 0 {
					if len(fakeKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
					}
					return
				}

				if len(fakeKubeClient.Actions()) != len(test.expectedActionsForMaster) {
					t.Fatalf("unexpected number of actions, expected to get %d but got %d, actions %v", len(test.expectedActionsForMaster), len(fakeKubeClient.Actions()), fakeKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range fakeKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForMaster[index], "roles") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.Role), test.expectedRolesForMaster[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedRolesForMaster[createActionIndex], createAction.GetObject().(*rbacv1.Role)))
					}
					createActionIndex++
				}
			}

			// validate seed clusters
			for i := 0; i < test.seedClusters; i++ {

				seedKubeClient, ok := seedClusterProviders[i].kubeClient.(*fake.Clientset)
				if !ok {
					t.Fatal("expected thatt seedClusterRESTClient will hold *fake.Clientset")
				}
				if len(test.expectedRolesForSeeds) == 0 {
					if len(seedKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", seedKubeClient.Actions())
					}
					return
				}

				if len(seedKubeClient.Actions()) != len(test.expectedActionsForSeeds) {
					t.Fatalf("unexpected number of actions, got %d, but expected to get %d, actions %v", len(seedKubeClient.Actions()), len(test.expectedActionsForSeeds), seedKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range seedKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForSeeds[index], "roles") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.Role), test.expectedRolesForSeeds[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedRolesForSeeds[createActionIndex], createAction.GetObject().(*rbacv1.Role)))
					}
					createActionIndex++
				}
			}
		})
	}
}

func TestEnsureProjectRBACRoleBindingForResources(t *testing.T) {
	tests := []struct {
		name                          string
		projectResourcesToSync        []projectResource
		projectToSync                 string
		expectedRoleBindingsForMaster []*rbacv1.RoleBinding
		existingRoleBindingsForMaster []*rbacv1.RoleBinding
		expectedActionsForMaster      []string
		seedClusters                  int
		expectedActionsForSeeds       []string
		expectedRoleBindingsForSeeds  []*rbacv1.RoleBinding
		existingRoleBindingsForSeeds  []*rbacv1.RoleBinding
	}{
		// scenario 1
		{
			name:                     "Scenario 1: Proper set of RBAC Bindings for secrets in sa-secret namespace are created on master and seed clusters",
			projectToSync:            "thunderball",
			expectedActionsForMaster: []string{"create"},
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:        "Secret",
					destination: destinationSeed,
					namespace:   "kubermatic",
				},

				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:      "Secret",
					namespace: "kubermatic",
				},
			},
			expectedRoleBindingsForMaster: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
			seedClusters:            1,
			expectedActionsForSeeds: []string{"create"},
			expectedRoleBindingsForSeeds: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
		},

		// scenario 2
		{
			name:                     "Scenario 2: Existing RBAC Bindings are properly updated when a new project is added",
			projectToSync:            "thunderball",
			expectedActionsForMaster: []string{"update"},
			projectResourcesToSync: []projectResource{
				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:        "Secret",
					destination: destinationSeed,
					namespace:   "kubermatic",
				},
				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:      "Secret",
					namespace: "kubermatic",
				},
			},
			existingRoleBindingsForMaster: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
			expectedRoleBindingsForMaster: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
						{

							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
			seedClusters:            1,
			expectedActionsForSeeds: []string{"update"},
			existingRoleBindingsForSeeds: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
			expectedRoleBindingsForSeeds: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-existing-project-1",
						},
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingRoleBinding := range test.existingRoleBindingsForMaster {
				objs = append(objs, existingRoleBinding)
				err := roleBindingsIndexer.Add(existingRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
			}
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			// manually set lister as we don't want to start informers in the tests
			fakeKubeInformerProviderForMaster := NewInformerProvider(fakeKubeClient, time.Minute*5)
			for _, res := range test.projectResourcesToSync {
				fakeInformerFactoryForClusterRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, res.namespace)
				fakeInformerFactoryForClusterRole.AddFakeRoleBindingInformer(roleBindingsIndexer)
				fakeKubeInformerProviderForMaster.kubeInformers[res.namespace] = fakeInformerFactoryForClusterRole
			}
			fakeKubeInformerProviderForMaster.started = true

			fakeMasterClusterProvider := &ClusterProvider{
				kubeClient:           fakeKubeClient,
				kubeInformerProvider: fakeKubeInformerProviderForMaster,
			}

			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
				for _, existingRoleBinding := range test.existingRoleBindingsForSeeds {
					objs = append(objs, existingRoleBinding)
					err := roleBindingsIndexer.Add(existingRoleBinding)
					if err != nil {
						t.Fatal(err)
					}
				}
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerProviderForSeed := NewInformerProvider(fakeSeedKubeClient, time.Minute*5)

				// manually set lister as we don't want to start informers in the tests
				for _, res := range test.projectResourcesToSync {
					fakeInformerFactoryForRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, res.namespace)
					fakeInformerFactoryForRole.AddFakeRoleBindingInformer(roleBindingsIndexer)
					fakeKubeInformerProviderForSeed.kubeInformers[res.namespace] = fakeInformerFactoryForRole
				}

				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerProviderForSeed, nil, nil)
				fakeKubeInformerProviderForSeed.started = true
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.projectResources = test.projectResourcesToSync
			target.seedClusterProviders = seedClusterProviders
			err := target.ensureRBACRoleBindingForResources(test.projectToSync)

			// validate master cluster
			{
				if err != nil {
					t.Fatal(err)
				}

				if len(test.expectedRoleBindingsForMaster) == 0 {
					if len(fakeKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
					}
					return
				}

				if len(fakeKubeClient.Actions()) != len(test.expectedActionsForMaster) {
					t.Fatalf("unexpected number of actions, expected %d, got %d, actions %v", len(test.expectedActionsForMaster), len(fakeKubeClient.Actions()), fakeKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range fakeKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForMaster[index], "rolebindings") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.RoleBinding), test.expectedRoleBindingsForMaster[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedRoleBindingsForMaster[createActionIndex], createAction.GetObject().(*rbacv1.RoleBinding)))
					}
					createActionIndex++
				}
			}

			// validate seed clusters
			for i := 0; i < test.seedClusters; i++ {

				seedKubeClient, ok := seedClusterProviders[i].kubeClient.(*fake.Clientset)
				if !ok {
					t.Fatal("expected thatt seedClusterRESTClient will hold *fake.Clientset")
				}
				if len(test.expectedRoleBindingsForSeeds) == 0 {
					if len(seedKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", seedKubeClient.Actions())
					}
					return
				}

				if len(seedKubeClient.Actions()) != len(test.expectedActionsForSeeds) {
					t.Fatalf("unexpected number of actions, expected %d, got %d, actions %v", len(test.expectedActionsForSeeds), len(seedKubeClient.Actions()), seedKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range seedKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForSeeds[index], "rolebindings") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.RoleBinding), test.expectedRoleBindingsForSeeds[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedRoleBindingsForSeeds[createActionIndex], createAction.GetObject().(*rbacv1.RoleBinding)))
					}
					createActionIndex++
				}
			}
		})
	}
}

func TestEnsureProjectCleanUpForRoleBindings(t *testing.T) {
	tests := []struct {
		name                          string
		projectResourcesToSync        []projectResource
		projectToSync                 *kubermaticv1.Project
		expectedRoleBindingsForMaster []*rbacv1.RoleBinding
		existingRoleBindingsForMaster []*rbacv1.RoleBinding
		expectedActionsForMaster      []string
		seedClusters                  int
		expectedActionsForSeeds       []string
		expectedRoleBindingsForSeeds  []*rbacv1.RoleBinding
		existingRoleBindingsForSeeds  []*rbacv1.RoleBinding
	}{
		// scenario 1
		{

			name:          "Scenario 1: When a project is removed corresponding Subject from the RBAC Binding are removed",
			projectToSync: test.CreateProject("plan9", test.CreateUser("James Bond")),
			projectResourcesToSync: []projectResource{

				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:        "Secret",
					destination: destinationSeed,
					namespace:   "kubermatic",
				},
				{
					gvr: schema.GroupVersionResource{
						Group:    k8scorev1.GroupName,
						Version:  k8scorev1.SchemeGroupVersion.Version,
						Resource: "secrets",
					},
					kind:      "Secret",
					namespace: "kubermatic",
				},
			},
			expectedActionsForMaster: []string{"get", "update"},
			expectedRoleBindingsForMaster: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
			existingRoleBindingsForMaster: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-plan9",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
			seedClusters:            1,
			expectedActionsForSeeds: []string{"get", "update"},
			expectedRoleBindingsForSeeds: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
			existingRoleBindingsForSeeds: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-plan9",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			kubermaticObjs := []runtime.Object{}
			projectIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			err := projectIndexer.Add(test.projectToSync)
			if err != nil {
				t.Fatal(err)
			}
			kubermaticObjs = append(kubermaticObjs, test.projectToSync)
			projectLister := kubermaticv1lister.NewProjectLister(projectIndexer)
			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			userLister := kubermaticv1lister.NewUserLister(userIndexer)

			roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingRoleBinding := range test.existingRoleBindingsForMaster {
				objs = append(objs, existingRoleBinding)
				err := roleBindingsIndexer.Add(existingRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
			}

			fakeMasterKubeClient := fake.NewSimpleClientset(objs...)
			fakeMasterKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)
			// manually set lister as we don't want to start informers in the tests
			fakeKubeInformerProviderForMaster := NewInformerProvider(fakeMasterKubeClient, time.Minute*5)
			for _, res := range test.projectResourcesToSync {
				fakeInformerFactoryForRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeMasterKubeClient, res.namespace)
				fakeInformerFactoryForRole.AddFakeRoleBindingInformer(roleBindingsIndexer)
				fakeKubeInformerProviderForMaster.kubeInformers[res.namespace] = fakeInformerFactoryForRole
			}
			fakeKubeInformerProviderForMaster.started = true

			fakeMasterClusterProvider := &ClusterProvider{
				kubeClient:           fakeMasterKubeClient,
				kubermaticClient:     fakeMasterKubermaticClient,
				kubeInformerProvider: fakeKubeInformerProviderForMaster,
			}
			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
				for _, existingRoleBinding := range test.existingRoleBindingsForSeeds {
					objs = append(objs, existingRoleBinding)
					err := roleBindingsIndexer.Add(existingRoleBinding)
					if err != nil {
						t.Fatal(err)
					}
				}

				clusterResourcesIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerProviderForSeed := NewInformerProvider(fakeSeedKubeClient, time.Minute*5)

				// manually set lister as we don't want to start informers in the tests
				for _, res := range test.projectResourcesToSync {
					fakeInformerFactoryForRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeSeedKubeClient, res.namespace)
					fakeInformerFactoryForRole.AddFakeRoleBindingInformer(roleBindingsIndexer)
					fakeKubeInformerProviderForSeed.kubeInformers[res.namespace] = fakeInformerFactoryForRole
				}

				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerProviderForSeed, nil, nil)
				fakeProvider.AddIndexerFor(clusterResourcesIndexer, schema.GroupVersionResource{Resource: kubermaticv1.ClusterResourceName})
				fakeKubeInformerProviderForSeed.started = true
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := projectController{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.projectResources = test.projectResourcesToSync
			target.seedClusterProviders = seedClusterProviders
			target.projectLister = projectLister
			target.userLister = userLister
			err = target.ensureProjectCleanup(test.projectToSync)

			// validate master cluster
			{
				if err != nil {
					t.Fatal(err)
				}

				fakeKubeClient, ok := fakeMasterClusterProvider.kubeClient.(*fake.Clientset)
				if !ok {
					t.Fatal("unable to cast fakeMasterClusterProvider.kubeCLient to *fake.Clientset")
				}

				if len(test.expectedRoleBindingsForMaster) == 0 {
					if len(fakeKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
					}
					return
				}

				if len(fakeKubeClient.Actions()) != len(test.expectedActionsForMaster) {
					t.Fatalf("unexpected actions %v", fakeKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range fakeKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForMaster[index], "rolebindings") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.RoleBinding), test.expectedRoleBindingsForMaster[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedRoleBindingsForMaster[createActionIndex], createAction.GetObject().(*rbacv1.RoleBinding)))
					}
					createActionIndex++
				}
			}

			// validate seed clusters
			for i := 0; i < test.seedClusters; i++ {

				seedKubeClient, ok := seedClusterProviders[i].kubeClient.(*fake.Clientset)
				if !ok {
					t.Fatal("expected thatt seedClusterRESTClient will hold *fake.Clientset")
				}
				if len(test.expectedRoleBindingsForSeeds) == 0 {
					if len(seedKubeClient.Actions()) != 0 {
						t.Fatalf("unexpected actions %#v", seedKubeClient.Actions())
					}
					return
				}

				if len(seedKubeClient.Actions()) != len(test.expectedActionsForSeeds) {
					t.Fatalf("unexpected actions %#v", seedKubeClient.Actions())
				}

				createActionIndex := 0
				for index, action := range seedKubeClient.Actions() {
					if !action.Matches(test.expectedActionsForSeeds[index], "rolebindings") {
						t.Fatalf("unexpected action %#v", action)
					}
					if action.GetVerb() == "get" {
						continue
					}
					// TODO: figure out why action.(clienttesting.GenericAction) does not work
					createAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.RoleBinding), test.expectedRoleBindingsForSeeds[createActionIndex]) {
						t.Fatalf("%v", diff.ObjectDiff(test.expectedRoleBindingsForSeeds[createActionIndex], createAction.GetObject().(*rbacv1.RoleBinding)))
					}
					createActionIndex++
				}
			}
		})
	}
}
