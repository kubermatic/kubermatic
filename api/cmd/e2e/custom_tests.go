package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	storageTestNamespace = "pvc-test"
	lbTestNamespace      = "lb-test"
)

func testStorageSupport(ctx *TestContext, r *R) {
	defer deleteNamespace(storageTestNamespace, ctx, r)

	// We're creating a own namespace, so the cleanup routine will take care as fallback
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: storageTestNamespace,
		},
	}
	if err := ctx.clusterContext.client.Create(ctx.ctx, ns); err != nil {
		r.Errorf("failed to create Namespace: %v", err)
		return
	}

	labels := map[string]string{"app": "data-writer"}
	// Creating a simple StatefulSet with 1 replica which writes to the PV. That way we know if storage can be provisioned and consumed
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "data-writer",
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
	if err := ctx.clusterContext.client.Create(ctx.ctx, statefulSet); err != nil {
		r.Errorf("failed to create StatefulSet: %v", err)
		return
	}

	err := wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
		select {
		case <-ctx.ctx.Done():
			return false, errors.New("parent context is closed")
		default:
		}

		currentSet := &appsv1.StatefulSet{}
		if err := ctx.clusterContext.client.Get(ctx.ctx, types.NamespacedName{Name: statefulSet.Name, Namespace: statefulSet.Namespace}, currentSet); err != nil {
			r.Logf("failed to load StatefulSet %s/%s from API server during PVC test: %v", ns.Name, statefulSet.Name, err)
			return false, nil
		}
		if currentSet.Status.ReadyReplicas == 1 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		r.Errorf("failed to check if StatefulSet is running: %v", err)
		return
	}
}

func testLBSupport(ctx *TestContext, r *R) {
	defer deleteNamespace(lbTestNamespace, ctx, r)

	// We're creating a own namespace, so the cleanup routine will take care as fallback
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: lbTestNamespace,
		},
	}
	if err := ctx.clusterContext.client.Create(ctx.ctx, ns); err != nil {
		r.Errorf("failed to create namespace: %v", err)
		return
	}

	labels := map[string]string{"app": "hello"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
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
	if err := ctx.clusterContext.client.Create(ctx.ctx, service); err != nil {
		r.Errorf("failed to create Service: %v", err)
		return
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "hello-kubernetes",
			Labels: labels,
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
	if err := ctx.clusterContext.client.Create(ctx.ctx, pod); err != nil {
		r.Errorf("failed to create Pod: %v", err)
		return
	}

	var host string
	err := wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
		select {
		case <-ctx.ctx.Done():
			return false, errors.New("parent context is closed")
		default:
		}

		currentService := &corev1.Service{}
		if err := ctx.clusterContext.client.Get(ctx.ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, currentService); err != nil {
			r.Logf("failed to load Service %s/%s from API server during LB test: %v", ns.Name, service.Name, err)
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
		r.Errorf("failed to check if Service is ready: %v", err)
		return
	}

	url := fmt.Sprintf("http://%s:80", host)
	err = wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
		select {
		case <-ctx.ctx.Done():
			return false, errors.New("parent context is closed")
		default:
		}

		resp, err := http.Get(url)
		if err != nil {
			r.Logf("Failed to call Pod via LB(%s) during LB test: %v", url, err)
			return false, nil
		}
		defer resp.Body.Close()

		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			r.Logf("Failed to read response body from Hello-Kubernetes Pod during LB test (%s): %v", url, err)
			return false, nil
		}

		if strings.Contains(string(contents), "Hello Kubernetes!") {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		r.Errorf("failed to check if Pod is available via LB: %v", err)
		return
	}
}
