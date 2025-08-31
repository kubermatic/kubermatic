package applicationcatalogmanager

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ApplicationCatalogManagerDeploymentName = "application-catalog-manager"

	ApplicationCatalogServiceAccountName = "application-catalog-manager"

	// Default image repository and tag
	DefaultImageRepository = "quay.io/kubermatic/application-catalog-manager"
	DefaultImageTag        = "c3221135593524a8641fdb5b4e18682f45465922"
)

var (
	// Default resource requirements for application-catalog-manager deployment.
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("500Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

func catalogManagerPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel: ApplicationCatalogManagerDeploymentName,
	}
}

func ServiceAccountReconciler() reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return ApplicationCatalogServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func catalogManagerClusterRoleName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:%s-application-catalog-manager", cfg.Namespace, cfg.Name)
}

func ClusterRoleReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleReconcilerFactory {
	name := catalogManagerClusterRoleName(cfg)

	return func() (string, reconciling.ClusterRoleReconciler) {
		return name, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"*"},
				},

				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"create", "patch"},
				},
				{
					APIGroups: []string{"apps.kubermatic.k8c.io"},
					Resources: []string{"applicationdefinitions"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"apps.kubermatic.k8c.io"},
					Resources: []string{"applicationdefinitions/status"},
					Verbs:     []string{"get", "update", "patch"},
				},
				{
					APIGroups: []string{"apps.kubermatic.k8c.io"},
					Resources: []string{"applicationdefinitions/finalizers"},
					Verbs:     []string{"get", "update", "patch", "delete"},
				},
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"kubermaticconfiguration"},
					Verbs:     []string{"get", "update", "list", "watch", "patch"},
				},
			}

			return cr, nil
		}
	}
}

func RoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return ApplicationCatalogServiceAccountName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get", "list"},
				},
			}

			return r, nil
		}
	}
}

func ClusterRoleBindingReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleBindingReconcilerFactory {
	name := catalogManagerClusterRoleName(cfg)

	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     name,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      ApplicationCatalogServiceAccountName,
					Namespace: cfg.Namespace,
				},
			}

			return crb, nil
		}
	}
}

func RoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return ApplicationCatalogServiceAccountName, func(crb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     ApplicationCatalogServiceAccountName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind: rbacv1.ServiceAccountKind,
					Name: ApplicationCatalogServiceAccountName,
				},
			}

			return crb, nil
		}
	}
}

func CatalogManagerDeploymentReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return ApplicationCatalogManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := catalogManagerPodLabels()

			d.Spec.Replicas = cfg.Spec.API.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			kubernetes.EnsureLabels(&d.Spec.Template, labels)
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "json_iso",
			})

			d.Spec.Template.Spec.ServiceAccountName = ApplicationCatalogServiceAccountName

			// TODO (buraksekili): debug log should come from cfg.
			args := []string{
				"--health-probe-address=0.0.0.0:8085",
				"--metrics-address=0.0.0.0:8080",
				fmt.Sprintf("--log-debug=%v", true),
			}

			d.Spec.Template.Spec.SecurityContext = &common.PodSecurityContext

			image := getImage(cfg)
			resources := getResources(cfg)

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "application-catalog-manager",
					Image:   image,
					Command: []string{"/usr/local/bin/manager"},
					Args:    args,
					Env:     common.KubermaticProxyEnvironmentVars(&cfg.Spec.Proxy),
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
					Resources: resources,
					ReadinessProbe: &corev1.Probe{
						InitialDelaySeconds: 3,
						TimeoutSeconds:      2,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
								Scheme: corev1.URISchemeHTTP,
								Port:   intstr.FromInt(8080),
							},
						},
					},
					LivenessProbe: &corev1.Probe{
						InitialDelaySeconds: 10,
						TimeoutSeconds:      10,
						PeriodSeconds:       15,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Scheme: corev1.URISchemeHTTP,
								Port:   intstr.FromInt(8080),
							},
						},
					},
					SecurityContext: &common.ContainerSecurityContext,
				},
			}

			return d, nil
		}
	}
}

func getResources(cfg *kubermaticv1.KubermaticConfiguration) corev1.ResourceRequirements {
	if cfg.Spec.Applications.CatalogManager.Resources.Requests != nil || cfg.Spec.Applications.CatalogManager.Resources.Limits != nil {
		return cfg.Spec.Applications.CatalogManager.Resources
	}

	return defaultResourceRequirements
}

func getImage(cfg *kubermaticv1.KubermaticConfiguration) string {
	repository := DefaultImageRepository
	if cfg.Spec.Applications.CatalogManager.Image.Repository != "" {
		repository = cfg.Spec.Applications.CatalogManager.Image.Repository
	}

	tag := DefaultImageTag
	if cfg.Spec.Applications.CatalogManager.Image.Tag != "" {
		tag = cfg.Spec.Applications.CatalogManager.Image.Tag
	}

	return repository + ":" + tag
}
