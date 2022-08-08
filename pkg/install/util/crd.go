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

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/crd"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func DeployCRDs(ctx context.Context, kubeClient ctrlruntimeclient.Client, log logrus.FieldLogger, directory string, versions *kubermaticversion.Versions) error {
	crds, err := crd.LoadFromDirectory(directory)
	if err != nil {
		return fmt.Errorf("failed to load CRDs: %w", err)
	}

	for _, crd := range crds {
		log.WithField("name", crd.GetName()).Debug("Creating CRD…")

		if versions != nil {
			// inject the current KKP version, so the operator and other controllers
			// can react to the changed CRDs (the seed-operator will do the same when
			// updating CRDs on seed clusters)
			annotations := crd.GetAnnotations()
			if annotations == nil {
				annotations = map[string]string{}
			}
			annotations[resources.VersionLabel] = versions.KubermaticCommit
			crd.SetAnnotations(annotations)
		}

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

// DeleteOldApplicationInstallationCrd removes old applicationInstallation crd from cluster.
// See pkg/install/stack/kubermatic-master/stack.go::InstallKubermaticCRDs() for more information.
// TODO REMOVE AFTER release v2.21.
func DeleteOldApplicationInstallationCrd(ctx context.Context, kubeClient ctrlruntimeclient.Client) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := kubeClient.Get(ctx, types.NamespacedName{Name: appskubermaticv1.ApplicationInstallationsFQDNName}, crd)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get CRD %s: %w", appskubermaticv1.ApplicationInstallationsFQDNName, err)
	}

	// No action is required
	if err != nil && apierrors.IsNotFound(err) {
		return nil
	}

	// CRD exists, now we need to check if it's namespaced scoped
	if crd.Spec.Scope == apiextensionsv1.NamespaceScoped {
		return nil
	}

	// Crd is cluster scoped, so we need to delete it.
	if err := kubeClient.Delete(ctx, crd); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete CRD %s: %w", crd.Name, err)
	}

	if err := WaitForCRDGone(ctx, kubeClient, appskubermaticv1.ApplicationInstallationsFQDNName, 10*time.Second); err != nil {
		return fmt.Errorf(" %s could not be deleted, please check for remaining resources and remove any finalizers", appskubermaticv1.ApplicationInstallationsFQDNName)
	}
	return nil
}

func DeployCRD(ctx context.Context, kubeClient ctrlruntimeclient.Client, crd ctrlruntimeclient.Object) error {
	err := kubeClient.Create(ctx, crd)
	if err == nil {
		return nil // success!
	}

	// CRD does not exist already, but creating failed for another reason
	if !apierrors.IsAlreadyExists(err) {
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
		if apierrors.IsNotFound(err) {
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
			if apierrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		return false, nil
	})
}
