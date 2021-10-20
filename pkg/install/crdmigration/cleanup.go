/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package crdmigration

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func RemoveOldResources(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	// remove seed cluster resources
	for seedName, seedClient := range opt.SeedClients {
		if err := removeResourcesInCluster(ctx, logger.WithField("seed", seedName), seedClient); err != nil {
			return fmt.Errorf("processing the seed cluster failed: %w", err)
		}
	}

	// remove master cluster resources
	if err := removeResourcesInCluster(ctx, logger.WithField("master", true), opt.MasterClient); err != nil {
		return fmt.Errorf("processing the master cluster failed: %w", err)
	}

	return nil
}

func removeResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) error {
	logger.Info("Removing resources in old API group…")

	oldAPIVersion := oldAPIGroup + "/v1"

	// reversing the kind is allows to unravel the owner references
	// without objects getting "stuck" and instead being deleted immediately
	for _, kind := range reverseKinds(allKubermaticKinds) {
		removeResourcesOfKindInCluster(ctx, logger.WithField("kind", kind.Name), client, oldAPIVersion, kind)
	}

	return nil
}

func removeResourcesOfKindInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, apiVersion string, kind Kind) error {
	logger.Debug("Removing…")

	list := &unstructured.UnstructuredList{}
	list.SetKind(kind.Name)
	list.SetAPIVersion(apiVersion)

	if err := client.List(ctx, list); err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	for _, item := range list.Items {
		if err := removeObject(ctx, client, &item); err != nil {
			return fmt.Errorf("failed to remove %s: %w", item.GetName(), err)
		}
	}

	return nil
}

func removeObject(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object) error {
	// remove finalizers
	obj.SetFinalizers(nil)

	// ...and owner refs just in case we missed an ownership when setting up the list of kinds
	obj.SetOwnerReferences(nil)

	if err := client.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to remove finalizers/owners from object: %w", err)
	}

	// delete the object
	if err := client.Delete(ctx, obj); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
