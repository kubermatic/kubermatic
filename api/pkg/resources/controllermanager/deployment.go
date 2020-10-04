package controllermanager

import (
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	name = "controller-manager"
)

// DeploymentCreator returns the function to create and update the controller manager deployment
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.ControllerManagerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.ControllerManagerDeploymentName
			dep.Labels = resources.BaseAppLabel(name, nil)

			flags, err := getFlags(data)
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
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			if data.Cluster().Spec.Cloud.GCP != nil {
				serviceAccountVolume := corev1.Volume{
					Name: resources.GoogleServiceAccountVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.GoogleServiceAccountSecretName,
						},
					},
				}
				volumes = append(volumes, serviceAccountVolume)
			}

			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels:      podLabels,
				Annotations: getPodAnnotations(data),
			}

			// Configure user cluster DNS resolver for this pod.
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Volumes = volumes

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
			if data.Cluster().Spec.Cloud.GCP != nil {
				serviceAccountMount := corev1.VolumeMount{
					Name:      resources.GoogleServiceAccountVolumeName,
					MountPath: "/etc/gcp",
					ReadOnly:  true,
				}
				controllerManagerMounts = append(controllerManagerMounts, serviceAccountMount)
			}

			envVars, err := GetEnvVars(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:    resources.ControllerManagerDeploymentName,
					Image:   data.ImageRegistry(resources.RegistryK8SGCR) + "/hyperkube-amd64:v" + data.Cluster().Spec.Version.String(),
					Command: []string{"/hyperkube", "kube-controller-manager"},
					Args:    flags,
					Env:     envVars,
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
					VolumeMounts: controllerManagerMounts,
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

func getFlags(data *resources.TemplateData) ([]string, error) {
	flags := []string{
		"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"--service-account-private-key-file", "/etc/kubernetes/service-account-key/sa.key",
		"--root-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-cert-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--cluster-signing-key-file", "/etc/kubernetes/pki/ca/ca.key",
		"--cluster-cidr", data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		"--allocate-node-cidrs=true",
		"--controllers", "*,bootstrapsigner,tokencleaner",
		"--use-service-account-credentials=true",
	}

	featureGates := []string{"RotateKubeletClientCertificate=true",
		"RotateKubeletServerCertificate=true"}
	// This is required for Kubelets < 1.11, they don't start DaemonSet
	// pods scheduled by the scheduler: https://github.com/kubernetes/kubernetes/issues/69346
	// TODO: Remove once we don't support Kube 1.10 anymore
	// TODO: Before removing, add check that prevents upgrading to 1.12 when
	// there is still a node < 1.11
	if data.Cluster().Spec.Version.Semver().Minor() >= 12 && data.Cluster().Spec.Version.Semver().Minor() < 17 {
		featureGates = append(featureGates, "ScheduleDaemonSetPods=false")
	}
	if len(featureGates) > 0 {
		flags = append(flags, "--feature-gates")
		flags = append(flags, strings.Join(featureGates, ","))
	}

	cloudProviderName := data.GetKubernetesCloudProviderName()
	if cloudProviderName != "" && cloudProviderName != "external" {
		flags = append(flags, "--cloud-provider", cloudProviderName)
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
		if cloudProviderName == "azure" && data.Cluster().Spec.Version.Semver().Minor() >= 15 {
			// Required so multiple clusters using the same resource group can allocate public IPs.
			// Ref: https://github.com/kubernetes/kubernetes/pull/77630
			flags = append(flags, "--cluster-name", data.Cluster().Name)
		}
	}

	if val := CloudRoutesFlagVal(data.Cluster().Spec.Cloud); val != nil {
		flags = append(flags, fmt.Sprintf("--configure-cloud-routes=%t", *val))
	}

	if data.Cluster().Spec.Version.Semver().Minor() >= 12 {
		// New flag in v1.12 which gets used to perform permission checks for tokens
		flags = append(flags, "--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
		// New flag in v1.12 which gets used to perform permission checks for certs
		flags = append(flags, "--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt")
	}

	// With 1.13 we're using the secure port for scraping metrics as the insecure port got marked deprecated
	if data.Cluster().Spec.Version.Semver().Minor() >= 13 {
		flags = append(flags, "--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
		flags = append(flags, "--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig")
		// We're going to use the https endpoints for scraping the metrics starting from 1.12. Thus we can deactivate the http endpoint
		flags = append(flags, "--port", "0")
		// Force the authentication lookup to succeed, otherwise if it fails all requests will be treated as anonymous and thus fail
		if data.Cluster().Spec.Version.Semver().Patch() > 0 {
			// Force the authentication lookup to succeed, otherwise if it fails all requests will be treated as anonymous and thus fail
			// Both the flag and the issue only exist in 1.13.1 and above
			flags = append(flags, "--authentication-tolerate-lookup-failure=false")
		}
	}

	// This is required in 1.12.0 as a workaround for
	// https://github.com/kubernetes/kubernetes/issues/68986, the patch
	// got cherry-picked onto 1.12.1
	if data.Cluster().Spec.Version.Semver().Minor() == 12 && data.Cluster().Spec.Version.Semver().Patch() == 0 {
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
					SecretName: resources.CASecretName,
				},
			},
		},
		{
			Name: resources.ServiceAccountKeySecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ServiceAccountKeySecretName,
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
				},
			},
		},
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		},
		{
			Name: resources.ControllerManagerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ControllerManagerKubeconfigSecretName,
				},
			},
		},
	}
}

type kubeControllerManagerEnvData interface {
	resources.CredentialsData
	Seed() *kubermaticv1.Seed
}

func GetEnvVars(data kubeControllerManagerEnvData) ([]corev1.EnvVar, error) {
	credentials, err := resources.GetCredentials(data)
	if err != nil {
		return nil, err
	}
	cluster := data.Cluster()

	var vars []corev1.EnvVar
	if cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: credentials.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: credentials.AWS.SecretAccessKey})
		vars = append(vars, corev1.EnvVar{Name: "AWS_VPC_ID", Value: cluster.Spec.Cloud.AWS.VPCID})
	}
	if cluster.Spec.Cloud.GCP != nil {
		vars = append(vars, corev1.EnvVar{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/etc/gcp/serviceAccount"})
	}
	vars = append(vars, resources.GetHTTPProxyEnvVarsFromSeed(data.Seed(), data.Cluster().Address.InternalName)...)
	return vars, nil
}

func getPodAnnotations(data *resources.TemplateData) map[string]string {
	annotations := map[string]string{
		"prometheus.io/path": "/metrics",
	}
	// With 1.13 we're using the secure port for scraping metrics as the insecure port got marked deprecated
	if data.Cluster().Spec.Version.Minor() >= 13 {
		annotations["prometheus.io/scrape_with_kube_cert"] = "true"
		annotations["prometheus.io/port"] = "10257"
	} else {
		annotations["prometheus.io/scrape"] = "true"
		annotations["prometheus.io/port"] = "10252"
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
		action.Port = intstr.FromInt(10257)
	} else {
		action.Scheme = corev1.URISchemeHTTP
		action.Port = intstr.FromInt(10252)
	}
	return action
}

func CloudRoutesFlagVal(cloudSpec kubermaticv1.CloudSpec) *bool {
	if cloudSpec.AWS != nil {
		return utilpointer.BoolPtr(false)
	}
	if cloudSpec.Openstack != nil {
		return utilpointer.BoolPtr(false)
	}
	if cloudSpec.VSphere != nil {
		return utilpointer.BoolPtr(false)
	}
	if cloudSpec.Azure != nil {
		return utilpointer.BoolPtr(false)
	}
	if cloudSpec.GCP != nil {
		return utilpointer.BoolPtr(true)
	}
	return nil
}
