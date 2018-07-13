package dns

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getTemplatePodLabels(data *resources.TemplateData) (map[string]string, error) {
	podLabels := map[string]string{
		resources.AppLabelKey: resources.DNSResolverDeploymentName,
	}
	configRevision, err := data.ConfigMapRevision(resources.DNSResolverConfigMapName)
	if err != nil {
		return nil, err
	}
	podLabels[fmt.Sprintf("%s-configmap-revision", resources.DNSResolverConfigMapName)] = configRevision

	return podLabels, err
}

func Service(data *resources.TemplateData, existing *corev1.Service) (*corev1.Service, error) {
	var svc *corev1.Service
	if existing != nil {
		svc = existing
	} else {
		svc = &corev1.Service{}
	}
	svc.Name = resources.DNSResolverServiceName
	svc.Spec.Selector = map[string]string{
		resources.AppLabelKey: resources.DNSResolverDeploymentName,
	}
	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:     "dns",
			Protocol: corev1.ProtocolUDP,
			Port:     int32(53),
		},
	}

	return svc, nil
}

func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.DNSResolverDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	dep.Labels = resources.GetLabels(resources.DNSResolverDeploymentName)
	dep.Spec.Replicas = resources.Int32(2)

	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			resources.AppLabelKey: resources.DNSResolverDeploymentName,
		},
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

	podLabels, err := getTemplatePodLabels(data)
	if err != nil {
		return nil, fmt.Errorf("failed to get podlabels: %v", err)
	}

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{Labels: podLabels}
	openvpnSidecar, err := resources.OpenVPNSidecarContainer(data, "openvpn-client")
	if err != nil {
		return nil, fmt.Errorf("failed to get openvpn sidecar for dns resolver: %v", err)
	}

	dep.Spec.Template.Spec.Containers = []corev1.Container{
		*openvpnSidecar,
		{
			Name:  resources.DNSResolverDeploymentName,
			Image: data.ImageRegistry(resources.RegistryKubernetesGCR) + "/google_containers/coredns:1.1.3",
			Args:  []string{"-conf", "/etc/coredns/Corefile"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      resources.DNSResolverConfigMapName,
					MountPath: "/etc/coredns",
					ReadOnly:  true,
				},
			},
		},
	}
	dep.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: resources.DNSResolverConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.DNSResolverConfigMapName,
					},
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
			Name: resources.CACertSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CACertSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}

	return dep, nil
}

// ConfigMap returns a ConfigMap containing the cloud-config for the supplied data
func ConfigMap(data *resources.TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	var cm *corev1.ConfigMap
	if existing != nil {
		cm = existing
	} else {
		cm = &corev1.ConfigMap{}
	}
	dnsIP, err := resources.UserClusterDNSResolverIP(data.Cluster)
	if err != nil {
		return nil, err
	}
	cm.Name = resources.DNSResolverConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Data = map[string]string{
		"Corefile": fmt.Sprintf(`
%s {
    forward . %s
    errors
}
. {
  forward . /etc/resolv.conf
  errors
  health
}
`, data.Cluster.Spec.ClusterNetwork.DNSDomain, dnsIP)}

	return cm, nil
}
