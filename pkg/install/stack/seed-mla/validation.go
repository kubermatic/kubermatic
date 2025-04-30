/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

func (*MonitoringStack) ValidateState(ctx context.Context, opt stack.DeployOptions) []error {
	failures := []error{}

	return failures
}

func (*MonitoringStack) ValidateConfiguration(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document, opt stack.DeployOptions, logger logrus.FieldLogger) (*kubermaticv1.KubermaticConfiguration, *yamled.Document, []error) {
	helmFailures := validateHelmValues(helmValues, opt)
	for idx, e := range helmFailures {
		helmFailures[idx] = prefixError("Helm values: ", e)
	}

	return config, helmValues, helmFailures
}

func validateHelmValues(helmValues *yamled.Document, opt stack.DeployOptions) []error {
	failures := []error{}

	if opt.MLAIncludeIap {
		path := yamled.Path{"iap", "deployments", "grafana", "encryption_key"}
		grafanaEncryptionKey, _ := helmValues.GetString(path)
		if err := ValidateIapBlockSecret(grafanaEncryptionKey, path.String()); err != nil {
			failures = append(failures, err)
		}

		path = yamled.Path{"iap", "deployments", "alertmanager", "encryption_key"}
		alertmanagerEncryptionKey, _ := helmValues.GetString(path)
		if err := ValidateIapBlockSecret(alertmanagerEncryptionKey, path.String()); err != nil {
			failures = append(failures, err)
		}

		path = yamled.Path{"iap", "deployments", "prometheus", "encryption_key"}
		prometheusEncryptionKey, _ := helmValues.GetString(path)
		if err := ValidateIapBlockSecret(prometheusEncryptionKey, path.String()); err != nil {
			failures = append(failures, err)
		}
	}

	return failures
}

func prefixError(prefix string, e error) error {
	return fmt.Errorf("%s%w", prefix, e)
}

func ValidateIapBlockSecret(value string, path string) error {
	if value == "" || !isBlockSecret(value) {
		secret, err := randomString()
		if err == nil {
			return fmt.Errorf("%s must be a non-empty secret of 16, 24 or 32 characters, for example: %s", path, secret)
		}

		return fmt.Errorf("%s must be a non-empty secret", path)
	}

	return nil
}

// isBlockSecret checks if the provided value is a valid block secret based on its length.
// A valid block secret must have a length of 16, 24, or 32 characters to match common
// encryption key lengths (e.g., AES-128, AES-192, AES-256).
func isBlockSecret(value string) bool {
	switch len(value) {
	case 16, 24, 32:
		return true
	}
	return false
}

func randomString() (string, error) {
	c := 24
	b := make([]byte, c)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
