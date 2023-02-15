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
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/stack/common"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

func (m *SeedStack) ValidateState(ctx context.Context, opt stack.DeployOptions) []error {
	return nil
}

func (*SeedStack) ValidateConfiguration(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document, opt stack.DeployOptions, logger logrus.FieldLogger) (*kubermaticv1.KubermaticConfiguration, *yamled.Document, []error) {
	helmFailures := validateHelmValues(helmValues)
	for idx, e := range helmFailures {
		helmFailures[idx] = prefixError("Helm values: ", e)
	}

	return config, helmValues, helmFailures
}

func validateHelmValues(helmValues *yamled.Document) []error {
	if helmValues.IsEmpty() {
		return []error{fmt.Errorf("No Helm Values file was provided, or the file was empty; installation cannot proceed. Please use the flag --helm-values=<valuesfile.yaml>")}
	}

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
	return fmt.Errorf("%s%w", prefix, e)
}
