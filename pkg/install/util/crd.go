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

package util

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/crd/util"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func DeployCRDs(ctx context.Context, kubeClient ctrlruntimeclient.Client, log logrus.FieldLogger, directory string) error {
	log.Info("Deploying Custom Resource Definitions…")

	crds, err := util.LoadFromDirectory(directory)
	if err != nil {
		return fmt.Errorf("failed to load CRDs: %v", err)
	}

	for _, crd := range crds {
		if err := kubeClient.Create(ctx, &crd); err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to deploy CRD %s: %v", crd.Name, err)
		}
	}

	return nil
}
