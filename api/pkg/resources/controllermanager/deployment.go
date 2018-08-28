package controllermanager

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("250m"),
		},
	}
)

const (
	name = "controller-manager"
)

// Deployment returns the kubernetes Controller-Manager Deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.ControllerManagerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	dep.Labels = resources.BaseAppLabel(name, nil)

	dep.Spec.Replicas = resources.Int32(1)
	if data.Cluster.Spec.ComponentsOverride.ControllerManager.Replicas != nil {
		dep.Spec.Replicas = data.Cluster.Spec.ComponentsOverride.ControllerManager.Replicas
	}

	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: resources.BaseAppLabel(name, nil),
	}
	dep.Spec.Strategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
		MaxSurge: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 1,
		},
		MaxUnavailable: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 0,
		},
	}

	volumes := getVolumes()
	podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
	if err != nil {
		return nil, err
	}

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: podLabels,
		Annotations: map[string]string{
			"prometheus.io-0/scrape": "true",
			"prometheus.io-0/path":   "/metrics",
			"prometheus.io-0/port":   "10252",
		},
	}

	// get clusterIP of apiserver
	apiAddress, err := data.InClusterApiserverAddress()
	if err != nil {
		return nil, err
	}

	// Configure user cluster DNS resolver for this pod.
	dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
	if err != nil {
		return nil, err
	}

	kcDir := "/etc/kubernetes/controllermanager"
	dep.Spec.Template.Spec.Volumes = volumes
	dep.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:            "apiserver-running",
			Image:           data.ImageRegistry(resources.RegistryDocker) + "/busybox",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command: []string{
				"/bin/sh",
				"-ec",
				fmt.Sprintf("until wget -T 1 https://%s/healthz; do echo waiting for apiserver; sleep 2; done;", apiAddress),
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		},
	}

	openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
	if err != nil {
		return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
	}

	controllerManagerMounts := []corev1.VolumeMount{
		{
			Name:      resources.CASecretName,
			MountPath: "/etc/kubernetes/pki/ca",
			ReadOnly:  true,
		},
		{
			Name:      resources.ServiceAccountKeySecretName,
			MountPath: "/etc/kubernetes/service-account-key",
			ReadOnly:  true,
		},
		{
			Name:      resources.CloudConfigConfigMapName,
			MountPath: "/etc/kubernetes/cloud",
			ReadOnly:  true,
		},
		{
			Name:      resources.ControllerManagerKubeconfigSecretName,
			MountPath: kcDir,
			ReadOnly:  true,
		},
	}
	if data.Cluster.Spec.Cloud.VSphere != nil {
		fakeVMWareUUIDMount := corev1.VolumeMount{
			Name:      resources.CloudConfigConfigMapName,
			SubPath:   cloudconfig.FakeVMWareUUIDKeyName,
			MountPath: "/sys/class/dmi/id/product_serial",
			ReadOnly:  true,
		}
		// Required because of https://github.com/kubernetes/kubernetes/issues/65145
		controllerManagerMounts = append(controllerManagerMounts, fakeVMWareUUIDMount)
	}

	resourceRequirements := defaultResourceRequirements
	if data.Cluster.Spec.ComponentsOverride.ControllerManager.Resources != nil {
		resourceRequirements = *data.Cluster.Spec.ComponentsOverride.ControllerManager.Resources
	}
	dep.Spec.Template.Spec.Containers = []corev1.Container{
		*openvpnSidecar,
		{
			Name:            name,
			Image:           data.ImageRegistry(resources.RegistryKubernetesGCR) + "/google_containers/hyperkube-amd64:v" + data.Cluster.Spec.Version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/hyperkube", "controller-manager"},
			Args:            getFlags(data, kcDir),
			Env:             getEnvVars(data),
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources:                resourceRequirements,
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt(10252),
					},
				},
				FailureThreshold: 3,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				TimeoutSeconds:   15,
			},
			LivenessProbe: &corev1.Probe{
				FailureThreshold: 8,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt(10252),
					},
				},
				InitialDelaySeconds: 15,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      15,
			},
			VolumeMounts: controllerManagerMounts,
		},
	}

	return dep, nil
}

func getFlags(data *resources.TemplateData, kcDir string) []string {
	flags := []string{
		"--kubeconfig", fmt.Sprintf("%s/%s", kcDir, resources.ControllerManagerKubeconfigSecretName),
		"--service-account-private-key-file", "/etc/kubernetes/service-account-key/sa.key",
		"--root-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-cert-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-key-file", "/etc/kubernetes/pki/ca/ca.key",
		"--cluster-cidr", data.Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		"--configure-cloud-routes=false",
		"--allocate-node-cidrs=true",
		"--controllers", "*,bootstrapsigner,tokencleaner",
		"--feature-gates", "RotateKubeletClientCertificate=true,RotateKubeletServerCertificate=true",
		"--v", "4",
	}
	if data.Cluster.Spec.Cloud.AWS != nil {
		flags = append(flags, "--cloud-provider", "aws")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if data.Cluster.Spec.Cloud.Openstack != nil {
		flags = append(flags, "--cloud-provider", "openstack")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if data.Cluster.Spec.Cloud.VSphere != nil {
		flags = append(flags, "--cloud-provider", "vsphere")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if data.Cluster.Spec.Cloud.Azure != nil {
		flags = append(flags, "--cloud-provider", "azure")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	return flags
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.ServiceAccountKeySecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ServiceAccountKeySecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.CloudConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CloudConfigConfigMapName,
					},
					DefaultMode: resources.Int32(420),
				},
			},
		},
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.OpenVPNClientCertificatesSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.ControllerManagerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ControllerManagerKubeconfigSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}
}

func getEnvVars(data *resources.TemplateData) []corev1.EnvVar {
	var vars []corev1.EnvVar
	if data.Cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: data.Cluster.Spec.Cloud.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: data.Cluster.Spec.Cloud.AWS.SecretAccessKey})
		vars = append(vars, corev1.EnvVar{Name: "AWS_VPC_ID", Value: data.Cluster.Spec.Cloud.AWS.VPCID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_AVAILABILITY_ZONE", Value: data.Cluster.Spec.Cloud.AWS.AvailabilityZone})
	}
	return vars
}
