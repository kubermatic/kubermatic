//go:build dualstack

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

package dualstack

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	alibabatypes "k8c.io/machine-controller/sdk/cloudprovider/alibaba"
	awstypes "k8c.io/machine-controller/sdk/cloudprovider/aws"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"

	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	alibabaCredentials      jig.AlibabaCredentials
	awsCredentials          jig.AWSCredentials
	azureCredentials        jig.AzureCredentials
	digitaloceanCredentials jig.DigitaloceanCredentials
	equinixMetalCredentials jig.EquinixMetalCredentials
	gcpCredentials          jig.GCPCredentials
	hetznerCredentials      jig.HetznerCredentials
	openstackCredentials    jig.OpenstackCredentials
	vsphereCredentials      jig.VSphereCredentials
)

func addRHELSubscriptionInfo(osSpec interface{}) interface{} {
	rhelSpec, ok := osSpec.(rhel.Config)
	if !ok {
		panic(fmt.Sprintf("Expected RHEL os Spec, but got %T", osSpec))
	}

	rhelSpec.RHELSubscriptionManagerUser = os.Getenv("OS_RHEL_USERNAME")
	rhelSpec.RHELSubscriptionManagerPassword = os.Getenv("OS_RHEL_PASSWORD")
	rhelSpec.RHSMOfflineToken = os.Getenv("OS_RHEL_OFFLINE_TOKEN")

	return rhelSpec
}

type CreateJigFunc func(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig

func newAlibabaTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	jig := jig.NewAlibabaCluster(seedClient, log, alibabaCredentials, 1)
	jig.MachineJig.WithCloudProviderSpecPatch(func(providerSpec interface{}) interface{} {
		alibabaSpec := providerSpec.(alibabatypes.RawConfig)
		alibabaSpec.VSwitchID = providerconfig.ConfigVarString{Value: "vsw-gw876svgsv52bk0c95krn"}
		return alibabaSpec
	})

	return jig
}

func newAWSTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	jig := jig.NewAWSCluster(seedClient, log, awsCredentials, 1, nil)
	jig.ClusterJig.WithPatch(func(c *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
		return c
	})
	jig.MachineJig.WithCloudProviderSpecPatch(func(providerSpec interface{}) interface{} {
		awsSpec := providerSpec.(awstypes.RawConfig)
		awsSpec.AssignPublicIP = ptr.To(true)
		return awsSpec
	})

	return jig
}

func newAzureTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	return jig.NewAzureCluster(seedClient, log, azureCredentials, 1)
}

func newDigitaloceanTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	return jig.NewDigitaloceanCluster(seedClient, log, digitaloceanCredentials, 1)
}

func newEquinixMetalTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	return jig.NewEquinixMetalCluster(seedClient, log, equinixMetalCredentials, 1)
}

func newGCPTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	jig := jig.NewGCPCluster(seedClient, log, gcpCredentials, 1)
	jig.ClusterJig.WithPatch(func(c *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
		c.Cloud.GCP.Network = "global/networks/dualstack"
		c.Cloud.GCP.Subnetwork = "projects/kubermatic-dev/regions/europe-west3/subnetworks/dualstack-europe-west3"
		return c
	})

	return jig
}

func newHetznerTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	return jig.NewHetznerCluster(seedClient, log, hetznerCredentials, 1)
}

func newOpenstackTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	jig := jig.NewOpenstackCluster(seedClient, log, openstackCredentials, 1)
	jig.ClusterJig.WithPatch(func(c *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
		c.Cloud.Openstack.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
			CIDRBlocks: []string{"0.0.0.0/0", "::/0"},
		}
		c.Cloud.Openstack.SecurityGroups = "default"
		return c
	})

	return jig
}

func newVSphereTestJig(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *jig.TestJig {
	return jig.NewVSphereCluster(seedClient, log, vsphereCredentials, 1)
}
