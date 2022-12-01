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

package nodedeploymentmigration

import (
	"os"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/generator"
)

var (
	datacenter = &kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			Alibaba: &kubermaticv1.DatacenterSpecAlibaba{
				Region: "test",
			},
			Hetzner: &kubermaticv1.DatacenterSpecHetzner{
				Datacenter: "dummy",
			},
		},
	}
)

func TestParseNodeDeployment(t *testing.T) {
	body, err := os.ReadFile("nodedeployment.json")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	cluster := generator.GenCluster("test", "test", "projectName", time.Now(), func(c *kubermaticv1.Cluster) {
		c.Spec.Cloud.ProviderName = string(kubermaticv1.AlibabaCloudProvider)
		c.Spec.Cloud.Fake = nil
		c.Spec.Cloud.Alibaba = &kubermaticv1.AlibabaCloudSpec{}
	})

	md, migrated, err := ParseNodeOrMachineDeployment(cluster, datacenter, string(body))
	if err != nil {
		t.Fatalf("Failed to parse body: %v", err)
	}

	if !migrated {
		t.Fatal("Expected migrated to be true, but got false.")
	}

	if expected := "trusting-wiles"; md.Name != expected {
		t.Fatalf("Expected name to be %q, but got %q", expected, md.Name)
	}
}

func TestParseMachineDeployment(t *testing.T) {
	body, err := os.ReadFile("machinedeployment.json")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	cluster := generator.GenCluster("test", "test", "projectName", time.Now(), func(c *kubermaticv1.Cluster) {
		c.Spec.Cloud.ProviderName = string(kubermaticv1.HetznerCloudProvider)
		c.Spec.Cloud.Fake = nil
		c.Spec.Cloud.Hetzner = &kubermaticv1.HetznerCloudSpec{}
	})

	md, migrated, err := ParseNodeOrMachineDeployment(cluster, datacenter, string(body))
	if err != nil {
		t.Fatalf("Failed to parse body: %v", err)
	}

	if migrated {
		t.Fatal("Expected migrated to be faalse, but got true.")
	}

	if expected := "elastic-yalow"; md.Name != expected {
		t.Fatalf("Expected name to be %q, but got %q", expected, md.Name)
	}
}
