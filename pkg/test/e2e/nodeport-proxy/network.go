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
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"

	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
)

const (
	hostTestPodName       = "host-test-container-pod"
	hostTestContainerName = "agnhost-container"
)

type networkingTestConfig struct {
	e2eutils.TestPodConfig
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
func (n *networkingTestConfig) DialFromNode(targetIP string, targetPort, maxTries, minTries int, expectedEps sets.String, https bool, args ...string) sets.String {
	ipPort := net.JoinHostPort(targetIP, strconv.Itoa(targetPort))
	// The current versions of curl included in CentOS and RHEL distros
	// misinterpret square brackets around IPv6 as globbing, so use the -g
	// argument to disable globbing to handle the IPv6 case.
	c := []string{
		"curl",
		"-k",
		"-g",
		"-q",
		"-s",
		"--max-time 15",
		"--connect-timeout 1",
	}

	c = append(c, args...)
	if https {
		c = append(c, fmt.Sprintf("https://%s/hostName", ipPort))
	} else {
		c = append(c, fmt.Sprintf("http://%s/hostName", ipPort))
	}
	cmd := strings.Join(c, " ")

	// TODO: This simply tells us that we can reach the endpoints. Check that
	// the probability of hitting a specific endpoint is roughly the same as
	// hitting any other.
	eps := sets.NewString()
	diff := sets.NewString()

	filterCmd := fmt.Sprintf("%s | grep -v '^\\s*$'", cmd)

	for i := 0; i < maxTries; i++ {
		stdout, stderr, err := n.Exec(hostTestContainerName, "/bin/sh", "-c", filterCmd)
		if err != nil || len(stderr) > 0 {
			// A failure to exec command counts as a try, not a hard fail.
			// Also note that we will keep failing for maxTries in tests where
			// we confirm unreachability.
			n.Log.Infof("Failed to execute %q: %v, stdout: %q, stderr: %q", filterCmd, err, stdout, stderr)
		} else {
			trimmed := strings.TrimSpace(stdout)
			n.Log.Debugf("Got response: %q", trimmed)
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

// newAgnhostPod returns a pod using the HostNetwork to be used to shoot the
// nodeport proxy service.
func newAgnhostPod(ns string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostTestPodName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            hostTestContainerName,
					Image:           AgnhostImage,
					Args:            []string{"pause"},
					SecurityContext: &corev1.SecurityContext{},
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
			HostNetwork:                   true,
			SecurityContext:               &corev1.PodSecurityContext{},
			TerminationGracePeriodSeconds: pointer.Int64Ptr(0),
		},
	}
	return pod
}
