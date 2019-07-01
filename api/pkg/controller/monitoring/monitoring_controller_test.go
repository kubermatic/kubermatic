package monitoring

import (
	"testing"

	kubermaticscheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

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

	dcs := buildDatacenterMeta()

	dynamicClient := ctrlruntimefakeclient.NewFakeClient(objects...)
	reconciler := &Reconciler{
		Client:               dynamicClient,
		dcs:                  dcs,
		dc:                   TestDC,
		nodeAccessNetwork:    "192.0.2.0/24",
		dockerPullConfigJSON: []byte{},
		features:             Features{},
	}

	return reconciler
}

func buildDatacenterMeta() map[string]provider.DatacenterMeta {
	seedAlias := "alias-europe-west3-c"
	return map[string]provider.DatacenterMeta{
		"us-central1": {
			Location: "us-central",
			Country:  "US",
			IsSeed:   true,
			Spec: kubermaticv1.DatacenterSpec{
				Digitalocean: &kubermaticv1.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"us-central1-byo": {
			Location: "us-central",
			Country:  "US",
			Seed:     "us-central1",
			Spec: kubermaticv1.DatacenterSpec{
				BringYourOwn: &kubermaticv1.BringYourOwnSpec{},
			},
		},
		"private-do1": {
			Location: "US ",
			Seed:     "us-central1",
			Country:  "NL",
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"regular-do1": {
			Location: "Amsterdam",
			Seed:     "us-central1",
			Country:  "NL",
			Spec: kubermaticv1.DatacenterSpec{
				Digitalocean: &kubermaticv1.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"dns-override-do2": {
			Location:         "Amsterdam",
			Seed:             "us-central1",
			Country:          "NL",
			SeedDNSOverwrite: &seedAlias,
			Spec: kubermaticv1.DatacenterSpec{
				Digitalocean: &kubermaticv1.DigitaloceanSpec{
					Region: "ams3",
				},
			},
		},
	}
}
