/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package metering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// The helpers in this file follow the ones in testkit/mla. They are kept
// separate until both suites settle, then they should move to a shared
// package.

// meteringPrometheus returns a Prometheus API client for the metering
// Prometheus, through the kube-apiserver service proxy.
func meteringPrometheus(in Input) (promv1.API, error) {
	httpClient, err := rest.HTTPClientFor(in.SeedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for the seed: %w", err)
	}
	httpClient.Timeout = 2 * time.Minute

	host, err := url.Parse(in.SeedConfig.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse seed host %q: %w", in.SeedConfig.Host, err)
	}

	proxyPath := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:80/proxy", in.SeedNamespace, prometheusName)

	client, err := promapi.NewClient(promapi.Config{
		Address: host.JoinPath(proxyPath).String(),
		Client:  httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return promv1.NewAPI(client), nil
}

// queryVector runs an instant query and returns the result as a vector.
func queryVector(ctx context.Context, prom promv1.API, query string, ts time.Time) (model.Vector, error) {
	result, _, err := prom.Query(ctx, query, ts)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", query, err)
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("query %s: expected a vector, got %s", query, result.Type())
	}

	return vector, nil
}

// patchWithRevert applies mutate to the current state of obj as a JSON merge
// patch that only holds the changed fields, and returns a function that
// applies the inverse patch. Fields the caller did not touch, including ones a
// controller changes in between, are left alone on revert.
func patchWithRevert(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, mutate func()) (func(context.Context) error, error) {
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), obj); err != nil {
		return nil, fmt.Errorf("failed to get %T %s: %w", obj, obj.GetName(), err)
	}

	original, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	mutate()

	modified, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	forward, err := jsonpatch.CreateMergePatch(original, modified)
	if err != nil {
		return nil, fmt.Errorf("failed to compute patch: %w", err)
	}

	reverse, err := jsonpatch.CreateMergePatch(modified, original)
	if err != nil {
		return nil, fmt.Errorf("failed to compute revert patch: %w", err)
	}

	if err := client.Patch(ctx, obj, ctrlruntimeclient.RawPatch(types.MergePatchType, forward)); err != nil {
		return nil, fmt.Errorf("failed to patch %T %s: %w", obj, obj.GetName(), err)
	}

	target := obj.DeepCopyObject().(ctrlruntimeclient.Object)

	return func(ctx context.Context) error {
		if err := client.Patch(ctx, target, ctrlruntimeclient.RawPatch(types.MergePatchType, reverse)); err != nil {
			return fmt.Errorf("failed to revert %T %s: %w", target, target.GetName(), err)
		}

		return nil
	}, nil
}

// revertContext is used for reverts, which must also run when the check's own
// context was cancelled or timed out.
func revertContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

// serviceAddress splits an in-cluster Service host name such as
// "minio.minio.svc.cluster.local" into the Service name and namespace. ok is
// false for any other host.
func serviceAddress(host string) (name, namespace string, ok bool) {
	parts := strings.Split(host, ".")
	if len(parts) < 3 || parts[2] != "svc" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// forwardServicePort opens a port-forward to a ready pod behind the Service
// and returns the local port. Callers must call the returned stop function.
func forwardServicePort(ctx context.Context, in Input, namespace, serviceName string, servicePort int32) (uint16, func(), error) {
	pod, podPort, err := podForServicePort(ctx, in.SeedClient, namespace, serviceName, servicePort)
	if err != nil {
		return 0, nil, err
	}

	transport, upgrader, err := spdy.RoundTripperFor(in.SeedConfig)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create SPDY transport: %w", err)
	}

	host, err := url.Parse(in.SeedConfig.Host)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse seed host %q: %w", in.SeedConfig.Host, err)
	}
	pfURL := host.JoinPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, pod.Name))

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, pfURL)

	stopCh := make(chan struct{})
	readyCh := make(chan struct{})

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("0:%d", podPort)}, stopCh, readyCh, io.Discard, io.Discard)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create port-forwarder: %w", err)
	}

	var once sync.Once
	stop := func() { once.Do(func() { close(stopCh) }) }

	errCh := make(chan error, 1)
	go func() {
		errCh <- forwarder.ForwardPorts()
	}()

	select {
	case <-readyCh:
	case err := <-errCh:
		return 0, nil, fmt.Errorf("port-forward to %s/%s failed: %w", namespace, pod.Name, err)
	case <-ctx.Done():
		stop()
		return 0, nil, ctx.Err()
	}

	ports, err := forwarder.GetPorts()
	if err != nil {
		stop()
		return 0, nil, fmt.Errorf("failed to get forwarded port: %w", err)
	}
	if len(ports) == 0 {
		stop()
		return 0, nil, errors.New("port-forward reports no ports")
	}

	return ports[0].Local, stop, nil
}

// podForServicePort returns a ready pod selected by the Service and the
// container port the given Service port targets.
func podForServicePort(ctx context.Context, client ctrlruntimeclient.Client, namespace, serviceName string, servicePort int32) (*corev1.Pod, int32, error) {
	service := &corev1.Service{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: serviceName}, service); err != nil {
		return nil, 0, fmt.Errorf("failed to get Service %s/%s: %w", namespace, serviceName, err)
	}

	var targetPort *intstr.IntOrString
	for _, port := range service.Spec.Ports {
		if port.Port == servicePort {
			targetPort = &port.TargetPort
			break
		}
	}
	if targetPort == nil {
		return nil, 0, fmt.Errorf("Service %s/%s has no port %d", namespace, serviceName, servicePort)
	}

	pods := &corev1.PodList{}
	if err := client.List(ctx, pods,
		ctrlruntimeclient.InNamespace(namespace),
		ctrlruntimeclient.MatchingLabelsSelector{Selector: labels.SelectorFromSet(service.Spec.Selector)},
	); err != nil {
		return nil, 0, fmt.Errorf("failed to list pods for Service %s/%s: %w", namespace, serviceName, err)
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.DeletionTimestamp != nil || !podReady(pod) {
			continue
		}

		if targetPort.Type == intstr.Int {
			if targetPort.IntVal == 0 {
				return pod, servicePort, nil
			}
			return pod, targetPort.IntVal, nil
		}

		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				if port.Name == targetPort.StrVal {
					return pod, port.ContainerPort, nil
				}
			}
		}
	}

	return nil, 0, fmt.Errorf("no ready pod found for Service %s/%s", namespace, serviceName)
}

func podReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}
