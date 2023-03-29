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

package cloudcontroller

import (
	"fmt"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/api/v3/pkg/semver"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	AzureCCMDeploymentName = "azure-cloud-controller-manager"
)

var (
	azureResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("128Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("4"),
		},
	}
)

func azureDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return AzureCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = AzureCCMDeploymentName
			dep.Labels = resources.BaseAppLabels(AzureCCMDeploymentName, nil)
			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(OpenstackCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(OpenstackCCMDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			version, err := getAzureVersion(data.Cluster().Status.Versions.ControlPlane)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.AutomountServiceAccountToken = pointer.Bool(false)
			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), true)
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:         ccmContainerName,
					Image:        registry.Must(data.RewriteImage(resources.RegistryMCR + "/oss/kubernetes/azure-cloud-controller-manager:v" + version)),
					Command:      []string{"cloud-controller-manager"},
					Args:         getAzureFlags(data),
					Env:          getEnvVars(),
					VolumeMounts: getVolumeMounts(true),
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Scheme: corev1.URISchemeHTTP,
								Path:   "/healthz",
								Port:   intstr.FromInt(10258),
							},
						},
						SuccessThreshold:    1,
						FailureThreshold:    3,
						InitialDelaySeconds: 20,
						PeriodSeconds:       10,
						TimeoutSeconds:      5,
					},
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName: azureResourceRequirements.DeepCopy(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getAzureVersion(version semver.Semver) (string, error) {
	// reminder: do not forget to update addons/azure-cloud-node-manager as well!
	switch version.MajorMinor() {
	case v123:
		return "1.23.28", nil
	case v124:
		return "1.24.15", nil
	case v125:
		return "1.25.9", nil
	case v126:
		fallthrough
	default:
		return "1.26.5", nil
	}
}

func getAzureFlags(data *resources.TemplateData) []string {
	flags := []string{
		// "false" as we use IPAM in kube-controller-manager
		"--allocate-node-cidrs=false",
		// "false" as we use VXLAN overlay for pod network for all clusters ATM
		"--configure-cloud-routes=false",
		"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		"--v=4",
		"--cloud-config=/etc/kubernetes/cloud/config",
		"--cloud-provider=azure",
		"--leader-elect=true",
		"--route-reconciliation-period=10s",
		// This configures the secure port, but the CCM allows unauthenticated
		// access to /healthz, /readyz and /livez for the health checks.
		"--secure-port=10258",
		"--controllers=*,-cloud-node",
	}
	if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] {
		flags = append(flags, "--cluster-name", data.Cluster().Name)
	}
	return flags
}
