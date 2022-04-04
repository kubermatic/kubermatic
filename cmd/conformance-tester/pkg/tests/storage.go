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

package tests

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func supportsStorage(cluster *kubermaticv1.Cluster) bool {
	// list does not contain AWS because since 2022-Mar-02, we were
	// expecting problems while detaching volumes from instances
	return cluster.Spec.Cloud.Openstack != nil ||
		cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.VSphere != nil ||
		cluster.Spec.Cloud.GCP != nil ||
		cluster.Spec.Cloud.Hetzner != nil ||
		cluster.Spec.Cloud.Nutanix != nil
}

func TestStorage(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, attempt int) error {
	if !supportsStorage(cluster) {
		return nil
	}

	log.Info("Testing storage support...")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("pvc-test-%d", attempt),
		},
	}
	if err := userClusterClient.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	log.Info("Creating a StatefulSet with PVC...")
	labels := map[string]string{"app": "data-writer"}
	// Creating a simple StatefulSet with 1 replica which writes to the PV. That way we know if storage can be provisioned and consumed
	set := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "data-writer",
			Namespace: ns.Name,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "busybox",
							Image: "k8s.gcr.io/busybox",
							Args: []string{
								"/bin/sh",
								"-c",
								`echo "alive" > /data/healthy; sleep 3600`,
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 0,
								SuccessThreshold:    3,
								PeriodSeconds:       5,
								TimeoutSeconds:      1,
								FailureThreshold:    1,
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"cat",
											"/data/healthy",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
						},
					},
				},
			},
		},
	}
	if err := userClusterClient.Create(ctx, set); err != nil {
		return fmt.Errorf("failed to create statefulset: %w", err)
	}

	log.Info("Waiting until the StatefulSet is ready...")
	err := wait.Poll(10*time.Second, opts.CustomTestTimeout, func() (done bool, err error) {
		currentSet := &appsv1.StatefulSet{}
		name := types.NamespacedName{Namespace: ns.Name, Name: set.Name}
		if err := userClusterClient.Get(ctx, name, currentSet); err != nil {
			log.Warnf("Failed to fetch StatefulSet %s/%s: %v", ns.Name, set.Name, err)
			return false, nil
		}

		if currentSet.Status.ReadyReplicas == 1 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if StatefulSet is running: %w", err)
	}

	log.Info("Successfully validated storage support")
	return nil
}
