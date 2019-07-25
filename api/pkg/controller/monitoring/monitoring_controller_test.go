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
		seedGetter:           seed,
		nodeAccessNetwork:    "192.0.2.0/24",
		dockerPullConfigJSON: []byte{},
		features:             Features{},
	}

	return reconciler
}

func seed() (*kubermaticv1.Seed, error) {
	return &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name: "us-central1",
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				"us-central1-byo": {
					Location: "us-central",
					Country:  "US",
					Spec: kubermaticv1.DatacenterSpec{
						BringYourOwn: &kubermaticv1.DatacenterSpecBringYourOwn{},
					},
				},
				"private-do1": {
					Location: "US ",
					Country:  "NL",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "ams2",
						},
					},
				},
				"regular-do1": {
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
	}, nil
}
