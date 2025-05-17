//go:build ee

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

package kubermaticmaster

import (
	"context"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/applicationdefinitions"
	appcat "k8c.io/kubermatic/v2/pkg/ee/default-application-catalog"
	"k8c.io/kubermatic/v2/pkg/install/stack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func deployDefaultApplicationCatalog(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	return appcat.DeployDefaultApplicationCatalog(ctx, logger, kubeClient, opt)
}

func deploySystemApplications(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	return applicationdefinitions.DeploySystemApplications(ctx, logger, kubeClient, opt)
}
