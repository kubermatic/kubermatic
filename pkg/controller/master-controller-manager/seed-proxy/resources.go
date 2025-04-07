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

package seedproxy

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
)

func secretName(seed *kubermaticv1.Seed) string {
	return fmt.Sprintf("seed-proxy-%s", seed.Name)
}

func deploymentName(seed *kubermaticv1.Seed) string {
	return fmt.Sprintf("seed-proxy-%s", seed.Name)
}

func serviceName(seed *kubermaticv1.Seed) string {
	return fmt.Sprintf("seed-proxy-%s", seed.Name)
}

func seedMonitoringRoleName(seed *kubermaticv1.Seed) string {
	return fmt.Sprintf("seed-proxy-%s", seed.Namespace)
}

func seedMonitoringRoleBindingName(seed *kubermaticv1.Seed) string {
	return fmt.Sprintf("seed-proxy-%s", seed.Namespace)
}

func ensureDefaultLabels(existing map[string]string, name string, instance string) map[string]string {
	if existing == nil {
		existing = map[string]string{}
	}

	existing[NameLabel] = name
	existing[ManagedByLabel] = ControllerName

	if instance != "" {
		existing[InstanceLabel] = instance
	}

	return existing
}

func ownerReferences(secret *corev1.Secret) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(secret, secret.GroupVersionKind()),
	}
}

func seedServiceAccountReconciler(seed *kubermaticv1.Seed) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return SeedServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = ensureDefaultLabels(sa.Labels, SeedServiceAccountName, "")

			return sa, nil
		}
	}
}

func seedSecretReconciler(seed *kubermaticv1.Seed) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return SeedSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Labels = ensureDefaultLabels(s.Labels, SeedSecretName, seed.Name)

			// ensure Kubernetes has enough info to fill in the SA token
			s.Type = corev1.SecretTypeServiceAccountToken

			if s.Annotations == nil {
				s.Annotations = map[string]string{}
			}

			s.Annotations[corev1.ServiceAccountNameKey] = SeedServiceAccountName

			return s, nil
		}
	}
}

func seedMonitoringRoleReconciler(seed *kubermaticv1.Seed) reconciling.NamedRoleReconcilerFactory {
	name := seedMonitoringRoleName(seed)

	return func() (string, reconciling.RoleReconciler) {
		return name, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = ensureDefaultLabels(r.Labels, name, "")

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Verbs:     []string{"get"},
					Resources: []string{"services/proxy"},
					ResourceNames: []string{
						SeedPrometheusService,
						SeedAlertmanagerService,
					},
				},
			}

			return r, nil
		}
	}
}

func seedMonitoringRoleBindingReconciler(seed *kubermaticv1.Seed) reconciling.NamedRoleBindingReconcilerFactory {
	name := seedMonitoringRoleBindingName(seed)

	return func() (string, reconciling.RoleBindingReconciler) {
		return name, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = ensureDefaultLabels(rb.Labels, name, "")

			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     seedMonitoringRoleName(seed),
			}

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      SeedServiceAccountName,
					Namespace: seed.Namespace,
				},
			}

			return rb, nil
		}
	}
}

func masterSecretReconciler(seed *kubermaticv1.Seed, kubeconfig *rest.Config, credentials *corev1.Secret) reconciling.NamedSecretReconcilerFactory {
	name := secretName(seed)
	host := kubeconfig.Host
	var proxy string
	if kubeconfig.Proxy != nil {
		if proxyUrl, err := kubeconfig.Proxy(nil); err == nil && proxyUrl != nil {
			proxy = proxyUrl.String()
		}
	}

	return func() (string, reconciling.SecretReconciler) {
		return name, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Labels = ensureDefaultLabels(s.Labels, "seed-proxy", seed.Name)

			if s.Data == nil {
				s.Data = make(map[string][]byte)
			}

			// convert the service account CA and token into a handy kubeconfig so that
			// consuming the credentials becomes easier later on
			kubeconfig, err := convertServiceAccountToKubeconfig(host, credentials, &proxy)

			if err != nil {
				return s, fmt.Errorf("failed to create kubeconfig: %w", err)
			}

			s.Data["kubeconfig"] = kubeconfig

			return s, nil
		}
	}
}

func convertServiceAccountToKubeconfig(host string, credentials *corev1.Secret, proxyUrl *string) ([]byte, error) {
	clusterName := "seed"
	contextName := "default"
	authName := "token-based"

	cluster := api.NewCluster()
	cluster.CertificateAuthorityData = credentials.Data[corev1.ServiceAccountRootCAKey]
	cluster.Server = host
	if proxyUrl != nil {
		cluster.ProxyURL = *proxyUrl
	}

	context := api.NewContext()
	context.Cluster = clusterName
	context.AuthInfo = authName

	user := api.NewAuthInfo()
	user.Token = string(credentials.Data[corev1.ServiceAccountTokenKey])

	kubeconfig := api.NewConfig()
	kubeconfig.Clusters[clusterName] = cluster
	kubeconfig.Contexts[contextName] = context
	kubeconfig.AuthInfos[authName] = user
	kubeconfig.CurrentContext = contextName

	return clientcmd.Write(*kubeconfig)
}

func masterDeploymentReconciler(seed *kubermaticv1.Seed, secret *corev1.Secret, imageRewriter registry.ImageRewriter) reconciling.NamedDeploymentReconcilerFactory {
	name := deploymentName(seed)

	return func() (string, reconciling.DeploymentReconciler) {
		return name, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := func() map[string]string {
				return map[string]string{
					NameLabel:     MasterDeploymentName,
					InstanceLabel: seed.Name,
				}
			}

			probe := corev1.Probe{
				InitialDelaySeconds: 3,
				TimeoutSeconds:      2,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				FailureThreshold:    3,
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.Parse("http"),
					},
				},
			}

			d.OwnerReferences = ownerReferences(secret)
			d.Labels = ensureDefaultLabels(d.Labels, MasterDeploymentName, seed.Name)

			d.Spec.Replicas = ptr.To[int32](1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels(),
			}

			d.Spec.Template.Labels = labels()
			d.Spec.Template.Labels["prometheus.io/scrape"] = "true"
			d.Spec.Template.Labels["prometheus.io/port"] = fmt.Sprintf("%d", KubectlProxyPort)

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "proxy",
					Image:   registry.Must(imageRewriter(resources.RegistryQuay + "/kubermatic/util:2.5.0")),
					Command: []string{"/bin/bash"},
					Args:    []string{"-c", strings.TrimSpace(proxyScript)},
					Env: []corev1.EnvVar{
						{
							Name:  "KUBERNETES_CONTEXT",
							Value: seed.Name,
						},
						{
							Name:  "KUBECONFIG",
							Value: "/opt/kubeconfig/kubeconfig",
						},
						{
							Name:  "PROXY_PORT",
							Value: fmt.Sprintf("%d", KubectlProxyPort),
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: KubectlProxyPort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: "/opt/kubeconfig",
							Name:      "kubeconfig",
							ReadOnly:  true,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("24Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
					ReadinessProbe: &probe,
				},
			}

			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "kubeconfig",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secret.Name,
						},
					},
				},
			}

			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: resources.ImagePullSecretName,
				},
			}

			return d, nil
		}
	}
}

func masterServiceReconciler(seed *kubermaticv1.Seed, secret *corev1.Secret) reconciling.NamedServiceReconcilerFactory {
	name := serviceName(seed)

	return func() (string, reconciling.ServiceReconciler) {
		return name, func(s *corev1.Service) (*corev1.Service, error) {
			s.OwnerReferences = ownerReferences(secret)

			s.Labels = ensureDefaultLabels(s.Labels, MasterServiceName, seed.Name)

			s.Spec.Ports = []corev1.ServicePort{
				{
					Name: "http",
					Port: KubectlProxyPort,
					TargetPort: intstr.IntOrString{
						IntVal: KubectlProxyPort,
					},
					Protocol: corev1.ProtocolTCP,
				},
			}

			s.Spec.Selector = map[string]string{
				NameLabel:     MasterServiceName,
				InstanceLabel: seed.Name,
			}

			return s, nil
		}
	}
}

func (r *Reconciler) masterGrafanaConfigmapReconciler(seeds map[string]*kubermaticv1.Seed) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return MasterGrafanaConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			c.Data = make(map[string]string)

			c.Labels = ensureDefaultLabels(c.Labels, MasterGrafanaConfigMapName, "")

			for seedName, seed := range seeds {
				filename := fmt.Sprintf("prometheus-%s.yaml", seedName)

				config, err := buildGrafanaDatasource(seed)
				if err != nil {
					return nil, fmt.Errorf("failed to build Grafana config for seed %s: %w", seedName, err)
				}

				c.Data[filename] = config
			}

			return c, nil
		}
	}
}

func buildGrafanaDatasource(seed *kubermaticv1.Seed) (string, error) {
	data := map[string]interface{}{
		"ContextName":             seed.Name,
		"ServiceName":             serviceName(seed),
		"ServiceNamespace":        seed.Namespace,
		"ProxyPort":               KubectlProxyPort,
		"SeedMonitoringNamespace": SeedMonitoringNamespace,
		"SeedPrometheusService":   SeedPrometheusService,
	}

	var buffer bytes.Buffer

	tpl := template.Must(template.New("base").Parse(grafanaDatasource))
	err := tpl.Execute(&buffer, data)

	return strings.TrimSpace(buffer.String()), err
}

const proxyScript = `
set -euo pipefail

echo "Starting kubectl proxy for $KUBERNETES_CONTEXT on port $PROXY_PORT..."

kubectl proxy \
  --address=0.0.0.0 \
  --port=$PROXY_PORT \
  --accept-hosts='^.*'
`

const grafanaDatasource = `
# This file has been generated by the Kubermatic master-controller-manager.
apiVersion: 1
datasources:
- version: 1
  name: Seed {{ .ContextName }}
  org_id: 1
  type: prometheus
  access: proxy
  url: http://{{ .ServiceName }}.{{ .ServiceNamespace }}.svc.cluster.local:{{ .ProxyPort }}/api/v1/namespaces/{{ .SeedMonitoringNamespace }}/services/{{ .SeedPrometheusService }}/proxy/
  editable: false
`
