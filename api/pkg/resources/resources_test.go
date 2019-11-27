package resources

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

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

func TestSetResourceRequirements(t *testing.T) {
	defaultResourceRequirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("20m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}

	tests := []struct {
		name                 string
		containers           []corev1.Container
		annotations          map[string]string
		defaultRequirements  map[string]*corev1.ResourceRequirements
		expectedRequirements map[string]*corev1.ResourceRequirements
	}{
		{
			name: "test with valid resource requirements json",
			containers: []corev1.Container{
				{
					Name: "test",
				},
			},
			annotations: map[string]string{
				kubermaticv1.UpdatedByVPALabelKey: `[{"name":"test","requires":{"limits":{"cpu":"1","memory":"512Mi"},"requests":{"cpu":"20m","memory":"64Mi"}}}]`,
			},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test": &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("64Mi"),
						corev1.ResourceCPU:    resource.MustParse("20m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// err := SetResourceRequirements(tc.containers, tc.defaultRequirements, tc.annotations)
			// if err != nil {
			// 	t.Fatalf("error: %v", err)
			// }
			// for _, container := range tc.containers {
			// 	fmt.Println(container.Resources.Limits.Memory().MilliValue())
			// 	fmt.Println(tc.expectedRequirements["test"].Limits.Memory().MilliValue())
			// 	if !reflect.DeepEqual(container.Resources, tc.expectedRequirements) {
			// 		t.Errorf("invalid resource requirements. expected: %#v, but got %#v", tc.expectedRequirements, container.Resources)
			// 	}
			// }
		})
	}
}
