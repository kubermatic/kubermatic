/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package reconciling

import (
	"context"
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	appskubermaticv1 "k8c.io/api/v3/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

// CustomResourceDefinitionReconciler defines an interface to create/update CustomResourceDefinitions.
type CustomResourceDefinitionReconciler = func(existing *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error)

// NamedCustomResourceDefinitionReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedCustomResourceDefinitionReconcilerFactory = func() (name string, reconciler CustomResourceDefinitionReconciler)

// CustomResourceDefinitionObjectWrapper adds a wrapper so the CustomResourceDefinitionReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func CustomResourceDefinitionObjectWrapper(reconciler CustomResourceDefinitionReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*apiextensionsv1.CustomResourceDefinition))
		}
		return reconciler(&apiextensionsv1.CustomResourceDefinition{})
	}
}

// ReconcileCustomResourceDefinitions will create and update the CustomResourceDefinitions coming from the passed CustomResourceDefinitionReconciler slice.
func ReconcileCustomResourceDefinitions(ctx context.Context, namedFactories []NamedCustomResourceDefinitionReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := CustomResourceDefinitionObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &apiextensionsv1.CustomResourceDefinition{}, false); err != nil {
			return fmt.Errorf("failed to ensure CustomResourceDefinition %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// APIServiceReconciler defines an interface to create/update APIServices.
type APIServiceReconciler = func(existing *apiregistrationv1.APIService) (*apiregistrationv1.APIService, error)

// NamedAPIServiceReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedAPIServiceReconcilerFactory = func() (name string, reconciler APIServiceReconciler)

// APIServiceObjectWrapper adds a wrapper so the APIServiceReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func APIServiceObjectWrapper(reconciler APIServiceReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*apiregistrationv1.APIService))
		}
		return reconciler(&apiregistrationv1.APIService{})
	}
}

// ReconcileAPIServices will create and update the APIServices coming from the passed APIServiceReconciler slice.
func ReconcileAPIServices(ctx context.Context, namedFactories []NamedAPIServiceReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := APIServiceObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &apiregistrationv1.APIService{}, false); err != nil {
			return fmt.Errorf("failed to ensure APIService %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// AddonReconciler defines an interface to create/update Addons.
type AddonReconciler = func(existing *kubermaticv1.Addon) (*kubermaticv1.Addon, error)

// NamedAddonReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedAddonReconcilerFactory = func() (name string, reconciler AddonReconciler)

// AddonObjectWrapper adds a wrapper so the AddonReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func AddonObjectWrapper(reconciler AddonReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.Addon))
		}
		return reconciler(&kubermaticv1.Addon{})
	}
}

// ReconcileAddons will create and update the Addons coming from the passed AddonReconciler slice.
func ReconcileAddons(ctx context.Context, namedFactories []NamedAddonReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := AddonObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.Addon{}, false); err != nil {
			return fmt.Errorf("failed to ensure Addon %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// AddonConfigReconciler defines an interface to create/update AddonConfigs.
type AddonConfigReconciler = func(existing *kubermaticv1.AddonConfig) (*kubermaticv1.AddonConfig, error)

// NamedAddonConfigReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedAddonConfigReconcilerFactory = func() (name string, reconciler AddonConfigReconciler)

// AddonConfigObjectWrapper adds a wrapper so the AddonConfigReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func AddonConfigObjectWrapper(reconciler AddonConfigReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.AddonConfig))
		}
		return reconciler(&kubermaticv1.AddonConfig{})
	}
}

// ReconcileAddonConfigs will create and update the AddonConfigs coming from the passed AddonConfigReconciler slice.
func ReconcileAddonConfigs(ctx context.Context, namedFactories []NamedAddonConfigReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := AddonConfigObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.AddonConfig{}, false); err != nil {
			return fmt.Errorf("failed to ensure AddonConfig %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// ClusterReconciler defines an interface to create/update Clusters.
type ClusterReconciler = func(existing *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

// NamedClusterReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedClusterReconcilerFactory = func() (name string, reconciler ClusterReconciler)

// ClusterObjectWrapper adds a wrapper so the ClusterReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func ClusterObjectWrapper(reconciler ClusterReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.Cluster))
		}
		return reconciler(&kubermaticv1.Cluster{})
	}
}

// ReconcileClusters will create and update the Clusters coming from the passed ClusterReconciler slice.
func ReconcileClusters(ctx context.Context, namedFactories []NamedClusterReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := ClusterObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.Cluster{}, false); err != nil {
			return fmt.Errorf("failed to ensure Cluster %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// ClusterTemplateReconciler defines an interface to create/update ClusterTemplates.
type ClusterTemplateReconciler = func(existing *kubermaticv1.ClusterTemplate) (*kubermaticv1.ClusterTemplate, error)

// NamedClusterTemplateReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedClusterTemplateReconcilerFactory = func() (name string, reconciler ClusterTemplateReconciler)

// ClusterTemplateObjectWrapper adds a wrapper so the ClusterTemplateReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func ClusterTemplateObjectWrapper(reconciler ClusterTemplateReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.ClusterTemplate))
		}
		return reconciler(&kubermaticv1.ClusterTemplate{})
	}
}

// ReconcileClusterTemplates will create and update the ClusterTemplates coming from the passed ClusterTemplateReconciler slice.
func ReconcileClusterTemplates(ctx context.Context, namedFactories []NamedClusterTemplateReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := ClusterTemplateObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.ClusterTemplate{}, false); err != nil {
			return fmt.Errorf("failed to ensure ClusterTemplate %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// DatacenterReconciler defines an interface to create/update Datacenters.
type DatacenterReconciler = func(existing *kubermaticv1.Datacenter) (*kubermaticv1.Datacenter, error)

// NamedDatacenterReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedDatacenterReconcilerFactory = func() (name string, reconciler DatacenterReconciler)

// DatacenterObjectWrapper adds a wrapper so the DatacenterReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func DatacenterObjectWrapper(reconciler DatacenterReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.Datacenter))
		}
		return reconciler(&kubermaticv1.Datacenter{})
	}
}

// ReconcileDatacenters will create and update the Datacenters coming from the passed DatacenterReconciler slice.
func ReconcileDatacenters(ctx context.Context, namedFactories []NamedDatacenterReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := DatacenterObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.Datacenter{}, false); err != nil {
			return fmt.Errorf("failed to ensure Datacenter %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// KubermaticConfigurationReconciler defines an interface to create/update KubermaticConfigurations.
type KubermaticConfigurationReconciler = func(existing *kubermaticv1.KubermaticConfiguration) (*kubermaticv1.KubermaticConfiguration, error)

// NamedKubermaticConfigurationReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedKubermaticConfigurationReconcilerFactory = func() (name string, reconciler KubermaticConfigurationReconciler)

// KubermaticConfigurationObjectWrapper adds a wrapper so the KubermaticConfigurationReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func KubermaticConfigurationObjectWrapper(reconciler KubermaticConfigurationReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.KubermaticConfiguration))
		}
		return reconciler(&kubermaticv1.KubermaticConfiguration{})
	}
}

// ReconcileKubermaticConfigurations will create and update the KubermaticConfigurations coming from the passed KubermaticConfigurationReconciler slice.
func ReconcileKubermaticConfigurations(ctx context.Context, namedFactories []NamedKubermaticConfigurationReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := KubermaticConfigurationObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.KubermaticConfiguration{}, false); err != nil {
			return fmt.Errorf("failed to ensure KubermaticConfiguration %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// PresetReconciler defines an interface to create/update Presets.
type PresetReconciler = func(existing *kubermaticv1.Preset) (*kubermaticv1.Preset, error)

// NamedPresetReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedPresetReconcilerFactory = func() (name string, reconciler PresetReconciler)

// PresetObjectWrapper adds a wrapper so the PresetReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func PresetObjectWrapper(reconciler PresetReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.Preset))
		}
		return reconciler(&kubermaticv1.Preset{})
	}
}

// ReconcilePresets will create and update the Presets coming from the passed PresetReconciler slice.
func ReconcilePresets(ctx context.Context, namedFactories []NamedPresetReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := PresetObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.Preset{}, false); err != nil {
			return fmt.Errorf("failed to ensure Preset %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// RuleGroupReconciler defines an interface to create/update RuleGroups.
type RuleGroupReconciler = func(existing *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error)

// NamedRuleGroupReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedRuleGroupReconcilerFactory = func() (name string, reconciler RuleGroupReconciler)

// RuleGroupObjectWrapper adds a wrapper so the RuleGroupReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func RuleGroupObjectWrapper(reconciler RuleGroupReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.RuleGroup))
		}
		return reconciler(&kubermaticv1.RuleGroup{})
	}
}

// ReconcileRuleGroups will create and update the RuleGroups coming from the passed RuleGroupReconciler slice.
func ReconcileRuleGroups(ctx context.Context, namedFactories []NamedRuleGroupReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := RuleGroupObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.RuleGroup{}, false); err != nil {
			return fmt.Errorf("failed to ensure RuleGroup %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// UserReconciler defines an interface to create/update Users.
type UserReconciler = func(existing *kubermaticv1.User) (*kubermaticv1.User, error)

// NamedUserReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedUserReconcilerFactory = func() (name string, reconciler UserReconciler)

// UserObjectWrapper adds a wrapper so the UserReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func UserObjectWrapper(reconciler UserReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.User))
		}
		return reconciler(&kubermaticv1.User{})
	}
}

// ReconcileUsers will create and update the Users coming from the passed UserReconciler slice.
func ReconcileUsers(ctx context.Context, namedFactories []NamedUserReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := UserObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.User{}, false); err != nil {
			return fmt.Errorf("failed to ensure User %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// UserSSHKeyReconciler defines an interface to create/update UserSSHKeys.
type UserSSHKeyReconciler = func(existing *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error)

// NamedUserSSHKeyReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedUserSSHKeyReconcilerFactory = func() (name string, reconciler UserSSHKeyReconciler)

// UserSSHKeyObjectWrapper adds a wrapper so the UserSSHKeyReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func UserSSHKeyObjectWrapper(reconciler UserSSHKeyReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*kubermaticv1.UserSSHKey))
		}
		return reconciler(&kubermaticv1.UserSSHKey{})
	}
}

// ReconcileUserSSHKeys will create and update the UserSSHKeys coming from the passed UserSSHKeyReconciler slice.
func ReconcileUserSSHKeys(ctx context.Context, namedFactories []NamedUserSSHKeyReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := UserSSHKeyObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &kubermaticv1.UserSSHKey{}, false); err != nil {
			return fmt.Errorf("failed to ensure UserSSHKey %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// ApplicationDefinitionReconciler defines an interface to create/update ApplicationDefinitions.
type ApplicationDefinitionReconciler = func(existing *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error)

// NamedApplicationDefinitionReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedApplicationDefinitionReconcilerFactory = func() (name string, reconciler ApplicationDefinitionReconciler)

// ApplicationDefinitionObjectWrapper adds a wrapper so the ApplicationDefinitionReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func ApplicationDefinitionObjectWrapper(reconciler ApplicationDefinitionReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*appskubermaticv1.ApplicationDefinition))
		}
		return reconciler(&appskubermaticv1.ApplicationDefinition{})
	}
}

// ReconcileApplicationDefinitions will create and update the ApplicationDefinitions coming from the passed ApplicationDefinitionReconciler slice.
func ReconcileApplicationDefinitions(ctx context.Context, namedFactories []NamedApplicationDefinitionReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := ApplicationDefinitionObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &appskubermaticv1.ApplicationDefinition{}, false); err != nil {
			return fmt.Errorf("failed to ensure ApplicationDefinition %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// ApplicationInstallationReconciler defines an interface to create/update ApplicationInstallations.
type ApplicationInstallationReconciler = func(existing *appskubermaticv1.ApplicationInstallation) (*appskubermaticv1.ApplicationInstallation, error)

// NamedApplicationInstallationReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedApplicationInstallationReconcilerFactory = func() (name string, reconciler ApplicationInstallationReconciler)

// ApplicationInstallationObjectWrapper adds a wrapper so the ApplicationInstallationReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func ApplicationInstallationObjectWrapper(reconciler ApplicationInstallationReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*appskubermaticv1.ApplicationInstallation))
		}
		return reconciler(&appskubermaticv1.ApplicationInstallation{})
	}
}

// ReconcileApplicationInstallations will create and update the ApplicationInstallations coming from the passed ApplicationInstallationReconciler slice.
func ReconcileApplicationInstallations(ctx context.Context, namedFactories []NamedApplicationInstallationReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := ApplicationInstallationObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &appskubermaticv1.ApplicationInstallation{}, false); err != nil {
			return fmt.Errorf("failed to ensure ApplicationInstallation %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// OperatingSystemProfileReconciler defines an interface to create/update OperatingSystemProfiles.
type OperatingSystemProfileReconciler = func(existing *osmv1alpha1.OperatingSystemProfile) (*osmv1alpha1.OperatingSystemProfile, error)

// NamedOperatingSystemProfileReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedOperatingSystemProfileReconcilerFactory = func() (name string, reconciler OperatingSystemProfileReconciler)

// OperatingSystemProfileObjectWrapper adds a wrapper so the OperatingSystemProfileReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func OperatingSystemProfileObjectWrapper(reconciler OperatingSystemProfileReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*osmv1alpha1.OperatingSystemProfile))
		}
		return reconciler(&osmv1alpha1.OperatingSystemProfile{})
	}
}

// ReconcileOperatingSystemProfiles will create and update the OperatingSystemProfiles coming from the passed OperatingSystemProfileReconciler slice.
func ReconcileOperatingSystemProfiles(ctx context.Context, namedFactories []NamedOperatingSystemProfileReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := OperatingSystemProfileObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &osmv1alpha1.OperatingSystemProfile{}, false); err != nil {
			return fmt.Errorf("failed to ensure OperatingSystemProfile %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// OperatingSystemConfigReconciler defines an interface to create/update OperatingSystemConfigs.
type OperatingSystemConfigReconciler = func(existing *osmv1alpha1.OperatingSystemConfig) (*osmv1alpha1.OperatingSystemConfig, error)

// NamedOperatingSystemConfigReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedOperatingSystemConfigReconcilerFactory = func() (name string, reconciler OperatingSystemConfigReconciler)

// OperatingSystemConfigObjectWrapper adds a wrapper so the OperatingSystemConfigReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func OperatingSystemConfigObjectWrapper(reconciler OperatingSystemConfigReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*osmv1alpha1.OperatingSystemConfig))
		}
		return reconciler(&osmv1alpha1.OperatingSystemConfig{})
	}
}

// ReconcileOperatingSystemConfigs will create and update the OperatingSystemConfigs coming from the passed OperatingSystemConfigReconciler slice.
func ReconcileOperatingSystemConfigs(ctx context.Context, namedFactories []NamedOperatingSystemConfigReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := OperatingSystemConfigObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &osmv1alpha1.OperatingSystemConfig{}, false); err != nil {
			return fmt.Errorf("failed to ensure OperatingSystemConfig %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// VirtualMachineInstancetypeReconciler defines an interface to create/update VirtualMachineInstancetypes.
type VirtualMachineInstancetypeReconciler = func(existing *instancetypev1alpha1.VirtualMachineInstancetype) (*instancetypev1alpha1.VirtualMachineInstancetype, error)

// NamedVirtualMachineInstancetypeReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedVirtualMachineInstancetypeReconcilerFactory = func() (name string, reconciler VirtualMachineInstancetypeReconciler)

// VirtualMachineInstancetypeObjectWrapper adds a wrapper so the VirtualMachineInstancetypeReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func VirtualMachineInstancetypeObjectWrapper(reconciler VirtualMachineInstancetypeReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*instancetypev1alpha1.VirtualMachineInstancetype))
		}
		return reconciler(&instancetypev1alpha1.VirtualMachineInstancetype{})
	}
}

// ReconcileVirtualMachineInstancetypes will create and update the VirtualMachineInstancetypes coming from the passed VirtualMachineInstancetypeReconciler slice.
func ReconcileVirtualMachineInstancetypes(ctx context.Context, namedFactories []NamedVirtualMachineInstancetypeReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := VirtualMachineInstancetypeObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &instancetypev1alpha1.VirtualMachineInstancetype{}, false); err != nil {
			return fmt.Errorf("failed to ensure VirtualMachineInstancetype %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// VirtualMachinePreferenceReconciler defines an interface to create/update VirtualMachinePreferences.
type VirtualMachinePreferenceReconciler = func(existing *instancetypev1alpha1.VirtualMachinePreference) (*instancetypev1alpha1.VirtualMachinePreference, error)

// NamedVirtualMachinePreferenceReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedVirtualMachinePreferenceReconcilerFactory = func() (name string, reconciler VirtualMachinePreferenceReconciler)

// VirtualMachinePreferenceObjectWrapper adds a wrapper so the VirtualMachinePreferenceReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func VirtualMachinePreferenceObjectWrapper(reconciler VirtualMachinePreferenceReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*instancetypev1alpha1.VirtualMachinePreference))
		}
		return reconciler(&instancetypev1alpha1.VirtualMachinePreference{})
	}
}

// ReconcileVirtualMachinePreferences will create and update the VirtualMachinePreferences coming from the passed VirtualMachinePreferenceReconciler slice.
func ReconcileVirtualMachinePreferences(ctx context.Context, namedFactories []NamedVirtualMachinePreferenceReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := VirtualMachinePreferenceObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &instancetypev1alpha1.VirtualMachinePreference{}, false); err != nil {
			return fmt.Errorf("failed to ensure VirtualMachinePreference %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}

// DataVolumeReconciler defines an interface to create/update DataVolumes.
type DataVolumeReconciler = func(existing *cdiv1beta1.DataVolume) (*cdiv1beta1.DataVolume, error)

// NamedDataVolumeReconcilerFactory returns the name of the resource and the corresponding Reconciler function.
type NamedDataVolumeReconcilerFactory = func() (name string, reconciler DataVolumeReconciler)

// DataVolumeObjectWrapper adds a wrapper so the DataVolumeReconciler matches ObjectReconciler.
// This is needed as Go does not support function interface matching.
func DataVolumeObjectWrapper(reconciler DataVolumeReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		if existing != nil {
			return reconciler(existing.(*cdiv1beta1.DataVolume))
		}
		return reconciler(&cdiv1beta1.DataVolume{})
	}
}

// ReconcileDataVolumes will create and update the DataVolumes coming from the passed DataVolumeReconciler slice.
func ReconcileDataVolumes(ctx context.Context, namedFactories []NamedDataVolumeReconcilerFactory, namespace string, client ctrlruntimeclient.Client, objectModifiers ...reconciling.ObjectModifier) error {
	for _, factory := range namedFactories {
		name, reconciler := factory()
		reconcileObject := DataVolumeObjectWrapper(reconciler)
		reconcileObject = reconciling.CreateWithNamespace(reconcileObject, namespace)
		reconcileObject = reconciling.CreateWithName(reconcileObject, name)

		for _, objectModifier := range objectModifiers {
			reconcileObject = objectModifier(reconcileObject)
		}

		if err := reconciling.EnsureNamedObject(ctx, types.NamespacedName{Namespace: namespace, Name: name}, reconcileObject, client, &cdiv1beta1.DataVolume{}, false); err != nil {
			return fmt.Errorf("failed to ensure DataVolume %s/%s: %w", namespace, name, err)
		}
	}

	return nil
}
