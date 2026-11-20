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

package kubermaticmaster

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/stack"
)

func TestDeployCertManagerSkips(t *testing.T) {
	withIssuer := &kubermaticv1.KubermaticConfiguration{}
	withIssuer.Spec.Ingress.CertificateIssuer.Name = "letsencrypt-prod"

	headless := withIssuer.DeepCopy()
	headless.Spec.FeatureGates = map[string]bool{features.HeadlessInstallation: true}

	// None of these cases may touch the cluster or Helm, so nil clients are fine.
	testCases := map[string]stack.DeployOptions{
		"skip-charts":           {KubermaticConfiguration: withIssuer, SkipCharts: []string{CertManagerChartName}},
		"headless installation": {KubermaticConfiguration: headless},
		"no certificate issuer": {KubermaticConfiguration: &kubermaticv1.KubermaticConfiguration{}},
	}

	for name, opt := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := DeployCertManager(context.Background(), logrus.NewEntry(logrus.New()), nil, nil, opt); err != nil {
				t.Fatalf("DeployCertManager() = %v, want nil", err)
			}
		})
	}
}
