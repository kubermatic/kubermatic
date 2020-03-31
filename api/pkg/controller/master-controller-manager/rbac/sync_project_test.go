package rbac

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac/test"
	fakeInformerProvider "github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac/test/fake"
	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	k8scorev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeruntime "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
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
				project.ObjectMeta.ResourceVersion = "1"
				return project
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			objs = append(objs, test.expectedProject)
			masterClient := fakeruntime.NewFakeClient(objs...)

			// act
			target := projectController{
				ctx:    context.Background(),
				client: masterClient,
			}
			err := target.ensureProjectIsInActivePhase(test.projectToSync)
			assert.Nil(t, err)

			// validate
			var projectList kubermaticv1.ProjectList
			err = masterClient.List(context.Background(), &projectList)
			assert.NoError(t, err)

			assert.Len(t, projectList.Items, 1)
			assert.Equal(t, projectList.Items[0], *test.expectedProject)
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
				project.ObjectMeta.ResourceVersion = "1"
				return project
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			objs = append(objs, test.expectedProject)
			masterClient := fakeruntime.NewFakeClient(objs...)

			// act
			target := projectController{
				ctx:    context.Background(),
				client: masterClient,
			}
			err := target.ensureCleanupFinalizerExists(test.projectToSync)
			assert.NoError(t, err)

			// validate
			var projectList kubermaticv1.ProjectList
			err = masterClient.List(context.Background(), &projectList)
			assert.NoError(t, err)

			assert.Len(t, projectList.Items, 1)
			assert.Equal(t, projectList.Items[0], *test.expectedProject)
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
						Name:            "kubermatic:usersshkeies:owners",
						ResourceVersion: "1",
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
						Name:            "kubermatic:usersshkeies:editors",
						ResourceVersion: "1",
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
						Name:            "kubermatic:clusters:owners",
						ResourceVersion: "1",
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
						Name:            "kubermatic:clusters:editors",
						ResourceVersion: "1",
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
						Name:            "kubermatic:usersshkeies:owners",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
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
						Name:            "kubermatic:usersshkeies:editors",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
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
						Name:            "kubermatic:clusters:owners",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
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
						Name:            "kubermatic:clusters:editors",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
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

			fakeMasterClient := fakeruntime.NewFakeClient(objs...)

			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			seedClientMap := make(map[string]client.Client)
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
				seedClientMap[strconv.Itoa(i)] = fakeruntime.NewFakeClient(objs...)
			}

			// act
			target := projectController{
				ctx:              context.Background(),
				client:           fakeMasterClient,
				seedClientMap:    seedClientMap,
				projectResources: test.projectResourcesToSync,
			}
			err := target.ensureClusterRBACRoleBindingForResources(test.projectToSync)
			assert.NoError(t, err)

			// validate master cluster
			{
				var clusterRoleBindingList rbacv1.ClusterRoleBindingList
				err := fakeMasterClient.List(context.Background(), &clusterRoleBindingList)
				assert.NoError(t, err)

			expectedBindingLoop:
				for _, expectedBinding := range test.expectedClusterRoleBindingsForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingBinding := range clusterRoleBindingList.Items {
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoop
						}
					}
					t.Fatalf("expected ClusteRoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, clusterRoleBindingList.Items, len(test.expectedClusterRoleBindingsForMaster),
					"cluster contains more ClusterRoleBindings than expected (%d > %d)", len(clusterRoleBindingList.Items), len(test.expectedClusterRoleBindingsForMaster))
			}

			// validate seed clusters
			for _, seedClient := range seedClientMap {
				var clusterRoleBindingList rbacv1.ClusterRoleBindingList
				err := seedClient.List(context.Background(), &clusterRoleBindingList)
				assert.NoError(t, err)

			expectedBindingLoopSeed:
				for _, expectedBinding := range test.expectedClusterRoleBindingsForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingBinding := range clusterRoleBindingList.Items {
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoopSeed
						}
					}
					t.Fatalf("expected ClusteRoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, clusterRoleBindingList.Items, len(test.expectedClusterRoleBindingsForSeeds),
					"cluster contains more ClusterRoleBindings than expected (%d > %d)", len(clusterRoleBindingList.Items), len(test.expectedClusterRoleBindingsForSeeds))
			}
		})
	}
}

// TestEnsureClusterResourcesCleanup test if cluster resources for the given
// project were removed from all physical locations
func TestEnsureClusterResourcesCleanup(t *testing.T) {
	tests := []struct {
		name                string
		projectToSync       *kubermaticv1.Project
		existingClustersOn  map[string][]*kubermaticv1.Cluster
		remainingClustersOn map[string][]string
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
			remainingClustersOn: map[string][]string{
				"a": {"abcd"},
				"b": {},
				"c": {"cat", "bat"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// prepare test data
			seedClusterProviders := make([]*ClusterProvider, len(test.existingClustersOn))
			seedClientMap := make(map[string]client.Client, len(test.existingClustersOn))
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
					seedClientMap[providerName] = fakeruntime.NewFakeClient(kubermaticObjs...)
					index++
				}
			}
			fakeKubermaticMasterClient := kubermaticfakeclientset.NewSimpleClientset(test.projectToSync)
			fakeMasterClusterClient := fakeruntime.NewFakeClient(test.projectToSync)
			fakeMasterClusterProvider := &ClusterProvider{
				kubermaticClient: fakeKubermaticMasterClient,
			}

			// act
			target := projectController{
				ctx:                   context.Background(),
				seedClusterProviders:  seedClusterProviders,
				masterClusterProvider: fakeMasterClusterProvider,
				client:                fakeMasterClusterClient,
				seedClientMap:         seedClientMap,
			}
			if err := target.ensureProjectCleanup(test.projectToSync); err != nil {
				t.Fatal(err)
			}

			// validate
			for providerName, expectedClusterResources := range test.remainingClustersOn {
				cli := seedClientMap[providerName]

				var clusterList kubermaticv1.ClusterList
				err := cli.List(context.Background(), &clusterList)
				assert.NoError(t, err)

				remainingClusters := []string{}
				for _, c := range clusterList.Items {
					remainingClusters = append(remainingClusters, c.Name)
				}

				sort.Strings(expectedClusterResources)
				sort.Strings(remainingClusters)

				assert.Equal(t, expectedClusterResources, remainingClusters)
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
						Name:            "kubermatic:usersshkeies:owners",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: nil,
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeies:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeies:editors",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: nil,
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
						Name:            "kubermatic:clusters:owners",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: nil,
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:clusters:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:clusters:editors",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: nil,
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
			allObjs := []runtime.Object{}
			projectIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.projectToSync != nil {
				err := projectIndexer.Add(test.projectToSync)
				if err != nil {
					t.Fatal(err)
				}
				kubermaticObjs = append(kubermaticObjs, test.projectToSync)
			}

			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.existingUser != nil {
				err := userIndexer.Add(test.existingUser)
				if err != nil {
					t.Fatal(err)
				}
				kubermaticObjs = append(kubermaticObjs, test.projectToSync)
			}

			for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForMaster {
				objs = append(objs, existingClusterRoleBinding)
			}

			// merge vanilla and Kubermatic objects into one slice for the controller-runtime fake client
			allObjs = append(allObjs, objs...)
			allObjs = append(allObjs, kubermaticObjs...)

			fakeMasterKubeClient := fake.NewSimpleClientset(objs...)
			fakeMasterKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)
			fakeMasterClusterClient := fakeruntime.NewFakeClient(allObjs...)
			fakeMasterClusterProvider := &ClusterProvider{
				kubeClient:       fakeMasterKubeClient,
				kubermaticClient: fakeMasterKubermaticClient,
			}
			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			seedClusterClientMap := make(map[string]client.Client)
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

				seedClusterClientMap[strconv.Itoa(i)] = fakeruntime.NewFakeClient(objs...)
			}

			// act
			target := projectController{
				ctx:                   context.Background(),
				masterClusterProvider: fakeMasterClusterProvider,
				projectResources:      test.projectResourcesToSync,
				seedClusterProviders:  seedClusterProviders,
				client:                fakeMasterClusterClient,
				seedClientMap:         seedClusterClientMap,
			}
			err := target.ensureProjectCleanup(test.projectToSync)
			assert.NoError(t, err)

			// validate master cluster
			{
				var clusterRoleBindingList rbacv1.ClusterRoleBindingList
				err := fakeMasterClusterClient.List(context.Background(), &clusterRoleBindingList)
				assert.NoError(t, err)

			expectedBindingLoop:
				for _, expectedBinding := range test.expectedClusterRoleBindingsForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingBinding := range clusterRoleBindingList.Items {
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoop
						}
					}
					t.Fatalf("expected ClusteRoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, clusterRoleBindingList.Items, len(test.expectedClusterRoleBindingsForMaster),
					"cluster contains more ClusterRoleBindings than expected (%d > %d)", len(clusterRoleBindingList.Items), len(test.expectedClusterRoleBindingsForMaster))
			}

			// validate seed clusters
			for _, seedClient := range seedClusterClientMap {
				var clusterRoleBindingList rbacv1.ClusterRoleBindingList
				err := seedClient.List(context.Background(), &clusterRoleBindingList)
				assert.NoError(t, err)

			expectedBindingLoopSeed:
				for _, expectedBinding := range test.expectedClusterRoleBindingsForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingBinding := range clusterRoleBindingList.Items {
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoopSeed
						}
					}
					t.Fatalf("expected ClusteRoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, clusterRoleBindingList.Items, len(test.expectedClusterRoleBindingsForSeeds),
					"cluster contains more ClusterRoleBindings than expected (%d > %d)", len(clusterRoleBindingList.Items), len(test.expectedClusterRoleBindingsForSeeds))
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
						Name:            "kubermatic:clusters:owners",
						ResourceVersion: "1",
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
						Name:            "kubermatic:clusters:editors",
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"clusters"},
							Verbs:     []string{"create"},
						},
					},
				},
			},

			expectedClusterRolesForMaster: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeies:owners",
						ResourceVersion: "1",
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
						Name:            "kubermatic:usersshkeies:editors",
						ResourceVersion: "1",
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
						Name:            "kubermatic:userprojectbindings:owners",
						ResourceVersion: "1",
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
			fakeMasterClient := fakeruntime.NewFakeClient(objs...)
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
			seedClients := make(map[string]client.Client)
			for i := 0; i < test.seedClusters; i++ {
				objs := []runtime.Object{}
				fakeSeedKubeClient := fake.NewSimpleClientset(objs...)
				fakeKubeInformerProvider := NewInformerProvider(fakeSeedKubeClient, time.Minute*5)
				fakeProvider := NewClusterProvider(strconv.Itoa(i), fakeSeedKubeClient, fakeKubeInformerProvider, nil, nil)
				seedClusterProviders[i] = fakeProvider
				seedClients[strconv.Itoa(i)] = fakeruntime.NewFakeClient(objs...)
			}

			// act
			target := projectController{
				ctx:                   context.Background(),
				masterClusterProvider: fakeMasterClusterProvider,
				projectResources:      test.projectResourcesToSync,
				seedClusterProviders:  seedClusterProviders,
				client:                fakeMasterClient,
				seedClientMap:         seedClients,
			}
			err := target.ensureClusterRBACRoleForResources()
			assert.Nil(t, err)

			// validate master cluster
			{
				var clusterRoleList rbacv1.ClusterRoleList
				err = fakeMasterClient.List(context.Background(), &clusterRoleList)
				assert.NoError(t, err)

			expectedClusterRoleLoop:
				for _, expectedClusterRole := range test.expectedClusterRolesForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range clusterRoleList.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedClusterRoleLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}

				assert.Len(t, clusterRoleList.Items, len(test.expectedClusterRolesForMaster),
					"cluster contains more ClusterRoles than expected (%d > %d)", len(clusterRoleList.Items), len(test.expectedClusterRolesForMaster))
			}

			// validate seed clusters
			for _, fakeSeedClient := range seedClients {
				var clusterRoleList rbacv1.ClusterRoleList
				err = fakeSeedClient.List(context.Background(), &clusterRoleList)
				assert.NoError(t, err)

			expectedSeecClusterRoleLoop:
				for _, expectedClusterRole := range test.expectedClusterRolesForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range clusterRoleList.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedSeecClusterRoleLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}

				assert.Len(t, clusterRoleList.Items, len(test.expectedClusterRolesForSeeds),
					"cluster contains more ClusterRoles than expected (%d > %d)", len(clusterRoleList.Items), len(test.expectedClusterRolesForSeeds))
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
				binding.ObjectMeta.ResourceVersion = "1"
				return binding
			}(),
		},
		{
			name:            "scenario 2: no op when the owner of the project was set.",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingUser:    test.CreateUser("James Bond"),
			existingBinding: test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond"))),
			expectedBinding: func() *kubermaticv1.UserProjectBinding {
				binding := test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond")))
				return binding
			}(),
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
			masterClient := fakeruntime.NewFakeClient(objs...)

			// act
			target := projectController{
				ctx:    context.Background(),
				client: masterClient,
			}
			err := target.ensureProjectOwner(test.projectToSync)
			assert.NoError(t, err)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			var userProjectBindingList kubermaticv1.UserProjectBindingList
			err = masterClient.List(context.Background(), &userProjectBindingList)
			assert.NoError(t, err)

			assert.Len(t, userProjectBindingList.Items, 1)
			// Hack around the fact that the bindings' names are random
			userProjectBindingList.Items[0].ObjectMeta.Name = test.expectedBinding.ObjectMeta.Name
			assert.Equal(t, userProjectBindingList.Items[0], *test.expectedBinding)
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
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

			fakeMasterClient := fakeruntime.NewFakeClient(objs...)

			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			seedClientMap := make(map[string]client.Client)
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
				seedClientMap[strconv.Itoa(i)] = fakeruntime.NewFakeClient(objs...)
			}

			// act
			target := projectController{
				ctx:                  context.Background(),
				client:               fakeMasterClient,
				seedClientMap:        seedClientMap,
				projectResources:     test.projectResourcesToSync,
				seedClusterProviders: seedClusterProviders,
			}
			err := target.ensureRBACRoleForResources()
			assert.Nil(t, err)

			// validate master cluster
			{
				var roleList rbacv1.RoleList
				err = fakeMasterClient.List(context.Background(), &roleList)
				assert.NoError(t, err)

			expectedRoleLoop:
				for _, expectedClusterRole := range test.expectedRolesForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range roleList.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedRoleLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}

				assert.Len(t, roleList.Items, len(test.expectedRolesForMaster),
					"cluster contains more ClusterRoles than expected (%d > %d)", len(roleList.Items), len(test.expectedRolesForMaster))
			}

			// validate seed clusters
			for _, fakeSeedClient := range seedClientMap {
				var roleList rbacv1.RoleList
				err = fakeSeedClient.List(context.Background(), &roleList)
				assert.NoError(t, err)

			expectedSeecClusterRoleLoop:
				for _, expectedClusterRole := range test.expectedRolesForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range roleList.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedSeecClusterRoleLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}

				assert.Len(t, roleList.Items, len(test.expectedRolesForSeeds),
					"cluster contains more ClusterRoles than expected (%d > %d)", len(roleList.Items), len(test.expectedRolesForSeeds))
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
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

			fakeMasterClient := fakeruntime.NewFakeClient(objs...)

			seedClusterProviders := make([]*ClusterProvider, test.seedClusters)
			seedClusterClientMap := make(map[string]client.Client)
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

				seedClusterClientMap[strconv.Itoa(i)] = fakeruntime.NewFakeClient(objs...)
			}

			// act
			target := projectController{
				ctx:              context.Background(),
				client:           fakeMasterClient,
				seedClientMap:    seedClusterClientMap,
				projectResources: test.projectResourcesToSync,
			}
			err := target.ensureRBACRoleBindingForResources(test.projectToSync)
			assert.Nil(t, err)

			// validate master cluster
			{
				var roleBingingList rbacv1.RoleBindingList
				err = fakeMasterClient.List(context.Background(), &roleBingingList)
				assert.NoError(t, err)

			expectedRoleLoop:
				for _, expectedClusterRole := range test.expectedRoleBindingsForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range roleBingingList.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedRoleLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}

				assert.Len(t, roleBingingList.Items, len(test.expectedRoleBindingsForMaster),
					"cluster contains more ClusterRoles than expected (%d > %d)", len(roleBingingList.Items), len(test.expectedRoleBindingsForMaster))
			}

			// validate seed clusters
			for _, fakeSeedClient := range seedClusterClientMap {
				var roleBingingList rbacv1.RoleBindingList
				err = fakeSeedClient.List(context.Background(), &roleBingingList)
				assert.NoError(t, err)

			expectedSeecClusterRoleLoop:
				for _, expectedClusterRole := range test.expectedRoleBindingsForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range roleBingingList.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedSeecClusterRoleLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}

				assert.Len(t, roleBingingList.Items, len(test.expectedRoleBindingsForSeeds),
					"cluster contains more ClusterRoles than expected (%d > %d)", len(roleBingingList.Items), len(test.expectedRoleBindingsForSeeds))
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: nil,
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
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: nil,
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
			allObjs := []runtime.Object{}
			projectIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			err := projectIndexer.Add(test.projectToSync)
			if err != nil {
				t.Fatal(err)
			}
			kubermaticObjs = append(kubermaticObjs, test.projectToSync)

			roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingRoleBinding := range test.existingRoleBindingsForMaster {
				objs = append(objs, existingRoleBinding)
				err := roleBindingsIndexer.Add(existingRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
			}

			// merge vanilla and Kubermatic objects into one slice for the controller-runtime fake client
			allObjs = append(allObjs, objs...)
			allObjs = append(allObjs, kubermaticObjs...)

			fakeMasterKubeClient := fake.NewSimpleClientset(objs...)
			fakeMasterKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)
			fakeMasterClusterClient := fakeruntime.NewFakeClient(allObjs...)
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
			seedClusterClientMap := make(map[string]client.Client)
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

				seedClusterClientMap[strconv.Itoa(i)] = fakeruntime.NewFakeClient(objs...)
			}

			// act
			target := projectController{
				ctx:                   context.Background(),
				masterClusterProvider: fakeMasterClusterProvider,
				client:                fakeMasterClusterClient,
				seedClientMap:         seedClusterClientMap,
				projectResources:      test.projectResourcesToSync,
				seedClusterProviders:  seedClusterProviders,
			}
			err = target.ensureProjectCleanup(test.projectToSync)
			assert.NoError(t, err)

			// validate master cluster
			{
				var roleBindingList rbacv1.RoleBindingList
				err := fakeMasterClusterClient.List(context.Background(), &roleBindingList)
				assert.NoError(t, err)

			expectedBindingLoop:
				for _, expectedBinding := range test.expectedRoleBindingsForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingBinding := range roleBindingList.Items {
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoop
						}
					}
					t.Fatalf("expected RoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, roleBindingList.Items, len(test.expectedRoleBindingsForMaster),
					"cluster contains more RoleBindings than expected (%d > %d)", len(roleBindingList.Items), len(test.expectedRoleBindingsForMaster))
			}

			// validate seed clusters
			for _, seedClient := range seedClusterClientMap {
				var roleBindingList rbacv1.RoleBindingList
				err := seedClient.List(context.Background(), &roleBindingList)
				assert.NoError(t, err)

			expectedBindingLoopSeed:
				for _, expectedBinding := range test.expectedRoleBindingsForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingBinding := range roleBindingList.Items {
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoopSeed
						}
					}
					t.Fatalf("expected RoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, roleBindingList.Items, len(test.expectedRoleBindingsForSeeds),
					"cluster contains more RoleBindings than expected (%d > %d)", len(roleBindingList.Items), len(test.expectedRoleBindingsForSeeds))
			}
		})
	}
}
