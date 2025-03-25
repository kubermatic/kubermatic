/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package aws

import (
	"bytes"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/ini"
)

type CloudConfig struct {
	Global GlobalOpts
}

func ForCluster(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter) CloudConfig {
	cc := CloudConfig{
		// Dummy AZ, so that K8S can extract the region from it.
		// https://github.com/kubernetes/kubernetes/blob/v1.15.0/staging/src/k8s.io/legacy-cloud-providers/aws/aws.go#L1199
		// https://github.com/kubernetes/kubernetes/blob/v1.15.0/staging/src/k8s.io/legacy-cloud-providers/aws/aws.go#L1174
		Global: GlobalOpts{
			Zone:                        dc.Spec.AWS.Region + "x",
			VPC:                         cluster.Spec.Cloud.AWS.VPCID,
			KubernetesClusterID:         cluster.Name,
			DisableSecurityGroupIngress: false,
			RouteTableID:                cluster.Spec.Cloud.AWS.RouteTableID,
			RoleARN:                     cluster.Spec.Cloud.AWS.ControlPlaneRoleARN,
		},
	}

	if cluster.IsDualStack() {
		cc.Global.NodeIPFamilies = []string{"ipv4", "ipv6"}
	}

	return cc
}

func (c *CloudConfig) String() (string, error) {
	out := ini.New()

	global := out.Section("global", "")
	c.Global.toINI(global)

	buf := &bytes.Buffer{}
	if err := out.Render(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type GlobalOpts struct {
	Zone                        string
	VPC                         string
	SubnetID                    string
	RouteTableID                string
	RoleARN                     string
	KubernetesClusterID         string
	ElbSecurityGroup            string
	DisableSecurityGroupIngress bool
	NodeIPFamilies              []string
}

func (o *GlobalOpts) toINI(section ini.Section) {
	section.AddStringKey("Zone", o.Zone)
	section.AddStringKey("VPC", o.VPC)
	section.AddStringKey("SubnetID", o.SubnetID)
	section.AddStringKey("RouteTableID", o.RouteTableID)
	section.AddStringKey("RoleARN", o.RoleARN)
	section.AddStringKey("KubernetesClusterID", o.KubernetesClusterID)
	section.AddBoolKey("DisableSecurityGroupIngress", o.DisableSecurityGroupIngress)
	section.AddStringKey("ElbSecurityGroup", o.ElbSecurityGroup)

	for _, family := range o.NodeIPFamilies {
		section.AddStringKey("NodeIPFamilies", family)
	}
}
