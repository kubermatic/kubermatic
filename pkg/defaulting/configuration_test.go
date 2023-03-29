/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package defaulting_test

import (
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/validation"
)

func TestDefaultConfigurationIsValid(t *testing.T) {
	errs := validation.ValidateKubermaticVersioningConfiguration(defaulting.DefaultKubernetesVersioning, nil)
	for _, err := range errs {
		t.Error(err)
	}
}

func TestDefaultResources(t *testing.T) {
	config := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			API: &kubermaticv1.KubermaticAPIConfiguration{},
		},
	}

	defaulted, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		t.Fatal(err)
	}

	if defaulted.Spec.API == nil {
		t.Fatal("Expected .API not to be nil anymore.")
	}

	if defaulted.Spec.API.Resources == nil {
		t.Fatal("Expected .API.Resources not to be nil anymore.")
	}

	if defaulted.Spec.API.Resources.Requests.Cpu().Cmp(*defaulting.DefaultAPIResources.Requests.Cpu()) != 0 {
		t.Fatalf("Expected %v, but got %v.", defaulting.DefaultAPIResources.Requests.Cpu(), defaulted.Spec.API.Resources.Requests.Cpu())
	}
}
