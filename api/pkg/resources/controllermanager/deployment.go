package controllermanager

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
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
func Deployment(data resources.DeploymentDataProvider, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.ControllerManagerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	dep.Labels = resources.BaseAppLabel(name, nil)

	flags, err := getFlags(data.Cluster())
	if err != nil {
		return nil, err
	}

	dep.Spec.Replicas = resources.Int32(1)
	if data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas != nil {
		dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas
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

	// Configure user cluster DNS resolver for this pod.
	dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
	if err != nil {
		return nil, err
	}

	dep.Spec.Template.Spec.Volumes = volumes

	apiserverIsRunningContainer, err := apiserver.IsRunningInitContainer(data)
	if err != nil {
		return nil, err
	}
	dep.Spec.Template.Spec.InitContainers = []corev1.Container{*apiserverIsRunningContainer}

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
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
	}
	if data.Cluster().Spec.Cloud.VSphere != nil {
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
	if data.Cluster().Spec.ComponentsOverride.ControllerManager.Resources != nil {
		resourceRequirements = *data.Cluster().Spec.ComponentsOverride.ControllerManager.Resources
	}
	dep.Spec.Template.Spec.Containers = []corev1.Container{
		*openvpnSidecar,
		{
			Name:                     name,
			Image:                    data.ImageRegistry(resources.RegistryGCR) + "/google_containers/hyperkube-amd64:v" + data.Cluster().Spec.Version,
			ImagePullPolicy:          corev1.PullIfNotPresent,
			Command:                  []string{"/hyperkube", "controller-manager"},
			Args:                     flags,
			Env:                      getEnvVars(data.Cluster()),
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

	dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(resources.AppClusterLabel(name, data.Cluster().Name, nil))

	return dep, nil
}

func getFlags(cluster *kubermaticv1.Cluster) ([]string, error) {
	clusterVersionSemVer, err := semver.NewVersion(cluster.Spec.Version)
	if err != nil {
		return nil, err
	}

	flags := []string{
		"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"--service-account-private-key-file", "/etc/kubernetes/service-account-key/sa.key",
		"--root-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-cert-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-key-file", "/etc/kubernetes/pki/ca/ca.key",
		"--cluster-cidr", cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		"--configure-cloud-routes=false",
		"--allocate-node-cidrs=true",
		"--controllers", "*,bootstrapsigner,tokencleaner",
		"--use-service-account-credentials=true",
		"--v", "4",
	}

	featureGates := []string{"RotateKubeletClientCertificate=true",
		"RotateKubeletServerCertificate=true"}
	if clusterVersionSemVer.Minor() >= 12 {
		featureGates = append(featureGates, "ScheduleDaemonSetPods=false")
	}
	if len(featureGates) > 0 {
		flags = append(flags, "--feature-gates")
		flags = append(flags, strings.Join(featureGates, ","))
	}

	if cluster.Spec.Cloud.AWS != nil {
		flags = append(flags, "--cloud-provider", "aws")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if cluster.Spec.Cloud.Openstack != nil {
		flags = append(flags, "--cloud-provider", "openstack")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if cluster.Spec.Cloud.VSphere != nil {
		flags = append(flags, "--cloud-provider", "vsphere")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if cluster.Spec.Cloud.Azure != nil {
		flags = append(flags, "--cloud-provider", "azure")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}

	// This is required in 1.12.0-rc.2 as a workaround for
	// https://github.com/kubernetes/kubernetes/issues/68986
	// TODO: Check on 1.12 release/patch releases if the issue
	// got fixed and if yes, remove the check/make it more granular
	// Cherry-pick got created but not merged for 1.12.0:
	// https://github.com/kubernetes/kubernetes/pull/69117
	if clusterVersionSemVer.Minor() == 12 {
		flags = append(flags, "--authentication-skip-lookup=true")
	}
	return flags, nil
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

func getEnvVars(cluster *kubermaticv1.Cluster) []corev1.EnvVar {
	var vars []corev1.EnvVar
	if cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: cluster.Spec.Cloud.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: cluster.Spec.Cloud.AWS.SecretAccessKey})
		vars = append(vars, corev1.EnvVar{Name: "AWS_VPC_ID", Value: cluster.Spec.Cloud.AWS.VPCID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_AVAILABILITY_ZONE", Value: cluster.Spec.Cloud.AWS.AvailabilityZone})
	}
	return vars
}
