package rbac

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
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
			projectToSync: createProject("thunderball", createUser("James Bond")),
			expectedProject: func() *kubermaticv1.Project {
				project := createProject("thunderball", createUser("James Bond"))
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

			// act
			target := Controller{}
			target.kubermaticMasterClient = kubermaticFakeClient
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
			projectToSync: createProject("thunderball", createUser("James Bond")),
			expectedProject: func() *kubermaticv1.Project {
				project := createProject("thunderball", createUser("James Bond"))
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

			// act
			target := Controller{}
			target.kubermaticMasterClient = kubermaticFakeClient
			err := target.ensureProjectInitialized(test.projectToSync)

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
			name:                     "Scenario 1: Proper set of RBAC Bindings for project's resources are created on \"master\" and seed clusters",
			projectToSync:            "thunderball",
			expectedActionsForMaster: []string{"create", "create", "create", "create"},
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
			},
			existingClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:editors",
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
						Name:     "kubermatic:clusters:editors",
					},
				},
			},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:clusters:editors",
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
						Name:     "kubermatic:clusters:editors",
					},
				},
			},
			seedClusters:            2,
			expectedActionsForSeeds: []string{"update", "update"},
			existingClusterRoleBindingsForSeeds: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
			roleBindingsLister := rbaclister.NewClusterRoleBindingLister(roleBindingsIndexer)

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
				fakeKubeInformerFactory := kuberinformers.NewSharedInformerFactory(fakeSeedKubeClient, time.Minute*5)
				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerFactory, nil, nil)

				// manually set lister as we don't want to start informers in the tests
				roleBindingsLister := rbaclister.NewClusterRoleBindingLister(roleBindingsIndexer)
				fakeProvider.rbacClusterRoleBindingLister = roleBindingsLister
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := Controller{}
			target.kubeMasterClient = fakeKubeClient
			target.projectResources = test.projectResourcesToSync
			target.rbacClusterRoleBindingMasterLister = roleBindingsLister
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
					createActionIndex = createActionIndex + 1
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
					createActionIndex = createActionIndex + 1
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
			projectToSync: createProject("plan9", createUser("bob")),
			existingClustersOn: map[string][]*kubermaticv1.Cluster{

				// cluster resources that are on "a" physical location
				"a": []*kubermaticv1.Cluster{

					// cluster "abcd" that belongs to "thunderball" project
					&kubermaticv1.Cluster{
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
					&kubermaticv1.Cluster{
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
				"b": []*kubermaticv1.Cluster{

					// cluster "xyz" that belongs to "plan9" project
					&kubermaticv1.Cluster{
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
					&kubermaticv1.Cluster{
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
				"c": []*kubermaticv1.Cluster{

					// cluster "cat" that belongs to "acme" project
					&kubermaticv1.Cluster{
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
					&kubermaticv1.Cluster{
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
				"a": []string{"ab"},
				"b": []string{"xyz", "zzz"},
				"c": []string{},
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
					fakeKubeInformerFactory := kuberinformers.NewSharedInformerFactory(fakeKubeClient, time.Minute*5)
					fakeKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)
					fakeProvider := NewClusterProvider(providerName, fakeKubeClient, fakeKubeInformerFactory, fakeKubermaticClient, nil)
					fakeProvider.AddIndexerFor(clusterResourcesIndexer, schema.GroupVersionResource{Resource: kubermaticv1.ClusterResourceName})
					seedClusterProviders[index] = fakeProvider
					index = index + 1
				}
			}
			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			userLister := kubermaticv1lister.NewUserLister(userIndexer)
			fakeKubermaticMasterClient := kubermaticfakeclientset.NewSimpleClientset(test.projectToSync)

			// act
			target := Controller{}
			target.seedClusterProviders = seedClusterProviders
			target.userLister = userLister
			target.kubermaticMasterClient = fakeKubermaticMasterClient
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

// TestEnsureUserProjectCleanup extends TestEnsureProjectCleanup in a way
// that also checks if the project being removed is removed from "Spec.Project" array for all users that belong the the project
func TestEnsureUsersProjectCleanup(t *testing.T) {
	tests := []struct {
		name            string
		projectToSync   *kubermaticv1.Project
		existingUsers   []*kubermaticv1.User
		expectedActions []string
		expectedUsers   []*kubermaticv1.User
	}{
		// scenario 1
		{
			name:          "Scenario 1: bob's projects entries are updated when the project he belongs to is removed",
			projectToSync: createProject("plan9", createUser("bob")),
			existingUsers: []*kubermaticv1.User{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticv1.UserSpec{
						Name:  "bob",
						Email: "bob@acme.com",
						Projects: []kubermaticv1.ProjectGroup{
							{
								Group: "editors-myFirstProjectName",
								Name:  "myFirstProjectName",
							},
							{
								Group: "owners-plan9",
								Name:  "plan9",
							},
							{
								Group: "editors-myThirdProjectInternalName",
								Name:  "myThirdProjectInternalName",
							},
						},
					},
				},
			},
			expectedActions: []string{"update", "update"},
			expectedUsers: []*kubermaticv1.User{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticv1.UserSpec{
						Name:  "bob",
						Email: "bob@acme.com",
						Projects: []kubermaticv1.ProjectGroup{
							{
								Group: "editors-myFirstProjectName",
								Name:  "myFirstProjectName",
							},
							{
								Group: "editors-myThirdProjectInternalName",
								Name:  "myThirdProjectInternalName",
							},
						},
					},
				},
			},
		},

		// scenario 2
		{
			name:          "Scenario 2: only bob's projects entries are updated when the project he belongs to is removed",
			projectToSync: createProject("plan9", createUser("bob")),
			existingUsers: []*kubermaticv1.User{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticv1.UserSpec{
						Name:  "bob",
						Email: "bob@acme.com",
						Projects: []kubermaticv1.ProjectGroup{
							{
								Group: "editors-plan9",
								Name:  "plan9",
							},
						},
					},
				},

				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alice",
					},
					Spec: kubermaticv1.UserSpec{
						Name:  "alice",
						Email: "alice@acme.com",
						Projects: []kubermaticv1.ProjectGroup{
							{
								Group: "editors-placeX",
								Name:  "placeX",
							},
						},
					},
				},
			},
			expectedActions: []string{"update", "update"},
			expectedUsers: []*kubermaticv1.User{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticv1.UserSpec{
						Name:     "bob",
						Email:    "bob@acme.com",
						Projects: []kubermaticv1.ProjectGroup{},
					},
				},
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alice",
					},
					Spec: kubermaticv1.UserSpec{
						Name:  "alice",
						Email: "alice@acme.com",
						Projects: []kubermaticv1.ProjectGroup{
							{
								Group: "editors-placeX",
								Name:  "placeX",
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
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
			for _, existingUser := range test.existingUsers {
				err := userIndexer.Add(existingUser)
				if err != nil {
					t.Fatal(err)
				}
				kubermaticObjs = append(kubermaticObjs, existingUser)
			}
			userLister := kubermaticv1lister.NewUserLister(userIndexer)
			fakeKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)

			// act
			target := Controller{}
			target.kubermaticMasterClient = fakeKubermaticClient
			target.projectLister = projectLister
			target.userLister = userLister
			err := target.ensureProjectCleanup(test.projectToSync)
			if err != nil {
				t.Fatal(err)
			}

			// validate
			actualActions := fakeKubermaticClient.Actions()
			if len(test.expectedActions) != len(actualActions) {
				t.Fatalf("expected to get exactly %d actions but got %d, actions = %v", len(test.expectedActions), len(actualActions), actualActions)
			}

			// verifiedActions is equal to one because the last update action is updating the project
			// not the users and this is something we don't want to validate
			verifiedActions := 1
			for index, actualAction := range actualActions {
				if actualAction.Matches(test.expectedActions[index], "users") {
					updateAction, ok := actualAction.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("cannot cast actualAction to CreateActon")
					}
					updatedUser := updateAction.GetObject().(*kubermaticv1.User)
					if !equality.Semantic.DeepEqual(updatedUser, test.expectedUsers[index]) {
						t.Fatalf("%v", diff.ObjectDiff(updatedUser, test.expectedUsers[index]))
					}
					verifiedActions = verifiedActions + 1
				}
			}
			if verifiedActions != len(test.expectedActions) {
				t.Fatalf("expected to verify %d actions but only %d were verified", verifiedActions, len(test.expectedActions))
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

			name:          "Scenario 1: When a project is removed corresponding Subject from the RBAC Binding are removed",
			projectToSync: createProject("plan9", createUser("James Bond")),
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
			},
			expectedActionsForMaster: []string{"get", "update", "get", "update"},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
			existingClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
			seedClusters:            2,
			expectedActionsForSeeds: []string{"get", "update", "get", "update"},
			expectedClusterRoleBindingsForSeeds: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
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

			fakeKubeClient := fake.NewSimpleClientset(objs...)
			fakeKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)
			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForSeeds {
					objs = append(objs, existingClusterRoleBinding)
				}
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerFactory := kuberinformers.NewSharedInformerFactory(fakeSeedKubeClient, time.Minute*5)
				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerFactory, nil, nil)
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := Controller{}
			target.kubeMasterClient = fakeKubeClient
			target.kubermaticMasterClient = fakeKubermaticClient
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
					createActionIndex = createActionIndex + 1
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
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
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
					createActionIndex = createActionIndex + 1
				}
			}
		})
	}
}

func TestEnsureProjectClusterRBACRoleBindingForNamedResource(t *testing.T) {
	tests := []struct {
		name                        string
		projectToSync               *kubermaticv1.Project
		expectedClusterRoleBindings []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindings []*rbacv1.ClusterRoleBinding
		expectedActions             []string
	}{
		// scenario 1
		{
			name:            "scenario 1: desired RBAC Role Bindings for a project resource are created",
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			expectedActions: []string{"create", "create", "create"},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
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
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
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
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},

		// scenario 2
		{
			name:          "scenario 2: no op when desicred RBAC Role Bindings exist",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			existingClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
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
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
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
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: update when existing binding doesn't match desired ones",
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			expectedActions: []string{"update", "update", "update"},
			existingClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
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
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "wrong-subject-name",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "wrong-subject-name",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
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
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			clusterRoleBindingIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingClusterRoleBinding := range test.existingClusterRoleBindings {
				err := clusterRoleBindingIndexer.Add(existingClusterRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingClusterRoleBinding)
			}
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			clusterRoleBindingLister := rbaclister.NewClusterRoleBindingLister(clusterRoleBindingIndexer)

			// act
			target := Controller{}
			err := target.ensureClusterRBACRoleBindingForNamedResource(test.projectToSync.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, test.projectToSync.GetObjectMeta(), fakeKubeClient, clusterRoleBindingLister)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			if len(test.expectedClusterRoleBindings) == 0 {
				if len(fakeKubeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
				}
				return
			}

			if len(fakeKubeClient.Actions()) != len(test.expectedClusterRoleBindings) {
				t.Fatalf("unexpected actions %v", fakeKubeClient.Actions())
			}
			for index, action := range fakeKubeClient.Actions() {
				if !action.Matches(test.expectedActions[index], "clusterrolebindings") {
					t.Fatalf("unexpected action %#v", action)
				}
				// TODO: figure out why action.(clienttesting.GenericAction) does not work
				createAction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRoleBinding), test.expectedClusterRoleBindings[index]) {
					t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoleBindings[index], createAction.GetObject().(*rbacv1.ClusterRoleBinding)))
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
			expectedActionsForMaster: []string{"create", "create", "create", "create"},
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
				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
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
				&rbacv1.ClusterRole{
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
				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
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
				&rbacv1.ClusterRole{
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
			roleLister := rbaclister.NewClusterRoleLister(roleIndexer)

			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerFactory := kuberinformers.NewSharedInformerFactory(fakeSeedKubeClient, time.Minute*5)
				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerFactory, nil, nil)
				seedClusterProviders[i] = fakeProvider
			}

			// act
			target := Controller{}
			target.kubeMasterClient = fakeKubeClient
			target.projectResources = test.projectResourcesToSync
			target.rbacClusterRoleMasterLister = roleLister
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
					createActionIndex = createActionIndex + 1
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
					createActionIndex = createActionIndex + 1
				}
			}
		})
	}
}

func TestEnsureProjectClusterRBACRoleForNamedResource(t *testing.T) {
	tests := []struct {
		name                 string
		projectToSync        *kubermaticv1.Project
		expectedClusterRoles []*rbacv1.ClusterRole
		existingClusterRoles []*rbacv1.ClusterRole
		expectedActions      []string
	}{
		// scenario 1
		{
			name:            "scenario 1: desired RBAC Roles for a project resource are created",
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			expectedActions: []string{"create", "create", "create"},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
		},

		// scenario 2
		{
			name:          "scenario 2: no op when desicred RBAC Roles exist",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			existingClusterRoles: []*rbacv1.ClusterRole{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: update when desired are not the same as expected RBAC Roles",
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			expectedActions: []string{"update", "update"},
			existingClusterRoles: []*rbacv1.ClusterRole{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
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
			clusterRoleIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingClusterRole := range test.existingClusterRoles {
				err := clusterRoleIndexer.Add(existingClusterRole)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingClusterRole)
			}
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			clusterRoleLister := rbaclister.NewClusterRoleLister(clusterRoleIndexer)

			// act
			target := Controller{}
			err := target.ensureClusterRBACRoleForNamedResource(test.projectToSync.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, test.projectToSync.GetObjectMeta(), fakeKubeClient, clusterRoleLister)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			if len(test.expectedClusterRoles) == 0 {
				if len(fakeKubeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
				}
				return
			}

			if len(fakeKubeClient.Actions()) != len(test.expectedClusterRoles) {
				t.Fatalf("unexpected actions %#v ", fakeKubeClient.Actions())
			}

			for index, action := range fakeKubeClient.Actions() {
				if !action.Matches(test.expectedActions[index], "clusterroles") {
					t.Fatalf("unexpected action %#v", action)
				}
				// TODO: figure out why action.(clienttesting.GenericAction) does not work
				createAction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRole), test.expectedClusterRoles[index]) {
					t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoles[index], createAction.GetObject().(*rbacv1.ClusterRole)))
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
			name:            "scenario 1: make sure, that the owner of the newly created project is set properly.",
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			existingUser:    createUser("James Bond"),
			expectedBinding: createExpectedOwnerBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
		},
		{
			name:            "scenario 2: no op when the owner of the project was set.",
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			existingUser:    createUser("James Bond"),
			existingBinding: createExpectedOwnerBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
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
			userLister := kubermaticv1lister.NewUserLister(userIndexer)
			bindingLister := kubermaticv1lister.NewUserProjectBindingLister(bindingIndexer)

			// act
			target := Controller{}
			target.kubermaticMasterClient = kubermaticFakeClient
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

func createProject(name string, owner *kubermaticv1.User) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       kubermaticv1.ProjectKindName,
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID(name) + "ID",
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
					UID:        owner.GetUID(),
					Name:       owner.Name,
				},
			},
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: name,
		},
	}
}

func createUser(name string) *kubermaticv1.User {
	return &kubermaticv1.User{
		TypeMeta: metav1.TypeMeta{
			Kind: kubermaticv1.UserKindName,
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:  "",
			Name: name,
		},
		Spec: kubermaticv1.UserSpec{
			Email: fmt.Sprintf("%s@acme.com", name),
		},
	}
}

func createExpectedOwnerBinding(userName string, project *kubermaticv1.Project) *kubermaticv1.UserProjectBinding {
	user := createUser(userName)
	return &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				},
			},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			UserEmail: user.Spec.Email,
			ProjectID: project.Name,
			Group:     fmt.Sprintf("owners-%s", project.Name),
		},
	}
}
