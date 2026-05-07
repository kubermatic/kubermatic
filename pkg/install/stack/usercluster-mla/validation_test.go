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

package userclustermla

import (
	"strings"
	"testing"

	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

func TestValidateHelmValuesPreservesIAPGatewayValues(t *testing.T) {
	doc, err := yamled.Load(strings.NewReader(`
migrateGatewayAPI: true
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
iap:
  deployments:
    grafana:
      encryption_key: "1234567890123456"
    alertmanager:
      encryption_key: "1234567890123456"
`))
	if err != nil {
		t.Fatalf("failed to load Helm values: %v", err)
	}

	failures := validateHelmValues(doc, stack.DeployOptions{MLAIncludeIap: true})
	if len(failures) > 0 {
		t.Fatalf("expected no validation failures, got %v", failures)
	}

	gatewayName, _ := doc.GetString(yamled.Path{"httpRoute", "gatewayName"})
	if gatewayName != "kubermatic" {
		t.Fatalf("expected usercluster MLA IAP Gateway name to remain kubermatic, got %s", gatewayName)
	}

	gatewayNamespace, _ := doc.GetString(yamled.Path{"httpRoute", "gatewayNamespace"})
	if gatewayNamespace != "kubermatic" {
		t.Fatalf("expected usercluster MLA IAP Gateway namespace to remain kubermatic, got %s", gatewayNamespace)
	}
}
