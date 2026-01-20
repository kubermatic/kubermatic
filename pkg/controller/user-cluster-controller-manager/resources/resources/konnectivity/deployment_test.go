/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package konnectivity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func TestGetArgs(t *testing.T) {
	tests := []struct {
		name          string
		cluster       *kubermaticv1.Cluster
		keepaliveTime string
		expectedArgs  []string
	}{
		{
			name:          "default xfr-channel-size is set",
			cluster:       &kubermaticv1.Cluster{},
			keepaliveTime: "1m",
			expectedArgs: []string{
				"--logtostderr=true",
				"-v=3",
				"--sync-forever=true",
				"--ca-cert=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
				"--proxy-server-host=test",
				"--proxy-server-port=8132",
				"--admin-server-port=8133",
				"--health-server-port=8134",
				fmt.Sprintf("--service-account-token-path=/var/run/secrets/tokens/%s", resources.KonnectivityAgentToken),
				"--keepalive-time=1m",
				fmt.Sprintf("--xfr-channel-size=%d", kubermaticv1.DefaultKonnectivityXfrChannelSize),
			},
		},
		{
			name: "user xfr-channel-size overrides default without duplicates",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: []string{"--xfr-channel-size=300"},
						},
					},
				},
			},
			keepaliveTime: "1m",
			expectedArgs: []string{
				"--logtostderr=true",
				"-v=3",
				"--sync-forever=true",
				"--ca-cert=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
				"--proxy-server-host=test",
				"--proxy-server-port=8132",
				"--admin-server-port=8133",
				"--health-server-port=8134",
				fmt.Sprintf("--service-account-token-path=/var/run/secrets/tokens/%s", resources.KonnectivityAgentToken),
				"--keepalive-time=1m",
				"--xfr-channel-size=300",
			},
		},
		{
			name:          "custom keepalive time is used",
			cluster:       &kubermaticv1.Cluster{},
			keepaliveTime: "30s",
			expectedArgs: []string{
				"--logtostderr=true",
				"-v=3",
				"--sync-forever=true",
				"--ca-cert=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
				"--proxy-server-host=test",
				"--proxy-server-port=8132",
				"--admin-server-port=8133",
				"--health-server-port=8134",
				fmt.Sprintf("--service-account-token-path=/var/run/secrets/tokens/%s", resources.KonnectivityAgentToken),
				"--keepalive-time=30s",
				fmt.Sprintf("--xfr-channel-size=%d", kubermaticv1.DefaultKonnectivityXfrChannelSize),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := getArgs(tt.cluster, "test", tt.keepaliveTime, 8132)
			assert.ElementsMatch(t, tt.expectedArgs, args)
		})
	}
}
