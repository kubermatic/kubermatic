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

package openshift

import (
	"flag"
	"net"
	"os"
	"testing"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/apis/plugin"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	testhelper "k8c.io/kubermatic/v2/pkg/test"

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
				PauseImage:            "",
			}

			userdata, err := p.UserData(req)
			if err != nil {
				t.Fatalf("failed to call p.Userdata: %v", err)
			}

			userdata = "# This file has been generated, DO NOT EDIT.\n\n" + userdata

			testhelper.CompareOutput(t, test.name, userdata, *update, ".yaml")
		})
	}
}
