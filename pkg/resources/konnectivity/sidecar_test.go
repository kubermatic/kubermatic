/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestKnpServerArgs(t *testing.T) {
	// getDefaultArgs defines a function that returns the default args that are always present.
	getDefaultArgs := func(serverCount int32, keepAliveTime string) []string {
		return []string{
			"--logtostderr=true",
			"-v=3",
			fmt.Sprintf("--cluster-key=/etc/kubernetes/pki/%s.key", resources.KonnectivityProxyTLSSecretName),
			fmt.Sprintf("--cluster-cert=/etc/kubernetes/pki/%s.crt", resources.KonnectivityProxyTLSSecretName),
			"--uds-name=/etc/kubernetes/konnectivity-server/konnectivity-server.socket",
			fmt.Sprintf("--kubeconfig=/etc/kubernetes/kubeconfig/%s", resources.KonnectivityServerConf),
			fmt.Sprintf("--server-count=%d", serverCount),
			"--mode=grpc",
			"--server-port=0",
			"--agent-port=8132",
			"--admin-port=8133",
			"--admin-bind-address=0.0.0.0",
			"--health-port=8134",
			"--agent-namespace=kube-system",
			fmt.Sprintf("--agent-service-account=%s", resources.KonnectivityServiceAccountName),
			"--delete-existing-uds-file=true",
			"--authentication-audience=system:konnectivity-server",
			"--proxy-strategies=default",
			fmt.Sprintf("--keepalive-time=%s", keepAliveTime),
		}
	}

	tests := []struct {
		name              string
		seed              *kubermaticv1.Seed
		cluster           *kubermaticv1.Cluster
		objects           []ctrlruntimeclient.Object
		serverCount       int32
		expectedKeepAlive string
		expectedArgs      []string
		expectedErrMsg    string
	}{
		{
			name:              "error when seed is nil",
			seed:              nil,
			cluster:           &kubermaticv1.Cluster{},
			expectedKeepAlive: kubermaticv1.DefaultKonnectivityKeepaliveTime,
			expectedErrMsg:    "invalid cluster template, seed cluster template is nil",
		},
		{
			name: "use konnectivity args from seed with default keepalive",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: []string{"--custom-arg1=value1", "--custom-arg2=value2"},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{},
				},
			},
			serverCount:       3,
			expectedKeepAlive: kubermaticv1.DefaultKonnectivityKeepaliveTime,
			expectedArgs: append(
				getDefaultArgs(3, kubermaticv1.DefaultKonnectivityKeepaliveTime),
				"--custom-arg1=value1", "--custom-arg2=value2",
			),
		},
		{
			name: "use konnectivity args from seed with custom keepalive",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: []string{"--custom-arg1=value1", "--custom-arg2=value2"},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							KeepaliveTime: "10s",
						},
					},
				},
			},
			serverCount:       3,
			expectedKeepAlive: "10s",
			expectedArgs: append(
				getDefaultArgs(3, "10s"),
				"--custom-arg1=value1", "--custom-arg2=value2",
			),
		},
		{
			name: "use only default args when seed has nil konnectivity args",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: nil,
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							KeepaliveTime: "15s",
						},
					},
				},
			},
			serverCount:       4,
			expectedKeepAlive: "15s",
			expectedArgs:      getDefaultArgs(4, "15s"),
		},
		{
			name: "use template konnectivity args",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: nil,
						},
					},
					DefaultClusterTemplate: "test-template",
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							KeepaliveTime: "20s",
						},
					},
				},
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.ClusterTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"scope": kubermaticv1.SeedTemplateScope,
						},
					},
					Spec: kubermaticv1.ClusterSpec{
						ComponentsOverride: kubermaticv1.ComponentSettings{
							KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
								Args: []string{"--template-arg1=value1", "--template-arg2=value2"},
							},
						},
					},
				},
			},
			serverCount:       5,
			expectedKeepAlive: "20s",
			expectedArgs: append(
				getDefaultArgs(5, "20s"),
				"--template-arg1=value1", "--template-arg2=value2",
			),
		},
		{
			name: "template not found error",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: nil,
						},
					},
					DefaultClusterTemplate: "non-existent-template",
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{},
				},
			},
			serverCount:       1,
			expectedKeepAlive: kubermaticv1.DefaultKonnectivityKeepaliveTime,
			expectedErrMsg:    "failed to get ClusterTemplate for konnectivity:",
		},
		{
			name: "empty keepalive time uses default",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					DefaultComponentSettings: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							Args: []string{"--custom-arg=value"},
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ComponentsOverride: kubermaticv1.ComponentSettings{
						KonnectivityProxy: kubermaticv1.KonnectivityProxySettings{
							KeepaliveTime: "",
						},
					},
				},
			},
			serverCount:       2,
			expectedKeepAlive: kubermaticv1.DefaultKonnectivityKeepaliveTime,
			expectedArgs: append(
				getDefaultArgs(2, kubermaticv1.DefaultKonnectivityKeepaliveTime),
				"--custom-arg=value",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []ctrlruntimeclient.Object
			if tt.seed != nil {
				objects = append(objects, tt.seed)
			}
			if tt.cluster != nil {
				objects = append(objects, tt.cluster)
			}
			objects = append(objects, tt.objects...)

			c := fake.NewClientBuilder().
				WithObjects(objects...).
				Build()

			data := resources.NewTemplateDataBuilder().
				WithClient(c).
				WithSeed(tt.seed).
				WithCluster(tt.cluster).
				Build()

			keepAliveTime := data.GetKonnectivityKeepAliveTime()
			assert.Equal(t, tt.expectedKeepAlive, keepAliveTime, "konnectivity keepalive time should match expected value")

			args, err := knpServerArgs(data, tt.serverCount)

			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedArgs, args)
			}
		})
	}
}
