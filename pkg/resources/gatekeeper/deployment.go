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

package gatekeeper

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/servingcerthelper"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

const (
	controllerName = resources.GatekeeperControllerDeploymentName
	auditName      = resources.GatekeeperAuditDeploymentName
	imageName      = "open-policy-agent/gatekeeper"
	tag            = "v3.1.0-beta.9"
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
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
		},
		auditName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
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

// gatekeeperData is the data needed to construct the Gatekeeper components
type gatekeeperData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
}

// ControllerDeploymentCreator returns the function to create and update the Gatekeeper controller deployment
func ControllerDeploymentCreator(data gatekeeperData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return controllerName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = controllerName
			dep.Labels = resources.BaseAppLabels(controllerName, gatekeeperControllerLabels)

			if dep.Annotations == nil {
				dep.Annotations = make(map[string]string)
			}
			dep.Annotations["container.seccomp.security.alpha.kubernetes.io/manager"] = "runtime/default"

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(controllerName, gatekeeperControllerLabels),
			}

			volumes := getControllerVolumes()
			podLabels, err := data.GetPodTemplateLabels(controllerName, volumes, gatekeeperControllerLabels)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(60)
			dep.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}
			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = getControllerContainers(data, dep.Spec.Template.Spec.Containers)
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}
			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(controllerName, data.Cluster().Status.NamespaceName)

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(controllerName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

// AuditDeploymentCreator returns the function to create and update the Gatekeeper audit deployment
func AuditDeploymentCreator(data gatekeeperData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return auditName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = auditName
			dep.Labels = resources.BaseAppLabels(auditName, gatekeeperAuditLabels)

			if dep.Annotations == nil {
				dep.Annotations = make(map[string]string)
			}
			dep.Annotations["container.seccomp.security.alpha.kubernetes.io/manager"] = "runtime/default"

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(auditName, gatekeeperAuditLabels),
			}

			volumes := getAuditVolumes()
			podLabels, err := data.GetPodTemplateLabels(auditName, volumes, gatekeeperAuditLabels)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(60)
			dep.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}
			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = getAuditContainers(data, dep.Spec.Template.Spec.Containers)
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}
			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(auditName, data.Cluster().Status.NamespaceName)

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(auditName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

func getControllerContainers(data gatekeeperData, existingContainers []corev1.Container) []corev1.Container {

	return []corev1.Container{{
		Name:            controllerName,
		Image:           fmt.Sprintf("%s/%s:%s", data.ImageRegistry(resources.RegistryQuay), imageName, tag),
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"/manager"},
		Args: []string{
			"--port=8443",
			"--logtostderr",
			fmt.Sprintf("--exempt-namespace=%s", data.Cluster().Status.NamespaceName),
			"--operation=webhook",
			"--disable-cert-rotation=true",
			"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      resources.GatekeeperWebhookServerCertSecretName,
				MountPath: "/certs",
				ReadOnly:  true,
			},
			{
				Name:      resources.AdminKubeconfigSecretName,
				MountPath: "/etc/kubernetes/kubeconfig",
				ReadOnly:  true,
			},
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
		Env: []corev1.EnvVar{
			{Name: "POD_NAMESPACE",
				Value: resources.GatekeeperNamespace},
			{Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				}},
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
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
			Handler: corev1.Handler{
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
			AllowPrivilegeEscalation: pointer.BoolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"all",
				},
			},
			RunAsGroup:   pointer.Int64Ptr(999),
			RunAsNonRoot: pointer.BoolPtr(true),
			RunAsUser:    pointer.Int64Ptr(1000),
		},
	}}
}

func getAuditContainers(data gatekeeperData, existingContainers []corev1.Container) []corev1.Container {

	return []corev1.Container{{
		Name:            auditName,
		Image:           fmt.Sprintf("%s/%s:%s", data.ImageRegistry(resources.RegistryQuay), imageName, tag),
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"/manager"},
		Args: []string{
			"--logtostderr",
			"--operation=audit",
			"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      resources.AdminKubeconfigSecretName,
				MountPath: "/etc/kubernetes/kubeconfig",
				ReadOnly:  true,
			},
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
			{Name: "POD_NAMESPACE",
				Value: resources.GatekeeperNamespace},
			{Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				}},
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
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
			Handler: corev1.Handler{
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
			AllowPrivilegeEscalation: pointer.BoolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"all",
				},
			},
			RunAsGroup:   pointer.Int64Ptr(999),
			RunAsNonRoot: pointer.BoolPtr(true),
			RunAsUser:    pointer.Int64Ptr(1000),
		},
	}}
}

func getControllerVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.AdminKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.AdminKubeconfigSecretName,
				},
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
}

func getAuditVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.AdminKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.AdminKubeconfigSecretName,
				},
			},
		},
	}
}

type gatekeeperWebhookCertificateCreatorData interface {
	GetRootCA() (*triple.KeyPair, error)
	Cluster() *kubermaticv1.Cluster
}

// TLSServingCertSecretCreator returns a function to manage the TLS serving cert for gatkeeper webhook
func TLSServingCertSecretCreator(data gatekeeperWebhookCertificateCreatorData) reconciling.NamedSecretCreatorGetter {
	commonName := fmt.Sprintf("%s.%s.svc.cluster.local", resources.GatekeeperWebhookServiceName, data.Cluster().Status.NamespaceName)
	return servingcerthelper.ServingCertSecretCreator(data.GetRootCA,
		resources.GatekeeperWebhookServerCertSecretName,
		// Must match what's configured in the gatekeeper webhook
		commonName,
		[]string{commonName},
		nil)
}
