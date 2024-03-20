/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package gatekeeper

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	controllerName         = resources.GatekeeperControllerDeploymentName
	auditName              = resources.GatekeeperAuditDeploymentName
	auditEmptyDirName      = "tmp-volume"
	auditEmptyDirMountPath = "/tmp/audit"
	imageName              = "openpolicyagent/gatekeeper"
	tag                    = "v3.12.0"
	// Namespace used by Dashboard to find required resources.
	webhookServerPort  = 8443
	metricsPort        = 8888
	healthzPort        = 9090
	serviceAccountName = "gatekeeper-admin"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		controllerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		},
		auditName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		},
	}

	gatekeeperControllerLabels = map[string]string{
		"control-plane":           "controller-manager",
		"gatekeeper.sh/operation": "webhook",
		"gatekeeper.sh/system":    "yes",
	}

	gatekeeperAuditLabels = map[string]string{
		"control-plane":           "audit",
		"gatekeeper.sh/operation": "audit",
		"gatekeeper.sh/system":    "yes",
	}
)

// ControllerDeploymentReconciler returns the function to create and update the Gatekeeper controller deployment.
func ControllerDeploymentReconciler(enableMutation bool, imageRewriter registry.ImageRewriter, resourceOverride *corev1.ResourceRequirements) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return controllerName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(controllerName, gatekeeperControllerLabels)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			kubernetes.EnsureLabels(&dep.Spec.Template, baseLabels)
			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: auditEmptyDirName,
			})

			dep.Spec.Template.Spec.TerminationGracePeriodSeconds = ptr.To[int64](60)
			dep.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}
			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			dep.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"
			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}
			dep.Spec.Template.Spec.Containers = getControllerContainers(enableMutation, imageRewriter)
			var err error
			if resourceOverride != nil {
				overridesRequirements := map[string]*corev1.ResourceRequirements{
					controllerName: resourceOverride.DeepCopy(),
				}
				err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, overridesRequirements, dep.Annotations)
			} else {
				err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.GatekeeperWebhookServerCertSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.GatekeeperWebhookServerCertSecretName,
						},
					},
				},
			}

			return dep, nil
		}
	}
}

// AuditDeploymentReconciler returns the function to create and update the Gatekeeper audit deployment.
func AuditDeploymentReconciler(registryWithOverwrite registry.ImageRewriter, resourceOverride *corev1.ResourceRequirements) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return auditName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = auditName
			dep.Labels = resources.BaseAppLabels(auditName, gatekeeperAuditLabels)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(auditName, gatekeeperAuditLabels),
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: resources.BaseAppLabels(auditName, gatekeeperAuditLabels),
			}

			dep.Spec.Template.Spec.TerminationGracePeriodSeconds = ptr.To[int64](60)
			dep.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}
			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(true)
			dep.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"
			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}
			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: auditEmptyDirName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: resources.GatekeeperWebhookServerCertSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.GatekeeperWebhookServerCertSecretName,
						},
					},
				},
			}

			dep.Spec.Template.Spec.Containers = getAuditContainers(registryWithOverwrite)
			var err error
			if resourceOverride != nil {
				overridesRequirements := map[string]*corev1.ResourceRequirements{
					auditName: resourceOverride.DeepCopy(),
				}
				err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, overridesRequirements, dep.Annotations)
			} else {
				err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getControllerContainers(enableMutation bool, imageRewriter registry.ImageRewriter) []corev1.Container {
	return []corev1.Container{{
		Name:            controllerName,
		Image:           registry.Must(imageRewriter(fmt.Sprintf("%s:%s", imageName, tag))),
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"/manager"},
		Args: []string{
			"--port=8443",
			"--logtostderr",
			fmt.Sprintf("--exempt-namespace=%s", resources.GatekeeperNamespace),
			fmt.Sprintf("--exempt-namespace=%s", metav1.NamespaceSystem),
			"--operation=webhook",
			fmt.Sprintf("--enable-mutation=%t", enableMutation),
			"--admission-events-involved-namespace=false",
			"--disable-opa-builtin={http.send}",
			"--enable-generator-resource-expansion=false",
			"--log-mutations=false",
			"--max-serving-threads=-1",
			"--metrics-backend=prometheus",
			"--mutation-annotations=false",
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: webhookServerPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				ContainerPort: metricsPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				ContainerPort: healthzPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      resources.GatekeeperWebhookServerCertSecretName,
				MountPath: "/certs",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "POD_NAMESPACE",
				Value: resources.GatekeeperNamespace,
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name:  "NAMESPACE",
				Value: resources.GatekeeperNamespace,
			},
			{
				Name:  "CONTAINER_NAME",
				Value: "manager",
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/healthz",
					Port:   intstr.FromInt(healthzPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			FailureThreshold:    3,
			InitialDelaySeconds: 15,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      15,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/readyz",
					Port:   intstr.FromInt(healthzPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			FailureThreshold:    3,
			InitialDelaySeconds: 15,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      15,
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
			ReadOnlyRootFilesystem: ptr.To(true),
			RunAsGroup:             ptr.To[int64](999),
			RunAsNonRoot:           ptr.To(true),
			RunAsUser:              ptr.To[int64](1000),
		},
	}}
}

func getAuditContainers(imageRewriter registry.ImageRewriter) []corev1.Container {
	return []corev1.Container{{
		Name:            auditName,
		Image:           registry.Must(imageRewriter(fmt.Sprintf("%s:%s", imageName, tag))),
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"/manager"},
		Args: []string{
			"--logtostderr",
			"--operation=audit",
			"--operation=status",
			"--operation=mutation-status",
			fmt.Sprintf("--constraint-violations-limit=%d", resources.ConstraintViolationsLimit),
			"--audit-events-involved-namespace=false",
			fmt.Sprintf("--audit-match-kind-only=%t", resources.AuditMatchKindOnly),
			fmt.Sprintf("--validating-webhook-configuration-name=%s", resources.GatekeeperValidatingWebhookConfigurationName),
			fmt.Sprintf("--mutating-webhook-configuration-name=%s", resources.GatekeeperMutatingWebhookConfigurationName),
			"--enable-generator-resource-expansion=false",
			"--metrics-backend=prometheus",
			"--disable-cert-rotation=true",
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: metricsPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				ContainerPort: healthzPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "POD_NAMESPACE",
				Value: resources.GatekeeperNamespace,
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name:  "NAMESPACE",
				Value: resources.GatekeeperNamespace,
			},
			{
				Name:  "CONTAINER_NAME",
				Value: "manager",
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/healthz",
					Port:   intstr.FromInt(healthzPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			FailureThreshold:    3,
			InitialDelaySeconds: 15,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      15,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/readyz",
					Port:   intstr.FromInt(healthzPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			FailureThreshold:    3,
			InitialDelaySeconds: 15,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      15,
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
			ReadOnlyRootFilesystem: ptr.To(true),
			RunAsGroup:             ptr.To[int64](999),
			RunAsNonRoot:           ptr.To(true),
			RunAsUser:              ptr.To[int64](1000),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      auditEmptyDirName,
				MountPath: auditEmptyDirMountPath,
			},
			{
				Name:      resources.GatekeeperWebhookServerCertSecretName,
				MountPath: "/certs",
				ReadOnly:  true,
			},
		},
	}}
}
