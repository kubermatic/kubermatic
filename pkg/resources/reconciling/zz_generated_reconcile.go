// This file is generated. DO NOT EDIT.
package reconciling

import (
	gatekeeperv1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

// NamespaceCreator defines an interface to create/update Namespaces
type NamespaceCreator = GenericObjectCreator[*corev1.Namespace]

// NamedNamespaceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedNamespaceCreatorGetter = GenericNamedObjectCreator[*corev1.Namespace]

// ServiceCreator defines an interface to create/update Services
type ServiceCreator = GenericObjectCreator[*corev1.Service]

// NamedServiceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedServiceCreatorGetter = GenericNamedObjectCreator[*corev1.Service]

// SecretCreator defines an interface to create/update Secrets
type SecretCreator = GenericObjectCreator[*corev1.Secret]

// NamedSecretCreatorGetter returns the name of the resource and the corresponding creator function
type NamedSecretCreatorGetter = GenericNamedObjectCreator[*corev1.Secret]

// ConfigMapCreator defines an interface to create/update ConfigMaps
type ConfigMapCreator = GenericObjectCreator[*corev1.ConfigMap]

// NamedConfigMapCreatorGetter returns the name of the resource and the corresponding creator function
type NamedConfigMapCreatorGetter = GenericNamedObjectCreator[*corev1.ConfigMap]

// ServiceAccountCreator defines an interface to create/update ServiceAccounts
type ServiceAccountCreator = GenericObjectCreator[*corev1.ServiceAccount]

// NamedServiceAccountCreatorGetter returns the name of the resource and the corresponding creator function
type NamedServiceAccountCreatorGetter = GenericNamedObjectCreator[*corev1.ServiceAccount]

// EndpointsCreator defines an interface to create/update Endpoints
type EndpointsCreator = GenericObjectCreator[*corev1.Endpoints]

// NamedEndpointsCreatorGetter returns the name of the resource and the corresponding creator function
type NamedEndpointsCreatorGetter = GenericNamedObjectCreator[*corev1.Endpoints]

// EndpointSliceCreator defines an interface to create/update EndpointSlices
type EndpointSliceCreator = GenericObjectCreator[*discovery.EndpointSlice]

// NamedEndpointSliceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedEndpointSliceCreatorGetter = GenericNamedObjectCreator[*discovery.EndpointSlice]

// StatefulSetCreator defines an interface to create/update StatefulSets
type StatefulSetCreator = GenericObjectCreator[*appsv1.StatefulSet]

// NamedStatefulSetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedStatefulSetCreatorGetter = GenericNamedObjectCreator[*appsv1.StatefulSet]

// DeploymentCreator defines an interface to create/update Deployments
type DeploymentCreator = GenericObjectCreator[*appsv1.Deployment]

// NamedDeploymentCreatorGetter returns the name of the resource and the corresponding creator function
type NamedDeploymentCreatorGetter = GenericNamedObjectCreator[*appsv1.Deployment]

// DaemonSetCreator defines an interface to create/update DaemonSets
type DaemonSetCreator = GenericObjectCreator[*appsv1.DaemonSet]

// NamedDaemonSetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedDaemonSetCreatorGetter = GenericNamedObjectCreator[*appsv1.DaemonSet]

// PodDisruptionBudgetCreator defines an interface to create/update PodDisruptionBudgets
type PodDisruptionBudgetCreator = GenericObjectCreator[*policyv1.PodDisruptionBudget]

// NamedPodDisruptionBudgetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedPodDisruptionBudgetCreatorGetter = GenericNamedObjectCreator[*policyv1.PodDisruptionBudget]

// VerticalPodAutoscalerCreator defines an interface to create/update VerticalPodAutoscalers
type VerticalPodAutoscalerCreator = GenericObjectCreator[*autoscalingv1.VerticalPodAutoscaler]

// NamedVerticalPodAutoscalerCreatorGetter returns the name of the resource and the corresponding creator function
type NamedVerticalPodAutoscalerCreatorGetter = GenericNamedObjectCreator[*autoscalingv1.VerticalPodAutoscaler]

// ClusterRoleBindingCreator defines an interface to create/update ClusterRoleBindings
type ClusterRoleBindingCreator = GenericObjectCreator[*rbacv1.ClusterRoleBinding]

// NamedClusterRoleBindingCreatorGetter returns the name of the resource and the corresponding creator function
type NamedClusterRoleBindingCreatorGetter = GenericNamedObjectCreator[*rbacv1.ClusterRoleBinding]

// ClusterRoleCreator defines an interface to create/update ClusterRoles
type ClusterRoleCreator = GenericObjectCreator[*rbacv1.ClusterRole]

// NamedClusterRoleCreatorGetter returns the name of the resource and the corresponding creator function
type NamedClusterRoleCreatorGetter = GenericNamedObjectCreator[*rbacv1.ClusterRole]

// RoleCreator defines an interface to create/update Roles
type RoleCreator = GenericObjectCreator[*rbacv1.Role]

// NamedRoleCreatorGetter returns the name of the resource and the corresponding creator function
type NamedRoleCreatorGetter = GenericNamedObjectCreator[*rbacv1.Role]

// RoleBindingCreator defines an interface to create/update RoleBindings
type RoleBindingCreator = GenericObjectCreator[*rbacv1.RoleBinding]

// NamedRoleBindingCreatorGetter returns the name of the resource and the corresponding creator function
type NamedRoleBindingCreatorGetter = GenericNamedObjectCreator[*rbacv1.RoleBinding]

// CustomResourceDefinitionCreator defines an interface to create/update CustomResourceDefinitions
type CustomResourceDefinitionCreator = GenericObjectCreator[*apiextensionsv1.CustomResourceDefinition]

// NamedCustomResourceDefinitionCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCustomResourceDefinitionCreatorGetter = GenericNamedObjectCreator[*apiextensionsv1.CustomResourceDefinition]

// CronJobCreator defines an interface to create/update CronJobs
type CronJobCreator = GenericObjectCreator[*batchv1beta1.CronJob]

// NamedCronJobCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCronJobCreatorGetter = GenericNamedObjectCreator[*batchv1beta1.CronJob]

// MutatingWebhookConfigurationCreator defines an interface to create/update MutatingWebhookConfigurations
type MutatingWebhookConfigurationCreator = GenericObjectCreator[*admissionregistrationv1.MutatingWebhookConfiguration]

// NamedMutatingWebhookConfigurationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedMutatingWebhookConfigurationCreatorGetter = GenericNamedObjectCreator[*admissionregistrationv1.MutatingWebhookConfiguration]

// ValidatingWebhookConfigurationCreator defines an interface to create/update ValidatingWebhookConfigurations
type ValidatingWebhookConfigurationCreator = GenericObjectCreator[*admissionregistrationv1.ValidatingWebhookConfiguration]

// NamedValidatingWebhookConfigurationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedValidatingWebhookConfigurationCreatorGetter = GenericNamedObjectCreator[*admissionregistrationv1.ValidatingWebhookConfiguration]

// APIServiceCreator defines an interface to create/update APIServices
type APIServiceCreator = GenericObjectCreator[*apiregistrationv1.APIService]

// NamedAPIServiceCreatorGetter returns the name of the resource and the corresponding creator function
type NamedAPIServiceCreatorGetter = GenericNamedObjectCreator[*apiregistrationv1.APIService]

// IngressCreator defines an interface to create/update Ingresses
type IngressCreator = GenericObjectCreator[*networkingv1.Ingress]

// NamedIngressCreatorGetter returns the name of the resource and the corresponding creator function
type NamedIngressCreatorGetter = GenericNamedObjectCreator[*networkingv1.Ingress]

// KubermaticConfigurationCreator defines an interface to create/update KubermaticConfigurations
type KubermaticConfigurationCreator = GenericObjectCreator[*kubermaticv1.KubermaticConfiguration]

// NamedKubermaticConfigurationCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticConfigurationCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.KubermaticConfiguration]

// SeedCreator defines an interface to create/update Seeds
type SeedCreator = GenericObjectCreator[*kubermaticv1.Seed]

// NamedSeedCreatorGetter returns the name of the resource and the corresponding creator function
type NamedSeedCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.Seed]

// EtcdBackupConfigCreator defines an interface to create/update EtcdBackupConfigs
type EtcdBackupConfigCreator = GenericObjectCreator[*kubermaticv1.EtcdBackupConfig]

// NamedEtcdBackupConfigCreatorGetter returns the name of the resource and the corresponding creator function
type NamedEtcdBackupConfigCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.EtcdBackupConfig]

// ConstraintTemplateCreator defines an interface to create/update ConstraintTemplates
type ConstraintTemplateCreator = GenericObjectCreator[*gatekeeperv1.ConstraintTemplate]

// NamedConstraintTemplateCreatorGetter returns the name of the resource and the corresponding creator function
type NamedConstraintTemplateCreatorGetter = GenericNamedObjectCreator[*gatekeeperv1.ConstraintTemplate]

// KubermaticV1ConstraintTemplateCreator defines an interface to create/update ConstraintTemplates
type KubermaticV1ConstraintTemplateCreator = GenericObjectCreator[*kubermaticv1.ConstraintTemplate]

// NamedKubermaticV1ConstraintTemplateCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ConstraintTemplateCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.ConstraintTemplate]

// KubermaticV1ProjectCreator defines an interface to create/update Projects
type KubermaticV1ProjectCreator = GenericObjectCreator[*kubermaticv1.Project]

// NamedKubermaticV1ProjectCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ProjectCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.Project]

// KubermaticV1UserProjectBindingCreator defines an interface to create/update UserProjectBindings
type KubermaticV1UserProjectBindingCreator = GenericObjectCreator[*kubermaticv1.UserProjectBinding]

// NamedKubermaticV1UserProjectBindingCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1UserProjectBindingCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.UserProjectBinding]

// KubermaticV1ConstraintCreator defines an interface to create/update Constraints
type KubermaticV1ConstraintCreator = GenericObjectCreator[*kubermaticv1.Constraint]

// NamedKubermaticV1ConstraintCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ConstraintCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.Constraint]

// KubermaticV1UserCreator defines an interface to create/update Users
type KubermaticV1UserCreator = GenericObjectCreator[*kubermaticv1.User]

// NamedKubermaticV1UserCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1UserCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.User]

// KubermaticV1ClusterTemplateCreator defines an interface to create/update ClusterTemplates
type KubermaticV1ClusterTemplateCreator = GenericObjectCreator[*kubermaticv1.ClusterTemplate]

// NamedKubermaticV1ClusterTemplateCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ClusterTemplateCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.ClusterTemplate]

// NetworkPolicyCreator defines an interface to create/update NetworkPolicies
type NetworkPolicyCreator = GenericObjectCreator[*networkingv1.NetworkPolicy]

// NamedNetworkPolicyCreatorGetter returns the name of the resource and the corresponding creator function
type NamedNetworkPolicyCreatorGetter = GenericNamedObjectCreator[*networkingv1.NetworkPolicy]

// KubermaticV1RuleGroupCreator defines an interface to create/update RuleGroups
type KubermaticV1RuleGroupCreator = GenericObjectCreator[*kubermaticv1.RuleGroup]

// NamedKubermaticV1RuleGroupCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1RuleGroupCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.RuleGroup]

// AppsKubermaticV1ApplicationDefinitionCreator defines an interface to create/update ApplicationDefinitions
type AppsKubermaticV1ApplicationDefinitionCreator = GenericObjectCreator[*appskubermaticv1.ApplicationDefinition]

// NamedAppsKubermaticV1ApplicationDefinitionCreatorGetter returns the name of the resource and the corresponding creator function
type NamedAppsKubermaticV1ApplicationDefinitionCreatorGetter = GenericNamedObjectCreator[*appskubermaticv1.ApplicationDefinition]

// KubeVirtV1VirtualMachineInstancePresetCreator defines an interface to create/update VirtualMachineInstancePresets
type KubeVirtV1VirtualMachineInstancePresetCreator = GenericObjectCreator[*kubevirtv1.VirtualMachineInstancePreset]

// NamedKubeVirtV1VirtualMachineInstancePresetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubeVirtV1VirtualMachineInstancePresetCreatorGetter = GenericNamedObjectCreator[*kubevirtv1.VirtualMachineInstancePreset]

// KubermaticV1PresetCreator defines an interface to create/update Presets
type KubermaticV1PresetCreator = GenericObjectCreator[*kubermaticv1.Preset]

// NamedKubermaticV1PresetCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1PresetCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.Preset]

// CDIv1beta1DataVolumeCreator defines an interface to create/update DataVolumes
type CDIv1beta1DataVolumeCreator = GenericObjectCreator[*cdiv1beta1.DataVolume]

// NamedCDIv1beta1DataVolumeCreatorGetter returns the name of the resource and the corresponding creator function
type NamedCDIv1beta1DataVolumeCreatorGetter = GenericNamedObjectCreator[*cdiv1beta1.DataVolume]

// KubermaticV1ResourceQuotaCreator defines an interface to create/update ResourceQuotas
type KubermaticV1ResourceQuotaCreator = GenericObjectCreator[*kubermaticv1.ResourceQuota]

// NamedKubermaticV1ResourceQuotaCreatorGetter returns the name of the resource and the corresponding creator function
type NamedKubermaticV1ResourceQuotaCreatorGetter = GenericNamedObjectCreator[*kubermaticv1.ResourceQuota]
