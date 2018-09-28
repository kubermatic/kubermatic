package addoninstaller

import (
	"testing"

	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	clienttesting "k8s.io/client-go/testing"
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

			kubermaticObjs = append(kubermaticObjs, test.cluster)
			kubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)

			informerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, informer.DefaultInformerResyncPeriod)

			controller := Controller{}
			controller.client = kubermaticClient
			controller.defaultAddonList = addons
			controller.clusterLister = informerFactory.Kubermatic().V1().Clusters().Lister()
			controller.addonLister = informerFactory.Kubermatic().V1().Addons().Lister()

			informerFactory.Start(wait.NeverStop)
			informerFactory.WaitForCacheSync(wait.NeverStop)

			// act: create
			actions := controller.client.(*kubermaticfakeclientset.Clientset).Actions()
			beforeActionCount := len(actions)

			err := controller.sync(name)
			if err != nil {
				t.Error(err)
			}
			actions = controller.client.(*kubermaticfakeclientset.Clientset).Actions()

			if len(actions) == beforeActionCount {
				t.Fatal("expected action on the client but none was made during the controller sync")
			}

			for index, action := range actions {
				if index <= beforeActionCount {
					continue
				}

				if !action.Matches(test.expectedActions[index-beforeActionCount], "addons") {
					t.Fatalf("unexpected action %#v", action)
				}
				createaction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createaction.GetObject().(*kubermaticv1.Addon), test.expectedClusterAddons[index-beforeActionCount]) {
					t.Fatalf("Addon diff: %v", diff.ObjectDiff(test.expectedClusterAddons[index-beforeActionCount], createaction.GetObject().(*kubermaticv1.Addon)))
				}
			}

			if len(actions) != beforeActionCount+2 {
				t.Error("client did not made call to create addons")
			}
		})
	}
}
