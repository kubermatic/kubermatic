package scheduler

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("20m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}
)

const (
	name = "scheduler"
)

// DeploymentCreator returns the function to create and update the scheduler deployment
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.SchedulerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.SchedulerDeploymentName
			dep.Labels = resources.BaseAppLabel(name, nil)

			flags, err := getFlags(data.Cluster())
			if err != nil {
				return nil, err
			}

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.Scheduler.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.Scheduler.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(name, nil),
			}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels:      podLabels,
				Annotations: getPodAnnotations(data),
			}

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
			}

			// Configure user cluster DNS resolver for this pod.
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Volumes = volumes

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:    resources.SchedulerDeploymentName,
					Image:   data.ImageRegistry(resources.RegistryK8SGCR) + "/hyperkube-amd64:v" + data.Cluster().Spec.Version.String(),
					Command: []string{"/hyperkube", "kube-scheduler"},
					Args:    flags,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.SchedulerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      resources.CASecretName,
							MountPath: "/etc/kubernetes/pki/ca",
							ReadOnly:  true,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: getHealthGetAction(data),
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 8,
						Handler: corev1.Handler{
							HTTPGet: getHealthGetAction(data),
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
				},
			}
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				name:                defaultResourceRequirements.DeepCopy(),
				openvpnSidecar.Name: openvpnSidecar.Resources.DeepCopy(),
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, data.Cluster().Name)

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		},
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CASecretName,
					Items: []corev1.KeyToPath{
						{
							Path: resources.CACertSecretKey,
							Key:  resources.CACertSecretKey,
						},
					},
				},
			},
		},
		{
			Name: resources.SchedulerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.SchedulerKubeconfigSecretName,
				},
			},
		},
	}
}

func getFlags(cluster *kubermaticv1.Cluster) ([]string, error) {
	flags := []string{
		"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
	}

	// With 1.13 we're using the secure port for scraping metrics as the insecure port got marked deprecated
	if cluster.Spec.Version.Semver().Minor() >= 13 {
		// These are used to validate tokens
		flags = append(flags, "--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
		flags = append(flags, "--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
		// This is used to validate certs
		flags = append(flags, "--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt")
		// We're going to use the https endpoints for scraping the metrics starting from 1.13. Thus we can deactivate the http endpoint
		flags = append(flags, "--port", "0")
		if cluster.Spec.Version.Semver().Patch() > 0 {
			// Force the authentication lookup to succeed, otherwise if it fails all requests will be treated as anonymous and thus fail
			// Both the flag and the issue only exist in 1.13.1 and above
			flags = append(flags, "--authentication-tolerate-lookup-failure", "false")
		}
	}

	var featureGates []string
	// This is required for Kubelets < 1.11, they don't start DaemonSet
	// pods scheduled by the scheduler: https://github.com/kubernetes/kubernetes/issues/69346
	// TODO: Remove once we don't support Kube 1.10 anymore
	// TODO: Before removing, add check that prevents upgrading to 1.12 when
	// there is still a node < 1.11
	if cluster.Spec.Version.Semver().Minor() >= 12 && cluster.Spec.Version.Semver().Minor() < 17 {
		featureGates = append(featureGates, "ScheduleDaemonSetPods=false")
	}

	if len(featureGates) > 0 {
		flags = append(flags, "--feature-gates")
		flags = append(flags, strings.Join(featureGates, ","))
	}

	return flags, nil
}

func getPodAnnotations(data *resources.TemplateData) map[string]string {
	annotations := map[string]string{
		"prometheus.io/path": "/metrics",
	}
	// With 1.13 we're using the secure port for scraping metrics as the insecure port got marked deprecated
	if data.Cluster().Spec.Version.Minor() >= 13 {
		annotations["prometheus.io/scrape_with_kube_cert"] = "true"
		annotations["prometheus.io/port"] = "10259"
	} else {
		annotations["prometheus.io/scrape"] = "true"
		annotations["prometheus.io/port"] = "10251"
	}

	return annotations
}

func getHealthGetAction(data *resources.TemplateData) *corev1.HTTPGetAction {
	action := &corev1.HTTPGetAction{
		Path: "/healthz",
	}
	// With 1.13 we're using the secure port for scraping metrics as the insecure port got marked deprecated
	if data.Cluster().Spec.Version.Minor() >= 13 {
		action.Scheme = corev1.URISchemeHTTPS
		action.Port = intstr.FromInt(10259)
	} else {
		action.Scheme = corev1.URISchemeHTTP
		action.Port = intstr.FromInt(10251)
	}
	return action
}
