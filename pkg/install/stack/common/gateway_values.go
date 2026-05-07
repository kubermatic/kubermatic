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
)

// DefaultMasterHTTPRouteGatewayValues defaults master-cluster chart HTTPRoute parentRefs to
// spec.ingress.gateway.externalGateway when Gateway API migration is enabled and
// the chart still points at the built-in kubermatic Gateway. Do not use this for
// seed-scoped charts: in separate seed setups, the master externalGateway does
// not describe the seed Gateway.
func DefaultMasterHTTPRouteGatewayValues(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document, logger logrus.FieldLogger) bool {
	if config == nil {
		return false
	}

	gatewayConfig := config.Spec.Ingress.Gateway
	if !gatewayConfig.UsesExternalGateway() {
		return false
	}

	if migrateGatewayAPI, _ := helmValues.GetBool(yamled.Path{"migrateGatewayAPI"}); !migrateGatewayAPI {
		return false
	}

	gatewayNamePath := yamled.Path{"httpRoute", "gatewayName"}
	gatewayNamespacePath := yamled.Path{"httpRoute", "gatewayNamespace"}

	externalGatewayName := gatewayConfig.ExternalGateway.Name
	externalGatewayNamespace := gatewayConfig.ExternalGatewayNamespace(config.Namespace)

	currentGatewayName, _ := helmValues.GetString(gatewayNamePath)
	currentGatewayNamespace, _ := helmValues.GetString(gatewayNamespacePath)

	pointsAtDefaultGateway := (currentGatewayName == "" || currentGatewayName == defaulting.DefaultGatewayName) &&
		(currentGatewayNamespace == "" || currentGatewayNamespace == config.Namespace)
	if !pointsAtDefaultGateway {
		return false
	}

	logger.Warnf("Helm values: %s/%s point to the default Gateway, setting them to spec.ingress.gateway.externalGateway", gatewayNamePath.String(), gatewayNamespacePath.String())
	helmValues.Set(gatewayNamePath, externalGatewayName)
	helmValues.Set(gatewayNamespacePath, externalGatewayNamespace)

	return true
}
