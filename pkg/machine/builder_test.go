/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package machine

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"
)

func TestDetermineDatacenter(t *testing.T) {
	testCluster := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "foo",
				ProviderName:   string(kubermaticv1.AWSCloudProvider),
				AWS:            &kubermaticv1.AWSCloudSpec{},
			},
		},
	}

	fooAWSDC := &kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			AWS: &kubermaticv1.DatacenterSpecAWS{
				Region: "testregion",
			},
		},
	}

	testSeed := &kubermaticv1.Seed{
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				"foo": *fooAWSDC,
			},
		},
	}

	unrelatedSeed := &kubermaticv1.Seed{
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				"unrelated": {},
			},
		},
	}

	testcases := []struct {
		name           string
		datacenter     *kubermaticv1.Datacenter
		datacenterName string
		seed           *kubermaticv1.Seed
		cluster        *kubermaticv1.Cluster

		expectedDatacenter *kubermaticv1.Datacenter
		expectedError      bool
	}{
		// single test case for when nothing is given at all

		{
			name: "nothing given at all, should just result in nil",
		},

		// various valid combinations that the Builder can work with to deduce the target datacenter

		{
			name:               "explicit datacenter was configured",
			datacenter:         fooAWSDC,
			expectedDatacenter: fooAWSDC,
		},
		{
			name:               "explicit datacenter was configured and overrides anything else",
			datacenter:         fooAWSDC,
			seed:               unrelatedSeed, // should be ignored
			expectedDatacenter: fooAWSDC,
		},
		{
			name:               "cluster and seed given, should just return dc from cluster",
			cluster:            testCluster,
			seed:               testSeed,
			expectedDatacenter: fooAWSDC,
		},
		{
			name:               "datacenter name and seed given, should extract the datacenter from the seed",
			datacenterName:     "foo",
			seed:               testSeed,
			expectedDatacenter: fooAWSDC,
		},

		// all the invalid combinations that should result in an error

		{
			name:          "cluster but no seed, should return error",
			cluster:       testCluster,
			expectedError: true,
		},
		{
			name:          "seed but no cluster, should return error",
			seed:          testSeed,
			expectedError: true,
		},
		{
			name:           "datacenter name but no seed, should return error",
			datacenterName: "dcname",
			expectedError:  true,
		},
		{
			name:           "invalid datacenter name, should return error",
			datacenterName: "dcname",
			seed:           testSeed,
			expectedError:  true,
		},
		{
			name:           "explicit datacenter differs from datacenter in cluster, should return error",
			datacenterName: "dcname",
			cluster:        testCluster,
			expectedError:  true,
		},
		{
			name:          "cluster with invalid seed, should return error",
			cluster:       testCluster,
			seed:          unrelatedSeed,
			expectedError: true,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder().
				WithCluster(tt.cluster).
				WithSeed(tt.seed).
				WithDatacenter(tt.datacenter).
				WithDatacenterName(tt.datacenterName)

			determined, err := builder.determineDatacenter()
			if (err != nil) != tt.expectedError {
				t.Fatalf("ExpectedError=%v, but did not get the expected error, instead got: %v", tt.expectedError, err)
			}

			if tt.expectedDatacenter == nil && determined != nil {
				t.Fatalf("Expected no datacenter to be returned, but got %v.", *determined)
			}

			if changes := diff.ObjectDiff(&determined, tt.expectedDatacenter); changes != "" {
				t.Fatalf("got bad dc:\n%s", changes)
			}
		})
	}
}
