package metricsserver

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/servingcerthelper"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("150m"),
		},
	}
)

const (
	name = "metrics-server"
	// ServingCertSecretName is the name of the secret containing the metrics-server
	// serving cert.
	ServingCertSecretName  = "metrics-server-serving-cert"
	servingCertMountFolder = "/etc/serving-cert"

	tag = "v0.3.4"
)

// metricsServerData is the data needed to construct the metrics-server components
type metricsServerData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetRootCA() (*triple.KeyPair, error)
	ImageRegistry(string) string
	DNATControllerImage() string
	NodeAccessNetwork() string
}

// TLSServingCertSecretCreator returns a function to manage the TLS serving cert for the metrics
// server
func TLSServingCertSecretCreator(caGetter servingcerthelper.CAGetter) reconciling.NamedSecretCreatorGetter {
	dnsName := "metrics-server.kube-system.svc"
	return servingcerthelper.ServingCertSecretCreator(caGetter,
		ServingCertSecretName,
		// Must match whats configured in the apiservice in pkg/controller/usercluster/resources/metrics-server/external-name-service.go.
		// Can unfortunately not have a trailing dot, as thats only allowed in Kube 1.16+
		dnsName,
		[]string{dnsName},
		nil)
}

// DeploymentCreator returns the function to create and update the metrics server deployment
func DeploymentCreator(data metricsServerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.MetricsServerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.MetricsServerDeploymentName
			dep.Labels = resources.BaseAppLabel(name, nil)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(name, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.Volumes = volumes

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn-client sidecar: %v", err)
			}

			dnatControllerSidecar, err := vpnsidecar.DnatControllerContainer(data, "dnat-controller", "")
			if err != nil {
				return nil, fmt.Errorf("failed to get dnat-controller sidecar: %v", err)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    name,
					Image:   data.ImageRegistry(resources.RegistryGCR) + "/google_containers/metrics-server-amd64:" + tag,
					Command: []string{"/metrics-server"},
					Args: []string{
						"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--kubelet-port", "10250",
						"--kubelet-insecure-tls",
						// We use the same as the API server as we use the same dnat-controller
						"--kubelet-preferred-address-types", "ExternalIP,InternalIP",
						"--v", "1",
						"--logtostderr",
						"--tls-cert-file", servingCertMountFolder + "/" + resources.ServingCertSecretKey,
						"--tls-private-key-file", servingCertMountFolder + "/" + resources.ServingCertKeySecretKey,
					},
					Resources: defaultResourceRequirements,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.MetricsServerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      ServingCertSecretName,
							MountPath: servingCertMountFolder,
							ReadOnly:  true,
						},
					},
				},
				*openvpnSidecar,
				*dnatControllerSidecar,
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
			Name: resources.MetricsServerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.MetricsServerKubeconfigSecretName,
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
			Name: resources.KubeletDnatControllerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KubeletDnatControllerKubeconfigSecretName,
				},
			},
		},
		{
			Name: ServingCertSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ServingCertSecretName,
				},
			},
		},
	}
}
