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

package validation

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateKeyConfiguration(t *testing.T) {
	testCases := []struct {
		name        string
		config      *kubermaticv1.KeyConfiguration
		expectedErr bool
	}{
		{
			name:   "no configuration at all is valid",
			config: nil,
		},
		{
			name:   "an empty configuration is valid",
			config: &kubermaticv1.KeyConfiguration{},
		},
		{
			name: "RSA with a supported size",
			config: &kubermaticv1.KeyConfiguration{
				Certificates: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmRSA, RSAKeySize: 4096},
			},
		},
		{
			name: "ECDSA with a supported curve",
			config: &kubermaticv1.KeyConfiguration{
				ServiceAccountKey: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA, ECDSACurve: kubermaticv1.ECDSACurveP384},
			},
		},
		{
			name: "an RSA key size on an ECDSA key is rejected",
			config: &kubermaticv1.KeyConfiguration{
				Certificates: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA, RSAKeySize: 4096},
			},
			expectedErr: true,
		},
		{
			name: "a curve on an RSA key is rejected",
			config: &kubermaticv1.KeyConfiguration{
				Certificates: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmRSA, ECDSACurve: kubermaticv1.ECDSACurveP256},
			},
			expectedErr: true,
		},
		{
			name: "a curve without an algorithm is rejected, because the default is RSA",
			config: &kubermaticv1.KeyConfiguration{
				Certificates: &kubermaticv1.KeySpec{ECDSACurve: kubermaticv1.ECDSACurveP256},
			},
			expectedErr: true,
		},
		{
			name: "an unsupported curve is rejected",
			config: &kubermaticv1.KeyConfiguration{
				Certificates: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA, ECDSACurve: "P521"},
			},
			expectedErr: true,
		},
		{
			name: "an unsupported RSA key size is rejected",
			config: &kubermaticv1.KeyConfiguration{
				Certificates: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmRSA, RSAKeySize: 1024},
			},
			expectedErr: true,
		},
		{
			name: "an unsupported algorithm is rejected",
			config: &kubermaticv1.KeyConfiguration{
				ServiceAccountKey: &kubermaticv1.KeySpec{Algorithm: "Ed25519"},
			},
			expectedErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			errs := ValidateKeyConfiguration(test.config, field.NewPath("spec", "keyConfiguration"))
			if test.expectedErr && len(errs) == 0 {
				t.Error("expected the configuration to be rejected, but it was accepted")
			}
			if !test.expectedErr && len(errs) > 0 {
				t.Errorf("expected the configuration to be accepted, but got: %v", errs)
			}
		})
	}
}

func TestValidateKeyConfigurationUpdate(t *testing.T) {
	ecdsa := &kubermaticv1.KeyConfiguration{
		Certificates: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA},
	}
	rsa4096 := &kubermaticv1.KeyConfiguration{
		Certificates: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmRSA, RSAKeySize: 4096},
	}

	testCases := []struct {
		name        string
		oldConfig   *kubermaticv1.KeyConfiguration
		newConfig   *kubermaticv1.KeyConfiguration
		expectedErr bool
	}{
		{
			name:      "a legacy cluster stays legacy",
			oldConfig: nil,
			newConfig: nil,
		},
		{
			name:      "an unchanged configuration is accepted",
			oldConfig: ecdsa,
			newConfig: ecdsa.DeepCopy(),
		},
		{
			name:        "changing the configuration is rejected",
			oldConfig:   ecdsa,
			newConfig:   rsa4096,
			expectedErr: true,
		},
		{
			name:        "removing the configuration is rejected",
			oldConfig:   ecdsa,
			newConfig:   nil,
			expectedErr: true,
		},
		{
			name:        "adding the configuration to an existing cluster is rejected",
			oldConfig:   nil,
			newConfig:   ecdsa,
			expectedErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			errs := ValidateKeyConfigurationUpdate(test.oldConfig, test.newConfig, field.NewPath("spec", "keyConfiguration"))
			if test.expectedErr && len(errs) == 0 {
				t.Error("expected the update to be rejected, but it was accepted")
			}
			if !test.expectedErr && len(errs) > 0 {
				t.Errorf("expected the update to be accepted, but got: %v", errs)
			}
		})
	}
}
