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

package seedmla

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateConfigurationDefaultsSharedIAPGatewayValues(t *testing.T) {
	doc := seedMLAIAPValues(t)

	monitoringStack := &MonitoringStack{}
	_, gotDoc, failures := monitoringStack.ValidateConfiguration(seedMLAExternalGatewayConfig(), doc, stack.DeployOptions{MLAIncludeIap: true}, logrus.New())
	if len(failures) > 0 {
		t.Fatalf("expected no validation failures, got %v", failures)
	}

	assertIAPGatewayValues(t, gotDoc, "platform-gateway", "networking")
}

func TestValidateConfigurationPreservesSeparateSeedIAPGatewayValues(t *testing.T) {
	doc := seedMLAIAPValues(t)

	monitoringStack := &MonitoringStack{}
	_, gotDoc, failures := monitoringStack.ValidateConfiguration(seedMLAExternalGatewayConfig(), doc, stack.DeployOptions{MLAIncludeIap: true, SeparateSeed: true}, logrus.New())
	if len(failures) > 0 {
		t.Fatalf("expected no validation failures, got %v", failures)
	}

	assertIAPGatewayValues(t, gotDoc, "kubermatic", "kubermatic")
}

func seedMLAIAPValues(t *testing.T) *yamled.Document {
	t.Helper()

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
    prometheus:
      encryption_key: "1234567890123456"
`))
	if err != nil {
		t.Fatalf("failed to load Helm values: %v", err)
	}

	return doc
}

func seedMLAExternalGatewayConfig() *kubermaticv1.KubermaticConfiguration {
	return &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic"},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}
}

func assertIAPGatewayValues(t *testing.T, doc *yamled.Document, wantName, wantNamespace string) {
	t.Helper()

	gatewayName, _ := doc.GetString(yamled.Path{"httpRoute", "gatewayName"})
	if gatewayName != wantName {
		t.Fatalf("expected seed IAP Gateway name %s, got %s", wantName, gatewayName)
	}

	gatewayNamespace, _ := doc.GetString(yamled.Path{"httpRoute", "gatewayNamespace"})
	if gatewayNamespace != wantNamespace {
		t.Fatalf("expected seed IAP Gateway namespace %s, got %s", wantNamespace, gatewayNamespace)
	}
}
