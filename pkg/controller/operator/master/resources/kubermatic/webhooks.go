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

package kubermatic

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func UserSSHKeyMutatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationReconciler) {
		return common.UserSSHKeyAdmissionWebhookName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
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
					Name:                    "usersshkeys.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					ReinvocationPolicy:      &reinvocationPolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/mutate-kubermatic-k8c-io-v1-usersshkey"),
							Port:      ptr.To[int32](443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"usersshkeys"},
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

func UserSSHKeyValidatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return common.UserSSHKeyAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
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
					Name:                    "usersshkeys.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-kubermatic-k8c-io-v1-usersshkey"),
							Port:      ptr.To[int32](443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"usersshkeys"},
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

func UserValidatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return common.UserAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
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
					Name:                    "users.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-kubermatic-k8c-io-v1-user"),
							Port:      ptr.To[int32](443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"users"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
								admissionregistrationv1.Delete,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func ResourceQuotaValidatingWebhookConfigurationReconciler(ctx context.Context,
	cfg *kubermaticv1.KubermaticConfiguration,
	client ctrlruntimeclient.Client,
) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return common.ResourceQuotaAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
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
					Name:                    "resourcequotas.kubermatic.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-kubermatic-k8c-io-v1-resourcequota"),
							Port:      ptr.To[int32](443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"resourcequotas"},
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

func ExternalClusterMutatingWebhookConfigurationReconciler(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationReconciler) {
		return common.ExternalClusterAdmissionWebhookName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
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
					Name:                    "externalclusters.kubermatic.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					ReinvocationPolicy:      &reinvocationPolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/mutate-kubermatic-k8c-io-v1-externalcluster"),
							Port:      ptr.To[int32](443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"externalclusters"},
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

func GroupProjectBindingValidatingWebhookConfigurationReconciler(ctx context.Context,
	cfg *kubermaticv1.KubermaticConfiguration,
	client ctrlruntimeclient.Client,
) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return common.GroupProjectBindingAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
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
					Name:                    "groupprojectbindings.kubermatic.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      common.WebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      ptr.To("/validate-kubermatic-k8c-io-v1-groupprojectbinding"),
							Port:      ptr.To[int32](443),
						},
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{kubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"groupprojectbindings"},
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
