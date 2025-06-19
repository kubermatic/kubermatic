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

package stack

import (
	"context"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type DeployOptions struct {
	HelmClient                 helm.Client
	HelmValues                 *yamled.Document
	KubeClient                 ctrlruntimeclient.Client
	RestConfig                 *rest.Config
	StorageClassProvider       string
	KubermaticConfiguration    *kubermaticv1.KubermaticConfiguration
	RawKubermaticConfiguration *unstructured.Unstructured
	ForceHelmReleaseUpgrade    bool
	ChartsDirectory            string
	AllowEditionChange         bool
	SkipSeedValidation         sets.Set[string]

	SeedsGetter      provider.SeedsGetter
	SeedClientGetter provider.SeedClientGetter

	Versions kubermaticversion.Versions

	Logger                             *logrus.Entry
	EnableCertManagerV2Migration       bool
	EnableCertManagerUpstreamMigration bool
	EnableNginxIngressMigration        bool
	DisableTelemetry                   bool
	DisableDependencyUpdate            bool

	MLASkipMinio             bool
	MLASkipMinioLifecycleMgr bool
	MLAForceSecrets          bool
	MLAIncludeIap            bool
	MLASkipLogging           bool

	DeployDefaultAppCatalog bool

	SkipCharts []string

	DeployDefaultPolicyTemplateCatalog bool
}

type Stack interface {
	Name() string
	ValidateConfiguration(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document, opt DeployOptions, logger logrus.FieldLogger) (*kubermaticv1.KubermaticConfiguration, *yamled.Document, []error)
	ValidateState(ctx context.Context, opt DeployOptions) []error
	Deploy(ctx context.Context, opt DeployOptions) error
}
