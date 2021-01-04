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
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/servingcerthelper"
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
	name                               = "scheduler"
	kubeSchedulerConfigConfigMapName   = "kube-scheduler-config"
	kubeSchedulerServingCertSecretName = "kube-scheduler-serving-cert"
	kubeSchedulerConfig                = `
apiVersion: kubescheduler.config.k8s.io/v1alpha1
clientConnection:
  kubeconfig: /etc/openshift/kubeconfig/kubeconfig
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: true
  lockObjectNamespace: openshift-kube-scheduler
  resourceLock: configmaps
`
)

func KubeSchedulerConfigMapCreator() (string, reconciling.ConfigMapCreator) {
	return kubeSchedulerConfigConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["config.yaml"] = kubeSchedulerConfig
		return cm, nil
	}
}

func KubeSchedulerServingCertCreator(caGetter servingcerthelper.CAGetter) reconciling.NamedSecretCreatorGetter {
	return servingcerthelper.ServingCertSecretCreator(caGetter,
		kubeSchedulerServingCertSecretName,
		"scheduler.openshift-kube-scheduler.svc",
		[]string{"scheduler.openshift-kube-scheduler.svc", "scheduler.openshift-kube-scheduler.svc.cluster.local"},
		nil)
}

// DeploymentCreator returns the function to create and update the scheduler deployment
func KubeSchedulerDeploymentCreator(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.SchedulerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.SchedulerDeploymentName
			dep.Labels = resources.BaseAppLabels(name, nil)

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.Scheduler.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.Scheduler.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(name, nil),
			}

			dep.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/path":                  "/metrics",
					"prometheus.io/scrape_with_kube_cert": "true",
					"prometheus.io/port":                  "10259",
				},
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

			image, err := hyperkubeImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: openshiftImagePullSecretName}}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:    resources.SchedulerDeploymentName,
					Image:   image,
					Command: []string{"hyperkube", "kube-scheduler"},
					Args: []string{
						"--config=/etc/openshift/config/config.yaml",
						"--port=0",
						"--authentication-kubeconfig=/etc/openshift/kubeconfig/kubeconfig",
						"--authorization-kubeconfig=/etc/openshift/kubeconfig/kubeconfig",
						// TODO: Upstream also has RotateKubeletServerCertificate=true, check if we can use that
						"--feature-gates=ExperimentalCriticalPodAnnotation=true,LocalStorageCapacityIsolation=false,SupportPodPidsLimit=true",
						"-v=2",
						"--tls-cert-file=/etc/openshift/serving/serving.crt",
						"--tls-private-key-file=/etc/openshift/serving/serving.key",
						"--client-ca-file=/etc/openshift/ca/ca.crt",
						"--requestheader-client-ca-file=/etc/openshift/front-proxy-ca/ca.crt",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.SchedulerKubeconfigSecretName,
							MountPath: "/etc/openshift/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      resources.CASecretName,
							MountPath: "/etc/openshift/ca/ca.crt",
							SubPath:   "ca.crt",
							ReadOnly:  true,
						},
						{
							Name:      kubeSchedulerConfigConfigMapName,
							MountPath: "/etc/openshift/config",
							ReadOnly:  true,
						},
						{
							Name:      kubeSchedulerServingCertSecretName,
							MountPath: "/etc/openshift/serving",
							ReadOnly:  true,
						},
						{
							Name:      resources.ApiserverFrontProxyClientCertificateSecretName,
							MountPath: "/etc/openshift/front-proxy-ca/ca.crt",
							SubPath:   "ca.crt",
							ReadOnly:  true,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: getSchedulerHealthGetAction(),
						},
						FailureThreshold:    3,
						InitialDelaySeconds: 45,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      1,
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: getSchedulerHealthGetAction(),
						},
						FailureThreshold:    3,
						InitialDelaySeconds: 45,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      1,
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
		{
			Name: kubeSchedulerConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: kubeSchedulerConfigConfigMapName},
				},
			},
		},
		{
			Name: kubeSchedulerServingCertSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: kubeSchedulerServingCertSecretName,
				},
			},
		},
		{
			Name: resources.ApiserverFrontProxyClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverFrontProxyClientCertificateSecretName,
				},
			},
		},
	}
}

func getSchedulerHealthGetAction() *corev1.HTTPGetAction {
	return &corev1.HTTPGetAction{
		Path:   "/healthz",
		Scheme: corev1.URISchemeHTTPS,
		Port:   intstr.FromInt(10259),
	}
}
