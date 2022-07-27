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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func apiPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel: APIDeploymentName,
	}
}

func APIServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return apiServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func APIClusterRoleName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:%s-api", cfg.Namespace, cfg.Name)
}

func APIClusterRoleCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleCreatorGetter {
	name := APIClusterRoleName(cfg)

	return func() (string, reconciling.ClusterRoleCreator) {
		return name, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kubermatic.k8c.io", "apps.kubermatic.k8c.io"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"operatingsystemmanager.k8c.io"},
					Resources: []string{"operatingsystemprofiles"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"get", "list", "watch", "create", "patch"},
				},
				// TODO: Maybe split this out into a dedicated ClusterRole and
				// dynamically manage the resourceNames, so this isn't too broad
				{
					APIGroups: []string{""},
					Resources: []string{"users", "groups", "serviceaccounts"},
					Verbs:     []string{"impersonate"},
				},
			}

			return cr, nil
		}
	}
}

func APIRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return apiServiceAccountName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"*"},
				},
			}

			return r, nil
		}
	}
}

func APIClusterRoleBindingName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:%s-api", cfg.Namespace, cfg.Name)
}

func APIClusterRoleBindingCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleBindingCreatorGetter {
	name := APIClusterRoleBindingName(cfg)

	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     APIClusterRoleName(cfg),
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      apiServiceAccountName,
					Namespace: cfg.Namespace,
				},
			}

			return crb, nil
		}
	}
}

func APIRoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return apiServiceAccountName, func(crb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     apiServiceAccountName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind: rbacv1.ServiceAccountKind,
					Name: apiServiceAccountName,
				},
			}

			return crb, nil
		}
	}
}

func APIDeploymentCreator(cfg *kubermaticv1.KubermaticConfiguration, workerName string, versions kubermatic.Versions) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return APIDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			probe := corev1.Probe{
				InitialDelaySeconds: 3,
				TimeoutSeconds:      2,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				FailureThreshold:    3,
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/api/v1/healthz",
						Scheme: corev1.URISchemeHTTP,
						Port:   intstr.FromInt(8080),
					},
				},
			}

			d.Spec.Replicas = cfg.Spec.API.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: apiPodLabels(),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "json_iso",
			}

			d.Spec.Template.Spec.ServiceAccountName = apiServiceAccountName

			volumes := []corev1.Volume{
				{
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cfg.Spec.CABundle.Name,
							},
						},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					Name:      "ca-bundle",
					MountPath: "/opt/ca-bundle/",
					ReadOnly:  true,
				},
			}

			args := []string{
				"-logtostderr",
				"-address=0.0.0.0:8080",
				"-internal-address=0.0.0.0:8085",
				"-swagger=/opt/swagger.json",
				fmt.Sprintf("-ca-bundle=/opt/ca-bundle/%s", resources.CABundleConfigMapKey),
				fmt.Sprintf("-namespace=%s", cfg.Namespace),
				fmt.Sprintf("-oidc-url=%s", cfg.Spec.Auth.TokenIssuer),
				fmt.Sprintf("-oidc-authenticator-client-id=%s", cfg.Spec.Auth.ClientID),
				fmt.Sprintf("-oidc-skip-tls-verify=%v", cfg.Spec.Auth.SkipTokenIssuerTLSVerify),
				fmt.Sprintf("-domain=%s", cfg.Spec.Ingress.Domain),
				fmt.Sprintf("-service-account-signing-key=%s", cfg.Spec.Auth.ServiceAccountKey),
				fmt.Sprintf("-expose-strategy=%s", cfg.Spec.ExposeStrategy),
				fmt.Sprintf("-feature-gates=%s", common.StringifyFeatureGates(cfg)),
				fmt.Sprintf("-pprof-listen-address=%s", *cfg.Spec.API.PProfEndpoint),
			}

			if cfg.Spec.API.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			if cfg.Spec.FeatureGates[features.OIDCKubeCfgEndpoint] {
				args = append(
					args,
					fmt.Sprintf("-oidc-issuer-redirect-uri=%s", cfg.Spec.Auth.IssuerRedirectURL),
					fmt.Sprintf("-oidc-issuer-client-id=%s", cfg.Spec.Auth.IssuerClientID),
					fmt.Sprintf("-oidc-issuer-client-secret=%s", cfg.Spec.Auth.IssuerClientSecret),
					fmt.Sprintf("-oidc-issuer-cookie-hash-key=%s", cfg.Spec.Auth.IssuerCookieKey),
				)
			}

			if workerName != "" {
				args = append(args, fmt.Sprintf("-worker-name=%s", workerName))
			}

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "api",
					Image:   cfg.Spec.API.DockerRepository + ":" + versions.Kubermatic,
					Command: []string{"kubermatic-api"},
					Args:    args,
					Env:     common.ProxyEnvironmentVars(cfg),
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts:   volumeMounts,
					Resources:      cfg.Spec.API.Resources,
					ReadinessProbe: &probe,
				},
			}

			return d, nil
		}
	}
}

func APIPDBCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetCreatorGetter {
	name := "kubermatic-api"

	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return name, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			// To prevent the PDB from blocking node rotations, we accept
			// 0 minAvailable if the replica count is only 1.
			// NB: The cfg is defaulted, so Replicas==nil cannot happen.
			min := intstr.FromInt(1)
			if cfg.Spec.API.Replicas != nil && *cfg.Spec.API.Replicas < 2 {
				min = intstr.FromInt(0)
			}

			pdb.Spec.MinAvailable = &min
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: apiPodLabels(),
			}

			return pdb, nil
		}
	}
}

func APIServiceCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return apiServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeNodePort
			s.Spec.Selector = apiPodLabels()

			if len(s.Spec.Ports) < 2 {
				s.Spec.Ports = make([]corev1.ServicePort, 2)
			}

			s.Spec.Ports[0].Name = "http"
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromInt(8080)
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP

			s.Spec.Ports[1].Name = "metrics"
			s.Spec.Ports[1].Port = 8085
			s.Spec.Ports[1].TargetPort = intstr.FromInt(8080)
			s.Spec.Ports[1].Protocol = corev1.ProtocolTCP

			return s, nil
		}
	}
}
