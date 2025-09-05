//go:build !ee

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

package images

import (
	"time"

	"github.com/sirupsen/logrus"
	"iter"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/resources"
)

// getAdditionalImagesFromReconcilers returns the images used by the reconcilers for Enterprise Edition addons/components.
// Since this is the Community Edition, this function is no-op and would always return nil,nil.
func getAdditionalImagesFromReconcilers(_ *resources.TemplateData) ([]string, error) {
	return nil, nil
}

func DefaultAppsHelmCharts(
	config *kubermaticv1.KubermaticConfiguration,
	logger logrus.FieldLogger,
	helmClient helm.Client,
	helmTimeout time.Duration,
	registryPrefix string,
) iter.Seq2[*AppsHelmChart, error] {
	return func(yield func(*AppsHelmChart, error) bool) {}
}
