/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package networkpolicy

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	slim_metav1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cilium/cilium/pkg/policy/api"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
)

var apiServerLabels = map[string]string{"app": "apiserver"}

const (
	CiliumSeedApiserverAllow = "cilium-seed-apiserver-allow"
)

func SeedApiServerCiliumClusterwideNetworkPolicyReconciler() reconciling.NamedCiliumClusterwideNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.CiliumClusterwideNetworkPolicyReconciler) {
		return CiliumSeedApiserverAllow, func(np *ciliumv2.CiliumClusterwideNetworkPolicy) (*ciliumv2.CiliumClusterwideNetworkPolicy, error) {
			egressRule := []api.EgressRule{
				{
					EgressCommonRule: api.EgressCommonRule{
						ToEntities: api.EntitySlice{
							api.EntityKubeAPIServer,
						},
					},
				},
			}
			np.Spec = &api.Rule{
				EndpointSelector: api.EndpointSelector{
					LabelSelector: &slim_metav1.LabelSelector{
						MatchLabels: apiServerLabels,
					},
				},
				Egress: egressRule,
			}
			return np, nil
		}
	}
}

func CiliumCRDExists(ctx context.Context, client client.Client) bool {
	crd := apiextensionsv1.CustomResourceDefinition{}
	key := types.NamespacedName{Name: "ciliumclusterwidenetworkpolicies.cilium.io"}

	crdExists := true
	if err := client.Get(ctx, key, &crd); err != nil {
		if !apierrors.IsNotFound(err) {
			return false
		}
		crdExists = false
	}

	return crdExists
}
