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

package kubermatic

import (
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// NamespaceReconciler ensures namespaces have labels for Gateway access.
// Namespaces with the "kubermatic.io/gateway-access=true" label can have
// HTTPRoutes that attach to the KKP Gateway.
func NamespaceReconciler(namespaceName string) reconciling.NamedNamespaceReconcilerFactory {
	return func() (string, reconciling.NamespaceReconciler) {
		return namespaceName, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}

			// Label namespaces that should have HTTPRoute access
			ns.Labels[common.GatewayAccessLabelKey] = "true"
			return ns, nil
		}
	}
}
