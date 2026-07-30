//go:build e2e

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

package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	// ciliumExtraConfigExcludeLocalAddressKey mirrors the unexported constant in the
	// cni-application-installation-controller (controller.go).
	ciliumExtraConfigExcludeLocalAddressKey = "exclude-local-address"

	// nodeLocalDNSCIDR is the value the controller injects for the exclude-local-address
	// key: resources.NodeLocalDNSCacheAddress (the bare IP at pkg/resources/resources.go)
	// plus a /32 suffix appended at controller.go.
	nodeLocalDNSCIDR = resources.NodeLocalDNSCacheAddress + "/32"

	// cniAppInstallationName is the name the cni-application-installation-controller gives
	// the Cilium ApplicationInstallation in the user cluster's kube-system namespace: it is
	// the CNI plugin type string ("cilium"), a fixed lowercase name owned by the controller.
	cniAppInstallationName = "cilium"
	cniAppInstallationNs   = "kube-system"
)

// TestCiliumNodeLocalDNSExcludeLocalAddress guards PR #15996. The
// cni-application-installation-controller injects the NodeLocalDNS cache address into the
// Cilium ApplicationInstallation's extraConfig["exclude-local-address"] (reconciled into the
// user-cluster apiserver) when CNI=Cilium and NodeLocalDNSCacheEnabled is not explicitly
// false, so NodeLocalDNS is not treated as a host identity and egress network policy does
// not drop DNS.
//
// The base BYO cluster is created with NodeLocalDNSCacheEnabled at its default (true), so
// this is a positive-only assertion: the CIDR must be present in the reconciled values.
// The field is immutable post-create (KKP cluster-validation webhook rejects changes), so
// it cannot be toggled on a shared cluster; a negative-branch test would need a second
// cluster created with the field disabled, which is not worth the extra cluster. Reverting
// the #15996 fix makes the CIDR disappear and this assertion fails.
//
// This is a Tier C1 test: it asserts the reconciled Helm values in the user-cluster
// ApplicationInstallation, NOT that Cilium Pods are running. The BYO base cluster has no
// worker Nodes, so a "Cilium pods running" assertion would be Tier C2 and out of scope.
func TestCiliumNodeLocalDNSExcludeLocalAddress(t *testing.T) {
	uc := requireUserCluster(t)

	feature := features.New("Cilium NodeLocalDNS exclude-local-address is reconciled").
		Assess("extraConfig.exclude-local-address contains the NodeLocalDNS cache CIDR", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)

			g.Eventually(func(g gomega.Gomega) {
				extraConfig := getExtraConfig(g, getCiliumAppValues(g, ctx, uc.user))
				g.Expect(extraConfig).To(gomega.HaveKey(ciliumExtraConfigExcludeLocalAddressKey),
					"expected extraConfig.%s to be present for a Cilium cluster with NodeLocalDNSCacheEnabled", ciliumExtraConfigExcludeLocalAddressKey)
				raw := fmt.Sprintf("%v", extraConfig[ciliumExtraConfigExcludeLocalAddressKey])
				g.Expect(raw).To(gomega.ContainSubstring(nodeLocalDNSCIDR),
					"expected exclude-local-address %q to contain %q", raw, nodeLocalDNSCIDR)
			}).WithContext(ctx).WithTimeout(5 * time.Minute).WithPolling(2 * time.Second).Should(gomega.Succeed())

			return ctx
		}).
		Feature()

	testEnv.Test(t, feature)
}

// getExtraConfig returns the extraConfig sub-map from the parsed Cilium values. The
// controller writes exclude-local-address under extraConfig (controller.go), nested one
// level below the top-level Helm values.
func getExtraConfig(g gomega.Gomega, values map[string]any) map[string]any {
	ec, ok := values["extraConfig"].(map[string]any)
	g.Expect(ok).To(gomega.BeTrue(), "expected top-level values to contain an extraConfig map")
	return ec
}

// getCiliumAppValues returns the parsed Helm values of the user-cluster CNI
// ApplicationInstallation. The controller writes values to Spec.ValuesBlock (YAML) and
// resets Spec.Values to {}, so GetParsedValues (which parses ValuesBlock when Values is
// empty) is the correct read path.
func getCiliumAppValues(g gomega.Gomega, ctx context.Context, userClient ctrlruntimeclient.Client) map[string]any {
	app := &appskubermaticv1.ApplicationInstallation{}
	g.Expect(userClient.Get(ctx, types.NamespacedName{Name: cniAppInstallationName, Namespace: cniAppInstallationNs}, app)).To(gomega.Succeed(),
		"failed to get CNI ApplicationInstallation %s/%s", cniAppInstallationNs, cniAppInstallationName)
	values, err := app.Spec.GetParsedValues()
	g.Expect(err).To(gomega.Succeed(), "failed to parse CNI ApplicationInstallation values")
	return values
}

// requireUserCluster returns the shared base cluster, failing the test if it was not
// provisioned (i.e. the run omitted -with-user-cluster).
func requireUserCluster(t *testing.T) *clusterFixture {
	t.Helper()
	if userCluster == nil {
		t.Fatalf("test requires a user cluster but -with-user-cluster was not set")
	}
	return userCluster
}
