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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/pointer"
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

func defaultLabels(name string, instance string) map[string]string {
	labels := map[string]string{
		NameLabel:      name,
		ManagedByLabel: ControllerName,
	}

	if instance != "" {
		labels[InstanceLabel] = instance
	}

	return labels
}

func ownerReferences(secret *corev1.Secret) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(secret, secret.GroupVersionKind()),
	}
}

func seedServiceAccountCreator(seed *kubermaticv1.Seed) reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return SeedServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = defaultLabels(SeedServiceAccountName, "")

			return sa, nil
		}
	}
}

func seedMonitoringRoleCreator(seed *kubermaticv1.Seed) reconciling.NamedRoleCreatorGetter {
	name := seedMonitoringRoleName(seed)

	return func() (string, reconciling.RoleCreator) {
		return name, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = defaultLabels(name, "")

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

func seedMonitoringRoleBindingCreator(seed *kubermaticv1.Seed) reconciling.NamedRoleBindingCreatorGetter {
	name := seedMonitoringRoleBindingName(seed)

	return func() (string, reconciling.RoleBindingCreator) {
		return name, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = defaultLabels(name, "")

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

func masterSecretCreator(seed *kubermaticv1.Seed, kubeconfig *rest.Config, credentials *corev1.Secret) reconciling.NamedSecretCreatorGetter {
	name := secretName(seed)
	host := kubeconfig.Host

	return func() (string, reconciling.SecretCreator) {
		return name, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Labels = defaultLabels("seed-proxy", seed.Name)

			if s.Data == nil {
				s.Data = make(map[string][]byte)
			}

			// convert the service account CA and token into a handy kubeconfig so that
			// consuming the credentials becomes easier later on
			kubeconfig, err := convertServiceAccountToKubeconfig(host, credentials)
			if err != nil {
				return s, fmt.Errorf("failed to create kubeconfig: %v", err)
			}

			s.Data["kubeconfig"] = kubeconfig

			return s, nil
		}
	}
}

func convertServiceAccountToKubeconfig(host string, credentials *corev1.Secret) ([]byte, error) {
	clusterName := "seed"
	contextName := "default"
	authName := "token-based"

	cluster := api.NewCluster()
	cluster.CertificateAuthorityData = credentials.Data["ca.crt"]
	cluster.Server = host

	context := api.NewContext()
	context.Cluster = clusterName
	context.AuthInfo = authName

	user := api.NewAuthInfo()
	user.Token = string(credentials.Data["token"])

	kubeconfig := api.NewConfig()
	kubeconfig.Clusters[clusterName] = cluster
	kubeconfig.Contexts[contextName] = context
	kubeconfig.AuthInfos[authName] = user
	kubeconfig.CurrentContext = contextName

	return clientcmd.Write(*kubeconfig)
}

func masterDeploymentCreator(seed *kubermaticv1.Seed, secret *corev1.Secret) reconciling.NamedDeploymentCreatorGetter {
	name := deploymentName(seed)

	return func() (string, reconciling.DeploymentCreator) {
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
				Handler: corev1.Handler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.Parse("http"),
					},
				},
			}

			d.OwnerReferences = ownerReferences(secret)
			d.Labels = labels()
			d.Labels[ManagedByLabel] = ControllerName

			d.Spec.Replicas = pointer.Int32Ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels(),
			}

			d.Spec.Template.Labels = labels()
			d.Spec.Template.Labels["prometheus.io/scrape"] = "true"
			d.Spec.Template.Labels["prometheus.io/port"] = fmt.Sprintf("%d", KubectlProxyPort)

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "proxy",
					Image:   "quay.io/kubermatic/util:2.0.0",
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
							corev1.ResourceMemory: resource.MustParse("32Mi"),
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

			return d, nil
		}
	}
}

func masterServiceCreator(seed *kubermaticv1.Seed, secret *corev1.Secret) reconciling.NamedServiceCreatorGetter {
	name := serviceName(seed)

	return func() (string, reconciling.ServiceCreator) {
		return name, func(s *corev1.Service) (*corev1.Service, error) {
			s.OwnerReferences = ownerReferences(secret)
			s.Labels = map[string]string{
				NameLabel:      MasterServiceName,
				InstanceLabel:  seed.Name,
				ManagedByLabel: ControllerName,
			}

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

func (r *Reconciler) masterGrafanaConfigmapCreator(seeds map[string]*kubermaticv1.Seed) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return MasterGrafanaConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			labels := func() map[string]string {
				return map[string]string{
					NameLabel: MasterGrafanaConfigMapName,
				}
			}

			c.Data = make(map[string]string)

			c.Labels = labels()
			c.Labels[ManagedByLabel] = ControllerName

			for seedName, seed := range seeds {
				filename := fmt.Sprintf("prometheus-%s.yaml", seedName)

				config, err := buildGrafanaDatasource(seed)
				if err != nil {
					return nil, fmt.Errorf("failed to build Grafana config for seed %s: %v", seedName, err)
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
  --accept-hosts=''
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
