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

package usercluster

import (
	"encoding/json"
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("25m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		},
	}
)

const name = "usercluster-controller"

// userclusterControllerData is the subet of the deploymentData interface
// that is actually required by the usercluster deployment
// This makes importing the the deployment elsewhere (openshift controller)
// easier as only have to implement the parts that are actually in use
type userclusterControllerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
	GetOpenVPNServerPort() (int32, error)
	KubermaticAPIImage() string
	GetKubernetesCloudProviderName() string
	CloudCredentialSecretTemplate() ([]byte, error)
}

// DeploymentCreator returns the function to create and update the user cluster controller deployment
func DeploymentCreator(data userclusterControllerData, openshift bool) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.UserClusterControllerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.UserClusterControllerDeploymentName
			dep.Labels = resources.BaseAppLabels(name, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(name, nil),
			}
			dep.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
			dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
				MaxSurge: &intstr.IntOrString{
					Type: intstr.Int,
					// The readiness probe only turns ready if a sync succeeded.
					// That requires that the controller acquires the leader lock, which only happens if the other instance stops
					IntVal: 1,
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			openvpnServerPort, err := data.GetOpenVPNServerPort()
			if err != nil {
				return nil, err
			}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/path":   "/metrics",
					"prometheus.io/port":   "8085",
				},
			}

			dep.Spec.Template.Spec.Volumes = volumes

			dnsClusterIP, err := resources.UserClusterDNSResolverIP(data.Cluster())
			if err != nil {
				return nil, err
			}
			args := append([]string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"-metrics-listen-address", "0.0.0.0:8085",
				"-health-listen-address", "0.0.0.0:8086",
				"-namespace", "$(NAMESPACE)",
				"-cluster-url", data.Cluster().Address.URL,
				"-dns-cluster-ip", dnsClusterIP,
				"-openvpn-server-port", fmt.Sprint(openvpnServerPort),
				"-overwrite-registry", data.ImageRegistry(""),
				fmt.Sprintf("-openshift=%t", openshift),
				"-version", data.Cluster().Spec.Version.String(),
				"-cloud-provider-name", data.GetKubernetesCloudProviderName(),
				"-owner-email", data.Cluster().Status.UserEmail,
				fmt.Sprintf("-opa-integration=%t", data.Cluster().Spec.OPAIntegration != nil && data.Cluster().Spec.OPAIntegration.Enabled),
			}, getNetworkArgs(data)...)

			if openshiftConsoleCallbackURI := data.Cluster().Address.OpenshiftConsoleCallBack; openshiftConsoleCallbackURI != "" {
				args = append(args, "-openshift-console-callback-uri", openshiftConsoleCallbackURI)
			}

			if data.Cluster().Spec.UpdateWindow != nil && data.Cluster().Spec.UpdateWindow.Length != "" && data.Cluster().Spec.UpdateWindow.Start != "" {
				args = append(args, "-update-window-start", data.Cluster().Spec.UpdateWindow.Start, "-update-window-length", data.Cluster().Spec.UpdateWindow.Length)
			}

			labelArgsValue, err := getLabelsArgValue(data.Cluster())
			if err != nil {
				return nil, fmt.Errorf("failed to get label args value: %v", err)
			}
			if labelArgsValue != "" {
				args = append(args, "-node-labels", labelArgsValue)
			}

			cloudCredentialSecretTemplate, err := data.CloudCredentialSecretTemplate()
			if err != nil {
				return nil, fmt.Errorf("failed to get cloud-credential-secret-template: %v", err)
			}
			if cloudCredentialSecretTemplate != nil {
				args = append(args, "-cloud-credential-secret-template", string(cloudCredentialSecretTemplate))
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    name,
					Image:   data.KubermaticAPIImage() + ":" + resources.KUBERMATICCOMMIT,
					Command: []string{"/usr/local/bin/user-cluster-controller-manager"},
					Args:    args,
					Env: []corev1.EnvVar{
						{
							Name: "NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath:  "metadata.namespace",
									APIVersion: "v1",
								},
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/ready",
								Port:   intstr.FromInt(8086),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold: 5,
						PeriodSeconds:    5,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
					},
				},
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}
			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName

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
			Name: resources.InternalUserClusterAdminKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
				},
			},
		},
	}
}

func getNetworkArgs(data userclusterControllerData) []string {
	networkFlags := make([]string, len(data.Cluster().Spec.MachineNetworks)*2)
	i := 0

	for _, n := range data.Cluster().Spec.MachineNetworks {
		networkFlags[i] = "--ipam-controller-network"
		i++
		networkFlags[i] = fmt.Sprintf("%s,%s,%s", n.CIDR, n.Gateway, strings.Join(n.DNSServers, ","))
		i++
	}

	return networkFlags
}

func getLabelsArgValue(cluster *kubermaticv1.Cluster) (string, error) {
	labelsToApply := map[string]string{}
	for key, value := range cluster.Labels {
		if kubermaticv1.ProtectedClusterLabels.Has(key) {
			continue
		}
		labelsToApply[key] = value
	}

	if len(labelsToApply) == 0 {
		return "", nil
	}

	bytes, err := json.Marshal(labelsToApply)
	if err != nil {
		return "", fmt.Errorf("failed to marshal labels: %v", err)
	}
	return string(bytes), nil
}
