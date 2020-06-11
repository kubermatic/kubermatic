package openshift

import (
	"flag"
	"net"
	"os"
	"testing"

	testhelper "github.com/kubermatic/kubermatic/pkg/test"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/apis/plugin"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var update = flag.Bool("update", false, "Update test fixtures")

func TestUserdataGeneration(t *testing.T) {
	tests := []struct {
		name              string
		cloudProviderName string
		spec              clusterv1alpha1.MachineSpec
	}{
		{
			name:              "aws-v4.1.7",
			cloudProviderName: string(providerconfig.CloudProviderAWS),
			spec: clusterv1alpha1.MachineSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-aws-node",
				},
				ProviderSpec: clusterv1alpha1.ProviderSpec{
					Value: &runtime.RawExtension{Raw: []byte("{}")},
				},
				Versions: clusterv1alpha1.MachineVersionInfo{
					Kubelet: "v4.1.7",
				},
			},
		},
		{
			name:              "aws-v4.2.0",
			cloudProviderName: string(providerconfig.CloudProviderAWS),
			spec: clusterv1alpha1.MachineSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-aws-node",
				},
				ProviderSpec: clusterv1alpha1.ProviderSpec{
					Value: &runtime.RawExtension{Raw: []byte("{}")},
				},
				Versions: clusterv1alpha1.MachineVersionInfo{
					Kubelet: "v4.2.0",
				},
			},
		},
		{
			name:              "vsphere-v4.1.7",
			cloudProviderName: string(providerconfig.CloudProviderVsphere),
			spec: clusterv1alpha1.MachineSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-vsphere-node",
				},
				ProviderSpec: clusterv1alpha1.ProviderSpec{
					Value: &runtime.RawExtension{Raw: []byte("{}")},
				},
				Versions: clusterv1alpha1.MachineVersionInfo{
					Kubelet: "v4.1.7",
				},
			},
		},
	}

	if err := os.Setenv(DockerCFGEnvKey, `{"registry": {"user": "user", "pass": "pss"}}`); err != nil {
		t.Fatalf("failed to set dockercfg: %v", err)
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

			req := plugin.UserDataRequest{
				MachineSpec:       test.spec,
				Kubeconfig:        kubeconfig,
				CloudConfig:       "dummy-cloud-config",
				CloudProviderName: test.cloudProviderName,
				DNSIPs: []net.IP{
					net.ParseIP("8.8.8.8"),
					net.ParseIP("1.2.3.4"),
				},
				ExternalCloudProvider: false,
				HTTPProxy:             "",
				NoProxy:               "",
				InsecureRegistries:    []string{},
				PauseImage:            "",
			}

			userdata, err := p.UserData(req)
			if err != nil {
				t.Fatalf("failed to call p.Userdata: %v", err)
			}

			testhelper.CompareOutput(t, test.name, userdata, *update, ".yaml")
		})
	}
}
