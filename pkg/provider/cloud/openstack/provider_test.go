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

package openstack

import (
	"context"
	"crypto/x509"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	ostesting "k8c.io/kubermatic/v2/pkg/provider/cloud/openstack/internal/testing"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/test"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestIgnoreRouterAlreadyHasPortInSubnetError(t *testing.T) {
	const subnetID = "123"
	testCases := []struct {
		name            string
		inErr           error
		expectReturnErr bool
	}{
		{
			name: "Matches",
			inErr: gophercloud.ErrDefault400{
				ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
					Body: []byte("Router already has a port on subnet " + subnetID),
				},
			},
			expectReturnErr: false,
		},
		{
			name: "Doesn't match",
			inErr: gophercloud.ErrDefault400{
				ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
					Body: []byte("Need moar permissions"),
				},
			},
			expectReturnErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ignoreRouterAlreadyHasPortInSubnetError(tc.inErr, subnetID); (err != nil) != tc.expectReturnErr {
				t.Errorf("expect return err: %t, but got err: %v", tc.expectReturnErr, err)
			}
		})
	}
}

func TestInitializeCloudProvider(t *testing.T) {
	tests := []struct {
		name         string
		dc           *kubermaticv1.DatacenterSpecOpenstack
		cluster      *kubermaticv1.Cluster
		resources    []ostesting.Resource
		wantErr      bool
		wantCluster  kubermaticv1.Cluster
		wantRequests map[ostesting.Request]int
	}{
		{
			name: "Create all",
			dc:   &kubermaticv1.DatacenterSpecOpenstack{},
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
			},
			resources: []ostesting.Resource{&ostesting.ExternalNetwork},
			wantCluster: kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
					Finalizers: []string{
						SecurityGroupCleanupFinalizer,
						NetworkCleanupFinalizer,
						SubnetCleanupFinalizer,
						RouterCleanupFinalizer,
						RouterSubnetLinkCleanupFinalizer,
					},
					Annotations: map[string]string{
						FloatingIPPoolIDAnnotation: ostesting.ExternalNetwork.ID,
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							SecurityGroups: "kubernetes-cluster-xyz",
							FloatingIPPool: "external-network",
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							RouterID:       ostesting.RouterID,
						},
					},
				},
			},
			wantErr: false,
			wantRequests: map[ostesting.Request]int{
				{Method: http.MethodPost, Path: ostesting.SecurityGroupsEndpoint}:                        1,
				{Method: http.MethodPost, Path: ostesting.NetworksEndpoint}:                              1,
				{Method: http.MethodPost, Path: ostesting.SubnetsEndpoint}:                               1,
				{Method: http.MethodPost, Path: ostesting.RoutersEndpoint}:                               1,
				{Method: http.MethodPut, Path: ostesting.AddRouterInterfaceEndpoint(ostesting.RouterID)}: 1,
			},
		},
		{
			name: "Create all - dual stack",
			dc:   &kubermaticv1.DatacenterSpecOpenstack{},
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16", "fd00::/64"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
			},
			resources: []ostesting.Resource{&ostesting.ExternalNetwork},
			wantCluster: kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
					Finalizers: []string{
						SecurityGroupCleanupFinalizer,
						NetworkCleanupFinalizer,
						SubnetCleanupFinalizer,
						IPv6SubnetCleanupFinalizer,
						RouterCleanupFinalizer,
						RouterSubnetLinkCleanupFinalizer,
						RouterIPv6SubnetLinkCleanupFinalizer,
					},
					Annotations: map[string]string{
						FloatingIPPoolIDAnnotation: ostesting.ExternalNetwork.ID,
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16", "fd00::/64"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							SecurityGroups: "kubernetes-cluster-xyz",
							FloatingIPPool: "external-network",
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							IPv6SubnetID:   ostesting.SubnetID,
							RouterID:       ostesting.RouterID,
						},
					},
				},
			},
			wantErr: false,
			wantRequests: map[ostesting.Request]int{
				{Method: http.MethodPost, Path: ostesting.SecurityGroupsEndpoint}:                        1,
				{Method: http.MethodPost, Path: ostesting.NetworksEndpoint}:                              1,
				{Method: http.MethodPost, Path: ostesting.SubnetsEndpoint}:                               2,
				{Method: http.MethodGet, Path: ostesting.SubnetPoolsEndpoint}:                            1,
				{Method: http.MethodPost, Path: ostesting.RoutersEndpoint}:                               1,
				{Method: http.MethodPut, Path: ostesting.AddRouterInterfaceEndpoint(ostesting.RouterID)}: 2,
			},
		},
		{
			name: "Create nothing",
			dc:   &kubermaticv1.DatacenterSpecOpenstack{},
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							SecurityGroups: "kubernetes-cluster-xyz",
							FloatingIPPool: "external-network",
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							RouterID:       ostesting.RouterID,
						},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
						CloudProviderInfrastructure: kubermaticv1.HealthStatusUp,
					},
				},
			},
			resources: []ostesting.Resource{
				&ostesting.ExternalNetwork,
				&ostesting.InternalNetwork,
				&ostesting.SubnetTest,
				&ostesting.Router{
					ID: ostesting.RouterID,
				},
			},
			wantCluster: kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
					Annotations: map[string]string{
						FloatingIPPoolIDAnnotation: ostesting.ExternalNetwork.ID,
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							SecurityGroups: "kubernetes-cluster-xyz",
							FloatingIPPool: "external-network",
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							RouterID:       ostesting.RouterID,
						},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
						CloudProviderInfrastructure: kubermaticv1.HealthStatusUp,
					},
				},
			},
			wantRequests: map[ostesting.Request]int{
				{Method: http.MethodPost, Path: ostesting.SecurityGroupsEndpoint}:                        0,
				{Method: http.MethodPost, Path: ostesting.NetworksEndpoint}:                              0,
				{Method: http.MethodPost, Path: ostesting.SubnetsEndpoint}:                               0,
				{Method: http.MethodPost, Path: ostesting.RoutersEndpoint}:                               0,
				{Method: http.MethodPut, Path: ostesting.AddRouterInterfaceEndpoint(ostesting.RouterID)}: 0,
			},
			wantErr: false,
		},
		{
			name: "Network provided",
			dc:   &kubermaticv1.DatacenterSpecOpenstack{},
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							Network: "kubernetes-cluster-xyz",
						},
					},
				},
			},
			resources: []ostesting.Resource{
				&ostesting.ExternalNetwork,
				&ostesting.InternalNetwork,
			},
			wantCluster: kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
					Finalizers: []string{
						SecurityGroupCleanupFinalizer,
						SubnetCleanupFinalizer,
						RouterCleanupFinalizer,
						RouterSubnetLinkCleanupFinalizer,
					},
					Annotations: map[string]string{
						FloatingIPPoolIDAnnotation: ostesting.ExternalNetwork.ID,
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							SecurityGroups: "kubernetes-cluster-xyz",
							FloatingIPPool: "external-network",
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							RouterID:       ostesting.RouterID,
						},
					},
				},
			},
			wantErr: false,
			wantRequests: map[ostesting.Request]int{
				{Method: http.MethodPost, Path: ostesting.SecurityGroupsEndpoint}:                        1,
				{Method: http.MethodPost, Path: ostesting.NetworksEndpoint}:                              0,
				{Method: http.MethodPost, Path: ostesting.SubnetsEndpoint}:                               1,
				{Method: http.MethodPost, Path: ostesting.RoutersEndpoint}:                               1,
				{Method: http.MethodPut, Path: ostesting.AddRouterInterfaceEndpoint(ostesting.RouterID)}: 1,
			},
		},
		{
			name: "Network and subnet provided",
			dc:   &kubermaticv1.DatacenterSpecOpenstack{},
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							Network:  "kubernetes-cluster-xyz",
							SubnetID: ostesting.SubnetID,
						},
					},
				},
			},
			resources: []ostesting.Resource{
				&ostesting.ExternalNetwork,
				&ostesting.InternalNetwork,
				&ostesting.SubnetTest,
			},
			wantCluster: kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
					Finalizers: []string{
						SecurityGroupCleanupFinalizer,
						RouterCleanupFinalizer,
						RouterSubnetLinkCleanupFinalizer,
					},
					Annotations: map[string]string{
						FloatingIPPoolIDAnnotation: ostesting.ExternalNetwork.ID,
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							SecurityGroups: "kubernetes-cluster-xyz",
							FloatingIPPool: "external-network",
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							RouterID:       ostesting.RouterID,
						},
					},
				},
			},
			wantErr: false,
			wantRequests: map[ostesting.Request]int{
				{Method: http.MethodPost, Path: ostesting.SecurityGroupsEndpoint}:                        1,
				{Method: http.MethodPost, Path: ostesting.NetworksEndpoint}:                              0,
				{Method: http.MethodPost, Path: ostesting.SubnetsEndpoint}:                               0,
				{Method: http.MethodPost, Path: ostesting.RoutersEndpoint}:                               1,
				{Method: http.MethodPut, Path: ostesting.AddRouterInterfaceEndpoint(ostesting.RouterID)}: 1,
			},
		},
		{
			name: "Specified network not found",
			dc:   &kubermaticv1.DatacenterSpecOpenstack{},
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							Network:  "kubernetes-cluster-xyz",
							SubnetID: ostesting.SubnetID,
						},
					},
				},
			},
			resources: []ostesting.Resource{
				&ostesting.ExternalNetwork,
			},
			wantCluster: kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
					Finalizers: []string{
						SecurityGroupCleanupFinalizer,
						RouterCleanupFinalizer,
						RouterSubnetLinkCleanupFinalizer,
					},
					Annotations: map[string]string{
						FloatingIPPoolIDAnnotation: ostesting.ExternalNetwork.ID,
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							SecurityGroups: "kubernetes-cluster-xyz",
							FloatingIPPool: "external-network",
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							RouterID:       ostesting.RouterID,
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ostesting.NewSimulator(t).Add(tt.resources...)
			defer s.TearDown()

			os := &Provider{
				dc: tt.dc,
				getClientFunc: func(ctx context.Context, cluster kubermaticv1.CloudSpec, dc *kubermaticv1.DatacenterSpecOpenstack, secretKeySelector provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
					sc := s.GetClient()
					return sc, nil
				},
			}
			c, err := os.InitializeCloudProvider(context.Background(), tt.cluster, (&fakeClusterUpdater{c: tt.cluster}).update)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.InitializeCloudProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// No need to proceed with further tests if an error is expected.
			if tt.wantErr {
				return
			}
			// We cannot guarantee order if finalizers serialized using sets and deep.Equal fails if order is different,
			// thus we test finalizers equality separately.
			if w, g := sets.New(tt.wantCluster.Finalizers...), sets.New(c.Finalizers...); !w.Equal(g) {
				t.Errorf("Want finalizers: %v, got: %v", w, g)
			} else {
				tt.wantCluster.Finalizers = nil
				c.Finalizers = nil
			}

			if !diff.SemanticallyEqual(tt.wantCluster, *c) {
				t.Errorf("Diff found between actual and expected cluster:\n%v", diff.ObjectDiff(tt.wantCluster, *c))
			}

			rc := s.GetRequestCounters()
			for req, e := range tt.wantRequests {
				if a := rc[req]; a != e {
					t.Errorf("Wanted %d requests %s, but got %d", e, req, a)
				}
			}
		})
	}
}

func TestGetCredentialsForCluster(t *testing.T) {
	tests := []struct {
		name    string
		spec    *kubermaticv1.OpenstackCloudSpec
		mock    func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
		want    *resources.OpenstackCredentials
		wantErr bool
	}{
		// there are 3 kinds of auth mode for openstack which are mutually exclusive
		//   * domain + token
		//   * ApplicationCredential (ApplicationCredentialID and ApplicationCredentialSecret)
		//   * domain + user (ie  Username, Password, (Project or Tenant) and (ProjectID or tenantID))
		{
			name:    "valid spec with values - auth with token",
			spec:    &kubermaticv1.OpenstackCloudSpec{Domain: "domain", Token: "the_token", UseToken: true},
			mock:    test.ShouldNotBeCalled,
			want:    &resources.OpenstackCredentials{Domain: "domain", Token: "the_token"},
			wantErr: false,
		},
		{
			name:    "valid spec with values - auth with user with project",
			spec:    &kubermaticv1.OpenstackCloudSpec{Domain: "domain", ApplicationCredentialID: "", Username: "user", Password: "pass", Project: "the_project", ProjectID: "the_project_id"},
			mock:    test.ShouldNotBeCalled,
			want:    &resources.OpenstackCredentials{Username: "user", Password: "pass", Project: "the_project", ProjectID: "the_project_id", Domain: "domain", ApplicationCredentialID: "", ApplicationCredentialSecret: ""},
			wantErr: false,
		},
		{
			name:    "valid spec with values - auth with applicationCredential",
			spec:    &kubermaticv1.OpenstackCloudSpec{ApplicationCredentialID: "app_id", ApplicationCredentialSecret: "app_secret"},
			mock:    test.ShouldNotBeCalled,
			want:    &resources.OpenstackCredentials{ApplicationCredentialID: "app_id", ApplicationCredentialSecret: "app_secret"},
			wantErr: false,
		},
		{
			name:    "valid spec with CredentialsReference - auth with token",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "the_token", resources.OpenstackApplicationCredentialID: ""}),
			want:    &resources.OpenstackCredentials{Domain: "domain-value", Token: "the_token"},
			wantErr: false,
		},
		{
			name:    "valid spec with CredentialsReference - auth with user with project",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "", resources.OpenstackApplicationCredentialID: "", resources.OpenstackProject: "the_project", resources.OpenstackProjectID: "the_project_id"}),
			want:    &resources.OpenstackCredentials{Username: "username-value", Password: "password-value", Project: "the_project", ProjectID: "the_project_id", Domain: "domain-value", ApplicationCredentialID: "", ApplicationCredentialSecret: ""},
			wantErr: false,
		},
		{
			name:    "valid spec with CredentialsReference - auth with user with tenant( when project not defined it should fallback to tenant)",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "", resources.OpenstackApplicationCredentialID: "", resources.OpenstackProject: test.MissingKeyErr(resources.OpenstackProject), resources.OpenstackProjectID: test.MissingKeyErr(resources.OpenstackProjectID), resources.OpenstackTenant: "the_tenant", resources.OpenstackTenantID: "the_tenant_id"}),
			want:    &resources.OpenstackCredentials{Username: "username-value", Password: "password-value", Project: "the_tenant", ProjectID: "the_tenant_id", Domain: "domain-value", ApplicationCredentialID: "", ApplicationCredentialSecret: ""},
			wantErr: false,
		},
		{
			name:    "valid spec with CredentialsReference - auth with applicationCredential",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: ""}),
			want:    &resources.OpenstackCredentials{ApplicationCredentialID: "applicationCredentialID-value", ApplicationCredentialSecret: "applicationCredentialSecret-value"},
			wantErr: false,
		},

		{
			name:    "invalid spec CredentialsReference - missing Domain",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "", resources.OpenstackApplicationCredentialID: "", resources.OpenstackDomain: test.MissingKeyErr(resources.OpenstackDomain)}),
			want:    &resources.OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing ApplicationCredentialSecret",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "", resources.OpenstackApplicationCredentialID: "applicationCredentialID-value", resources.OpenstackApplicationCredentialSecret: test.MissingKeyErr(resources.OpenstackApplicationCredentialSecret)}),
			want:    &resources.OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing username",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "", resources.OpenstackApplicationCredentialID: "", resources.OpenstackUsername: test.MissingKeyErr(resources.OpenstackUsername)}),
			want:    &resources.OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing password",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "", resources.OpenstackApplicationCredentialID: "", resources.OpenstackPassword: test.MissingKeyErr(resources.OpenstackPassword)}),
			want:    &resources.OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing Project and tenant",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "", resources.OpenstackApplicationCredentialID: "", resources.OpenstackProject: test.MissingKeyErr(resources.OpenstackProject), resources.OpenstackTenant: test.MissingKeyErr(resources.OpenstackTenant)}),
			want:    &resources.OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing ProjectID and tenantID",
			spec:    &kubermaticv1.OpenstackCloudSpec{UseToken: false, CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{resources.OpenstackToken: "", resources.OpenstackApplicationCredentialID: "", resources.OpenstackProjectID: test.MissingKeyErr(resources.OpenstackProjectID), resources.OpenstackTenantID: test.MissingKeyErr(resources.OpenstackTenantID)}),
			want:    &resources.OpenstackCredentials{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCredentialsForCluster(kubermaticv1.CloudSpec{Openstack: tt.spec}, tt.mock)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCredentialsForCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCredentialsForCluster() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcile(t *testing.T) {
	tests := []struct {
		name         string
		dc           *kubermaticv1.DatacenterSpecOpenstack
		cluster      *kubermaticv1.Cluster
		resources    []ostesting.Resource
		wantErr      bool
		wantCluster  kubermaticv1.Cluster
		wantRequests map[ostesting.Request]int
	}{
		{
			name: "Network and subnet provided",
			dc:   &kubermaticv1.DatacenterSpecOpenstack{},
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							FloatingIPPool: "foo",
						},
					},
				},
			},
			resources: []ostesting.Resource{
				&ostesting.ExternalNetwork,
				&ostesting.ExternalNetworkFoo,
				&ostesting.InternalNetwork,
				&ostesting.SubnetTest,
			},
			wantCluster: kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-xyz",
					Finalizers: []string{
						SecurityGroupCleanupFinalizer,
						RouterCleanupFinalizer,
						RouterSubnetLinkCleanupFinalizer,
					},
					Annotations: map[string]string{
						FloatingIPPoolIDAnnotation: ostesting.ExternalNetworkFoo.ID,
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
					},
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							SecurityGroups: "kubernetes-cluster-xyz",
							FloatingIPPool: "foo",
							Network:        "kubernetes-cluster-xyz",
							SubnetID:       ostesting.SubnetID,
							RouterID:       ostesting.RouterID,
						},
					},
				},
			},
			wantErr: false,
			wantRequests: map[ostesting.Request]int{
				{Method: http.MethodPost, Path: ostesting.SecurityGroupsEndpoint}:                        1,
				{Method: http.MethodPost, Path: ostesting.NetworksEndpoint}:                              0,
				{Method: http.MethodPost, Path: ostesting.SubnetsEndpoint}:                               0,
				{Method: http.MethodPost, Path: ostesting.RoutersEndpoint}:                               1,
				{Method: http.MethodPut, Path: ostesting.AddRouterInterfaceEndpoint(ostesting.RouterID)}: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := ostesting.NewSimulator(t).Add(tt.resources...)
			defer s.TearDown()
			os := &Provider{
				dc: tt.dc,
				getClientFunc: func(ctx context.Context, cluster kubermaticv1.CloudSpec, dc *kubermaticv1.DatacenterSpecOpenstack, secretKeySelector provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
					sc := s.GetClient()
					return sc, nil
				},
			}
			c, err := os.ReconcileCluster(ctx, tt.cluster, (&fakeClusterUpdater{c: tt.cluster}).update)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReconcileCluster() error = %v, wantErr %v", err, tt.wantErr)
			}
			// No need to proceed with further tests if an error is expected.
			if tt.wantErr {
				return
			}
			// We cannot guarantee order if finalizers serialized using sets and deep.Equal fails if order is different,
			// thus we test finalizers equality separately.
			if w, g := sets.New(tt.wantCluster.Finalizers...), sets.New(c.Finalizers...); !w.Equal(g) {
				t.Errorf("Want finalizers: %v, got: %v", w, g)
			} else {
				tt.wantCluster.Finalizers = nil
				c.Finalizers = nil
			}
			if !diff.SemanticallyEqual(tt.wantCluster, *c) {
				t.Errorf("Diff found between actual and expected cluster:\n%v", diff.ObjectDiff(tt.wantCluster, *c))
			}
			rc := s.GetRequestCounters()
			for req, e := range tt.wantRequests {
				if a := rc[req]; a != e {
					t.Errorf("Wanted %d requests %s, but got %d", e, req, a)
				}
			}
		})
	}
}

type fakeClusterUpdater struct {
	c *kubermaticv1.Cluster
}

func (f *fakeClusterUpdater) update(_ context.Context, _ string, updateFn func(c *kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error) {
	updateFn(f.c)
	return f.c, nil
}

func TestSplitString(t *testing.T) {
	testcases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "",
			expected: []string{},
		},
		{
			input:    "a",
			expected: []string{"a"},
		},
		{
			input:    "a b c",
			expected: []string{"a b c"},
		},
		{
			input:    "a,b , c ",
			expected: []string{"a", "b", "c"},
		},
		{
			input:    "a,",
			expected: []string{"a"},
		},
		{
			input:    ",,,",
			expected: []string{},
		},
		{
			input:    ",a,",
			expected: []string{"a"},
		},
		{
			input:    "a,b,a,b,a",
			expected: []string{"a", "b"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.input, func(t *testing.T) {
			split := splitString(tc.input)
			if !cmp.Equal(split, tc.expected) {
				t.Fatalf("Expected %v, got %v.", tc.expected, split)
			}
		})
	}
}
