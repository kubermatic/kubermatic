//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kubevirtnetworkcontroller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	clusterName    = "kv-netpol"
	datacenterName = "testdc"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                       string
		requestName                string
		expectedReconcileErrStatus metav1.StatusReason
		seedClient                 ctrlruntimeclient.Client
		seedGetter                 func() (*kubermaticv1.Seed, error)
		wantErr                    error
		wantNetPolicy              *networkingv1.NetworkPolicy
	}{
		{
			name:        "scenario 1: reconcile with network policy feature enabled in default allow mode",
			requestName: clusterName,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generator.GenDefaultProject(), generator.GenCluster(clusterName, clusterName, "projectName", time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
					cluster.Spec.Cloud.DatacenterName = datacenterName
				})).
				Build(),
			seedGetter: func() (*kubermaticv1.Seed, error) {
				return &kubermaticv1.Seed{
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							datacenterName: {
								Country:  "D",
								Location: "Hamburg",
								Spec: kubermaticv1.DatacenterSpec{
									Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
										NamespacedMode: &kubermaticv1.NamespacedMode{
											Enabled:   true,
											Namespace: "test",
										},
										ProviderNetwork: &kubermaticv1.ProviderNetwork{
											Name:                 "test",
											NetworkPolicyEnabled: true,
											NetworkPolicy: kubermaticv1.NetworkPolicy{
												Enabled: true,
												Mode:    kubermaticv1.NetworkPolicyModeAllow,
											},
											VPCs: []kubermaticv1.VPC{
												{
													Name: "dev",
													Subnets: []kubermaticv1.Subnet{
														{Name: "subnet-1"}, {Name: "subnet-2"},
													},
												},
											},
										},
									},
									RequiredEmails: []string{"example.com"},
								},
							},
						},
					},
				}, nil
			},
			wantNetPolicy: &networkingv1.NetworkPolicy{

				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"cluster.x-k8s.io/cluster-name": clusterName,
						},
					},
					Egress: []networkingv1.NetworkPolicyEgressRule{
						{
							Ports: []networkingv1.NetworkPolicyPort{},
							To: []networkingv1.NetworkPolicyPeer{
								{
									PodSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"cluster.x-k8s.io/cluster-name": clusterName,
										},
									},
								},
							},
						},
						{
							Ports: []networkingv1.NetworkPolicyPort{},
							To: []networkingv1.NetworkPolicyPeer{
								{
									IPBlock: &networkingv1.IPBlock{
										CIDR: "10.0.0.1/32",
									},
								},
							},
						},
						{
							Ports: []networkingv1.NetworkPolicyPort{},
							To: []networkingv1.NetworkPolicyPeer{
								{
									IPBlock: &networkingv1.IPBlock{
										CIDR: "10.0.1.1/32",
									},
								},
							},
						},
						{
							Ports: []networkingv1.NetworkPolicyPort{},
							To: []networkingv1.NetworkPolicyPeer{
								{
									IPBlock: &networkingv1.IPBlock{
										CIDR:   "0.0.0.0/0",
										Except: []string{"10.0.0.0/24", "10.0.1.0/24"},
									},
								},
							},
						},
					},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeEgress,
					},
				},
			},
		},
		{
			name:        "scenario 2: reconcile with network policy feature enabled in default deny mode",
			requestName: clusterName,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generator.GenDefaultProject(), generator.GenCluster(clusterName, clusterName, "projectName", time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
					cluster.Spec.Cloud.DatacenterName = datacenterName
				})).
				Build(),
			seedGetter: func() (*kubermaticv1.Seed, error) {
				return &kubermaticv1.Seed{
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							datacenterName: {
								Country:  "D",
								Location: "Hamburg",
								Spec: kubermaticv1.DatacenterSpec{
									Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
										NamespacedMode: &kubermaticv1.NamespacedMode{
											Enabled:   true,
											Namespace: "test",
										},
										DNSConfig: &corev1.PodDNSConfig{
											Nameservers: []string{"1.1.1.1"},
										},
										ProviderNetwork: &kubermaticv1.ProviderNetwork{
											Name:                 "test",
											NetworkPolicyEnabled: true,
											NetworkPolicy: kubermaticv1.NetworkPolicy{
												Enabled: true,
												Mode:    kubermaticv1.NetworkPolicyModeDeny,
											},
											VPCs: []kubermaticv1.VPC{
												{
													Name: "dev",
													Subnets: []kubermaticv1.Subnet{
														{Name: "subnet-1"}, {Name: "subnet-2"},
													},
												},
											},
										},
									},
									RequiredEmails: []string{"example.com"},
								},
							},
						},
					},
				}, nil
			},
			wantNetPolicy: &networkingv1.NetworkPolicy{

				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"cluster.x-k8s.io/cluster-name": clusterName,
						},
					},
					Egress: []networkingv1.NetworkPolicyEgressRule{
						{
							To: []networkingv1.NetworkPolicyPeer{
								{
									IPBlock: &networkingv1.IPBlock{
										CIDR: "35.194.142.199/32",
									},
								},
							},
						},
						{
							To: []networkingv1.NetworkPolicyPeer{
								{
									IPBlock: &networkingv1.IPBlock{
										CIDR: "1.1.1.1/32",
									},
								},
							},
						},
					},
					Ingress: []networkingv1.NetworkPolicyIngressRule{
						{
							From: []networkingv1.NetworkPolicyPeer{
								{
									IPBlock: &networkingv1.IPBlock{
										CIDR: "35.194.142.199/32",
									},
								},
							},
						},
					},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeEgress,
						networkingv1.PolicyTypeIngress,
					},
				},
			},
		},
		{
			name:        "scenario 3: reconcile with network policy feature disabled",
			requestName: clusterName,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generator.GenDefaultProject(), generator.GenCluster(clusterName, clusterName, "projectName", time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
					cluster.Spec.Cloud.DatacenterName = datacenterName
				})).
				Build(),
			seedGetter: func() (*kubermaticv1.Seed, error) {
				return &kubermaticv1.Seed{
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							datacenterName: {
								Country:  "D",
								Location: "Hamburg",
								Spec: kubermaticv1.DatacenterSpec{
									Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
										NamespacedMode: &kubermaticv1.NamespacedMode{
											Enabled:   true,
											Namespace: "test",
										},
										ProviderNetwork: &kubermaticv1.ProviderNetwork{
											Name:                 "test",
											NetworkPolicyEnabled: false,
											NetworkPolicy: kubermaticv1.NetworkPolicy{
												Enabled: false,
											},
											VPCs: []kubermaticv1.VPC{
												{
													Name: "dev",
													Subnets: []kubermaticv1.Subnet{
														{Name: "subnet-1"}, {Name: "subnet-2"},
													},
												},
											},
										},
									},
									RequiredEmails: []string{"example.com"},
								},
							},
						},
					},
				}, nil
			},
			wantNetPolicy: nil,
		},
		{
			name:        "scenario 4: reconcile with network policy feature enabled and namespaced mode disabled",
			requestName: clusterName,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generator.GenDefaultProject(), generator.GenCluster(clusterName, clusterName, "projectName", time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
					cluster.Spec.Cloud.DatacenterName = datacenterName
				})).
				Build(),
			seedGetter: func() (*kubermaticv1.Seed, error) {
				return &kubermaticv1.Seed{
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							datacenterName: {
								Country:  "D",
								Location: "Hamburg",
								Spec: kubermaticv1.DatacenterSpec{
									Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
										NamespacedMode: &kubermaticv1.NamespacedMode{
											Enabled: false,
										},
										ProviderNetwork: &kubermaticv1.ProviderNetwork{
											Name:                 "test",
											NetworkPolicyEnabled: true,
											NetworkPolicy: kubermaticv1.NetworkPolicy{
												Enabled: true,
												Mode:    kubermaticv1.NetworkPolicyModeAllow,
											},
											VPCs: []kubermaticv1.VPC{
												{
													Name: "dev",
													Subnets: []kubermaticv1.Subnet{
														{Name: "subnet-1"}, {Name: "subnet-2"},
													},
												},
											},
										},
									},
									RequiredEmails: []string{"example.com"},
								},
							},
						},
					},
				}, nil
			},
			wantNetPolicy: nil,
		},
		{
			name:        "scenario 5: reconcile with network policy feature enabled and no provider network specified",
			requestName: clusterName,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generator.GenDefaultProject(), generator.GenCluster(clusterName, clusterName, "projectName", time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
					cluster.Spec.Cloud.DatacenterName = datacenterName
				})).
				Build(),
			seedGetter: func() (*kubermaticv1.Seed, error) {
				return &kubermaticv1.Seed{
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							datacenterName: {
								Country:  "D",
								Location: "Hamburg",
								Spec: kubermaticv1.DatacenterSpec{
									Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
										NamespacedMode: &kubermaticv1.NamespacedMode{
											Enabled:   true,
											Namespace: "test",
										},
									},
									RequiredEmails: []string{"example.com"},
								},
							},
						},
					},
				}, nil
			},
			wantNetPolicy: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().WithObjects(generateSubnet("subnet-1", "10.0.0.0/24", "10.0.0.1"), generateSubnet("subnet-2", "10.0.1.0/24", "10.0.1.1")).Build()
			infraGetter := func(ctx context.Context, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			r := &Reconciler{
				log:         kubermaticlog.Logger,
				recorder:    &record.FakeRecorder{},
				Client:      tc.seedClient,
				seedGetter:  tc.seedGetter,
				infraGetter: infraGetter,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			_, err := r.Reconcile(ctx, request)
			if !errors.Is(tc.wantErr, err) {
				t.Fatalf("unexpected error occurred: %v, want error: %v", err, tc.wantErr)
			}

			currentNetPol := &networkingv1.NetworkPolicy{}
			if err := fakeClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("cluster-isolation-%s", clusterName), Namespace: "test"}, currentNetPol); err != nil {
				if tc.wantNetPolicy != nil {
					t.Fatalf("failed to fetch infra cluster network policies")
				}
				// when we do not expect any network policy to be created, we can continue with next test case
				return
			}
			if output := diff.ObjectDiff(tc.wantNetPolicy.Spec, currentNetPol.Spec); output != "" {
				t.Fatalf("expected network policies have different content: %v", output)
			}
		})
	}
}

func generateSubnet(name, cidr, gateway string) *unstructured.Unstructured {
	subnetUS := &unstructured.Unstructured{}
	subnetSpec := map[string]interface{}{
		"cidrBlock": cidr,
		"gateway":   gateway,
	}
	subnetMetadata := map[string]interface{}{
		"name": name,
	}
	subnetUS.SetUnstructuredContent(map[string]interface{}{
		"kind":       "Subnet",
		"apiVersion": "kubeovn.io/v1",
		"metadata":   subnetMetadata,
		"spec":       subnetSpec,
	})
	return subnetUS
}
