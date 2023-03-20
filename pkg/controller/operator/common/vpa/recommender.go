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

package vpa

import (
	"fmt"
	"strconv"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	RecommenderName = "vpa-recommender"
	recommenderPort = 8942
)

func RecommenderServiceAccountReconciler() reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return RecommenderName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func appPodLabels(appName string) map[string]string {
	return map[string]string{
		common.NameLabel: appName,
	}
}

func RecommenderDeploymentReconciler(cfg *kubermaticv1.KubermaticConfiguration, versions kubermatic.Versions) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return RecommenderName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = pointer.Int32(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: appPodLabels(RecommenderName),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   strconv.Itoa(recommenderPort),
				"fluentbit.io/parser":  "glog",
			}

			d.Spec.Template.Spec.ServiceAccountName = RecommenderName
			d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: pointer.Bool(true),
				RunAsUser:    pointer.Int64(65534),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "recommender",
					Image:   cfg.Spec.VerticalPodAutoscaler.Recommender.DockerRepository + ":" + versions.VPA,
					Command: []string{"/recommender"},
					Args: []string{
						fmt.Sprintf("--address=:%d", recommenderPort),
						"--kube-api-burst=20",
						"--kube-api-qps=10",
						"--storage=prometheus",
						"--prometheus-address=http://prometheus.monitoring.svc.cluster.local:9090",
						"--prometheus-cadvisor-job-name=cadvisor-vpa",
						"--metric-for-pod-labels=kube_pod_labels",
						"--pod-namespace-label=namespace",
						"--pod-name-label=pod",
						"--pod-label-prefix=label_",
						"--recommendation-margin-fraction=0",
						"--logtostderr",
						"--v=4",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: recommenderPort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			}

			if res := cfg.Spec.VerticalPodAutoscaler.Recommender.Resources; res != nil {
				d.Spec.Template.Spec.Containers[0].Resources = *res
			}

			return d, nil
		}
	}
}
