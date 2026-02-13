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

package openstack

import (
	"bytes"
	"regexp"
	"testing"

	v1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/ini"
)

func checkINI(i ini.File, t *testing.T) {
	var buffer bytes.Buffer
	i.Render(&buffer)

	if !regexp.MustCompile(`\[LoadBalancer\]`).MatchString(buffer.String()) {
		t.Error("CloudConfig INI does not produce \"[LoadBalancer]\" section")
		t.Error(buffer.String())
	}

	if !regexp.MustCompile(`create-monitor = true`).MatchString(buffer.String()) {
		t.Error("CloudConfig INI does not produce values: \"create-monitor = true\"")
		t.Error(buffer.String())
	}

	if !regexp.MustCompile(`monitor-delay = 1`).MatchString(buffer.String()) {
		t.Error("CloudConfig INI does not produce values: \"monitor-delay = 1\"")
		t.Error(buffer.String())
	}

	if !regexp.MustCompile(`monitor-max-retries = 2`).MatchString(buffer.String()) {
		t.Error("CloudConfig INI does not produce values: \"monitor-max-retries = 2\"")
		t.Error(buffer.String())
	}

	if !regexp.MustCompile(`monitor-max-retries-down = 3`).MatchString(buffer.String()) {
		t.Error("CloudConfig INI does not produce values: \"monitor-max-retries-down = 3\"")
		t.Error(buffer.String())
	}

	if !regexp.MustCompile(`monitor-timeout = 4`).MatchString(buffer.String()) {
		t.Error("CloudConfig INI does not produce values: \"monitor-timeout = 4\"")
		t.Error(buffer.String())
	}
}

func TestLoadBalancerOptsToINIEmpty(t *testing.T) {
	cluster := &v1.Cluster{
		Spec: v1.ClusterSpec{
			Cloud: v1.CloudSpec{
				Openstack: &v1.OpenstackCloudSpec{
					UseOctavia: nil,
				},
			},
		},
	}
	datacenter := &v1.Datacenter{
		Spec: v1.DatacenterSpec{
			Openstack: &v1.DatacenterSpecOpenstack{
				LoadBalancerClasses: nil,
				LoadBalancerMonitor: &v1.DatacenterSpecOpenstackLoadBalancerMonitor{},
			},
		},
	}
	credentials := resources.Credentials{}

	cc := ForCluster(cluster, datacenter, credentials)
	i := ini.New()
	section := i.Section("LoadBalancer", "")
	cc.LoadBalancer.toINI(section)
}

func TestLoadBalancerOptsToINIDatacenterOnly(t *testing.T) {
	cluster := &v1.Cluster{
		Spec: v1.ClusterSpec{
			Cloud: v1.CloudSpec{
				Openstack: &v1.OpenstackCloudSpec{
					UseOctavia: nil,
				},
			},
		},
	}
	datacenter := &v1.Datacenter{
		Spec: v1.DatacenterSpec{
			Openstack: &v1.DatacenterSpecOpenstack{
				LoadBalancerClasses: nil,
				LoadBalancerMonitor: &v1.DatacenterSpecOpenstackLoadBalancerMonitor{
					Create:         true,
					Delay:          1,
					MaxRetries:     2,
					MaxRetriesDown: 3,
					Timeout:        4,
				},
			},
		},
	}
	credentials := resources.Credentials{}

	cc := ForCluster(cluster, datacenter, credentials)
	i := ini.New()
	section := i.Section("LoadBalancer", "")
	cc.LoadBalancer.toINI(section)

	checkINI(i, t)
}

func TestLoadBalancerOptsToINIClusterOnly(t *testing.T) {
	cluster := &v1.Cluster{
		Spec: v1.ClusterSpec{
			Cloud: v1.CloudSpec{
				Openstack: &v1.OpenstackCloudSpec{
					UseOctavia: nil,
					LoadBalancerMonitor: &v1.OpenstackCloudLoadBalancerMonitorSpec{
						Create:         true,
						Delay:          1,
						MaxRetries:     2,
						MaxRetriesDown: 3,
						Timeout:        4,
					},
				},
			},
		},
	}
	datacenter := &v1.Datacenter{
		Spec: v1.DatacenterSpec{
			Openstack: &v1.DatacenterSpecOpenstack{
				LoadBalancerClasses: nil,
				LoadBalancerMonitor: &v1.DatacenterSpecOpenstackLoadBalancerMonitor{},
			},
		},
	}
	credentials := resources.Credentials{}

	cc := ForCluster(cluster, datacenter, credentials)
	i := ini.New()
	section := i.Section("LoadBalancer", "")
	cc.LoadBalancer.toINI(section)

	checkINI(i, t)
}

func TestLoadBalancerOptsToINIClusterOverridingDatacenter(t *testing.T) {
	cluster := &v1.Cluster{
		Spec: v1.ClusterSpec{
			Cloud: v1.CloudSpec{
				Openstack: &v1.OpenstackCloudSpec{
					UseOctavia: nil,
					LoadBalancerMonitor: &v1.OpenstackCloudLoadBalancerMonitorSpec{
						Create:         true,
						Delay:          1,
						MaxRetries:     2,
						MaxRetriesDown: 3,
						Timeout:        4,
					},
				},
			},
		},
	}
	datacenter := &v1.Datacenter{
		Spec: v1.DatacenterSpec{
			Openstack: &v1.DatacenterSpecOpenstack{
				LoadBalancerClasses: nil,
				LoadBalancerMonitor: &v1.DatacenterSpecOpenstackLoadBalancerMonitor{
					Create:         false,
					Delay:          11,
					MaxRetries:     12,
					MaxRetriesDown: 13,
					Timeout:        14,
				},
			},
		},
	}
	credentials := resources.Credentials{}

	cc := ForCluster(cluster, datacenter, credentials)
	i := ini.New()
	section := i.Section("LoadBalancer", "")
	cc.LoadBalancer.toINI(section)

	checkINI(i, t)
}
