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
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	e2eutils "k8c.io/kubermatic/v3/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	hostTestPodName       = "host-test-container-pod"
	hostTestContainerName = "agnhost-container"
)

type networkingTestConfig struct {
	e2eutils.TestPodConfig
}

type DialConfig struct {
	TargetIP           string
	TargetPort         int
	HTTPS              bool
	ExtraCurlArguments []string
}

// Dial execs `curl` inside the agn test Pod, trying to reach the
// target of the DialConfig; the target is agnhost's /hostname
// endpoint, which will return the target's hostname (i.e. the
// hostname of the pod backing the target service).
// If successful, the endpoint name is returned, otherwise an empty string.
func (n *networkingTestConfig) Dial(ctx context.Context, dc DialConfig) (string, error) {
	ipPort := net.JoinHostPort(dc.TargetIP, strconv.Itoa(dc.TargetPort))

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

	c = append(c, dc.ExtraCurlArguments...)
	if dc.HTTPS {
		c = append(c, fmt.Sprintf("https://%s/hostName", ipPort))
	} else {
		c = append(c, fmt.Sprintf("http://%s/hostName", ipPort))
	}

	cmd := strings.Join(c, " ")
	filterCmd := fmt.Sprintf("%s | grep -v '^\\s*$'", cmd)

	stdout, stderr, err := n.Exec(ctx, hostTestContainerName, "/bin/sh", "-c", filterCmd)
	if err != nil || len(stderr) > 0 {
		return "", fmt.Errorf("failed to execute test command; stdout: %q, stderr: %q", stdout, stderr)
	}

	return strings.TrimSpace(stdout), nil
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
			TerminationGracePeriodSeconds: pointer.Int64(0),
		},
	}
	return pod
}
