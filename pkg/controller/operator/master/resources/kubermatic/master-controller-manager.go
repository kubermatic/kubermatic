package kubermatic

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func masterControllerManagerPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel: common.MasterControllerManagerDeploymentName,
	}
}

func MasterControllerManagerDeploymentCreator(cfg *operatorv1alpha1.KubermaticConfiguration, workerName string, versions common.Versions) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return common.MasterControllerManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = cfg.Spec.MasterController.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: masterControllerManagerPodLabels(),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "json_iso",
			}

			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: common.DockercfgSecretName,
				},
			}

			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "seed-webhook-serving-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: common.SeedWebhookServingCertSecretName,
						},
					},
				},
			}

			args := []string{
				"-logtostderr",
				"-internal-address=0.0.0.0:8085",
				"-dynamic-datacenters=true",
				"-worker-count=20",
				fmt.Sprintf("-namespace=%s", cfg.Namespace),
				fmt.Sprintf("-pprof-listen-address=%s", *cfg.Spec.MasterController.PProfEndpoint),
				fmt.Sprintf("-seed-admissionwebhook-cert-file=/opt/seed-webhook-serving-cert/%s", resources.ServingCertSecretKey),
				fmt.Sprintf("-seed-admissionwebhook-key-file=/opt/seed-webhook-serving-cert/%s", resources.ServingCertKeySecretKey),
			}

			if cfg.Spec.MasterController.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			if workerName != "" {
				args = append(args, fmt.Sprintf("-worker-name=%s", workerName))
			}

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "controller-manager",
					Image:   cfg.Spec.MasterController.DockerRepository + ":" + versions.Kubermatic,
					Command: []string{"master-controller-manager"},
					Args:    args,
					Env:     common.ProxyEnvironmentVars(cfg),
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "seed-webhook-serving-cert",
							MountPath: "/opt/seed-webhook-serving-cert/",
							ReadOnly:  true,
						},
					},
					Resources: cfg.Spec.MasterController.Resources,
				},
			}

			return d, nil
		}
	}
}

func MasterControllerManagerPDBCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetCreatorGetter {
	name := "kubermatic-master-controller-manager"

	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return name, func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
			min := intstr.FromInt(1)

			pdb.Spec.MinAvailable = &min
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: masterControllerManagerPodLabels(),
			}

			return pdb, nil
		}
	}
}
