// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "k8c.io/kubermatic/v2/pkg/crd/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Addons returns a AddonInformer.
	Addons() AddonInformer
	// AddonConfigs returns a AddonConfigInformer.
	AddonConfigs() AddonConfigInformer
	// Alertmanagers returns a AlertmanagerInformer.
	Alertmanagers() AlertmanagerInformer
	// AllowedRegistries returns a AllowedRegistryInformer.
	AllowedRegistries() AllowedRegistryInformer
	// Clusters returns a ClusterInformer.
	Clusters() ClusterInformer
	// ClusterTemplates returns a ClusterTemplateInformer.
	ClusterTemplates() ClusterTemplateInformer
	// ClusterTemplateInstances returns a ClusterTemplateInstanceInformer.
	ClusterTemplateInstances() ClusterTemplateInstanceInformer
	// Constraints returns a ConstraintInformer.
	Constraints() ConstraintInformer
	// ConstraintTemplates returns a ConstraintTemplateInformer.
	ConstraintTemplates() ConstraintTemplateInformer
	// EtcdBackupConfigs returns a EtcdBackupConfigInformer.
	EtcdBackupConfigs() EtcdBackupConfigInformer
	// EtcdRestores returns a EtcdRestoreInformer.
	EtcdRestores() EtcdRestoreInformer
	// ExternalClusters returns a ExternalClusterInformer.
	ExternalClusters() ExternalClusterInformer
	// KubermaticSettings returns a KubermaticSettingInformer.
	KubermaticSettings() KubermaticSettingInformer
	// MLAAdminSettings returns a MLAAdminSettingInformer.
	MLAAdminSettings() MLAAdminSettingInformer
	// Projects returns a ProjectInformer.
	Projects() ProjectInformer
	// RuleGroups returns a RuleGroupInformer.
	RuleGroups() RuleGroupInformer
	// Users returns a UserInformer.
	Users() UserInformer
	// UserProjectBindings returns a UserProjectBindingInformer.
	UserProjectBindings() UserProjectBindingInformer
	// UserSSHKeys returns a UserSSHKeyInformer.
	UserSSHKeys() UserSSHKeyInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Addons returns a AddonInformer.
func (v *version) Addons() AddonInformer {
	return &addonInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// AddonConfigs returns a AddonConfigInformer.
func (v *version) AddonConfigs() AddonConfigInformer {
	return &addonConfigInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Alertmanagers returns a AlertmanagerInformer.
func (v *version) Alertmanagers() AlertmanagerInformer {
	return &alertmanagerInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// AllowedRegistries returns a AllowedRegistryInformer.
func (v *version) AllowedRegistries() AllowedRegistryInformer {
	return &allowedRegistryInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Clusters returns a ClusterInformer.
func (v *version) Clusters() ClusterInformer {
	return &clusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterTemplates returns a ClusterTemplateInformer.
func (v *version) ClusterTemplates() ClusterTemplateInformer {
	return &clusterTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterTemplateInstances returns a ClusterTemplateInstanceInformer.
func (v *version) ClusterTemplateInstances() ClusterTemplateInstanceInformer {
	return &clusterTemplateInstanceInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Constraints returns a ConstraintInformer.
func (v *version) Constraints() ConstraintInformer {
	return &constraintInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ConstraintTemplates returns a ConstraintTemplateInformer.
func (v *version) ConstraintTemplates() ConstraintTemplateInformer {
	return &constraintTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// EtcdBackupConfigs returns a EtcdBackupConfigInformer.
func (v *version) EtcdBackupConfigs() EtcdBackupConfigInformer {
	return &etcdBackupConfigInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// EtcdRestores returns a EtcdRestoreInformer.
func (v *version) EtcdRestores() EtcdRestoreInformer {
	return &etcdRestoreInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ExternalClusters returns a ExternalClusterInformer.
func (v *version) ExternalClusters() ExternalClusterInformer {
	return &externalClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// KubermaticSettings returns a KubermaticSettingInformer.
func (v *version) KubermaticSettings() KubermaticSettingInformer {
	return &kubermaticSettingInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// MLAAdminSettings returns a MLAAdminSettingInformer.
func (v *version) MLAAdminSettings() MLAAdminSettingInformer {
	return &mLAAdminSettingInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Projects returns a ProjectInformer.
func (v *version) Projects() ProjectInformer {
	return &projectInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// RuleGroups returns a RuleGroupInformer.
func (v *version) RuleGroups() RuleGroupInformer {
	return &ruleGroupInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Users returns a UserInformer.
func (v *version) Users() UserInformer {
	return &userInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// UserProjectBindings returns a UserProjectBindingInformer.
func (v *version) UserProjectBindings() UserProjectBindingInformer {
	return &userProjectBindingInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// UserSSHKeys returns a UserSSHKeyInformer.
func (v *version) UserSSHKeys() UserSSHKeyInformer {
	return &userSSHKeyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
