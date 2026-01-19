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

package metricsserver

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/servingcerthelper"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("200Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}
)

const (
	name = "metrics-server"
	// ServingCertSecretName is the name of the secret containing the metrics-server
	// serving cert.
	ServingCertSecretName  = "metrics-server-serving-cert"
	servingCertMountFolder = "/etc/serving-cert"

	tag = "v0.7.0"
)

// metricsServerData is the data needed to construct the metrics-server components.
type metricsServerData interface {
	Cluster() *kubermaticv1.Cluster
	GetRootCA() (*triple.KeyPair, error)
	RewriteImage(string) (string, error)
	NodeAccessNetwork() string
	IsKonnectivityEnabled() bool
}

// TLSServingCertSecretReconciler returns a function to manage the TLS serving cert for the metrics
// server.
func TLSServingCertSecretReconciler(caGetter servingcerthelper.CAGetter) reconciling.NamedSecretReconcilerFactory {
	dnsName := "metrics-server.kube-system.svc"
	return servingcerthelper.ServingCertSecretReconciler(caGetter,
		ServingCertSecretName,
		// Must match what's configured in the apiservice in pkg/controller/usercluster/resources/metrics-server/external-name-service.go.
		// Can unfortunately not have a trailing dot, as that's only allowed in Kube 1.16+
		dnsName,
		[]string{dnsName},
		nil)
}

// DeploymentReconciler returns the function to create and update the metrics server deployment.
func DeploymentReconciler(data metricsServerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.MetricsServerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(name, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
			})

			dep.Spec.Template.Spec.Volumes = getVolumes()

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    name,
					Image:   registry.Must(data.RewriteImage(resources.RegistryK8S + "/metrics-server/metrics-server:" + tag)),
					Command: []string{"/metrics-server"},
					Args: []string{
						"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--kubelet-insecure-tls",
						"--kubelet-use-node-status-port",
						"--secure-port", "10250",
						"--metric-resolution", "15s",
						"--kubelet-preferred-address-types", resources.GetKubeletPreferredAddressTypes(data.Cluster(), data.IsKonnectivityEnabled()),
						"--v", "1",
						"--tls-cert-file", servingCertMountFolder + "/" + resources.ServingCertSecretKey,
						"--tls-private-key-file", servingCertMountFolder + "/" + resources.ServingCertKeySecretKey,
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 10250,
							Name:          "https",
							Protocol:      corev1.ProtocolTCP,
						},
					},
					// Do not define a readiness probe, as the metrics-server will only get ready
					// when it has scraped a node or pod at least once, which might never happen in
					// clusters without nodes. An unready metrics-server would prevent the
					// SeedResourcesUpToDate condition to become true.
					// ReadinessProbe: nil,
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   1,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/livez",
								Port:   intstr.FromString("https"),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
					},
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
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
						AllowPrivilegeEscalation: resources.Bool(false),
						ReadOnlyRootFilesystem:   resources.Bool(true),
						RunAsNonRoot:             resources.Bool(true),
						RunAsUser:                resources.Int64(1000),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			}
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				name: defaultResourceRequirements.DeepCopy(),
			}

			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, kubermaticv1.AntiAffinityTypePreferred)

			dep.Spec.Template, err = apiserver.IsRunningWrapper(data, dep.Spec.Template, sets.New(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return dep, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	vs := []corev1.Volume{
		{
			Name: resources.MetricsServerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.MetricsServerKubeconfigSecretName,
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
	return vs
}
