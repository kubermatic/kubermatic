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

package provider

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
)

func TestKubevirtConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewKubevirtConfig().
		WithVCPUs("2").
		WithMemory("memory").
		WithPrimaryDiskOSImage("image").
		WithPrimaryDiskSize("size").
		WithPrimaryDiskStorageClassName("className").
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.VirtualMachine.Template.Memory.Value != "memory" {
		t.Fatal("Builder did not apply VM template memory to the config.")
	}
}

type kubevirtTestcase struct {
	baseTestcase[kubevirt.RawConfig, kubermaticv1.DatacenterSpecKubevirt]
}

func (tt *kubevirtTestcase) Run(cluster *kubermaticv1.Cluster) (*kubevirt.RawConfig, error) {
	return CompleteKubevirtProviderSpec(tt.Input(), cluster, tt.datacenter)
}

var _ testcase[kubevirt.RawConfig] = &kubevirtTestcase{}

func TestCompleteKubevirtProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecKubevirt{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteKubevirtProviderSpec(nil, cluster, datacenter); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Kubevirt = &kubermaticv1.KubevirtCloudSpec{}
		if _, err := CompleteKubevirtProviderSpec(nil, cluster, datacenter); err != nil {
			t.Errorf("Cluster is now matching Kubevirt, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.KubevirtCloudProvider),
		Kubevirt:     &kubermaticv1.KubevirtCloudSpec{},
	})
	goodCluster.Status.NamespaceName = "testns"

	defaultMachine := NewKubevirtConfig()
	goodMachine := cloneBuilder(defaultMachine).WithPrimaryDiskOSImage("testns/")

	testcases := []testcase[kubevirt.RawConfig]{
		&kubevirtTestcase{
			baseTestcase: baseTestcase[kubevirt.RawConfig, kubermaticv1.DatacenterSpecKubevirt]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecKubevirt{
					DNSPolicy: "test-dnspolicy",
				},
				expected: cloneBuilder(goodMachine).WithDNSPolicy("test-dnspolicy").WithClusterName(goodCluster.Name),
			},
		},
		&kubevirtTestcase{
			baseTestcase: baseTestcase[kubevirt.RawConfig, kubermaticv1.DatacenterSpecKubevirt]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecKubevirt{
					DNSPolicy: "test-dnspolicy",
				},
				inputSpec: cloneBuilder(defaultMachine).WithDNSPolicy("keep-me-kubevirt"),
				expected:  cloneBuilder(goodMachine).WithDNSPolicy("keep-me-kubevirt").WithClusterName(goodCluster.Name),
			},
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}

const (
	osImageURL              = "http://image-repo-http-server.kube-system.svc.cluster.local/images"
	osImageDataVolumeName   = "dv-name"
	osImageDataVolumeNsName = "namespace/dv-name"
	ns                      = "namespace"
)

func TestExtractKubeVirtOsImageURLOrDataVolumeNsName(t *testing.T) {
	type args struct {
		namespace string
		osImage   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "URL",
			args: args{
				osImage:   osImageURL,
				namespace: ns,
			},
			want: osImageURL, // no change, URL kept
		},
		{
			name: "DataVolumeName prefixed with namespace",
			args: args{
				osImage:   osImageDataVolumeNsName,
				namespace: ns,
			},
			want: osImageDataVolumeNsName, // no change, already right format
		},
		{
			name: "DataVolumeName",
			args: args{
				osImage:   osImageDataVolumeName,
				namespace: ns,
			},
			want: osImageDataVolumeNsName, // add the namespace/ prefix
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractKubeVirtOsImageURLOrDataVolumeNsName(tt.args.namespace, tt.args.osImage); got != tt.want {
				t.Errorf("extractKubeVirtOsImageURLOrDataVolumeNsName() = %v, want %v", got, tt.want)
			}
		})
	}
}
