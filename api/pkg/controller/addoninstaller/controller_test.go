package addoninstaller

import (
	"testing"

	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/rand"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

var addons = []string{"Foo", "Bar"}

func truePtr() *bool {
	b := true
	return &b
}
func TestCreateAddon(t *testing.T) {
	name := rand.String(10)
	tests := []struct {
		expectedActions       []string
		expectedClusterAddons []*kubermaticv1.Addon
		name                  string
		cluster               *kubermaticv1.Cluster
		err                   error
		ns                    *corev1.Namespace
	}{
		{
			expectedActions: []string{"create", "create"},
			expectedClusterAddons: []*kubermaticv1.Addon{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Foo",
						Namespace: "cluster-" + name,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "kubermatic.k8s.io/v1",
								Kind:               "Cluster",
								Name:               name,
								Controller:         truePtr(),
								BlockOwnerDeletion: truePtr(),
							},
						},
					},
					Spec: kubermaticv1.AddonSpec{
						Name: "Foo",
						Cluster: corev1.ObjectReference{
							Kind: "Cluster",
							Name: name,
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Bar",
						Namespace: "cluster-" + name,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "kubermatic.k8s.io/v1",
								Kind:               "Cluster",
								Name:               name,
								Controller:         truePtr(),
								BlockOwnerDeletion: truePtr(),
							},
						},
					},
					Spec: kubermaticv1.AddonSpec{
						Name: "Bar",
						Cluster: corev1.ObjectReference{
							Kind: "Cluster",
							Name: name,
						},
					},
				},
			},
			name: "successfully created",
			err:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kubermaticv1.WorkerNameLabelKey: "worker",
					},
					Name: name,
				},
				Spec:    kubermaticv1.ClusterSpec{},
				Address: kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					Health: kubermaticv1.ClusterHealth{
						ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
							Apiserver: true,
						},
					},
					NamespaceName: "cluster-" + name,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var objects []runtime.Object
			kubermaticObjs := []runtime.Object{}
			if test.ns != nil {
				objects = append(objects, test.ns)
			}

			clusterIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			err := clusterIndexer.Add(test.cluster)
			if err != nil {
				t.Fatal(err)
			}
			addonIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			err = addonIndexer.Add(test.cluster)
			if err != nil {
				t.Fatal(err)
			}
			kubermaticObjs = append(kubermaticObjs, test.cluster)
			kubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)

			controller := Controller{}
			controller.client = kubermaticClient
			controller.workerName = "worker"
			controller.defaultAddonList = addons
			controller.clusterLister = kubermaticv1lister.NewClusterLister(clusterIndexer)
			controller.addonLister = kubermaticv1lister.NewAddonLister(addonIndexer)

			// act: create
			actions := controller.client.(*kubermaticfakeclientset.Clientset).Actions()
			beforeActionCount := len(actions)
			if beforeActionCount != 0 {
				t.Errorf("action count is not 0 but %d", beforeActionCount)
			}
			err = controller.sync(name)
			if err != nil {
				t.Error(err)
			}
			actions = controller.client.(*kubermaticfakeclientset.Clientset).Actions()
			for index, action := range actions {
				if !action.Matches(test.expectedActions[index], "addons") {
					t.Fatalf("unexpected action %#v", action)
				}
				createaction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createaction.GetObject().(*kubermaticv1.Addon), test.expectedClusterAddons[index]) {
					t.Fatalf("Addon diff: %v", diff.ObjectDiff(test.expectedClusterAddons[index], createaction.GetObject().(*kubermaticv1.Addon)))
				}
			}

			if len(actions) != beforeActionCount+2 {
				t.Error("client did not made call to create addons")
			}
		})
	}
}
