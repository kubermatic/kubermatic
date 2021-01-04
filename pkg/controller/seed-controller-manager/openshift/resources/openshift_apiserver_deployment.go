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

package resources

import (
	"context"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
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
	OpenshiftAPIServerServiceName    = OpenshiftAPIServerDeploymentName
)

func OpenshiftAPIServiceCreator() (string, reconciling.ServiceCreator) {
	return OpenshiftAPIServerServiceName, func(svc *corev1.Service) (*corev1.Service, error) {
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		svc.Spec.Ports = []corev1.ServicePort{{
			Protocol:   corev1.ProtocolTCP,
			Port:       443,
			TargetPort: intstr.FromInt(8443),
		}}
		svc.Spec.Selector = resources.BaseAppLabels(OpenshiftAPIServerDeploymentName, nil)

		return svc, nil
	}
}

// OpenshiftAPIServerDeploymentCreator returns the deployment creator for the Openshift APIServer
// This can not be part of the openshift-kube-apiserver pod, because the openshift-apiserver needs some CRD
// definitions to work and get ready, however we can not talk to the API until at least one pod is ready, preventing
// us from creating those CRDs
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
				MatchLabels: resources.BaseAppLabels(OpenshiftAPIServerDeploymentName, nil),
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
			dep.Spec.Template.Labels = resources.BaseAppLabels(OpenshiftAPIServerDeploymentName, nil)
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: resources.ImagePullSecretName},
				{Name: openshiftImagePullSecretName},
			}
			dep.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.CASecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.CASecretName,
							Items: []corev1.KeyToPath{
								{
									Path: "ca-bundle.crt",
									Key:  resources.CACertSecretKey,
								},
							},
						},
					},
				},
				{
					Name: openshiftAPIServerTLSServingCertSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: openshiftAPIServerTLSServingCertSecretName,
						},
					},
				},
				{
					Name: resources.FrontProxyCASecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.FrontProxyCASecretName,
						},
					},
				},
				{
					Name: resources.ApiserverEtcdClientCertificateSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.ApiserverEtcdClientCertificateSecretName,
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
					Name: openshiftAPIServerConfigMapName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: openshiftAPIServerConfigMapName},
						},
					},
				},
				{
					Name: resources.InternalUserClusterAdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
						},
					},
				},
			}

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn-client sidecar: %v", err)
			}

			dnatControllerSidecar, err := vpnsidecar.DnatControllerContainer(data, "dnat-controller", "")
			if err != nil {
				return nil, fmt.Errorf("failed to get dnat-controller sidecar: %v", err)
			}

			// TODO: Make it cope with our registry overwriting
			image, err := hypershiftImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				*dnatControllerSidecar,
				{
					Name:    OpenshiftAPIServerDeploymentName,
					Image:   image,
					Command: []string{"hypershift", "openshift-apiserver"},
					Args:    []string{"--config=/etc/origin/master/master-config.yaml"},
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
							MountPath: "/var/run/configmaps/aggregator-client-ca",
							ReadOnly:  true,
						},
						{
							Name:      openshiftAPIServerConfigMapName,
							MountPath: "/etc/origin/master",
							ReadOnly:  true,
						},
						{
							Name:      resources.CASecretName,
							MountPath: "/var/run/configmaps/client-ca",
							ReadOnly:  true,
						},
						{
							Name:      openshiftAPIServerTLSServingCertSecretName,
							MountPath: "/var/run/secrets/serving-cert",
							ReadOnly:  true,
						},
						{
							Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
							MountPath: "/etc/origin/master/kubeconfig",
							ReadOnly:  true,
						},
					},
				},
			}
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				OpenshiftAPIServerDeploymentName: openshiftAPIServerDefaultResourceRequirements.DeepCopy(),
				openvpnSidecar.Name:              openvpnSidecar.Resources.DeepCopy(),
				dnatControllerSidecar.Name:       dnatControllerSidecar.Resources.DeepCopy(),
			}
			overrides := map[string]*corev1.ResourceRequirements{}
			if data.Cluster().Spec.ComponentsOverride.Apiserver.Resources != nil {
				overrides[OpenshiftAPIServerDeploymentName] = data.Cluster().Spec.ComponentsOverride.Apiserver.Resources.DeepCopy()
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, overrides, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}
			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(OpenshiftAPIServerDeploymentName, data.Cluster().Name)
			podLabels, err := data.GetPodTemplateLabels(OpenshiftAPIServerDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Labels = podLabels

			// The openshift apiserver needs the normal apiserver
			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(OpenshiftAPIServerDeploymentName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}
