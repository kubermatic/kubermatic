package monitoring

import (
	"testing"

	kubermaticscheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	TestDC = "regular-do1"
)

func newTestReconciler(t *testing.T, objects []runtime.Object) *Reconciler {
	if err := kubermaticscheme.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("failed to add kubermatic scheme: %v", err)
	}

	dynamicClient := ctrlruntimefakeclient.NewFakeClient(objects...)
	reconciler := &Reconciler{
		Client:               dynamicClient,
		dc:                   buildDatacenter(),
		nodeAccessNetwork:    "192.0.2.0/24",
		dockerPullConfigJSON: []byte{},
		features:             Features{},
	}

	return reconciler
}

func buildDatacenter() *kubermaticv1.SeedDatacenter {
	return &kubermaticv1.SeedDatacenter{
		ObjectMeta: metav1.ObjectMeta{
			Name: "us-central1",
		},
		Spec: kubermaticv1.SeedDatacenterSpec{
			NodeLocations: map[string]kubermaticv1.NodeLocation{
				"us-central1-byo": kubermaticv1.NodeLocation{
					Location: "us-central",
					Country:  "US",
					Spec: kubermaticv1.DatacenterSpec{
						BringYourOwn: &kubermaticv1.DatacenterSpecBringYourOwn{},
					},
				},
				"private-do1": kubermaticv1.NodeLocation{
					Location: "US ",
					Country:  "NL",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
				"regular-do1": kubermaticv1.NodeLocation{
					Location: "Amsterdam",
					Country:  "NL",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
			},
		},
	}
}
