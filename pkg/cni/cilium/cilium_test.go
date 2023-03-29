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

package cilium

import (
	"encoding/json"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/cni"
	"k8c.io/kubermatic/v3/pkg/resources"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

var testCluster = &kubermaticv1.Cluster{
	Spec: kubermaticv1.ClusterSpec{
		CNIPlugin: &kubermaticv1.CNIPluginSettings{
			Type:    kubermaticv1.CNIPluginTypeCilium,
			Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCilium),
		},
		ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
			Pods: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"192.168.0.0/24"},
			},
			NodeCIDRMaskSizeIPv4: pointer.Int32(16),
			ProxyMode:            resources.EBPFProxyMode,
		},
		ComponentsOverride: kubermaticv1.ComponentSettings{
			Apiserver: kubermaticv1.APIServerSettings{
				NodePortRange: "30000-31777",
			},
		},
	},
	Status: kubermaticv1.ClusterStatus{
		Address: kubermaticv1.ClusterAddress{
			ExternalName: "cluster.kubermatic.test",
			Port:         6443,
		},
	},
}

func TestGetCiliumAppInstallOverrideValues(t *testing.T) {
	testCases := []struct {
		name              string
		cluster           *kubermaticv1.Cluster
		overwriteRegistry string
		expectedValues    string
	}{
		{
			name:              "default values",
			cluster:           testCluster,
			overwriteRegistry: "",
			expectedValues:    `{"cni":{"exclusive":false},"ipam":{"operator":{"clusterPoolIPv4MaskSize":"16","clusterPoolIPv4PodCIDR":"192.168.0.0/24"}},"k8sServiceHost":"cluster.kubermatic.test","k8sServicePort":6443,"kubeProxyReplacement":"strict","nodePort":{"range":"30000,31777"},"operator":{"securityContext":{"seccompProfile":{"type":"RuntimeDefault"}}}}`,
		},
		{
			name:              "default values with overwrite registry",
			cluster:           testCluster,
			overwriteRegistry: "myregistry.io",
			expectedValues:    `{"certgen":{"image":{"repository":"myregistry.io/cilium/certgen","useDigest":false}},"cni":{"exclusive":false},"hubble":{"relay":{"image":{"repository":"myregistry.io/cilium/hubble-relay","useDigest":false}},"ui":{"backend":{"image":{"repository":"myregistry.io/cilium/hubble-ui-backend","useDigest":false}},"frontend":{"image":{"repository":"myregistry.io/cilium/hubble-ui","useDigest":false}}}},"image":{"repository":"myregistry.io/cilium/cilium","useDigest":false},"ipam":{"operator":{"clusterPoolIPv4MaskSize":"16","clusterPoolIPv4PodCIDR":"192.168.0.0/24"}},"k8sServiceHost":"cluster.kubermatic.test","k8sServicePort":6443,"kubeProxyReplacement":"strict","nodePort":{"range":"30000,31777"},"operator":{"image":{"repository":"myregistry.io/cilium/operator","useDigest":false},"securityContext":{"seccompProfile":{"type":"RuntimeDefault"}}}}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			values := GetAppInstallOverrideValues(testCluster, testCase.overwriteRegistry)
			rawValues, _ := json.Marshal(values)
			if string(rawValues) != testCase.expectedValues {
				t.Fatalf("values '%s' do not match expected values '%s'", rawValues, testCase.expectedValues)
			}
		})
	}
}

func TestValidateCiliumValuesUpdate(t *testing.T) {
	testCases := []struct {
		name               string
		expectedError      string
		testValuesModifier func(map[string]any)
	}{
		{
			name: "No value change",
			testValuesModifier: func(values map[string]any) {
				// NOOP
			},
			expectedError: "[]",
		},
		{
			name: "Allowed values change",
			testValuesModifier: func(values map[string]any) {
				values["allowed"] = "true"
			},
			expectedError: "[]",
		},
		{
			name: "Modified immutable value",
			testValuesModifier: func(values map[string]any) {
				values["ipv6"] = map[string]any{"enabled": "true"}
			},
			expectedError: "[spec.values.ipv6: Invalid value: map[string]interface {}{\"enabled\":\"true\"}: value is immutable]",
		},
		{
			name: "Removed immutable value",
			testValuesModifier: func(values map[string]any) {
				delete(values, "ipam")
			},
			expectedError: "[spec.values.ipam: Invalid value: \"null\": value is immutable]",
		},
		{
			name: "Change mandatory value",
			testValuesModifier: func(values map[string]any) {
				values["kubeProxyReplacement"] = "disabled"
			},
			expectedError: "[]",
		},
		{
			name: "Remove mandatory value",
			testValuesModifier: func(values map[string]any) {
				delete(values, "kubeProxyReplacement")
			},
			expectedError: "[spec.values.kubeProxyReplacement: Not found: \"null\"]",
		},
		{
			name: "Remove nested immutable value",
			testValuesModifier: func(values map[string]any) {
				o := values["operator"].(map[string]any)
				delete(o, "securityContext")
			},
			expectedError: "[spec.values.operator.securityContext: Invalid value: \"null\": value is immutable]",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// get values reconciled by KKP
			oldValues := GetAppInstallOverrideValues(testCluster, "")

			// copy oldValues to newValues to modify
			newValues := make(map[string]any)
			rawValues, _ := json.Marshal(oldValues)
			err := json.Unmarshal(rawValues, &newValues)
			if err != nil {
				t.Fatalf("values unmarshalling failed: %s", err)
			}

			// modify newValues
			testCase.testValuesModifier(newValues)

			// validate the update and check for expected errors
			errList := ValidateValuesUpdate(newValues, oldValues, field.NewPath("spec").Child("values"))
			if fmt.Sprint(errList) != testCase.expectedError {
				if testCase.expectedError == "[]" {
					testCase.expectedError = "nil"
				}
				t.Fatalf("expected error to be %s but got %v", testCase.expectedError, errList)
			}
		})
	}
}
