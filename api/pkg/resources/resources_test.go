package resources

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

func TestInClusterApiserverIP(t *testing.T) {
	testCases := []struct {
		name           string
		cidr           string
		expectedResult string
	}{
		{
			name:           "Parse /24",
			cidr:           "10.10.10.0/24",
			expectedResult: "10.10.10.1",
		},
		{
			name:           "Parse /20",
			cidr:           "10.240.20.0/20",
			expectedResult: "10.240.16.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cluster := &kubermaticv1.Cluster{}
			cluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{tc.cidr}

			result, err := InClusterApiserverIP(cluster)
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			if result.String() != tc.expectedResult {
				t.Errorf("wrong result, expected: %s, result: %s", tc.expectedResult, result.String())
			}
		})
	}
}

func TestUserClusterDNSResolverIP(t *testing.T) {
	testCases := []struct {
		name           string
		cidr           string
		expectedResult string
	}{
		{
			name:           "Parse /24",
			cidr:           "10.10.10.0/24",
			expectedResult: "10.10.10.10",
		},
		{
			name:           "Parse /20",
			cidr:           "10.240.20.0/20",
			expectedResult: "10.240.16.10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cluster := &kubermaticv1.Cluster{}
			cluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{tc.cidr}

			result, err := UserClusterDNSResolverIP(cluster)
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			if result != tc.expectedResult {
				t.Errorf("wrong result, expected: %s, result: %s", tc.expectedResult, result)
			}
		})
	}
}
