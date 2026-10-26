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
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/podexec"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	gatewayProbePodName   = "mla-gateway-probe"
	gatewayProbeContainer = "probe"
	gatewayProbeSecret    = gatewayProbePodName + "-tls"

	// gatewayName matches the unexported constant in
	// pkg/controller/seed-controller-manager/mla/resources.go and the value of
	// the app.kubernetes.io/name label on the gateway pods.
	gatewayName = "mla-gateway"

	// lokiPushJob is the log stream label the probe writes and later queries,
	// so it can be told apart from real agent traffic.
	lokiPushJob = "mla-gateway-e2e-probe"

	// lokiPushPath is the gateway write-path location for Loki
	// (mla/resources.go:89).
	lokiPushPath = "/loki/api/v1/push"

	// forgedTenantID is the tenant a client would write to if the gateway honored
	// a client-supplied X-Scope-OrgID.
	forgedTenantID = "mla-e2e-forged-tenant"

	// lokiSpoofJob is the stream pushed with the forged tenant header, kept apart
	// from the lokiPushJob stream.
	lokiSpoofJob = "mla-gateway-e2e-spoof"

	// lokiQueryFrontendAddr is the Loki read endpoint the gateway proxies to. It
	// is queried directly to ask for a tenant the gateway would overwrite.
	lokiQueryFrontendAddr = "loki-distributed-query-frontend.mla.svc.cluster.local:3100"

	// paths the staged mTLS material is mounted at inside the probe pod.
	probeTLSMountPath = "/etc/ssl/mla"
	probeCertPath     = probeTLSMountPath + "/client.crt"
	probeKeyPath      = probeTLSMountPath + "/client.key"
)

// gatewayProbe is the agnhost pod the data-plane checks curl from: it holds the
// monitoring-agent client cert, so it can talk to the gateway write port, and it
// runs in the seed cluster namespace, so it can also reach cluster-local
// Services (the gateway read port, the mla namespace). It is addressed by name,
// so every test process can exec into the same pod.
type gatewayProbe struct {
	in        Input
	namespace string
}

func (p *gatewayProbe) exec(ctx context.Context, command ...string) (string, string, error) {
	return podexec.ExecuteCommand(ctx, p.in.SeedConfig, types.NamespacedName{
		Namespace: p.namespace,
		Name:      gatewayProbePodName,
	}, gatewayProbeContainer, command...)
}

// ensureGatewayProbe makes sure the probe pod and its TLS Secret exist in the
// cluster namespace and the gateway rollout is complete. It is idempotent and
// safe to call from parallel test processes.
func ensureGatewayProbe(ctx context.Context, in Input) (*gatewayProbe, error) {
	clusterNS := in.Cluster.Status.NamespaceName
	if clusterNS == "" {
		return nil, errors.New("cluster has no namespace yet (Status.NamespaceName empty)")
	}

	// the agent mTLS client cert is reconciled into the user cluster and signed
	// by the seed-side mla-gateway-ca, the CA the gateway verifies clients
	// against. The CA itself is not needed here: the probe curls a pod IP, so
	// server cert verification is skipped (-k) and only the client cert matters.
	var clientCert, clientKey []byte
	if err := wait.Poll(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		certSecret := &corev1.Secret{}
		if err := in.UserClient.Get(ctx, types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLAMonitoringAgentCertificatesSecretName}, certSecret); err != nil {
			if apierrors.IsNotFound(err) {
				return err, nil
			}

			return nil, fmt.Errorf("failed to get monitoring agent cert secret: %w", err)
		}

		clientCert = certSecret.Data[resources.MLAMonitoringAgentClientCertSecretKey]
		clientKey = certSecret.Data[resources.MLAMonitoringAgentClientKeySecretKey]

		if len(clientCert) == 0 || len(clientKey) == 0 {
			return errors.New("mla material not yet populated"), nil
		}

		return nil, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to gather mTLS material: %w", err)
	}

	// stage the cert/key in the seed cluster-namespace as a Secret the probe
	// pod mounts, so curl can present them without writing them to a command line.
	if err := ensureProbeSecret(ctx, in.SeedClient, clusterNS, clientCert, clientKey); err != nil {
		return nil, err
	}

	if _, err := waitGatewayPodReady(ctx, in.SeedClient, clusterNS); err != nil {
		return nil, fmt.Errorf("failed to find ready gateway pod: %w", err)
	}

	if err := ensurePod(ctx, in.SeedClient, in, newGatewayProbePod(clusterNS)); err != nil {
		return nil, fmt.Errorf("failed to deploy gateway probe pod: %w", err)
	}

	return &gatewayProbe{in: in, namespace: clusterNS}, nil
}

func ensureProbeSecret(ctx context.Context, client ctrlruntimeclient.Client, namespace string, cert, key []byte) error {
	data := map[string][]byte{
		resources.MLAMonitoringAgentClientCertSecretKey: cert,
		resources.MLAMonitoringAgentClientKeySecretKey:  key,
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayProbeSecret,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	err := client.Create(ctx, secret)
	if err == nil {
		return nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create probe TLS secret: %w", err)
	}

	existing := &corev1.Secret{}
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(secret), existing); err != nil {
		return fmt.Errorf("failed to get probe TLS secret: %w", err)
	}

	oldExisting := existing.DeepCopy()
	existing.Data = data

	if err := client.Patch(ctx, existing, ctrlruntimeclient.MergeFrom(oldExisting)); err != nil {
		return fmt.Errorf("failed to update probe TLS secret: %w", err)
	}

	return nil
}

// CleanupGatewayProbe deletes the gateway probe pod and its TLS Secret.
func CleanupGatewayProbe(ctx context.Context, in Input) error {
	namespace := in.Cluster.Status.NamespaceName

	var errs []error

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: gatewayProbePodName, Namespace: namespace}}
	if err := in.SeedClient.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete gateway probe pod: %w", err))
	}

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: gatewayProbeSecret, Namespace: namespace}}
	if err := in.SeedClient.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		errs = append(errs, fmt.Errorf("failed to delete probe TLS secret: %w", err))
	}

	return errors.Join(errs...)
}

// pushCurl builds a write-path curl that presents the agent client cert and
// prints only the HTTP status code.
//
// -k: the server cert SAN does not cover a pod IP, but client-cert
// verification (ssl_verify_client on) still runs server-side regardless.
func pushCurl(podIP, path string, extra ...string) []string {
	curl := []string{
		"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"--connect-timeout", "3", "--max-time", "15",
		"--cert", probeCertPath, "--key", probeKeyPath, "-k",
	}
	curl = append(curl, extra...)

	return append(curl, gatewayPushURL(podIP, path))
}

// gatewayPushURL is the gateway write port, which has no in-cluster Service
// (mla-gateway ClusterIP fronts only the read port 8081), so callers curl a
// gateway pod IP directly.
func gatewayPushURL(podIP, path string) string {
	return fmt.Sprintf("https://%s%s", net.JoinHostPort(podIP, "8080"), path)
}

// lokiQueryRangeURL builds a query_range request for a single job label, over a
// window around the moment the stream was pushed.
func lokiQueryRangeURL(hostPort, job string, pushTime time.Time) string {
	return fmt.Sprintf(
		"http://%s/loki/api/v1/query_range?query=%s&start=%d&end=%d",
		hostPort,
		url.QueryEscape(fmt.Sprintf("{job=%q}", job)),
		pushTime.Add(-time.Minute).UnixNano(),
		pushTime.Add(10*time.Minute).UnixNano(),
	)
}

// gatewayReadAddr is the gateway read path: the mla-gateway Service on port 80,
// which fronts the internal read server on 8081.
func gatewayReadAddr(clusterNS string) string {
	return net.JoinHostPort(fmt.Sprintf("%s.%s.svc.cluster.local", gatewayName, clusterNS), "80")
}

// lokiQueryResponse is the part of a Loki query response the checks assert on.
type lokiQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     []json.RawMessage `json:"result"`
	} `json:"data"`
}

// parseLokiQueryResponse decodes a Loki query body and requires a successful
// response: error bodies echo the query, so a substring check for the job label
// alone would also pass on failures.
func parseLokiQueryResponse(body string) (*lokiQueryResponse, error) {
	response := &lokiQueryResponse{}
	if err := json.Unmarshal([]byte(body), response); err != nil {
		return nil, fmt.Errorf("unable to decode response %q: %w", body, err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("expected a successful response, got %q", body)
	}

	return response, nil
}

// curlGet is a plain GET from the probe pod, with the response body on stdout.
func curlGet(url string, extra ...string) []string {
	curl := []string{"curl", "-s", "--connect-timeout", "3", "--max-time", "15"}
	curl = append(curl, extra...)

	return append(curl, url)
}

// waitForLokiStream polls a Loki query from the probe pod until the stream
// with the given job label is visible.
func waitForLokiStream(ctx context.Context, probe *gatewayProbe, command []string, job string, timeout time.Duration) error {
	return wait.Poll(ctx, 2*time.Second, timeout, func(ctx context.Context) (error, error) {
		stdout, _, err := probe.exec(ctx, command...)
		if err != nil {
			return err, nil
		}

		response, err := parseLokiQueryResponse(stdout)
		if err != nil {
			return err, nil
		}

		if len(response.Data.Result) == 0 || !strings.Contains(stdout, job) {
			return errors.New("stream not yet visible"), nil
		}

		return nil, nil
	})
}

// pushLokiStream pushes body through the gateway write path and expects a 2xx.
// Pod readiness only proves the 8081 probe endpoint, so the first push can
// still hit a transient distributor or resolver blip; retry briefly. The pod IP
// is re-resolved per attempt in case a gateway rollout replaces the pod
// mid-check.
func pushLokiStream(ctx context.Context, probe *gatewayProbe, body string, extraHeaders ...string) error {
	return wait.Poll(ctx, 2*time.Second, 60*time.Second, func(ctx context.Context) (error, error) {
		podIP, err := readyGatewayPodIP(ctx, probe.in.SeedClient, probe.namespace)
		if err != nil {
			return err, nil
		}

		args := []string{"-XPOST", "-H", "Content-Type: application/json"}
		for _, header := range extraHeaders {
			args = append(args, "-H", header)
		}
		args = append(args, "-d", body)

		code, _, err := probe.exec(ctx, pushCurl(podIP, lokiPushPath, args...)...)
		if err != nil {
			return fmt.Errorf("push failed to execute: %w", err), nil
		}

		if !isSuccessPushCode(code) {
			return fmt.Errorf("push returned HTTP %s", code), nil
		}

		return nil, nil
	})
}

// CheckGatewayWritePath exercises the MLA Gateway data plane directly. It
// confirms two NGINX behaviors that an image bump could regress: that a real
// mTLS push round-trips to storage (W1) and that a client without a cert is
// rejected (W2, ssl_verify_client on).
//
// This covers the gateway's NGINX behavior across all expose strategies; the
// nodeport-proxy layer in front is out of scope.
func CheckGatewayWritePath(ctx context.Context, in Input) error {
	probe, err := ensureGatewayProbe(ctx, in)
	if err != nil {
		return err
	}

	var problems []string

	// a real mTLS push round-trips to storage. Push a log stream, then read
	// it back through the gateway read path (port 80 -> 8081, no TLS) and assert
	// the stream is visible. Proves proxy_pass on both ports plus the
	// X-Scope-OrgID tenant injection end to end.
	pushTime := time.Now()
	pushBody := fmt.Sprintf(
		`{"streams":[{"stream":{"job":%q},"values":[[%q,"probe"]]}]}`,
		lokiPushJob, strconv.FormatInt(pushTime.UnixNano(), 10),
	)

	if err := pushLokiStream(ctx, probe, pushBody); err != nil {
		problems = append(problems, fmt.Sprintf("W1 push did not succeed: %v", err))
	}

	// deliberately no X-Scope-OrgID header: the gateway read server overwrites
	// it with the cluster name (proxy_set_header at server scope), so getting
	// the pushed stream back proves the gateway's tenant injection works on
	// both paths, not just that loki honors a header the test set itself.
	readCmd := curlGet(lokiQueryRangeURL(gatewayReadAddr(probe.namespace), lokiPushJob, pushTime))
	if err := waitForLokiStream(ctx, probe, readCmd, lokiPushJob, 90*time.Second); err != nil {
		problems = append(problems, fmt.Sprintf("W1 pushed stream not readable via gateway read path: %v", err))
	}

	// the write-path checks above can outlive the pod resolved at the start,
	// so pick a live pod again for the no-cert probe.
	if noCertPodIP, err := readyGatewayPodIP(ctx, in.SeedClient, probe.namespace); err != nil {
		problems = append(problems, fmt.Sprintf("W2 could not resolve a ready gateway pod: %v", err))
	} else {
		noCertCurl := []string{
			"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
			"--connect-timeout", "3", "--max-time", "15", "-k",
			"-XPOST", "-H", "Content-Type: application/json", "-d", pushBody, gatewayPushURL(noCertPodIP, lokiPushPath),
		}

		noCertCode, _, noCertErr := probe.exec(ctx, noCertCurl...)
		switch {
		case noCertErr == nil && isSuccessPushCode(noCertCode):
			problems = append(problems, fmt.Sprintf("W2 push without client cert unexpectedly succeeded (HTTP %s); mTLS enforcement broken", noCertCode))
		case noCertErr == nil:
			// handshake completed but nginx refused the request at the HTTP level
			// (400 "No required SSL certificate was sent"); mTLS enforced.
		case isTLSRejection(noCertErr):
			// curl bailed during or right after the handshake; mTLS enforced.
		default:
			// an infrastructure failure (timeout, pod gone) must not pass as
			// proof of mTLS enforcement.
			problems = append(problems, fmt.Sprintf("W2 push without client cert failed for a non-TLS reason, cannot prove mTLS enforcement: %v", noCertErr))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("gateway write path checks failed:\n%s", strings.Join(problems, "\n"))
	}

	return nil
}

// CheckTenantHeaderSpoofing checks that a user cluster cannot write into or
// read from another tenant by setting X-Scope-OrgID itself: the gateway's
// server-scope proxy_set_header overwrites whatever the client sent.
//
// The read-back through the gateway carries the assertion. It sends no
// X-Scope-OrgID, and loki runs with auth_enabled, so both regressions fail it: a
// honored client header would put the stream under forgedTenantID, out of reach
// of a read tenanted to the cluster name, and a gateway that stopped setting the
// header at all would make the read a 401. The direct loki queries afterwards
// name the tenant the data actually landed in.
func CheckTenantHeaderSpoofing(ctx context.Context, in Input) error {
	probe, err := ensureGatewayProbe(ctx, in)
	if err != nil {
		return err
	}

	pushTime := time.Now()
	pushBody := fmt.Sprintf(
		`{"streams":[{"stream":{"job":%q},"values":[[%q,"spoof-probe"]]}]}`,
		lokiSpoofJob, strconv.FormatInt(pushTime.UnixNano(), 10),
	)
	forgedHeader := fmt.Sprintf("X-Scope-OrgID: %s", forgedTenantID)

	var problems []string

	// the gateway must ignore the forged header, not reject the request, so a
	// non-2xx here is a failure just like it is for W1.
	if err := pushLokiStream(ctx, probe, pushBody, forgedHeader); err != nil {
		problems = append(problems, fmt.Sprintf("push with a forged tenant header did not succeed: %v", err))
	}

	// no header is sent here, so the gateway is the only thing that can supply one
	gatewayReadCmd := curlGet(lokiQueryRangeURL(gatewayReadAddr(probe.namespace), lokiSpoofJob, pushTime))
	if err := waitForLokiStream(ctx, probe, gatewayReadCmd, lokiSpoofJob, 90*time.Second); err != nil {
		problems = append(problems, fmt.Sprintf("stream pushed with a forged tenant header is not readable under the real tenant %q: %v", in.Cluster.Name, err))
	}

	// control, and the precondition for the check after it: the same query
	// straight at loki, asking for the real tenant. Without it, an empty
	// forged-tenant result could just mean loki is unreachable or has not indexed
	// the stream yet.
	directURL := lokiQueryRangeURL(lokiQueryFrontendAddr, lokiSpoofJob, pushTime)
	realTenantCmd := curlGet(directURL, "-H", fmt.Sprintf("X-Scope-OrgID: %s", in.Cluster.Name))

	if err := waitForLokiStream(ctx, probe, realTenantCmd, lokiSpoofJob, 60*time.Second); err != nil {
		problems = append(problems, fmt.Sprintf("control query against %s for tenant %q never returned the stream, so the forged-tenant check would be inconclusive and was skipped: %v", lokiQueryFrontendAddr, in.Cluster.Name, err))
	} else {
		// not polled: this asserts data is *not* there, so retrying would only
		// widen the window for it to show up.
		forgedTenantCmd := curlGet(directURL, "-H", forgedHeader)

		stdout, _, execErr := probe.exec(ctx, forgedTenantCmd...)
		if execErr != nil {
			problems = append(problems, fmt.Sprintf("forged-tenant query failed to execute, cannot prove the tenant is empty: %v", execErr))
		} else if response, err := parseLokiQueryResponse(stdout); err != nil {
			problems = append(problems, fmt.Sprintf("forged-tenant query did not return a usable response: %v", err))
		} else if len(response.Data.Result) > 0 || strings.Contains(stdout, lokiSpoofJob) {
			problems = append(problems, fmt.Sprintf("stream is readable under the forged tenant %q, the gateway honored the client header: %s", forgedTenantID, stdout))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("tenant header spoofing checks failed:\n%s", strings.Join(problems, "\n"))
	}

	return nil
}

// promQueryResponse is the part of a Cortex instant-query response the metrics
// check asserts on.
type promQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// CheckMetricsRoundTrip exercises the Cortex data plane end to end: the
// annotated user-cluster pod is scraped by the monitoring agent, remote_written
// through the gateway write path to cortex-distributor and queried back out of
// the gateway read path. Requires EnsureMetricsProbe.
func CheckMetricsRoundTrip(ctx context.Context, in Input) error {
	if err := EnsureMetricsProbe(ctx, in); err != nil {
		return err
	}

	probe, err := ensureGatewayProbe(ctx, in)
	if err != nil {
		return err
	}

	// without a healthy agent nothing can be scraped, so check it first to get a
	// precise error instead of an opaque query timeout
	if err := utils.WaitForDeploymentReady(ctx, in.UserClient, in.Log, resources.UserClusterMLANamespace, resources.MLAMonitoringAgentDeploymentName, 5*time.Minute); err != nil {
		return fmt.Errorf("monitoring agent Deployment did not get ready: %w", err)
	}

	query := fmt.Sprintf("%s{run=%q, job=\"kubernetes-pods\"}", metricsProbeMetric, in.Cluster.Name)
	queryCmd := curlGet(fmt.Sprintf(
		"http://%s/api/prom/api/v1/query?query=%s",
		gatewayReadAddr(probe.namespace),
		url.QueryEscape(query),
	))

	// on failure this is the only evidence of what the read path answered
	lastBody := ""

	queryErr := wait.Poll(ctx, 5*time.Second, 6*time.Minute, func(ctx context.Context) (error, error) {
		stdout, _, execErr := probe.exec(ctx, queryCmd...)
		if execErr != nil {
			return execErr, nil
		}

		lastBody = stdout

		response := &promQueryResponse{}
		if err := json.Unmarshal([]byte(stdout), response); err != nil {
			return fmt.Errorf("unable to decode response %q: %w", stdout, err), nil
		}

		if response.Status != "success" {
			return fmt.Errorf("expected a successful response, got %q", stdout), nil
		}

		if response.Data.ResultType != "vector" {
			return fmt.Errorf("expected an instant vector, got resultType %q", response.Data.ResultType), nil
		}

		if len(response.Data.Result) == 0 {
			return errors.New("series not visible in Cortex yet"), nil
		}

		// the labels arrive with the series, so a mismatch will not fix itself; fail
		// right away with the actual label set. This catches a regression in the
		// agent's relabel pipeline, which emits kubernetes_namespace and
		// kubernetes_pod_name rather than the namespace/pod a stock Prometheus
		// setup would. A single series is expected: cortex runs the HA tracker, so
		// the agent's two replicas do not each produce one.
		metric := response.Data.Result[0].Metric
		expectedLabels := map[string]string{
			"__name__":             metricsProbeMetric,
			"job":                  "kubernetes-pods",
			"kubernetes_namespace": metav1.NamespaceDefault,
			"kubernetes_pod_name":  metricsProbePodName,
			"run":                  in.Cluster.Name,
		}

		for key, expected := range expectedLabels {
			if actual := metric[key]; actual != expected {
				return nil, fmt.Errorf("expected label %s=%q, got %q (full label set: %v)", key, expected, actual, metric)
			}
		}

		return nil, nil
	})
	if queryErr != nil {
		// the agent logs that tell an agent-side failure apart from a cortex-side
		// one are captured by protokol from the user cluster's mla-system namespace,
		// see hack/ci/run-mla-e2e-tests.sh
		return fmt.Errorf("%s not queryable through the gateway read path (last response: %q): %w", query, lastBody, queryErr)
	}

	return nil
}

// CheckLogsRoundTrip proves the user-cluster logging agent works end to end:
// the logs probe pod's marker line must travel agent -> mla-gateway -> Loki and
// be readable through the gateway read path. Requires EnsureLogsProbe.
func CheckLogsRoundTrip(ctx context.Context, in Input) error {
	if err := EnsureLogsProbe(ctx, in); err != nil {
		return err
	}

	probe, err := ensureGatewayProbe(ctx, in)
	if err != nil {
		return err
	}

	// without a healthy agent nothing is shipped, so check it first to get a
	// precise error instead of an opaque query timeout
	if err := utils.WaitForDaemonSetReady(ctx, in.UserClient, in.Log, resources.UserClusterMLANamespace, resources.MLALoggingAgentDaemonSetName, 5*time.Minute); err != nil {
		return fmt.Errorf("logging agent DaemonSet did not get ready: %w", err)
	}

	// the probe emits continuously, so a window around now contains marker lines
	readCmd := curlGet(lokiQueryRangeURL(gatewayReadAddr(probe.namespace), logsProbeJob, time.Now()))

	lastBody := ""

	queryErr := wait.Poll(ctx, 5*time.Second, 6*time.Minute, func(ctx context.Context) (error, error) {
		stdout, _, execErr := probe.exec(ctx, readCmd...)
		if execErr != nil {
			return execErr, nil
		}

		lastBody = stdout

		response, err := parseLokiQueryResponse(stdout)
		if err != nil {
			return err, nil
		}

		if len(response.Data.Result) == 0 {
			return errors.New("stream not visible in Loki yet"), nil
		}

		if !strings.Contains(stdout, logsProbeMarker) {
			return fmt.Errorf("stream found but marker %q missing (response: %q)", logsProbeMarker, stdout), nil
		}

		return nil, nil
	})
	if queryErr != nil {
		return fmt.Errorf("job %q not queryable through the gateway read path (last response: %q): %w", logsProbeJob, lastBody, queryErr)
	}

	return nil
}

// isTLSRejection reports whether an exec error looks like curl refusing at the
// TLS layer: exit 35 (SSL connect error) when the handshake is rejected
// outright, exit 56 (recv failure) when nginx completes the TLS 1.3 handshake
// and then closes the connection with a certificate-required alert.
func isTLSRejection(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "exit code 35") || strings.Contains(msg, "exit code 56")
}

// waitGatewayPodReady returns the pod IP of a ready mla-gateway pod in the
// cluster namespace. It first waits for the gateway Deployment rollout to be
// complete: rate limit changes rewrite the gateway nginx ConfigMap, and the
// checksum annotation forces a rolling update. A pod picked mid-rollout can
// belong to the losing ReplicaSet and be deleted while a check is still
// curling its IP.
func waitGatewayPodReady(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (string, error) {
	var gatewayPodIP string
	if err := wait.Poll(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		deployment := &appsv1.Deployment{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: gatewayName}, deployment); err != nil {
			return fmt.Errorf("failed to get gateway deployment: %w", err), nil
		}

		replicas := int32(1)
		if deployment.Spec.Replicas != nil {
			replicas = *deployment.Spec.Replicas
		}

		if deployment.Status.ObservedGeneration < deployment.Generation ||
			deployment.Status.UpdatedReplicas != replicas ||
			deployment.Status.ReadyReplicas != replicas ||
			deployment.Status.Replicas != replicas {
			return errors.New("gateway deployment rollout not complete yet"), nil
		}

		ip, err := readyGatewayPodIP(ctx, client, namespace)
		if err != nil {
			return err, nil
		}

		gatewayPodIP = ip
		return nil, nil
	}); err != nil {
		return "", err
	}

	return gatewayPodIP, nil
}

// readyGatewayPodIP returns the pod IP of a ready, non-terminating mla-gateway
// pod. Callers that curl the gateway over a longer period re-resolve the IP
// per attempt so a rollout between attempts does not leave them talking to a
// deleted pod.
func readyGatewayPodIP(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (string, error) {
	pods := &corev1.PodList{}
	if err := client.List(ctx, pods,
		ctrlruntimeclient.InNamespace(namespace),
		ctrlruntimeclient.MatchingLabels{"app.kubernetes.io/name": gatewayName}); err != nil {
		return "", fmt.Errorf("failed to list gateway pods: %w", err)
	}

	for i := range pods.Items {
		p := &pods.Items[i]
		if p.DeletionTimestamp == nil && p.Status.PodIP != "" && nginxReady(p) {
			return p.Status.PodIP, nil
		}
	}

	return "", errors.New("no ready gateway pod")
}

func nginxReady(p *corev1.Pod) bool {
	for _, c := range p.Status.ContainerStatuses {
		if c.Name == "nginx" {
			return c.Ready
		}
	}

	return false
}

// isSuccessPushCode treats any 2xx as a successful write (Loki push normally
// returns 204). An empty code means curl bailed before any HTTP response
// (TLS handshake failure), which is a failure for positive assertions and a
// pass for the mTLS-negative one.
func isSuccessPushCode(code string) bool {
	return len(code) == 3 && code[0] == '2'
}

// newGatewayProbePod returns an agnhost pod that stays alive (pause) and mounts
// the staged TLS material so curl inside it can present the client cert.
func newGatewayProbePod(namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayProbePodName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            gatewayProbeContainer,
					Image:           agnhostImage,
					Args:            []string{"pause"},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "tls", MountPath: probeTLSMountPath, ReadOnly: true},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "tls",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  gatewayProbeSecret,
							DefaultMode: ptr.To[int32](0400),
						},
					},
				},
			},
			TerminationGracePeriodSeconds: ptr.To[int64](0),
		},
	}
}
