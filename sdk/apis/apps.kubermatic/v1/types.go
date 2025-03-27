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

package v1

const (
	// ApplicationDefinitionSeedCleanupFinalizer indicates that synced application definition on seed clusters need cleanup.
	ApplicationDefinitionSeedCleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-application-definition"

	// ApplicationInstallationCleanupFinalizer indicates that application installed on user-cluster need cleanup
	// ie uninstall the application, remove  namespace where application were installed ...
	ApplicationInstallationCleanupFinalizer = "kubermatic.k8c.io/cleanup-application-installation"

	// ApplicationManagedByLabel indicates the ownership of the application definition / application installation.
	ApplicationManagedByLabel = "apps.kubermatic.k8c.io/managed-by"

	// ApplicationManagedByKKPValue can be used as a value for the ApplicationManagedByLabel to indicate that the
	// application definition / application installation is managed by KKP (i.e. it is KKP-internal).
	ApplicationManagedByKKPValue = "kkp"

	// ApplicationTypeLabel indicated the type of the application definition / application installation.
	ApplicationTypeLabel = "apps.kubermatic.k8c.io/type"

	// ApplicationTypeCNIValue can be used as a value for the ApplicationTypeLabel to indicate that the
	// application definition / application installation type if CNI (Container Network Interface).
	ApplicationTypeCNIValue = "cni"

	// ApplicationEnforcedAnnotation marks an ApplicationInstallation as enforced.
	ApplicationEnforcedAnnotation = "apps.kubermatic.k8c.io/enforced"

	// ApplicationDefaultedAnnotation marks an ApplicationInstallation as defaulted.
	ApplicationDefaultedAnnotation = "apps.kubermatic.k8c.io/defaulted"
)
