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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *testRunner) testPVC(ctx context.Context, log *zap.SugaredLogger, userClusterClient ctrlruntimeclient.Client, attempt int) error {
	log.Info("Testing support for PVC's...")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("pvc-test-%d", attempt),
		},
	}
	if err := userClusterClient.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
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
								Handler: corev1.Handler{
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
		return fmt.Errorf("failed to create statefulset: %v", err)
	}

	log.Info("Waiting until the StatefulSet is ready...")
	err := wait.Poll(10*time.Second, r.customTestTimeout, func() (done bool, err error) {
		currentSet := &appsv1.StatefulSet{}
		name := types.NamespacedName{Namespace: ns.Name, Name: set.Name}
		if err := userClusterClient.Get(ctx, name, currentSet); err != nil {
			log.Warnf("Failed to load StatefulSet %s/%s from API server during PVC test: %v", ns.Name, set.Name, err)
			return false, nil
		}

		if currentSet.Status.ReadyReplicas == 1 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if StatefulSet is running: %v", err)
	}

	log.Info("Successfully validated PVC support")
	return nil
}

func (r *testRunner) testLB(ctx context.Context, log *zap.SugaredLogger, userClusterClient ctrlruntimeclient.Client, attempt int) error {
	log.Info("Testing support for LB's...")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("lb-test-%d", attempt),
		},
	}
	if err := userClusterClient.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	log.Info("Creating a Service of type LoadBalancer...")
	labels := map[string]string{"app": "hello"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns.Name,
			Annotations: map[string]string{
				"load-balancer.hetzner.cloud/location": "nbg1",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
	if err := userClusterClient.Create(ctx, service); err != nil {
		return fmt.Errorf("failed to create Service: %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello-kubernetes",
			Namespace: ns.Name,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "hello-kubernetes",
					Image: "gcr.io/google-samples/node-hello:1.0",
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	if err := userClusterClient.Create(ctx, pod); err != nil {
		return fmt.Errorf("failed to create Pod: %v", err)
	}

	var host string
	log.Debug("Waiting until the Service has a public IP/Name...")
	err := wait.Poll(3*time.Second, r.customTestTimeout, func() (done bool, err error) {
		currentService := &corev1.Service{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: service.Name}, currentService); err != nil {
			log.Warnf("Failed to load Service %s/%s from API server during LB test: %v", ns.Name, service.Name, err)
			return false, nil
		}
		if len(currentService.Status.LoadBalancer.Ingress) > 0 {
			host = currentService.Status.LoadBalancer.Ingress[0].IP
			if host == "" {
				host = currentService.Status.LoadBalancer.Ingress[0].Hostname

				// We wait until we can actually resolve the name
				ips, err := net.LookupHost(host)
				if err != nil || len(ips) == 0 {
					return false, nil
				}
			}
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if Service is ready: %v", err)
	}
	log.Debug("The Service has an external IP/Name")

	hostURL := fmt.Sprintf("http://%s:80", host)
	log.Debug("Waiting until the pod is available via the LB...")
	err = wait.Poll(3*time.Second, r.customTestTimeout, func() (done bool, err error) {
		resp, err := http.Get(hostURL)
		if err != nil {
			log.Warnf("Failed to call Pod via LB (%s) during LB test: %v", hostURL, err)
			return false, nil
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Warnf("Failed to close response body from Hello-Kubernetes Pod during LB test (%s): %v", hostURL, err)
			}
		}()
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Warnf("Failed to read response body from Hello-Kubernetes Pod during LB test (%s): %v", hostURL, err)
			return false, nil
		}

		if strings.Contains(string(contents), "Hello Kubernetes!") {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if Pod is available via LB: %v", err)
	}

	log.Info("Successfully validated LB support")
	return nil
}

type metricsData struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

// testUserClusterMetrics ensures all expected metrics are actually collected
// in Prometheus. Note that this assumes that some time has passed between
// Prometheus' eployment and this test, so it can scrape all targets. This
// includes kubelets, so nodes must have been ready for at least 30 seconds
// before this can succeed.
func (r *testRunner) testUserClusterMetrics(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client) error {
	log.Info("Testing user cluster metrics availability...")

	res := r.seedGeneratedClient.CoreV1().RESTClient().Get().
		Namespace(cluster.Status.NamespaceName).
		Resource("pods").
		Name("prometheus-0:9090").
		SubResource("proxy").
		Suffix("api/v1/label/__name__/values").
		Do(ctx)

	if err := res.Error(); err != nil {
		return fmt.Errorf("request to Prometheus failed: %v", err)
	}

	code := 0
	res.StatusCode(&code)
	if code != http.StatusOK {
		return fmt.Errorf("Prometheus returned HTTP %d", code)
	}

	body, err := res.Raw()
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	data := &metricsData{}
	if err := json.Unmarshal(body, data); err != nil {
		return fmt.Errorf("failed to unmarshal metrics response: %v", err)
	}

	if data.Status != "success" {
		return fmt.Errorf("failed to get prometheus metrics with data status: %s", data.Status)
	}

	expected := sets.NewString(
		"etcd_disk_backend_defrag_duration_seconds_sum",
		"kube_daemonset_labels",
		"kubelet_runtime_operations_duration_seconds_count",
		"machine_controller_machines",
		"replicaset_controller_rate_limiter_use",
		"scheduler_e2e_scheduling_duration_seconds_count",
		"apiserver_request_total",
		"workqueue_retries_total",
	)
	fetched := sets.NewString(data.Data...)
	missing := expected.Difference(fetched)

	if missing.Len() > 0 {
		return fmt.Errorf("failed to get all expected metrics: got: %v, %v are missing", fetched.List(), missing.List())
	}

	log.Info("Successfully validated user cluster metrics.")
	return nil
}

func (r *testRunner) testUserClusterPodAndNodeMetrics(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster,
	userClusterClient ctrlruntimeclient.Client) error {
	log.Info("Testing user cluster pod and node metrics availability...")

	// check node metrics
	allNodeMetricsList := &v1beta1.NodeMetricsList{}
	if err := userClusterClient.List(ctx, allNodeMetricsList); err != nil {
		return fmt.Errorf("error getting node metrics list: %v", err)
	}
	if len(allNodeMetricsList.Items) == 0 {
		return fmt.Errorf("node metrics list is empty")
	}

	for _, nodeMetric := range allNodeMetricsList.Items {
		// check a metric to see if it works
		if nodeMetric.Usage.Memory().IsZero() {
			return fmt.Errorf("node %q memory usage metric is 0", nodeMetric.Name)
		}
	}

	// create pod to check metrics
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "check-metrics",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "hello-kubernetes",
					Image: "gcr.io/google-samples/node-hello:1.0",
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	if err := userClusterClient.Create(ctx, pod); err != nil {
		return fmt.Errorf("failed to create Pod: %v", err)
	}

	err := wait.Poll(r.userClusterPollInterval, r.customTestTimeout, func() (done bool, err error) {
		metricPod := &corev1.Pod{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, metricPod); err != nil {
			log.Warnw("Failed to get test metric pod", zap.Error(err))
			return false, nil
		}

		if !podIsReady(metricPod) {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if test metrics pod is ready: %v", err)
	}

	// check pod metrics
	err = wait.Poll(r.userClusterPollInterval, r.customTestTimeout, func() (done bool, err error) {
		podMetrics := &v1beta1.PodMetrics{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, podMetrics); err != nil {
			log.Warnw("Failed to get test metric pod metrics", zap.Error(err))
			return false, nil
		}
		for _, cont := range podMetrics.Containers {
			if cont.Usage.Memory().IsZero() {
				log.Warnw("Metrics test pod memory usage is 0", zap.Error(err))
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to get metric test pod metrics: %v", err)
	}

	log.Info("Successfully validated user cluster pod and node metrics.")
	return nil
}
