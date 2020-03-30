package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *testRunner) testPVC(log *zap.SugaredLogger, userClusterClient ctrlruntimeclient.Client, attempt int) error {
	log.Info("Testing support for PVC's...")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("pvc-test-%d", attempt),
		},
	}
	if err := userClusterClient.Create(context.Background(), ns); err != nil {
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
	if err := userClusterClient.Create(context.Background(), set); err != nil {
		return fmt.Errorf("failed to create statefulset: %v", err)
	}

	log.Info("Waiting until the StatefulSet is ready...")
	err := wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
		currentSet := &appsv1.StatefulSet{}
		name := types.NamespacedName{Namespace: ns.Name, Name: set.Name}
		if err := userClusterClient.Get(context.Background(), name, currentSet); err != nil {
			log.Warnf("failed to load StatefulSet %s/%s from API server during PVC test: %v", ns.Name, set.Name, err)
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

func (r *testRunner) testLB(log *zap.SugaredLogger, userClusterClient ctrlruntimeclient.Client, attempt int) error {
	log.Info("Testing support for LB's...")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("lb-test-%d", attempt),
		},
	}
	if err := userClusterClient.Create(context.Background(), ns); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	log.Info("Creating a Service of type LoadBalancer...")
	labels := map[string]string{"app": "hello"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns.Name,
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
	if err := userClusterClient.Create(context.Background(), service); err != nil {
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

	if err := userClusterClient.Create(context.Background(), pod); err != nil {
		return fmt.Errorf("failed to create Pod: %v", err)
	}

	var host string
	log.Debug("Waiting until the Service has a public IP/Name...")
	err := wait.Poll(10*time.Second, defaultTimeout, func() (done bool, err error) {
		currentService := &corev1.Service{}
		if err := userClusterClient.Get(context.Background(), types.NamespacedName{Namespace: ns.Name, Name: service.Name}, currentService); err != nil {
			log.Warnf("failed to load Service %s/%s from API server during LB test: %v", ns.Name, service.Name, err)
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
	err = wait.Poll(30*time.Second, defaultTimeout, func() (done bool, err error) {
		resp, err := http.Get(hostURL)
		if err != nil {
			log.Warnf("Failed to call Pod via LB(%s) during LB test: %v", hostURL, err)
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

func (r *testRunner) testUserClusterMetrics(log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client) error {
	log.Info("Testing user cluster metrics availability...")

	namespacedPod := types.NamespacedName{
		Namespace: cluster.Status.NamespaceName,
		Name:      "prometheus-0",
	}

	prometheusPod := &corev1.Pod{}
	if err := seedClient.Get(context.Background(), namespacedPod, prometheusPod); err != nil {
		return fmt.Errorf("failed to get prometheus pod: %v", err)
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", cluster.Status.NamespaceName, prometheusPod.Name)
	hostIP := strings.TrimLeft(r.seedRestConfig.Host, "htps:/")
	podURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	roundTripper, upgrader, err := spdy.RoundTripperFor(r.seedRestConfig)
	if err != nil {
		return fmt.Errorf("failed to create a roundTripper: %v", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &podURL)

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwarder, err := portforward.New(dialer, []string{"9090", "9090"}, stopChan, readyChan, out, errOut)
	if err != nil {
		return fmt.Errorf("failed creating a new port-forwarder: %v", err)
	}

	go handlePortForwardChan(r.log, readyChan, out, errOut)

	go func(logger *zap.SugaredLogger) {
		if err = forwarder.ForwardPorts(); err != nil {
			logger.Errorf("failed port-forwarding pod: %s with error: %v", prometheusPod.Name, err)
		}
	}(log)

	var (
		retries    = 0
		requestErr error
	)

	for {
		if retries >= 15 {
			return fmt.Errorf("failed to request metrics forever: %v", requestErr)
		}

		res, err := http.Get("http://localhost:9090/api/v1/label/__name__/values")
		if err != nil {
			requestErr = fmt.Errorf("failed to get prometheus metrics: %v", err)
			retries++
			time.Sleep(5 * time.Second)
			continue
		}

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get a proper response: response status code is %v", res.StatusCode)
		}

		data := &metricsData{}

		metricsBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed reading metrics response: %v", err)
		}

		if err := res.Body.Close(); err != nil {
			return fmt.Errorf("failed closing response: %v", err)
		}

		if err := json.Unmarshal(metricsBytes, data); err != nil {
			return fmt.Errorf("failed to unmarshal metrics response: %v", err)
		}

		if data.Status != "success" {
			return fmt.Errorf("failed to get prometheus metrics with data status: %s", data.Status)
		}

		if len(data.Data) == 0 {
			return errors.New("failed to get prometheus metrics: no metrics found")
		}

		fetchedMetricsSet := sets.NewString(data.Data...)

		if !fetchedMetricsSet.HasAll("machine_controller_machines", "kubelet_runtime_operations_latency_microseconds_count",
			"replicaset_controller_rate_limiter_use", "workqueue_retries_total", "ssh_tunnel_open_count", "scheduler_e2e_scheduling_duration_seconds_count",
			"kube_daemonset_labels", "etcd_disk_backend_defrag_duration_seconds_sum") {
			return errors.New("failed to get all expected metrics")
		}

		close(stopChan)

		return nil
	}
}

func handlePortForwardChan(logger *zap.SugaredLogger, readyChan chan struct{}, out, errOut fmt.Stringer) {
	for range readyChan {
	}
	if len(errOut.String()) != 0 {
		logger.Errorf("Error while port-forwarding: %s", errOut.String())
	}

	if len(out.String()) != 0 {
		logger.Infof("Port-forwarder returned: %s", errOut.String())
	}
}
