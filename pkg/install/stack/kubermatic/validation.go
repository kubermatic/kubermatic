/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

func ValidateConfiguration(config *operatorv1alpha1.KubermaticConfiguration, helmValues *yamled.Document, logger logrus.FieldLogger) (*operatorv1alpha1.KubermaticConfiguration, *yamled.Document, []error) {
	kubermaticFailures := validateKubermaticConfiguration(config)
	for idx, e := range kubermaticFailures {
		kubermaticFailures[idx] = prefixError("KubermaticConfiguration: ", e)
	}

	helmFailures := validateHelmValues(config, helmValues, logger)
	for idx, e := range helmFailures {
		helmFailures[idx] = prefixError("Helm values: ", e)
	}

	return config, helmValues, append(kubermaticFailures, helmFailures...)
}

func validateKubermaticConfiguration(config *operatorv1alpha1.KubermaticConfiguration) []error {
	failures := []error{}

	if config.Namespace != KubermaticOperatorNamespace {
		failures = append(failures, errors.New("the namespace must be \"kubermatic\""))
	}

	if !config.Spec.Ingress.Disable {
		if config.Spec.Ingress.Domain == "" {
			failures = append(failures, errors.New("spec.ingress.domain cannot be left empty"))
		}

		if config.Spec.Ingress.CertificateIssuer.Name == "" {
			failures = append(failures, errors.New("spec.ingress.certificateIssuer.name cannot be left empty"))
		}
	}

	failures = validateRandomSecret(config, config.Spec.Auth.ServiceAccountKey, "spec.auth.serviceAccountKey", failures)

	if config.Spec.FeatureGates.Has(features.OIDCKubeCfgEndpoint) {
		failures = validateRandomSecret(config, config.Spec.Auth.IssuerClientSecret, "spec.auth.issuerClientSecret", failures)
		failures = validateRandomSecret(config, config.Spec.Auth.IssuerCookieKey, "spec.auth.issuerCookieKey", failures)
	}

	return failures
}

func validateRandomSecret(config *operatorv1alpha1.KubermaticConfiguration, value string, path string, failures []error) []error {
	if value == "" {
		secret, err := randomString()
		if err == nil {
			failures = append(failures, fmt.Errorf("%s must be a non-empty secret, for example: %s", path, secret))
		} else {
			failures = append(failures, fmt.Errorf("%s must be a non-empty secret", path))
		}
	}

	return failures
}

func validateHelmValues(config *operatorv1alpha1.KubermaticConfiguration, helmValues *yamled.Document, logger logrus.FieldLogger) []error {
	failures := []error{}

	path := yamled.Path{"dex", "ingress", "host"}
	if domain, _ := helmValues.GetString(path); domain == "" {
		logger.WithField("domain", config.Spec.Ingress.Domain).Warnf("Helm values: %s is empty, setting to spec.ingress.domain from KubermaticConfiguration", path.String())
		helmValues.Set(path, config.Spec.Ingress.Domain)
	}

	path = yamled.Path{"kubermaticOperator", "imagePullSecret"}
	if value, _ := helmValues.GetString(path); value == "" {
		logger.Warnf("Helm values: %s is empty, setting to spec.imagePullSecret from KubermaticConfiguration", path.String())
		helmValues.Set(path, config.Spec.ImagePullSecret)
	}

	return failures
}

func prefixError(prefix string, e error) error {
	return fmt.Errorf("%s%v", prefix, e)
}

func randomString() (string, error) {
	c := 32
	b := make([]byte, c)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
