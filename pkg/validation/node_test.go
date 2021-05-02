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

package validation_test

import (
	"errors"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/validation"
)

// EqualError reports whether errors a and b are considered equal.
// They're equal if both are nil, or both are not nil and a.Error() == b.Error().
func EqualError(a, b error) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Error() == b.Error())
}

func TestValidateCreateNodeSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name       string
		Cluster    *kubermaticv1.Cluster
		Spec       *apiv1.NodeSpec
		Datacenter *kubermaticv1.Datacenter
		Expected   error
	}{
		{
			"should pass validation when openstack provider not used",
			&kubermaticv1.Cluster{},
			&apiv1.NodeSpec{},
			&kubermaticv1.Datacenter{},
			nil,
		},
		{
			"should pass validation when floating ip not enforced and empty",
			&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							FloatingIPPool: "",
						},
					},
				},
			},
			&apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Openstack: &apiv1.OpenstackNodeSpec{UseFloatingIP: false},
				},
			},
			&kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{EnforceFloatingIP: false},
				},
			},
			nil,
		},
		{
			"should fail validation when floating ip enforced and empty",
			&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							FloatingIPPool: "",
						},
					},
				},
			},
			&apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Openstack: &apiv1.OpenstackNodeSpec{UseFloatingIP: false},
				},
			},
			&kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{EnforceFloatingIP: true},
				},
			},
			errors.New("no floating ip pool specified"),
		},
		{
			"should fail validation when floating ip not enforced, but required by user and empty",
			&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							FloatingIPPool: "",
						},
					},
				},
			},
			&apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Openstack: &apiv1.OpenstackNodeSpec{UseFloatingIP: true},
				},
			},
			&kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{EnforceFloatingIP: false},
				},
			},
			errors.New("no floating ip pool specified"),
		},
		{
			"should pass validation when floating ip not enforced, but filled and required by user",
			&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							FloatingIPPool: "ext-network",
						},
					},
				},
			},
			&apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Openstack: &apiv1.OpenstackNodeSpec{UseFloatingIP: true},
				},
			},
			&kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{EnforceFloatingIP: false},
				},
			},
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			err := validation.ValidateCreateNodeSpec(c.Cluster, c.Spec, c.Datacenter)

			if !EqualError(err, c.Expected) {
				t.Fatalf("expected err to be '%v', but got '%v'", c.Expected, err)
			}
		})
	}
}
