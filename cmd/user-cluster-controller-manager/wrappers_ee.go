//go:build ee

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"fmt"

	"go.uber.org/zap"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	velerocontroller "k8c.io/kubermatic/v2/pkg/ee/cluster-backup/user-cluster/velero-controller"
	resourceusagecontroller "k8c.io/kubermatic/v2/pkg/ee/resource-usage-controller"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func setupControllers(
	log *zap.SugaredLogger,
	seedMgr, userMgr manager.Manager,
	clusterName string,
	versions kubermatic.Versions,
	overwriteRegistry string,
	caBundle *certificates.CABundle,
	clusterIsPaused userclustercontrollermanager.IsPausedChecker,
) error {
	if err := resourceusagecontroller.Add(log, seedMgr, userMgr, clusterName, caBundle, clusterIsPaused); err != nil {
		return fmt.Errorf("failed to create cluster-backup controller: %w", err)
	}

	if err := velerocontroller.Add(seedMgr, userMgr, log, clusterName, versions, overwriteRegistry); err != nil {
		return fmt.Errorf("failed to create cluster-backup controller: %w", err)
	}

	return nil
}
