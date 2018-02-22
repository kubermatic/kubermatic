package cluster

import (
	"errors"
	"testing"

	"github.com/go-test/deep"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidatingCheckDatacenter(t *testing.T) {
	tests := []struct {
		name    string
		cluster *kubermaticv1.Cluster
		ret     error
	}{
		{
			name: "not existing seed dc",
			ret:  errors.New("could not find given seed datacenter \"does-not-exist\""),
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
				Spec: kubermaticv1.ClusterSpec{
					SeedDatacenterName: "does-not-exist",
					Cloud: &kubermaticv1.CloudSpec{
						DatacenterName: "regular-do1",
					},
				},
			},
		},
		{
			name: "not existing node dc",
			ret:  errors.New("could not find node datacenter \"does-not-exist\""),
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
				Spec: kubermaticv1.ClusterSpec{
					SeedDatacenterName: "us-central1",
					Cloud: &kubermaticv1.CloudSpec{
						DatacenterName: "does-not-exist",
					},
				},
			},
		},
		{
			name: "node dc as seed dc",
			ret:  errors.New("given datacenter \"us-central1\" is not configured as a node datacenter"),
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
				Spec: kubermaticv1.ClusterSpec{
					SeedDatacenterName: "us-central1",
					Cloud: &kubermaticv1.CloudSpec{
						DatacenterName: "us-central1",
					},
				},
			},
		},
		{
			name: "seed dc as node dc",
			ret:  errors.New("given seed datacenter \"regular-do1\" is not configured as a seed datacenter"),
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
				Spec: kubermaticv1.ClusterSpec{
					SeedDatacenterName: "regular-do1",
					Cloud: &kubermaticv1.CloudSpec{
						DatacenterName: "regular-do1",
					},
				},
			},
		},
		{
			name: "successful",
			ret:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
				Spec: kubermaticv1.ClusterSpec{
					SeedDatacenterName: "us-central1",
					Cloud: &kubermaticv1.CloudSpec{
						DatacenterName: "regular-do1",
					},
				},
			},
		},
		{
			name: "successful - BringYourOwn",
			ret:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
				Spec: kubermaticv1.ClusterSpec{
					SeedDatacenterName: "us-central1",
					Cloud: &kubermaticv1.CloudSpec{
						DatacenterName: "us-central1-byo",
						BringYourOwn:   &kubermaticv1.BringYourOwnCloudSpec{},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
			err := f.controller.validateDatacenter(test.cluster)
			if diff := deep.Equal(err, test.ret); diff != nil {
				t.Errorf("expected to get %v instead got: %v", test.ret, err)
			}
		})
	}
}
