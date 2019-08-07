package seed

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	// We call this in init because even thought it is possible to register the same
	// scheme multiple times it is an unprotected concurrent map access and these tests
	// are very good at making that panic
	if err := kubermaticv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}

func TestValidate(t *testing.T) {

	testCases := []struct {
		name             string
		existingSeeds    map[string]*kubermaticv1.Seed
		seedToValidate   *kubermaticv1.Seed
		existingClusters []runtime.Object
		isDelete         bool
		errExpected      bool
	}{
		{
			name:           "Happy path, no error",
			seedToValidate: &kubermaticv1.Seed{},
		},
		{
			name: "DatacenterName already in use, error",
			existingSeeds: map[string]*kubermaticv1.Seed{
				"existing-seed": &kubermaticv1.Seed{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"in-use": kubermaticv1.Datacenter{},
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"in-use": kubermaticv1.Datacenter{},
					},
				},
			},
			errExpected: true,
		},
		{
			name:           "Removed datacenter still in use, error",
			seedToValidate: &kubermaticv1.Seed{},
			existingClusters: []runtime.Object{&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						DatacenterName: "keep-me",
					},
				},
			}},
			errExpected: true,
		},
		{
			name:             "Deletion when there are still clusters, error",
			seedToValidate:   &kubermaticv1.Seed{},
			existingClusters: []runtime.Object{&kubermaticv1.Cluster{}},
			isDelete:         true,
			errExpected:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sv := &seedValidator{
				listOpts: &ctrlruntimeclient.ListOptions{},
			}

			err := sv.validate(tc.seedToValidate,
				fakectrlruntimeclient.NewFakeClient(tc.existingClusters...),
				tc.existingSeeds, tc.isDelete)

			if (err != nil) != tc.errExpected {
				t.Fatalf("expected err: %t, but got err: %v", tc.errExpected, err)
			}
		})
	}

}
