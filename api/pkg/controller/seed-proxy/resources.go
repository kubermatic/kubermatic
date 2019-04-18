package seedproxy

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func secretName(contextName string) string {
	return fmt.Sprintf("seed-proxy-%s", contextName)
}

func deploymentName(contextName string) string {
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
	// secretName := secretName(contextName)

	return func() (string, reconciling.DeploymentCreator) {
		return name, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := map[string]string{
				"app.kubernetes.io/name":       "seed-proxy",
				"app.kubernetes.io/instance":   contextName,
				"app.kubernetes.io/managed-by": ControllerName,
			}

			d.Name = name
			d.Namespace = KubermaticNamespace
			d.Labels = labels

			d.Spec.Replicas = i32ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			d.Spec.Template.Labels = labels

			return d, nil
		}
	}
}

func i32ptr(i int32) *int32 {
	return &i
}
