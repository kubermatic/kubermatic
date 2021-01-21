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
	"fmt"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterWebhookServiceName   = "cluster-webhook"
	ClusterAdmissionWebhookName = "kubermatic-clusters"
)

func ClusterAdmissionWebhookCreator(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return ClusterAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.ClusterScope

			ca, err := common.WebhookCABundle(cfg, client)
			if err != nil {
				return nil, fmt.Errorf("cannot find Seed Admission CA bundle: %v", err)
			}

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "clusters.kubermatic.io", // this should be a FQDN
					AdmissionReviewVersions: []string{"v1beta1"},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: ca,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      clusterWebhookServiceName,
							Namespace: cfg.Namespace,
							Path:      pointer.StringPtr("/validate-kubermatic-k8s-io-cluster"),
							Port:      pointer.Int32Ptr(443),
						},
					},
					ObjectSelector: &metav1.LabelSelector{},
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

// ClusterAdmissionServiceCreator creates the Service for the Cluster Admission webhook.
// This service is created only on seed clusters.
func ClusterAdmissionServiceCreator(cfg *operatorv1alpha1.KubermaticConfiguration, client ctrlruntimeclient.Client) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return clusterWebhookServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeClusterIP

			if len(s.Spec.Ports) != 1 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = "https"
			s.Spec.Ports[0].Port = 443
			s.Spec.Ports[0].TargetPort = intstr.FromInt(8100)
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			s.Spec.Selector = map[string]string{
				common.NameLabel: common.SeedControllerManagerDeploymentName,
			}

			return s, nil
		}
	}
}
