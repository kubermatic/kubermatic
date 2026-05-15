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

package common

import (
	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	"k8s.io/apimachinery/pkg/types"
)

var (
	httpRouteGatewayNamePath      = yamled.Path{"httpRoute", "gatewayName"}
	httpRouteGatewayNamespacePath = yamled.Path{"httpRoute", "gatewayNamespace"}
	migrateGatewayAPIPath         = yamled.Path{"migrateGatewayAPI"}
)

// DefaultMasterHTTPRouteGatewayValues defaults master-cluster chart HTTPRoute parentRefs to
// spec.ingress.gateway.externalGateway when Gateway API migration is enabled and
// the chart still points at the built-in kubermatic Gateway. Do not use this for
// seed-scoped charts: in separate seed setups, the master externalGateway does
// not describe the seed Gateway.
func DefaultMasterHTTPRouteGatewayValues(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document, logger logrus.FieldLogger) bool {
	if !shouldUseExternalMasterHTTPRouteGateway(config, helmValues) {
		return false
	}

	externalGateway := masterExternalGatewayReference(config)

	logger.Infof("Helm values: %s/%s point to the default Gateway, setting them to spec.ingress.gateway.externalGateway", httpRouteGatewayNamePath.String(), httpRouteGatewayNamespacePath.String())
	helmValues.Set(httpRouteGatewayNamePath, externalGateway.Name)
	helmValues.Set(httpRouteGatewayNamespacePath, externalGateway.Namespace)

	return true
}

// MasterHTTPRouteGatewayReference returns the effective Gateway reference for
// master-scoped HTTPRoute charts. It mirrors DefaultMasterHTTPRouteGatewayValues
// without mutating Helm values, so installer cleanup code can wait for the same
// Gateway that the chart will attach to.
func MasterHTTPRouteGatewayReference(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document) types.NamespacedName {
	if shouldUseExternalMasterHTTPRouteGateway(config, helmValues) {
		return masterExternalGatewayReference(config)
	}

	return currentMasterHTTPRouteGatewayReference(config, helmValues)
}

func shouldUseExternalMasterHTTPRouteGateway(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document) bool {
	if config == nil || helmValues == nil {
		return false
	}

	gatewayConfig := config.Spec.Ingress.Gateway
	if !gatewayConfig.UsesExternalGateway() {
		return false
	}

	if migrateGatewayAPI, _ := helmValues.GetBool(migrateGatewayAPIPath); !migrateGatewayAPI {
		return false
	}

	currentGateway := currentMasterHTTPRouteGatewayReference(config, helmValues)
	return currentGateway.Name == defaulting.DefaultGatewayName && currentGateway.Namespace == config.Namespace
}

func currentMasterHTTPRouteGatewayReference(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document) types.NamespacedName {
	key := types.NamespacedName{
		Name: defaulting.DefaultGatewayName,
	}
	if config != nil {
		key.Namespace = config.Namespace
	}
	if helmValues == nil {
		return key
	}

	if name, _ := helmValues.GetString(httpRouteGatewayNamePath); name != "" {
		key.Name = name
	}
	if namespace, _ := helmValues.GetString(httpRouteGatewayNamespacePath); namespace != "" {
		key.Namespace = namespace
	}

	return key
}

func masterExternalGatewayReference(config *kubermaticv1.KubermaticConfiguration) types.NamespacedName {
	gatewayConfig := config.Spec.Ingress.Gateway
	return types.NamespacedName{
		Name:      gatewayConfig.ExternalGateway.Name,
		Namespace: gatewayConfig.ExternalGatewayNamespace(config.Namespace),
	}
}
