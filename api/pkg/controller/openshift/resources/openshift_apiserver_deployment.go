package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilpointer "k8s.io/utils/pointer"
)

var (
	openshiftAPIServerDefaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("200Mi"),
			corev1.ResourceCPU:    resource.MustParse("150m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("4Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	OpenshiftAPIServerDeploymentName = "openshift-apiserver"
)

func OpenshiftAPIServerDeploymentCreator(ctx context.Context, data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return OpenshiftAPIServerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			var err error
			dep.Name = OpenshiftAPIServerDeploymentName

			dep.Spec.Replicas = utilpointer.Int32Ptr(1)
			if data.Cluster().Spec.ComponentsOverride.Apiserver.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.Apiserver.Replicas
			}
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(OpenshiftAPIServerDeploymentName, nil),
			}

			dep.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
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
			dep.Spec.Template.Labels = resources.BaseAppLabel(OpenshiftAPIServerDeploymentName, nil)
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: resources.ImagePullSecretName},
				{Name: openshiftImagePullSecretName},
			}
			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.FrontProxyCASecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  resources.FrontProxyCASecretName,
							DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
						},
					},
				},
				{
					Name: resources.ApiserverEtcdClientCertificateSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  resources.ApiserverEtcdClientCertificateSecretName,
							DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
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
					Name: resources.KubeletDnatControllerKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  resources.KubeletDnatControllerKubeconfigSecretName,
							DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
						},
					},
				},
			}

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			etcdEndpoints := etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName)
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:  "etcd-running",
					Image: data.ImageRegistry(resources.RegistryGCR) + "/etcd-development/etcd:" + etcd.ImageTag,
					Command: []string{
						"/bin/sh",
						"-ec",
						fmt.Sprintf("until ETCDCTL_API=3 /usr/local/bin/etcdctl --cacert=/etc/etcd/pki/client/ca.crt --cert=/etc/etcd/pki/client/apiserver-etcd-client.crt --key=/etc/etcd/pki/client/apiserver-etcd-client.key --dial-timeout=2s --endpoints='%s' endpoint health; do echo waiting for etcd; sleep 2; done;", strings.Join(etcdEndpoints, ",")),
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.ApiserverEtcdClientCertificateSecretName,
							MountPath: "/etc/etcd/pki/client",
							ReadOnly:  true,
						},
					},
				},
			}

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn-client sidecar: %v", err)
			}

			dnatControllerSidecar, err := vpnsidecar.DnatControllerContainer(data, "dnat-controller", "")
			if err != nil {
				return nil, fmt.Errorf("failed to get dnat-controller sidecar: %v", err)
			}

			resourceRequirements := openshiftAPIServerDefaultResourceRequirements.DeepCopy()
			if data.Cluster().Spec.ComponentsOverride.Apiserver.Resources != nil {
				resourceRequirements = data.Cluster().Spec.ComponentsOverride.Apiserver.Resources
			}

			// TODO: Make it cope with our registry overwriting
			image, err := openshiftAPIServerImage(data.Cluster().Spec.Version.String())
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				*dnatControllerSidecar,
				{
					Name:      OpenshiftAPIServerDeploymentName,
					Image:     image,
					Command:   []string{"hypershift", "openshift-apiserver"},
					Args:      []string{"--config=/etc/origin/master/master-config.yaml"},
					Resources: *resourceRequirements,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 8443,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(8443),
								Scheme: "HTTPS",
							},
						},
						FailureThreshold: 10,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   1,
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(8443),
								Scheme: "HTTPS",
							},
						},
						FailureThreshold: 10,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   1,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.ApiserverEtcdClientCertificateSecretName,
							MountPath: "/etc/etcd/pki/client",
							ReadOnly:  true,
						},
						{
							Name:      resources.FrontProxyCASecretName,
							MountPath: "/etc/kubernetes/pki/front-proxy/ca",
							ReadOnly:  true,
						},
					},
				},
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(OpenshiftAPIServerDeploymentName, data.Cluster().Name)

			return dep, nil
		}
	}
}

func openshiftAPIServerImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case "4.1.9":
		return "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad", nil
	default:
		return "", fmt.Errorf("no image available for openshift version %q", openshiftVersion)
	}

}
