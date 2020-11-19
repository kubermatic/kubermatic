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

package nodeportproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	hostTestPodName       = "host-test-container-pod"
	hostTestContainerName = "agnhost-container"
)

type NetworkingTestConfig struct {
	Log           *zap.SugaredLogger
	Namespace     string
	Client        ctrlclient.Client
	PodRestClient rest.Interface
	Config        *rest.Config

	// HostTestContainerPod is a pod running using the hostexec image.
	HostTestContainerPod *corev1.Pod
}

// DeployTestPod deploys the pod to be used to shoot the requests to the
// nodeport proxy service.
func (n *NetworkingTestConfig) DeployTestPod() error {
	hostTestContainerPod := n.newAgnhostPod(n.Namespace)
	if err := n.Client.Create(context.TODO(), hostTestContainerPod); err != nil {
		return errors.Wrap(err, "failed to create host test pod")
	}

	if err := n.waitForPodsReady(hostTestPodName); err != nil {
		return errors.Wrap(err, "timeout occurred while waiting for host test pod readiness")
	}

	if err := n.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: hostTestContainerPod.Namespace,
		Name:      hostTestContainerPod.Name,
	}, hostTestContainerPod); err != nil {
		return errors.Wrap(err, "failed to get host test pod")
	}
	n.HostTestContainerPod = hostTestContainerPod
	return nil
}

// newAgnhostPod returns a pod using the HostNetwork to be used to shoot the
// nodeport proxy service.
func (n *NetworkingTestConfig) newAgnhostPod(ns string) *corev1.Pod {
	immediate := int64(0)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostTestPodName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            hostTestContainerName,
					Image:           AgnosImage,
					Args:            []string{"pause"},
					SecurityContext: &corev1.SecurityContext{},
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
			HostNetwork:                   true,
			SecurityContext:               &corev1.PodSecurityContext{},
			TerminationGracePeriodSeconds: &immediate,
		},
	}
	return pod
}

// CleanUp deletes the resources.
func (n *NetworkingTestConfig) CleanUp() error {
	return n.Client.Delete(context.TODO(), n.HostTestContainerPod)
}

// Based on:
// https://github.com/mgdevstack/kubernetes/blob/9eced040142454a20255ae323279a38dc6b2bc1a/test/e2e/framework/network/utils.go#L347
// DialFromNode executes a tcp request based on protocol via kubectl exec
// in a test container running with host networking.
// - minTries is the minimum number of curl attempts required before declaring
//   success. Set to 0 if you'd like to return as soon as all endpoints respond
//   at least once.
// - maxTries is the maximum number of curl attempts. If this many attempts pass
//   and we don't see all expected endpoints, the test fails.
// maxTries == minTries will confirm that we see the expected endpoints and no
// more for maxTries. Use this if you want to eg: fail a readiness check on a
// pod and confirm it doesn't show up as an endpoint.
func (n *NetworkingTestConfig) DialFromNode(targetIP string, targetPort, maxTries, minTries int, expectedEps sets.String) sets.String {
	ipPort := net.JoinHostPort(targetIP, strconv.Itoa(targetPort))
	// The current versions of curl included in CentOS and RHEL distros
	// misinterpret square brackets around IPv6 as globbing, so use the -g
	// argument to disable globbing to handle the IPv6 case.
	cmd := fmt.Sprintf("curl -g -q -s --max-time 15 --connect-timeout 1 http://%s/hostName", ipPort)

	// TODO: This simply tells us that we can reach the endpoints. Check that
	// the probability of hitting a specific endpoint is roughly the same as
	// hitting any other.
	eps := sets.NewString()
	diff := sets.NewString()

	filterCmd := fmt.Sprintf("%s | grep -v '^\\s*$'", cmd)

	for i := 0; i < maxTries; i++ {
		stdout, stderr, err := n.exec("/bin/sh", "-c", filterCmd)
		if err != nil || len(stderr) > 0 {
			// A failure to exec command counts as a try, not a hard fail.
			// Also note that we will keep failing for maxTries in tests where
			// we confirm unreachability.
			n.Log.Infof("Failed to execute %q: %v, stdout: %q, stderr: %q", filterCmd, err, stdout, stderr)
		} else {
			trimmed := strings.TrimSpace(stdout)
			if trimmed != "" {
				eps.Insert(trimmed)
			}
		}

		diff = expectedEps.Difference(eps)

		// Check against i+1 so we exit if minTries == maxTries.
		if eps.Equal(expectedEps) && i+1 >= minTries {
			n.Log.Infof("Found all expected endpoints: %+v", eps.List())
			return diff
		}

		n.Log.Infof("Waiting for %+v endpoints (expected=%+v, actual=%+v)", diff, expectedEps.List(), eps.List())

		// TODO: get rid of this delay #36281
		time.Sleep(hitEndpointRetryDelay)
	}

	return diff
}

// exec executes the command in the host network container.
func (n *NetworkingTestConfig) exec(command ...string) (string, string, error) {

	const tty = false

	req := n.PodRestClient.Post().
		Resource("pods").
		Name(hostTestPodName).
		Namespace(n.Namespace).
		SubResource("exec").
		Param("container", hostTestContainerName)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: hostTestContainerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       tty,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err := execute("POST", req.URL(), n.Config, nil, &stdout, &stderr, tty)

	return stdout.String(), stderr.String(), err
}

func (n *NetworkingTestConfig) waitForPodsReady(pods ...string) error {
	if !CheckPodsRunningReady(n.Client, n.Namespace, pods, podReadinessTimeout) {
		return fmt.Errorf("timeout waiting for %d pods to be ready", len(pods))
	}
	return nil
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
