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

package kubermatic

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterAdmissionWebhookName         = "kubermatic-clusters"
	AddonAdmissionWebhookName           = "kubermatic-addons"
	MLAAdminSettingAdmissionWebhookName = "kubermatic-mlaadminsettings"
	OSCAdmissionWebhookName             = "kubermatic-operating-system-configs"
	OSPAdmissionWebhookName             = "kubermatic-operating-system-profiles"
	IPAMPoolAdmissionWebhookName        = "kubermatic-ipampools"
)

func ClusterValidatingWebhookConfigurationCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return ClusterAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.ClusterScope

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "clusters.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-kubermatic-k8c-io-v1-cluster"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"clusters"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func ClusterMutatingWebhookConfigurationCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedMutatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.MutatingWebhookConfigurationCreator) {
		return ClusterAdmissionWebhookName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.ClusterScope

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.MutatingWebhook{
				{
					Name:                    "clusters.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					ReinvocationPolicy:      &reinvocationPolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/mutate-kubermatic-k8c-io-v1-cluster"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"clusters"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func AddonMutatingWebhookConfigurationCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedMutatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.MutatingWebhookConfigurationCreator) {
		return AddonAdmissionWebhookName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.NamespacedScope

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.MutatingWebhook{
				{
					Name:                    "addons.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					ReinvocationPolicy:      &reinvocationPolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(10),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/mutate-kubermatic-k8c-io-v1-addon"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"addons"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func MLAAdminSettingMutatingWebhookConfigurationCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedMutatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.MutatingWebhookConfigurationCreator) {
		return MLAAdminSettingAdmissionWebhookName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.NamespacedScope

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.MutatingWebhook{
				{
					Name:                    "mlaadminsettings.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					ReinvocationPolicy:      &reinvocationPolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(10),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/mutate-kubermatic-k8c-io-v1-mlaadminsetting"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"mlaadminsettings"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func OperatingSystemConfigValidatingWebhookConfigurationCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return OSCAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.AllScopes

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "operatingsystemconfigs.operatingsystemmanager.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-operating-system-config"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{osmv1alpha1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"operatingsystemconfigs"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func OperatingSystemProfileValidatingWebhookConfigurationCreator(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return OSPAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.AllScopes

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "operatingsystemprofiles.operatingsystemmanager.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-operating-system-profile"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{osmv1alpha1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"operatingsystemprofiles"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func IPAMPoolValidatingWebhookConfigurationCreator(ctx context.Context,
	cfg *kubermaticv1.KubermaticConfiguration,
	client ctrlruntimeclient.Client,
) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return IPAMPoolAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.ClusterScope

			ca, err := common.WebhookCABundle(ctx, cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find webhook CA bundle: %w", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "ipampools.kubermatic.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-kubermatic-k8c-io-v1-ipampool"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"ipampools"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}
