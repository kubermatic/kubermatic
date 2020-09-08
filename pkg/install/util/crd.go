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
	"time"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/crd/util"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func DeployCRDs(ctx context.Context, kubeClient ctrlruntimeclient.Client, log logrus.FieldLogger, directory string) error {
	crds, err := util.LoadFromDirectory(directory)
	if err != nil {
		return fmt.Errorf("failed to load CRDs: %v", err)
	}

	for _, crd := range crds {
		log.WithField("name", crd.Name).Debug("Creating CRD…")

		if err := kubeClient.Create(ctx, &crd); err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to deploy CRD %s: %v", crd.Name, err)
		}

		// wait for CRD to be established
		if err := wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
			retrievedCRD := &apiextensionsv1beta1.CustomResourceDefinition{}
			name := types.NamespacedName{Name: crd.Name}

			if err := kubeClient.Get(ctx, name, retrievedCRD); err != nil {
				// in theory this should never happen after the .Create() call above
				// has succeede, but it can happen temporarily
				if kerrors.IsNotFound(err) {
					return false, nil
				}

				return false, err
			}

			for _, condition := range retrievedCRD.Status.Conditions {
				if condition.Type == apiextensionsv1beta1.Established {
					return condition.Status == apiextensionsv1beta1.ConditionTrue, nil
				}
			}

			return false, nil
		}); err != nil {
			return fmt.Errorf("failed to wait for CRD %s to have Established=True condition: %v", crd.Name, err)
		}
	}

	return nil
}
