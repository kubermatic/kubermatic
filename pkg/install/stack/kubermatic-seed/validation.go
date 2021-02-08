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

package kubermaticseed

import (
	"fmt"

	"github.com/sirupsen/logrus"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/install/stack/common"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

func (*SeedStack) ValidateConfiguration(config *operatorv1alpha1.KubermaticConfiguration, helmValues *yamled.Document, logger logrus.FieldLogger) (*operatorv1alpha1.KubermaticConfiguration, *yamled.Document, []error) {
	helmFailures := validateHelmValues(helmValues)
	for idx, e := range helmFailures {
		helmFailures[idx] = prefixError("Helm values: ", e)
	}

	return config, helmValues, helmFailures
}

func validateHelmValues(helmValues *yamled.Document) []error {
	failures := []error{}

	path := yamled.Path{"minio", "credentials", "accessKey"}
	accessKey, _ := helmValues.GetString(path)
	if err := common.ValidateRandomSecret(accessKey, path.String()); err != nil {
		failures = append(failures, err)
	}

	path = yamled.Path{"minio", "credentials", "secretKey"}
	secretKey, _ := helmValues.GetString(path)
	if err := common.ValidateRandomSecret(secretKey, path.String()); err != nil {
		failures = append(failures, err)
	}

	return failures
}

func prefixError(prefix string, e error) error {
	return fmt.Errorf("%s%v", prefix, e)
}
