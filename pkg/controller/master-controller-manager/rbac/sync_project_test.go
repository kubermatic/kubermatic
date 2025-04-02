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
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac/test"
	fakeInformerProvider "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac/test/fake"
	kubermaticface "k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getFakeRestMapper(t *testing.T) meta.RESTMapper {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := kubermaticv1.AddToScheme(scheme); err != nil {
		t.Fatalf("getFakeRestMapper: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("getFakeRestMapper: %v", err)
	}
	return testrestmapper.TestOnlyStaticRESTMapper(scheme)
}

func getFakeClientset(objs ...ctrlruntimeclient.Object) *fake.Clientset {
	runtimeObjects := []runtime.Object{}
	for _, obj := range objs {
		runtimeObjects = append(runtimeObjects, obj.(runtime.Object))
	}

	return fake.NewSimpleClientset(runtimeObjects...)
}

func TestEnsureProjectInitialized(t *testing.T) {
	tests := []struct {
		name            string
		projectToSync   *kubermaticv1.Project
		expectedProject *kubermaticv1.Project
	}{
		{
			name:          "scenario 1: cleanup finializer is added to a project",
			projectToSync: test.CreateProject("thunderball"),
			expectedProject: func() *kubermaticv1.Project {
				project := test.CreateProject("thunderball")
				project.Finalizers = []string{"kubermatic.k8c.io/controller-manager-rbac-cleanup"}
				return project
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}
			objs = append(objs, test.expectedProject)
			masterClient := kubermaticface.NewClientBuilder().WithObjects(objs...).Build()

			// act
			target := projectController{
				client:     masterClient,
				restMapper: getFakeRestMapper(t),
				log:        zap.NewNop().Sugar(),
			}
			err := target.ensureCleanupFinalizerExists(ctx, test.projectToSync)
			assert.NoError(t, err)

			// validate
			var projectList kubermaticv1.ProjectList
			err = masterClient.List(ctx, &projectList)
			assert.NoError(t, err)

			projectList.Items[0].ResourceVersion = ""
			test.expectedProject.ResourceVersion = ""

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
					object: &kubermaticv1.Cluster{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ClusterKindName,
						},
					},
					destination: destinationSeed,
				},

				{
					object: &kubermaticv1.UserSSHKey{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.SSHKeyKind,
						},
					},
				},
			},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeys:owners",
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
						Name:     "kubermatic:usersshkeys:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeys:editors",
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
						Name:     "kubermatic:usersshkeys:editors",
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
					object: &kubermaticv1.Cluster{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ClusterKindName,
						},
					},
					destination: destinationSeed,
				},
				{
					object: &kubermaticv1.UserSSHKey{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.SSHKeyKind,
						},
					},
				},
			},
			existingClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeys:owners",
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
						Name:     "kubermatic:usersshkeys:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeys:editors",
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
						Name:     "kubermatic:usersshkeys:editors",
					},
				},
			},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeys:owners",
						ResourceVersion: "1",
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
						Name:     "kubermatic:usersshkeys:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeys:editors",
						ResourceVersion: "1",
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
						Name:     "kubermatic:usersshkeys:editors",
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

		// scenario 3
		{
			name:                     "Scenario 3: Proper set of RBAC Bindings for project's ExternalCluster created on master",
			projectToSync:            "thunderball",
			expectedActionsForMaster: []string{"create", "create"},
			projectResourcesToSync: []projectResource{
				{
					object: &kubermaticv1.ExternalCluster{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ExternalClusterKind,
						},
					},
				},
			},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:externalclusters:owners",
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
						Name:     "kubermatic:externalclusters:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:externalclusters:editors",
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
						Name:     "kubermatic:externalclusters:editors",
					},
				},
			},
			seedClusters:                        2,
			expectedActionsForSeeds:             []string{"create", "create"},
			expectedClusterRoleBindingsForSeeds: []*rbacv1.ClusterRoleBinding{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}
			roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForMaster {
				objs = append(objs, existingClusterRoleBinding)
				err := roleBindingsIndexer.Add(existingClusterRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
			}
			fakeKubeClient := getFakeClientset(objs...)
			// manually set lister as we don't want to start informers in the tests
			fakeKubeInformerProviderForMaster := NewInformerProvider(fakeKubeClient, time.Minute*5)
			fakeInformerFactoryForClusterRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, metav1.NamespaceAll)
			fakeInformerFactoryForClusterRole.AddFakeClusterRoleBindingInformer(roleBindingsIndexer)
			fakeKubeInformerProviderForMaster.kubeInformers[metav1.NamespaceAll] = fakeInformerFactoryForClusterRole

			fakeMasterClient := kubermaticface.NewClientBuilder().WithObjects(objs...).Build()

			seedClientMap := make(map[string]ctrlruntimeclient.Client)
			for i := range test.seedClusters {
				objs := []ctrlruntimeclient.Object{}
				for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForSeeds {
					objs = append(objs, existingClusterRoleBinding)
				}

				seedClientMap[strconv.Itoa(i)] = kubermaticface.NewClientBuilder().WithObjects(objs...).Build()
			}

			// act
			target := projectController{
				client:           fakeMasterClient,
				restMapper:       getFakeRestMapper(t),
				seedClientMap:    seedClientMap,
				projectResources: test.projectResourcesToSync,
				log:              zap.NewNop().Sugar(),
			}
			err := target.ensureClusterRBACRoleBindingForResources(ctx, test.projectToSync)
			assert.NoError(t, err)

			// validate master cluster
			{
				var clusterRoleBindingList rbacv1.ClusterRoleBindingList
				err := fakeMasterClient.List(ctx, &clusterRoleBindingList)
				assert.NoError(t, err)

			expectedBindingLoop:
				for _, expectedBinding := range test.expectedClusterRoleBindingsForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedBinding.ResourceVersion = ""

					for _, existingBinding := range clusterRoleBindingList.Items {
						existingBinding.ResourceVersion = ""
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoop
						}
					}

					t.Fatalf("expected ClusterRoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, clusterRoleBindingList.Items, len(test.expectedClusterRoleBindingsForMaster),
					"cluster contains more ClusterRoleBindings than expected (%d > %d)", len(clusterRoleBindingList.Items), len(test.expectedClusterRoleBindingsForMaster))
			}

			// validate seed clusters
			for _, seedClient := range seedClientMap {
				var clusterRoleBindingList rbacv1.ClusterRoleBindingList
				err := seedClient.List(ctx, &clusterRoleBindingList)
				assert.NoError(t, err)

			expectedBindingLoopSeed:
				for _, expectedBinding := range test.expectedClusterRoleBindingsForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedBinding.ResourceVersion = ""

					for _, existingBinding := range clusterRoleBindingList.Items {
						existingBinding.ResourceVersion = ""
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoopSeed
						}
					}
					t.Fatalf("expected ClusterRoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, clusterRoleBindingList.Items, len(test.expectedClusterRoleBindingsForSeeds),
					"cluster contains more ClusterRoleBindings than expected (%d > %d)", len(clusterRoleBindingList.Items), len(test.expectedClusterRoleBindingsForSeeds))
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
			projectToSync: test.CreateProject("plan9"),
			projectResourcesToSync: []projectResource{
				{
					object: &kubermaticv1.Cluster{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ClusterKindName,
						},
					},
					destination: destinationSeed,
				},
				{
					object: &kubermaticv1.UserSSHKey{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.SSHKeyKind,
						},
					},
				},
			},
			expectedActionsForMaster: []string{"get", "update", "get", "update"},
			expectedClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeys:owners",
						ResourceVersion: "1",
					},
					Subjects: nil,
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeys:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeys:editors",
						ResourceVersion: "1",
					},
					Subjects: nil,
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeys:editors",
					},
				},
			},
			existingClusterRoleBindingsForMaster: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeys:owners",
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
						Name:     "kubermatic:usersshkeys:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeys:editors",
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
						Name:     "kubermatic:usersshkeys:editors",
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
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}
			kubermaticObjs := []ctrlruntimeclient.Object{}
			allObjs := []ctrlruntimeclient.Object{}
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

			fakeMasterClusterClient := kubermaticface.NewClientBuilder().WithObjects(allObjs...).Build()

			seedClusterClientMap := make(map[string]ctrlruntimeclient.Client)
			for i := range test.seedClusters {
				objs := []ctrlruntimeclient.Object{}
				for _, existingClusterRoleBinding := range test.existingClusterRoleBindingsForSeeds {
					objs = append(objs, existingClusterRoleBinding)
				}

				seedClusterClientMap[strconv.Itoa(i)] = kubermaticface.NewClientBuilder().WithObjects(objs...).Build()
			}

			// act
			target := projectController{
				projectResources: test.projectResourcesToSync,
				client:           fakeMasterClusterClient,
				restMapper:       getFakeRestMapper(t),
				seedClientMap:    seedClusterClientMap,
				log:              zap.NewNop().Sugar(),
			}
			err := target.ensureProjectCleanup(ctx, test.projectToSync)
			assert.NoError(t, err)

			// validate master cluster
			{
				var clusterRoleBindingList rbacv1.ClusterRoleBindingList
				err := fakeMasterClusterClient.List(ctx, &clusterRoleBindingList)
				assert.NoError(t, err)

			expectedBindingLoop:
				for _, expectedBinding := range test.expectedClusterRoleBindingsForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedBinding.ResourceVersion = ""

					for _, existingBinding := range clusterRoleBindingList.Items {
						existingBinding.ResourceVersion = ""
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoop
						}
					}
					t.Fatalf("expected ClusterRoleBinding %q not found in cluster", expectedBinding.Name)
				}

				assert.Len(t, clusterRoleBindingList.Items, len(test.expectedClusterRoleBindingsForMaster),
					"cluster contains more ClusterRoleBindings than expected (%d > %d)", len(clusterRoleBindingList.Items), len(test.expectedClusterRoleBindingsForMaster))
			}

			// validate seed clusters
			for _, seedClient := range seedClusterClientMap {
				var clusterRoleBindingList rbacv1.ClusterRoleBindingList
				err := seedClient.List(ctx, &clusterRoleBindingList)
				assert.NoError(t, err)

			expectedBindingLoopSeed:
				for _, expectedBinding := range test.expectedClusterRoleBindingsForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedBinding.ResourceVersion = ""

					for _, existingBinding := range clusterRoleBindingList.Items {
						existingBinding.ResourceVersion = ""
						if reflect.DeepEqual(*expectedBinding, existingBinding) {
							continue expectedBindingLoopSeed
						}
					}
					t.Fatalf("expected ClusterRoleBinding %q not found in cluster", expectedBinding.Name)
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
					object: &kubermaticv1.Cluster{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ClusterKindName,
						},
					},
					destination: destinationSeed,
				},

				{
					object: &kubermaticv1.UserSSHKey{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.SSHKeyKind,
						},
					},
				},
			},

			expectedClusterRolesForSeeds: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:clusters:owners",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
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
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "editors",
						},
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
						Name:            "kubermatic:usersshkeys:owners",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"usersshkeys"},
							Verbs:     []string{"create"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:usersshkeys:editors",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "editors",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"usersshkeys"},
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
					object: &kubermaticv1.UserProjectBinding{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.UserProjectBindingKind,
						},
					},
				},
			},

			expectedClusterRolesForMaster: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:userprojectbindings:owners",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"userprojectbindings"},
							Verbs:     []string{"create"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:userprojectbindings:projectmanagers",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "projectmanagers",
						},
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

		// scenario 3
		{
			name:                     "Scenario 3: Proper set of RBAC Roles for ExternalCluster resource are created on \"master\" and seed clusters",
			expectedActionsForMaster: []string{"create"},
			// UserProjectBinding is a resource that is only on master cluster
			expectedActionsForSeeds: []string{},
			seedClusters:            2,
			projectResourcesToSync: []projectResource{
				{
					object: &kubermaticv1.ExternalCluster{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ExternalClusterKind,
						},
					},
				},
			},

			expectedClusterRolesForMaster: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:externalclusters:owners",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"externalclusters"},
							Verbs:     []string{"create"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:externalclusters:editors",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "editors",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{"externalclusters"},
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
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}
			fakeMasterClient := kubermaticface.NewClientBuilder().WithObjects(objs...).Build()

			seedClients := make(map[string]ctrlruntimeclient.Client)
			for i := range test.seedClusters {
				seedClients[strconv.Itoa(i)] = kubermaticface.NewClientBuilder().Build()
			}

			// act
			target := projectController{
				projectResources: test.projectResourcesToSync,
				client:           fakeMasterClient,
				restMapper:       getFakeRestMapper(t),
				seedClientMap:    seedClients,
				log:              zap.NewNop().Sugar(),
			}
			err := target.ensureClusterRBACRoleForResources(ctx)
			assert.Nil(t, err)

			// validate master cluster
			{
				var clusterRoleList rbacv1.ClusterRoleList
				err = fakeMasterClient.List(ctx, &clusterRoleList)
				assert.NoError(t, err)

			expectedClusterRoleLoop:
				for _, expectedClusterRole := range test.expectedClusterRolesForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedClusterRole.ResourceVersion = ""

					for _, existingClusterRole := range clusterRoleList.Items {
						existingClusterRole.ResourceVersion = ""
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
				err = fakeSeedClient.List(ctx, &clusterRoleList)
				assert.NoError(t, err)

			expectedSeecClusterRoleLoop:
				for _, expectedClusterRole := range test.expectedClusterRolesForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedClusterRole.ResourceVersion = ""

					for _, existingClusterRole := range clusterRoleList.Items {
						existingClusterRole.ResourceVersion = ""
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
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
					namespace: "kubermatic",
				},
				{
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
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
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
							Resources: []string{"secrets"},
							Verbs:     []string{"create"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "projectmanagers",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
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
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
							Resources: []string{"secrets"},
							Verbs:     []string{"create"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "projectmanagers",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
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
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
					namespace: "kubermatic",
				},
				{
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
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
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
							Resources: []string{"secrets"},
							Verbs:     []string{"create"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "projectmanagers",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
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
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
							Resources: []string{"secrets"},
							Verbs:     []string{"create"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "projectmanagers",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
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
						Labels: map[string]string{
							kubermaticv1.AuthZRoleLabel: "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{corev1.SchemeGroupVersion.Group},
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
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}
			fakeKubeClient := getFakeClientset(objs...)
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

			fakeMasterClient := kubermaticface.NewClientBuilder().WithObjects(objs...).Build()

			seedClientMap := make(map[string]ctrlruntimeclient.Client)
			for i := range test.seedClusters {
				seedClientMap[strconv.Itoa(i)] = kubermaticface.NewClientBuilder().Build()
			}

			// act
			target := projectController{
				client:           fakeMasterClient,
				restMapper:       getFakeRestMapper(t),
				seedClientMap:    seedClientMap,
				projectResources: test.projectResourcesToSync,
				log:              zap.NewNop().Sugar(),
			}
			err := target.ensureRBACRoleForResources(ctx)
			assert.Nil(t, err)

			// validate master cluster
			{
				var roleList rbacv1.RoleList
				err = fakeMasterClient.List(ctx, &roleList)
				assert.NoError(t, err)

			expectedRoleLoop:
				for _, expectedClusterRole := range test.expectedRolesForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedClusterRole.ResourceVersion = ""

					for _, existingClusterRole := range roleList.Items {
						existingClusterRole.ResourceVersion = ""
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
				err = fakeSeedClient.List(ctx, &roleList)
				assert.NoError(t, err)

			expectedSeecClusterRoleLoop:
				for _, expectedClusterRole := range test.expectedRolesForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedClusterRole.ResourceVersion = ""

					for _, existingClusterRole := range roleList.Items {
						existingClusterRole.ResourceVersion = ""
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
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
					destination: destinationSeed,
					namespace:   "kubermatic",
				},

				{
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
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
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
					destination: destinationSeed,
					namespace:   "kubermatic",
				},
				{
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:projectmanagers",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-existing-project-1",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
					},
				},
			},
			expectedRoleBindingsForMaster: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-existing-project-1",
						},
						{

							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
					},
				},
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
						Name:      "kubermatic:secrets:projectmanagers",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-existing-project-1",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
					},
				},
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
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-existing-project-1",
						},
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
					},
				},
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
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}
			roleBindingsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingRoleBinding := range test.existingRoleBindingsForMaster {
				objs = append(objs, existingRoleBinding)
				err := roleBindingsIndexer.Add(existingRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
			}
			fakeKubeClient := getFakeClientset(objs...)
			// manually set lister as we don't want to start informers in the tests
			fakeKubeInformerProviderForMaster := NewInformerProvider(fakeKubeClient, time.Minute*5)
			for _, res := range test.projectResourcesToSync {
				fakeInformerFactoryForClusterRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, res.namespace)
				fakeInformerFactoryForClusterRole.AddFakeRoleBindingInformer(roleBindingsIndexer)
				fakeKubeInformerProviderForMaster.kubeInformers[res.namespace] = fakeInformerFactoryForClusterRole
			}
			fakeKubeInformerProviderForMaster.started = true

			fakeMasterClient := kubermaticface.NewClientBuilder().WithObjects(objs...).Build()

			seedClusterClientMap := make(map[string]ctrlruntimeclient.Client)
			for i := range test.seedClusters {
				objs := []ctrlruntimeclient.Object{}
				for _, existingRoleBinding := range test.existingRoleBindingsForSeeds {
					objs = append(objs, existingRoleBinding)
				}

				seedClusterClientMap[strconv.Itoa(i)] = kubermaticface.NewClientBuilder().WithObjects(objs...).Build()
			}

			// act
			target := projectController{
				client:           fakeMasterClient,
				restMapper:       getFakeRestMapper(t),
				seedClientMap:    seedClusterClientMap,
				projectResources: test.projectResourcesToSync,
				log:              zap.NewNop().Sugar(),
			}
			err := target.ensureRBACRoleBindingForResources(ctx, test.projectToSync)
			assert.Nil(t, err)

			// validate master cluster
			{
				var roleBingingList rbacv1.RoleBindingList
				err = fakeMasterClient.List(ctx, &roleBingingList)
				assert.NoError(t, err)

			expectedRoleLoop:
				for _, expectedClusterRole := range test.expectedRoleBindingsForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedClusterRole.ResourceVersion = ""

					for _, existingClusterRole := range roleBingingList.Items {
						existingClusterRole.ResourceVersion = ""
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
				err = fakeSeedClient.List(ctx, &roleBingingList)
				assert.NoError(t, err)

			expectedSeecClusterRoleLoop:
				for _, expectedClusterRole := range test.expectedRoleBindingsForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedClusterRole.ResourceVersion = ""

					for _, existingClusterRole := range roleBingingList.Items {
						existingClusterRole.ResourceVersion = ""
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
			projectToSync: test.CreateProject("plan9"),
			projectResourcesToSync: []projectResource{
				{
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
					destination: destinationSeed,
					namespace:   "kubermatic",
				},
				{
					object: &corev1.Secret{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
					namespace: "kubermatic",
				},
			},
			expectedActionsForMaster: []string{"get", "update"},
			expectedRoleBindingsForMaster: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					Subjects: nil,
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
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
						Name:      "kubermatic:secrets:projectmanagers",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-plan9",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
					},
				},
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
						Name:            "kubermatic:secrets:projectmanagers",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
					},
					Subjects: nil,
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:secrets:owners",
						Namespace:       "kubermatic",
						ResourceVersion: "1",
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
						Name:      "kubermatic:secrets:projectmanagers",
						Namespace: "kubermatic",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "projectmanagers-plan9",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secrets:projectmanagers",
					},
				},
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
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}
			kubermaticObjs := []ctrlruntimeclient.Object{}
			allObjs := []ctrlruntimeclient.Object{}
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

			fakeMasterClusterClient := kubermaticface.NewClientBuilder().WithObjects(allObjs...).Build()
			// manually set lister as we don't want to start informers in the tests

			seedClusterClientMap := make(map[string]ctrlruntimeclient.Client)
			for i := range test.seedClusters {
				objs := []ctrlruntimeclient.Object{}
				for _, existingRoleBinding := range test.existingRoleBindingsForSeeds {
					objs = append(objs, existingRoleBinding)
				}

				seedClusterClientMap[strconv.Itoa(i)] = kubermaticface.NewClientBuilder().WithObjects(objs...).Build()
			}

			// act
			target := projectController{
				client:           fakeMasterClusterClient,
				restMapper:       getFakeRestMapper(t),
				seedClientMap:    seedClusterClientMap,
				projectResources: test.projectResourcesToSync,
				log:              zap.NewNop().Sugar(),
			}
			err = target.ensureProjectCleanup(ctx, test.projectToSync)
			assert.NoError(t, err)

			// validate master cluster
			{
				var roleBindingList rbacv1.RoleBindingList
				err := fakeMasterClusterClient.List(ctx, &roleBindingList)
				assert.NoError(t, err)

			expectedBindingLoop:
				for _, expectedBinding := range test.expectedRoleBindingsForMaster {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedBinding.ResourceVersion = ""

					for _, existingBinding := range roleBindingList.Items {
						existingBinding.ResourceVersion = ""
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
				err := seedClient.List(ctx, &roleBindingList)
				assert.NoError(t, err)

			expectedBindingLoopSeed:
				for _, expectedBinding := range test.expectedRoleBindingsForSeeds {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedBinding.ResourceVersion = ""

					for _, existingBinding := range roleBindingList.Items {
						existingBinding.ResourceVersion = ""
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
