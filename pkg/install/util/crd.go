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

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func DeployCRDs(ctx context.Context, kubeClient ctrlruntimeclient.Client, log logrus.FieldLogger, directory string) error {
	crds, err := util.LoadFromDirectory(directory)
	if err != nil {
		return fmt.Errorf("failed to load CRDs: %w", err)
	}

	for _, crd := range crds {
		log.WithField("name", crd.GetName()).Debug("Creating CRD…")

		if err := DeployCRD(ctx, kubeClient, crd); err != nil {
			return fmt.Errorf("failed to deploy CRD %s: %w", crd.GetName(), err)
		}
	}

	// wait for CRDs to be established
	for _, crd := range crds {
		if err := WaitForReadyCRD(ctx, kubeClient, crd.GetName(), 30*time.Second); err != nil {
			return fmt.Errorf("failed to wait for CRD %s to have Established=True condition: %w", crd.GetName(), err)
		}
	}

	return nil
}

func DeployCRD(ctx context.Context, kubeClient ctrlruntimeclient.Client, crd ctrlruntimeclient.Object) error {
	err := kubeClient.Create(ctx, crd)
	if err == nil {
		return nil // success!
	}

	// CRD does not exist already, but creating failed for another reason
	if !kerrors.IsAlreadyExists(err) {
		return err
	}

	// CRD exists already, time to update it
	existingCRD := crd.DeepCopyObject().(ctrlruntimeclient.Object)
	key := ctrlruntimeclient.ObjectKey{
		Name:      crd.GetName(),
		Namespace: crd.GetNamespace(),
	}

	if err = kubeClient.Get(ctx, key, existingCRD); err != nil {
		return fmt.Errorf("failed to retrieve existing CRD: %w", err)
	}

	// do not use mergo to merge the existing into the new CRD,
	// because this would bring back the "kubectl apply" semantics;
	// we want "kubectl replace" semantics instead, so we only keep
	// a few fields from the metadata intact and overwrite everything else

	crd.SetResourceVersion(existingCRD.GetResourceVersion())
	crd.SetGeneration(existingCRD.GetGeneration())

	return kubeClient.Update(ctx, crd)
}

func HasAllReadyCRDs(ctx context.Context, kubeClient ctrlruntimeclient.Client, crdNames []string) (bool, error) {
	for _, crdName := range crdNames {
		exists, err := HasReadyCRD(ctx, kubeClient, crdName)
		if err != nil || !exists {
			return false, err
		}
	}

	return true, nil
}

func HasReadyCRD(ctx context.Context, kubeClient ctrlruntimeclient.Client, crdName string) (bool, error) {
	retrievedCRD := &apiextensionsv1.CustomResourceDefinition{}
	name := types.NamespacedName{Name: crdName}

	if err := kubeClient.Get(ctx, name, retrievedCRD); err != nil {
		// in theory this should never happen after the .Create() call above
		// has succeeded, but it can happen temporarily
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	for _, condition := range retrievedCRD.Status.Conditions {
		if condition.Type == apiextensionsv1.Established {
			return condition.Status == apiextensionsv1.ConditionTrue, nil
		}
	}

	return false, nil
}

func WaitForReadyCRD(ctx context.Context, kubeClient ctrlruntimeclient.Client, crdName string, timeout time.Duration) error {
	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		return HasReadyCRD(ctx, kubeClient, crdName)
	})
}

func WaitForCRDGone(ctx context.Context, kubeClient ctrlruntimeclient.Client, crdName string, timeout time.Duration) error {
	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		retrievedCRD := &apiextensionsv1.CustomResourceDefinition{}
		name := types.NamespacedName{Name: crdName}

		if err := kubeClient.Get(ctx, name, retrievedCRD); err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		return false, nil
	})
}
