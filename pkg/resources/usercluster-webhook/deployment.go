/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package webhook

import (
	"fmt"
	"strconv"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

type webhookData interface {
	Cluster() *kubermaticv1.Cluster
	DC() *kubermaticv1.Datacenter
	KubermaticAPIImage() string
	KubermaticDockerTag() string
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
}

func webhookPodLabels() map[string]string {
	return map[string]string{
		resources.AppLabelKey: resources.UserClusterWebhookDeploymentName,
	}
}

// DeploymentCreator returns the function to create and update the user cluster webhook deployment.
func DeploymentCreator(data webhookData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.UserClusterWebhookDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Name = resources.UserClusterWebhookDeploymentName
			d.Spec.Replicas = pointer.Int32Ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: webhookPodLabels(),
			}
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8080",
				"fluentbit.io/parser":  "json_iso",
			}

			projectID, ok := data.Cluster().Labels[kubermaticv1.ProjectIDLabelKey]
			if !ok {
				return nil, fmt.Errorf("no project-id label on cluster %q", data.Cluster().Name)
			}

			args := []string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"-webhook-cert-dir=/opt/webhook-serving-cert/",
				fmt.Sprintf("-webhook-cert-name=%s", resources.ServingCertSecretKey),
				fmt.Sprintf("-webhook-key-name=%s", resources.ServingCertKeySecretKey),
				fmt.Sprintf("-ca-bundle=/opt/ca-bundle/%s", resources.CABundleConfigMapKey),
				fmt.Sprintf("-project-id=%s", projectID),
			}

			if data.Cluster().Spec.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			envVars, err := getEnvVars(data)
			if err != nil {
				return nil, err
			}

			volumes := []corev1.Volume{
				{
					Name: "webhook-serving-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.UserClusterWebhookServingCertSecretName,
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
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.CABundleConfigMapName,
							},
						},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					Name:      "webhook-serving-cert",
					MountPath: "/opt/webhook-serving-cert/",
					ReadOnly:  true,
				},
				{
					Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
					MountPath: "/etc/kubernetes/kubeconfig",
					ReadOnly:  true,
				},
				{
					Name:      "ca-bundle",
					MountPath: "/opt/ca-bundle/",
					ReadOnly:  true,
				},
			}

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.UserClusterControllerDeploymentName,
					Image:   data.KubermaticAPIImage() + ":" + data.KubermaticDockerTag(),
					Command: []string{"user-cluster-webhook"},
					Args:    args,
					Env:     envVars,
					Ports: []corev1.ContainerPort{
						{
							Name:          "admission",
							ContainerPort: 9443,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "metrics",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: volumeMounts,
					Resources:    defaultResourceRequirements,
					ReadinessProbe: &corev1.Probe{
						InitialDelaySeconds: 3,
						TimeoutSeconds:      2,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.Parse("metrics"),
							},
						},
					},
				},
			}

			return d, nil
		}
	}
}

func getEnvVars(data webhookData) ([]corev1.EnvVar, error) {
	credentials, err := resources.GetCredentials(data)
	if err != nil {
		return nil, err
	}

	var vars []corev1.EnvVar
	if data.Cluster().Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: credentials.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: credentials.AWS.SecretAccessKey})
		vars = append(vars, corev1.EnvVar{Name: "AWS_ASSUME_ROLE_ARN", Value: credentials.AWS.AssumeRoleARN})
		vars = append(vars, corev1.EnvVar{Name: "AWS_ASSUME_ROLE_EXTERNAL_ID", Value: credentials.AWS.AssumeRoleExternalID})
	}
	if data.Cluster().Spec.Cloud.Azure != nil {
		vars = append(vars, corev1.EnvVar{Name: "AZURE_CLIENT_ID", Value: credentials.Azure.ClientID})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_CLIENT_SECRET", Value: credentials.Azure.ClientSecret})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_TENANT_ID", Value: credentials.Azure.TenantID})
		vars = append(vars, corev1.EnvVar{Name: "AZURE_SUBSCRIPTION_ID", Value: credentials.Azure.SubscriptionID})
	}
	if data.Cluster().Spec.Cloud.Openstack != nil {
		vars = append(vars, corev1.EnvVar{Name: "OS_AUTH_URL", Value: data.DC().Spec.Openstack.AuthURL})
		vars = append(vars, corev1.EnvVar{Name: "OS_USER_NAME", Value: credentials.Openstack.Username})
		vars = append(vars, corev1.EnvVar{Name: "OS_PASSWORD", Value: credentials.Openstack.Password})
		vars = append(vars, corev1.EnvVar{Name: "OS_DOMAIN_NAME", Value: credentials.Openstack.Domain})
		vars = append(vars, corev1.EnvVar{Name: "OS_PROJECT_NAME", Value: credentials.Openstack.Project})
		vars = append(vars, corev1.EnvVar{Name: "OS_PROJECT_ID", Value: credentials.Openstack.ProjectID})
		vars = append(vars, corev1.EnvVar{Name: "OS_APPLICATION_CREDENTIAL_ID", Value: credentials.Openstack.ApplicationCredentialID})
		vars = append(vars, corev1.EnvVar{Name: "OS_APPLICATION_CREDENTIAL_SECRET", Value: credentials.Openstack.ApplicationCredentialSecret})
		vars = append(vars, corev1.EnvVar{Name: "OS_TOKEN", Value: credentials.Openstack.Token})
	}
	if data.Cluster().Spec.Cloud.Hetzner != nil {
		vars = append(vars, corev1.EnvVar{Name: "HZ_TOKEN", Value: credentials.Hetzner.Token})
	}
	if data.Cluster().Spec.Cloud.Digitalocean != nil {
		vars = append(vars, corev1.EnvVar{Name: "DO_TOKEN", Value: credentials.Digitalocean.Token})
	}
	if data.Cluster().Spec.Cloud.VSphere != nil {
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_ADDRESS", Value: data.DC().Spec.VSphere.Endpoint})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_USERNAME", Value: credentials.VSphere.Username})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_PASSWORD", Value: credentials.VSphere.Password})
	}
	if data.Cluster().Spec.Cloud.Packet != nil {
		vars = append(vars, corev1.EnvVar{Name: "PACKET_API_KEY", Value: credentials.Packet.APIKey})
		vars = append(vars, corev1.EnvVar{Name: "PACKET_PROJECT_ID", Value: credentials.Packet.ProjectID})
	}
	if data.Cluster().Spec.Cloud.GCP != nil {
		vars = append(vars, corev1.EnvVar{Name: "GOOGLE_SERVICE_ACCOUNT", Value: credentials.GCP.ServiceAccount})
	}
	if data.Cluster().Spec.Cloud.Kubevirt != nil {
		vars = append(vars, corev1.EnvVar{Name: "KUBEVIRT_KUBECONFIG", Value: credentials.Kubevirt.KubeConfig})
	}
	if data.Cluster().Spec.Cloud.Alibaba != nil {
		vars = append(vars, corev1.EnvVar{Name: "ALIBABA_ACCESS_KEY_ID", Value: credentials.Alibaba.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "ALIBABA_ACCESS_KEY_SECRET", Value: credentials.Alibaba.AccessKeySecret})
	}
	if data.Cluster().Spec.Cloud.Anexia != nil {
		vars = append(vars, corev1.EnvVar{Name: "ANEXIA_TOKEN", Value: credentials.Anexia.Token})
	}
	if data.Cluster().Spec.Cloud.Nutanix != nil {
		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_ENDPOINT", Value: data.DC().Spec.Nutanix.Endpoint})
		if port := data.DC().Spec.Nutanix.Port; port != nil {
			vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PORT", Value: strconv.Itoa(int(*port))})
		}
		if data.DC().Spec.Nutanix.AllowInsecure {
			vars = append(vars, corev1.EnvVar{Name: "NUTANIX_INSECURE", Value: "true"})
		}

		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_USERNAME", Value: credentials.Nutanix.Username})
		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PASSWORD", Value: credentials.Nutanix.Password})
		if proxyURL := credentials.Nutanix.ProxyURL; proxyURL != "" {
			vars = append(vars, corev1.EnvVar{Name: "NUTANIX_PROXY_URL", Value: proxyURL})
		}

		vars = append(vars, corev1.EnvVar{Name: "NUTANIX_CLUSTER_NAME", Value: data.Cluster().Spec.Cloud.Nutanix.ClusterName})
	}
	if data.Cluster().Spec.Cloud.VMwareCloudDirector != nil {
		vars = append(vars, corev1.EnvVar{Name: "VCD_URL", Value: data.DC().Spec.VMwareCloudDirector.URL})

		if data.DC().Spec.VMwareCloudDirector.AllowInsecure {
			vars = append(vars, corev1.EnvVar{Name: "VCD_ALLOW_UNVERIFIED_SSL", Value: "true"})
		}

		vars = append(vars, corev1.EnvVar{Name: "VCD_USER", Value: credentials.VMwareCloudDirector.Username})
		vars = append(vars, corev1.EnvVar{Name: "VCD_PASSWORD", Value: credentials.VMwareCloudDirector.Password})
		vars = append(vars, corev1.EnvVar{Name: "VCD_ORG", Value: credentials.VMwareCloudDirector.Organization})
		vars = append(vars, corev1.EnvVar{Name: "VCD_VDC", Value: credentials.VMwareCloudDirector.VDC})
	}

	vars = resources.SanitizeEnvVars(vars)

	return vars, nil
}
