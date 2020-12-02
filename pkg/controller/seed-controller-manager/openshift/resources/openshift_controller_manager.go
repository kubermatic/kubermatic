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
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/servingcerthelper"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

var (
	controllerManagerDefaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
	openshiftControllerManagerConfigTemplate = template.Must(template.New("base").Funcs(sprig.TxtFuncMap()).Parse(openshiftControllerManagerConfigTemplateRaw))
)

const (
	OpenshiftControllerManagerDeploymentName        = "openshift-controller-manager"
	openshiftControllerManagerConfigMapName         = "openshift-controller-manager-config"
	openshiftControllerManagerServingCertSecretName = "openshift-controller-manager-serving-cert"
	openshiftControllerManagerConfigMapKey          = "config.yaml"
	openshiftControllerManagerConfigTemplateRaw     = `
apiVersion: openshiftcontrolplane.config.openshift.io/v1
kind: OpenShiftControllerManagerConfig
build:
  imageTemplateFormat:
    format: {{ .BuildImageTemplateFormatImage }}
deployer:
  imageTemplateFormat:
    format: {{ .DeployerImageTemplateFormatImage }}
dockerPullSecret:
  internalRegistryHostname: image-registry.openshift-image-registry.svc:5000
kubeClientConfig:
  kubeConfig: /etc/kubernetes/kubeconfig/kubeconfig
servingInfo:
  certFile: /etc/openshift/pki/serving/serving.crt
  keyFile: /etc/openshift/pki/serving/serving.key
  clientCA: /etc/openshift/pki/ca/ca.crt
`
)

func OpenshiftControllerManagerConfigMapCreator(data openshiftData) reconciling.NamedConfigMapCreatorGetter {
	openshiftVersion := data.Cluster().Spec.Version.String()
	return func() (string, reconciling.ConfigMapCreator) {
		return openshiftControllerManagerConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			buildImageTemplateFormatImage, err := dockerBuilderImage(openshiftVersion, data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}
			deployerImageTemplateFormatImage, err := deployerImage(openshiftVersion, data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}

			vars := struct {
				BuildImageTemplateFormatImage    string
				DeployerImageTemplateFormatImage string
			}{
				BuildImageTemplateFormatImage:    buildImageTemplateFormatImage,
				DeployerImageTemplateFormatImage: deployerImageTemplateFormatImage,
			}
			templateBuffer := &bytes.Buffer{}
			if err := openshiftControllerManagerConfigTemplate.Execute(templateBuffer, vars); err != nil {
				return nil, fmt.Errorf("failed to execute template: %v", err)
			}

			cm.Data[openshiftControllerManagerConfigMapKey] = templateBuffer.String()
			return cm, nil
		}
	}
}

// OpenshiftControllerManagerServingCertSecretCreator returns the function to create and update the serving cert for the openshift controller manager
func OpenshiftControllerManagerServingCertSecretCreator(caGetter servingcerthelper.CAGetter) reconciling.NamedSecretCreatorGetter {
	return servingcerthelper.ServingCertSecretCreator(caGetter,
		openshiftControllerManagerServingCertSecretName,
		"controller-manager.openshift-controller-manager.svc",
		[]string{"controller-manager.openshift-controller-manager.svc", "controller-manager.openshift-controller-manager.svc.cluster.local"},
		nil)
}

// OpenshiftControllerManagerDeploymentCreator returns the function to create and update the controller manager deployment
func OpenshiftControllerManagerDeploymentCreator(ctx context.Context, data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return OpenshiftControllerManagerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.ControllerManagerDeploymentName
			dep.Labels = resources.BaseAppLabels(OpenshiftControllerManagerDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas
			}
			dep.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(OpenshiftControllerManagerDeploymentName, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: openshiftImagePullSecretName}}

			volumes := getControllerManagerVolumes()
			podLabels, err := data.GetPodTemplateLabelsWithContext(ctx, OpenshiftControllerManagerDeploymentName, volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/path":                  "/metrics",
					"prometheus.io/scrape_with_kube_cert": "true",
					"prometheus.io/port":                  "8444",
				},
			}

			// Configure user cluster DNS resolver for this pod.
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Volumes = volumes

			image, err := hypershiftImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:    OpenshiftControllerManagerDeploymentName,
					Image:   image,
					Command: []string{"hypershift", "openshift-controller-manager"},
					Args:    []string{"--config=/etc/origin/master/config.yaml", "-v=2"},
					Env: []corev1.EnvVar{{
						Name:  "KUBECONFIG",
						Value: "/etc/kubernetes/kubeconfig/kubeconfig",
					}},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      openshiftControllerManagerConfigMapName,
							MountPath: "/etc/origin/master",
						},
						{
							MountPath: "/etc/openshift/pki/ca/ca.crt",
							Name:      resources.CASecretName,
							SubPath:   "ca.crt",
							ReadOnly:  true,
						},
						{
							MountPath: "/etc/openshift/pki/serving",
							Name:      openshiftControllerManagerServingCertSecretName,
							ReadOnly:  true,
						},
					},
				},
			}
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				OpenshiftControllerManagerDeploymentName: controllerManagerDefaultResourceRequirements.DeepCopy(),
				openvpnSidecar.Name:                      openvpnSidecar.Resources.DeepCopy(),
			}
			overrides := map[string]*corev1.ResourceRequirements{}
			if data.Cluster().Spec.ComponentsOverride.ControllerManager.Resources != nil {
				overrides[OpenshiftAPIServerDeploymentName] = data.Cluster().Spec.ComponentsOverride.ControllerManager.Resources.DeepCopy()
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, overrides, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}
			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(OpenshiftControllerManagerDeploymentName, data.Cluster().Name)

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(OpenshiftControllerManagerDeploymentName), "OAuthClient,oauth.openshift.io/v1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

func getControllerManagerVolumes() []corev1.Volume {
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
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
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
		{
			Name: openshiftControllerManagerConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: openshiftControllerManagerConfigMapName},
				},
			},
		},
		{
			Name: openshiftControllerManagerServingCertSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: openshiftControllerManagerServingCertSecretName,
				},
			},
		},
	}
}
