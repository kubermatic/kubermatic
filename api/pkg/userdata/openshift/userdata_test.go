package openshift

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"testing"

	"github.com/pmezard/go-difflib/difflib"

	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var update = flag.Bool("update", false, "Update test fixtures")

func TestUserdataGeneration(t *testing.T) {
	tests := []struct {
		name              string
		cloudProviderName string
		spec              clusterv1alpha1.MachineSpec
	}{
		{
			name:              "simple-aws",
			cloudProviderName: "aws",
			spec: clusterv1alpha1.MachineSpec{
				ProviderSpec: clusterv1alpha1.ProviderSpec{
					Value: &runtime.RawExtension{Raw: []byte("{}")},
				},
				Versions: clusterv1alpha1.MachineVersionInfo{
					Kubelet: "1.2.3",
				},
			},
		},
	}

	for _, test := range tests {
		kubeconfig := &clientcmdapi.Config{
			Clusters: map[string]*clientcmdapi.Cluster{
				"": {
					Server:                   "https://server:443",
					CertificateAuthorityData: []byte("my-cert"),
				},
			},
			AuthInfos: map[string]*clientcmdapi.AuthInfo{
				"": {
					Token: "my-token",
				},
			},
		}

		t.Run(test.name, func(t *testing.T) {
			p := Provider{}
			userdata, err := p.UserData(test.spec,
				kubeconfig,
				"dummy-cloud-config",
				test.cloudProviderName,
				[]net.IP{net.ParseIP("8.8.8.8")},
				false)
			if err != nil {
				t.Fatalf("failed to call p.Userdata: %v", err)
			}

			fixturePath := fmt.Sprintf("testdata/%s.yaml", test.name)
			if *update {
				if err := ioutil.WriteFile(fixturePath, []byte(userdata), 0644); err != nil {
					t.Fatalf("failed to update fixture %q: %v", fixturePath, err)
				}
			}

			expected, err := ioutil.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("failed to read fixture %q: %v", fixturePath, err)
			}

			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(expected)),
				B:        difflib.SplitLines(userdata),
				FromFile: "Fixture",
				ToFile:   "Current",
				Context:  3,
			}
			diffStr, err := difflib.GetUnifiedDiffString(diff)
			if err != nil {
				t.Fatalf("failed to generate diff: %v", err)
			}

			if string(expected) != userdata {
				t.Errorf("Userdata file does not match the fixture anymore. You can update the fixture by running the tests with the `-update` flag. Diff: %s", diffStr)
			}
		})
	}
}
