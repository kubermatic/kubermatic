package vpa

import (
	"fmt"
	"strconv"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	UpdaterName = "vpa-updater"
	updaterPort = 8942
)

func UpdaterServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return UpdaterName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func UpdaterDeploymentCreator(cfg *operatorv1alpha1.KubermaticConfiguration, versions common.Versions) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return UpdaterName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = pointer.Int32Ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: appPodLabels(UpdaterName),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   strconv.Itoa(updaterPort),
				"fluentbit.io/parser":  "glog",
			}

			d.Spec.Template.Spec.ServiceAccountName = UpdaterName
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "updater",
					Image:   cfg.Spec.VerticalPodAutoscaler.Updater.DockerRepository + ":" + versions.VPA,
					Command: []string{"/updater"},
					Args: []string{
						fmt.Sprintf("--address=:%d", updaterPort),
						"--evict-after-oom-treshold=30m",
						"--updater-interval=10m",
						"--logtostderr",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: updaterPort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: cfg.Spec.VerticalPodAutoscaler.Updater.Resources,
				},
			}

			return d, nil
		}
	}
}
