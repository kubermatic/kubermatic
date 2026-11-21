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

package mla

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// mlaNamespace is where the usercluster-mla installer stack deploys the
	// MLA backends on the seed.
	mlaNamespace = "mla"

	grafanaSecretName  = "grafana"
	grafanaServiceName = "grafana"
	grafanaServicePort = 80

	cortexAlertmanagerService = "cortex-alertmanager"
	cortexAlertmanagerPort    = 8080
	cortexRulerService        = "cortex-ruler"
	cortexRulerPort           = 8080
	lokiRulerService          = "loki-distributed-ruler"
	lokiRulerPort             = 3100

	httpTimeout = 15 * time.Second
)

// serviceProxy talks to a seed Service through the kube-apiserver service
// proxy. It only works for backends that do not need an Authorization header:
// the kube-apiserver strips that header after it authenticated the caller.
type serviceProxy struct {
	client  *http.Client
	baseURL string
}

func newServiceProxy(cfg *rest.Config, namespace, service string, port int) (*serviceProxy, error) {
	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for the seed: %w", err)
	}
	httpClient.Timeout = httpTimeout

	host, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse seed host %q: %w", cfg.Host, err)
	}

	proxyPath := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy", namespace, service, port)

	return &serviceProxy{
		client:  httpClient,
		baseURL: host.JoinPath(proxyPath).String(),
	}, nil
}

// newPodProxy talks to one pod port through the kube-apiserver pod proxy.
func newPodProxy(cfg *rest.Config, namespace, pod string, port int) (*serviceProxy, error) {
	proxy, err := newServiceProxy(cfg, namespace, "unused", port)
	if err != nil {
		return nil, err
	}

	host, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}

	proxy.baseURL = host.JoinPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%d/proxy", namespace, pod, port)).String()

	return proxy, nil
}

// do sends a request to path on the proxied Service and returns the status
// code and the body. body may be nil.
func (p *serviceProxy) do(ctx context.Context, method, path string, header http.Header, body io.Reader) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return resp.StatusCode, respBody, nil
}

// grafanaConn is an admin connection to the user cluster MLA Grafana. Grafana
// needs basic auth, which cannot pass the kube-apiserver service proxy, so the
// connection goes through a port-forward to a Grafana pod.
type grafanaConn struct {
	client   *grafanasdk.Client
	baseURL  string
	user     string
	password string
	stop     func()
}

// newGrafanaClient returns a Grafana admin client. Callers must call the
// returned stop function when done.
func newGrafanaClient(ctx context.Context, in Input) (*grafanasdk.Client, func(), error) {
	conn, err := newGrafanaConn(ctx, in)
	if err != nil {
		return nil, nil, err
	}

	return conn.client, conn.stop, nil
}

func newGrafanaConn(ctx context.Context, in Input) (*grafanaConn, error) {
	secret := &corev1.Secret{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: mlaNamespace, Name: grafanaSecretName}, secret); err != nil {
		return nil, fmt.Errorf("failed to get Grafana Secret: %w", err)
	}

	adminName, ok := secret.Data[mla.GrafanaUserKey]
	if !ok {
		return nil, fmt.Errorf("Grafana Secret does not contain %s key", mla.GrafanaUserKey)
	}
	adminPass, ok := secret.Data[mla.GrafanaPasswordKey]
	if !ok {
		return nil, fmt.Errorf("Grafana Secret does not contain %s key", mla.GrafanaPasswordKey)
	}

	localPort, stop, err := forwardServicePort(ctx, in, mlaNamespace, grafanaServiceName, grafanaServicePort)
	if err != nil {
		return nil, fmt.Errorf("failed to port-forward to Grafana: %w", err)
	}

	conn := &grafanaConn{
		baseURL:  fmt.Sprintf("http://127.0.0.1:%d", localPort),
		user:     string(adminName),
		password: string(adminPass),
		stop:     stop,
	}

	conn.client, err = grafanasdk.NewClient(conn.baseURL, fmt.Sprintf("%s:%s", adminName, adminPass), &http.Client{Timeout: httpTimeout})
	if err != nil {
		stop()
		return nil, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	return conn, nil
}

// get sends an admin GET request in the given org and decodes the JSON body.
func (c *grafanaConn) get(ctx context.Context, orgID uint, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.user, c.password)
	req.Header.Set("X-Grafana-Org-Id", fmt.Sprintf("%d", orgID))

	resp, err := (&http.Client{Timeout: httpTimeout}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: expected HTTP 200, got HTTP %d: %s", path, resp.StatusCode, body)
	}

	return json.Unmarshal(body, out)
}

// forwardServicePort port-forwards a random local port to a ready pod behind
// the given Service port. The forwarding goroutine ends when stop is called.
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

	return nil, 0, errors.New("no ready pod found for the Service")
}

func podReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}
