//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resources

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubelbresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	controllerResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.KubeLBDeploymentName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		},
	}
)

const (
	imageName = "kubelb-ccm-ee"
	imageTag  = "v1.3.1"
)

type kubeLBData interface {
	Cluster() *kubermaticv1.Cluster
	RewriteImage(string) (string, error)
	DC() *kubermaticv1.Datacenter
	KubeLBImageRepository() string
	KubeLBImageTag() string
}

func NewKubeLBData(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client, overwriteRegistry string, dc kubermaticv1.Datacenter, kubeLB kubermaticv1.KubeLBConfiguration) *resources.TemplateData {
	return resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithCluster(cluster).
		WithClient(client).
		WithOverwriteRegistry(overwriteRegistry).
		WithDatacenter(&dc).
		WithKubeLBImageRepository(kubeLB.ImageRepository).
		WithKubeLBImageTag(kubeLB.ImageTag).
		Build()
}

// DeploymentReconciler returns the function to create and update the kubeLB  deployment.
func DeploymentReconciler(data kubeLBData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.KubeLBDeploymentName, func(in *appsv1.Deployment) (*appsv1.Deployment, error) {
			_, creator := DeploymentReconcilerWithoutInitWrapper(data)()
			deployment, err := creator(in)
			if err != nil {
				return nil, err
			}

			deployment.Spec.Template, err = apiserver.IsRunningWrapper(data, deployment.Spec.Template, sets.New(resources.KubeLBDeploymentName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return deployment, nil
		}
	}
}

// DeploymentReconcilerWithoutInitWrapper returns the function to create and update the kubeLB deployment without the
// wrapper that checks for apiserver availability. This allows to adjust the command.
func DeploymentReconcilerWithoutInitWrapper(data kubeLBData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.KubeLBDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Labels = resources.BaseAppLabels(resources.KubeLBDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.KubeLBDeploymentName, nil),
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/scrape":                 "true",
				"prometheus.io/path":                   "/metrics",
				"prometheus.io/port":                   "8082",
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
			})

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				getCCMKubeconfigVolume(),
				getKubeLBManagerKubeconfigVolume(),
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			repository := registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/" + imageName))
			if r := data.KubeLBImageRepository(); r != "" {
				repository = r
			}
			tag := imageTag
			if t := data.KubeLBImageTag(); t != "" {
				tag = t
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.KubeLBDeploymentName,
					Image:   repository + ":" + tag,
					Command: []string{"/ccm"},
					Args:    getFlags(data.Cluster().Name, data.DC().Spec.KubeLB, data.Cluster().Spec.KubeLB),
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(8085),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold:    3,
						InitialDelaySeconds: 15,
						PeriodSeconds:       20,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
								Port:   intstr.FromInt(8085),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold:    3,
						InitialDelaySeconds: 5,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.KubeLBCCMKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      resources.KubeLBManagerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubelb-kubeconfig",
							ReadOnly:  true,
						},
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 8082,
							Name:          "ccm-metrics",
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			}

			dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, controllerResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getFlags(name string, kubelb *kubermaticv1.KubeLBDatacenterSettings, clusterKubeLB *kubermaticv1.KubeLB) []string {
	flags := []string{
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"-kubelb-kubeconfig", "/etc/kubernetes/kubelb-kubeconfig/kubeconfig",
		"-health-probe-bind-address", "0.0.0.0:8085",
		"-metrics-addr", "0.0.0.0:8082",
		"-leader-election-namespace", metav1.NamespaceSystem,
		"-cluster-name", fmt.Sprintf(kubelbresources.TenantNamespacePattern, name),
	}

	if kubelb != nil {
		flags = append(flags, "-node-address-type", kubelb.NodeAddressType)
		if kubelb.EnableSecretSynchronizer {
			flags = append(flags, "-enable-secret-synchronizer")
		}
		if kubelb.DisableIngressClass {
			flags = append(flags, "-use-ingress-class=false")
		}
	}

	if clusterKubeLB != nil && clusterKubeLB.EnableGatewayAPI != nil && *clusterKubeLB.EnableGatewayAPI {
		flags = append(flags, "-enable-gateway-api")
		flags = append(flags, "-install-gateway-api-crds")
	}
	if clusterKubeLB != nil && clusterKubeLB.UseLoadBalancerClass != nil && *clusterKubeLB.UseLoadBalancerClass {
		flags = append(flags, "-use-loadbalancer-class")
	}

	// Cluster configuration has a higher precedence than datacenter configuration.
	if clusterKubeLB != nil && clusterKubeLB.ExtraArgs != nil {
		for k, v := range clusterKubeLB.ExtraArgs {
			flags = append(flags, fmt.Sprintf("-%s=%s", k, v))
		}
	} else if kubelb != nil && kubelb.ExtraArgs != nil {
		for k, v := range kubelb.ExtraArgs {
			flags = append(flags, fmt.Sprintf("-%s=%s", k, v))
		}
	}

	return flags
}
