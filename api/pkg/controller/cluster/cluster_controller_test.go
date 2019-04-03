package cluster

import (
	"testing"

	kubermaticscheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const TestClusterName = "fqpcvnc6v"
const TestDC = "europe-west3-c"
const TestExternalURL = "dev.kubermatic.io"
const TestExternalPort = 30000

func newTestReconciler(t *testing.T, objects []runtime.Object) *Reconciler {
	if err := kubermaticscheme.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("failed to add kubermatic scheme: %v", err)
	}

	dcs := buildDatacenterMeta()

	dynamicClient := ctrlruntimefakeclient.NewFakeClient(objects...)
	r := &Reconciler{
		Client:                         dynamicClient,
		userClusterConnProvider:        nil,
		externalURL:                    TestExternalURL,
		dc:                             TestDC,
		dcs:                            dcs,
		enableEtcdDataCorruptionChecks: true,
		enableVPA:                      true,
		etcdDiskSize:                   resource.MustParse("5Gi"),
		nodeAccessNetwork:              "192.0.2.0/24",
	}

	return r
}

func buildDatacenterMeta() map[string]provider.DatacenterMeta {
	seedAlias := "alias-europe-west3-c"
	return map[string]provider.DatacenterMeta{
		"us-central1": {
			Location: "us-central",
			Country:  "US",
			Private:  false,
			IsSeed:   true,
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"us-central1-byo": {
			Location: "us-central",
			Country:  "US",
			Private:  false,
			Seed:     "us-central1",
			Spec: provider.DatacenterSpec{
				BringYourOwn: &provider.BringYourOwnSpec{},
			},
		},
		"private-do1": {
			Location: "US ",
			Seed:     "us-central1",
			Country:  "NL",
			Private:  true,
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
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"dns-override-do2": {
			Location:         "Amsterdam",
			Seed:             "us-central1",
			Country:          "NL",
			SeedDNSOverwrite: &seedAlias,
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams3",
				},
			},
		},
	}
}
