package seedproxy

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func secretName(contextName string) string {
	return fmt.Sprintf("seed-proxy-%s", contextName)
}

func deploymentName(contextName string) string {
	return fmt.Sprintf("seed-proxy-%s", contextName)
}

func serviceName(contextName string) string {
	return fmt.Sprintf("seed-proxy-%s", contextName)
}

func seedServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return ServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = map[string]string{
				"app.kubernetes.io/name":       ServiceAccountName,
				"app.kubernetes.io/managed-by": ControllerName,
			}

			return sa, nil
		}
	}
}

func seedPrometheusRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return SeedPrometheusRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = SeedPrometheusRoleName
			r.Namespace = SeedPrometheusNamespace
			r.Labels = map[string]string{
				"app.kubernetes.io/name":       SeedPrometheusRoleName,
				"app.kubernetes.io/managed-by": ControllerName,
			}

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"services/proxy"},
					Verbs:     []string{"get", "list", "watch", "create"},
				},
			}

			return r, nil
		}
	}
}

func seedPrometheusRoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return SeedPrometheusRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Name = SeedPrometheusRoleBindingName
			rb.Namespace = SeedPrometheusNamespace
			rb.Labels = map[string]string{
				"app.kubernetes.io/name":       SeedPrometheusRoleName,
				"app.kubernetes.io/managed-by": ControllerName,
			}

			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     SeedPrometheusRoleName,
			}

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      SeedPrometheusRoleName,
					Namespace: ServiceAccountNamespace,
				},
			}

			return rb, nil
		}
	}
}

func masterSecretCreator(contextName string, credentials *corev1.Secret) reconciling.NamedSecretCreatorGetter {
	name := secretName(contextName)

	return func() (string, reconciling.SecretCreator) {
		return name, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Name = name
			s.Namespace = KubermaticNamespace
			s.Labels = map[string]string{
				"app.kubernetes.io/name":       "seed-proxy",
				"app.kubernetes.io/instance":   contextName,
				"app.kubernetes.io/managed-by": ControllerName,
			}

			if s.Data == nil {
				s.Data = make(map[string][]byte)
			}

			for k, v := range credentials.Data {
				s.Data[k] = v
			}

			return s, nil
		}
	}
}

func masterDeploymentCreator(contextName string) reconciling.NamedDeploymentCreatorGetter {
	name := deploymentName(contextName)
	secretName := secretName(contextName)

	return func() (string, reconciling.DeploymentCreator) {
		return name, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := func() map[string]string {
				return map[string]string{
					"app.kubernetes.io/name":     "seed-proxy",
					"app.kubernetes.io/instance": contextName,
				}
			}

			d.Name = name
			d.Namespace = KubermaticNamespace
			d.Labels = labels()
			d.Labels["app.kubernetes.io/managed-by"] = ControllerName

			d.Spec.Replicas = i32ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels(),
			}

			d.Spec.Template.Labels = labels()
			d.Spec.Template.Labels["prometheus.io/scrape"] = "true"
			d.Spec.Template.Labels["prometheus.io/port"] = "8001"

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "prometheus",
					Image:   "quay.io/kubermatic/util:1.0.0-2",
					Command: []string{"/bin/bash"},
					Args:    []string{"-c", strings.TrimSpace(proxyScript)},
					Env: []corev1.EnvVar{
						{
							Name:  "KUBERNETES_CONTEXT",
							Value: contextName,
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8001,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: "var/run/secrets/kubernetes.io/serviceaccount/",
							Name:      "serviceaccount",
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
				},
			}

			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "serviceaccount",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretName,
						},
					},
				},
			}

			return d, nil
		}
	}
}

func masterServiceCreator(contextName string) reconciling.NamedServiceCreatorGetter {
	name := serviceName(contextName)

	return func() (string, reconciling.ServiceCreator) {
		return name, func(s *corev1.Service) (*corev1.Service, error) {
			labels := func() map[string]string {
				return map[string]string{
					"app.kubernetes.io/name":     "seed-proxy",
					"app.kubernetes.io/instance": contextName,
				}
			}

			s.Name = name
			s.Namespace = KubermaticNamespace

			s.Labels = labels()
			s.Labels["app.kubernetes.io/managed-by"] = ControllerName

			s.Spec.Ports = []corev1.ServicePort{
				{
					Name: "http",
					Port: 8001,
					TargetPort: intstr.IntOrString{
						StrVal: "http",
					},
				},
			}

			s.Spec.Selector = labels()

			return s, nil
		}
	}
}

func i32ptr(i int32) *int32 {
	return &i
}

const proxyScript = `
set -euo pipefail

echo "Starting kubectl proxy for $KUBERNETES_CONTEXT on port 8001..."

while true; do
  kubectl proxy \
	 --address=0.0.0.0 \
	 --port=8001 \
	 --accept-hosts='' \
	 --accept-paths='^/api/v1/namespaces/monitoring/services/.*/proxy/.*$' \
	 --reject-methods='^(POST|PUT|PATCH|DELETE)$'
  sleep 1
done
`
