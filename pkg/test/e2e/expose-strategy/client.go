/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package exposestrategy

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/resources"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	openvpnImage        = "quay.io/kubermatic/openvpn:v2.5.2-r0"
	clientPodName       = "client-pod"
	clientContainerName = "openvpn"
	kubeconfigPath      = "/etc/kubernetes"
)

// KubeVersions is used to unmarshal the output of 'kubectl version -ojson'.
type KubeVersions struct {
	ServerVersion KubeVersion `json:"serverVersion"`
	ClientVersion KubeVersion `json:"clientVersion"`
}

type KubeVersion struct {
	Major      string                 `json:"major"`
	Minor      string                 `json:"minor"`
	GitVersion string                 `json:"gitVersion"`
	Other      map[string]interface{} `json:"-"`
}

func (k KubeVersion) MajorVersion() uint64 {
	if v, err := strconv.ParseInt(k.Major, 10, 64); err == nil {
		return uint64(v)
	}
	return 0
}

func (k KubeVersion) MinorVersion() uint64 {
	if v, err := strconv.ParseInt(k.Minor, 10, 64); err == nil {
		return uint64(v)
	}
	return 0
}

type clientJig struct {
	e2eutils.TestPodConfig
}

func (cj *clientJig) VerifyApiserverVersion(ctx context.Context, kasHostPort string, insecure bool, expectServerVersion semver.Semver) error {
	cmd := []string{"kubectl", "version", "--output", "json"}

	// If kasHostPort is provided we override the server address
	if kasHostPort != "" {
		cmd = append(cmd, "--server", fmt.Sprintf("https://%s", kasHostPort))
	}
	if insecure {
		cmd = append(cmd, "--insecure-skip-tls-verify=true")
	}

	return wait.PollImmediateLog(ctx, cj.Log, 1*time.Millisecond, 15*time.Second, func(ctx context.Context) (transient error, terminal error) {
		stdout, stderr, err := cj.Exec(ctx, clientContainerName, cmd...)
		if err != nil {
			return fmt.Errorf("failed to execute kubectl (stdout=%q, stderr=%q): %w", stdout, stderr, err), nil
		}

		rawVersion := strings.TrimSpace(stdout)
		cj.Log.Debugw("Got response", "body", rawVersion)

		v := KubeVersions{}
		if err := json.Unmarshal([]byte(rawVersion), &v); err != nil {
			return fmt.Errorf("failed to unmarshal output of kubectl version command: %w", err), nil
		}

		currentMajorMinor := fmt.Sprintf("%s.%s", v.ServerVersion.Major, v.ServerVersion.Minor)
		expectedMajorMinor := expectServerVersion.MajorMinor()

		if currentMajorMinor != expectedMajorMinor {
			return fmt.Errorf("expected serverVersion to be %q, but got %q", expectedMajorMinor, currentMajorMinor), nil
		}

		return nil, nil
	})
}

// newClientPod returns a pod that runs a container allowing to run kubectl and
// OpenVPN commands.
func newClientPod(ns string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientPodName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    clientContainerName,
					Image:   openvpnImage,
					Command: []string{"/bin/bash"},
					Args:    []string{"-c", "while true; do sleep 2073600; done"},
					Env: []corev1.EnvVar{
						{
							Name:  "KUBECONFIG",
							Value: fmt.Sprintf("%s/%s", kubeconfigPath, resources.KubeconfigSecretKey),
						},
					},
					SecurityContext: &corev1.SecurityContext{},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.AdminKubeconfigSecretName,
							MountPath: kubeconfigPath,
							ReadOnly:  true,
						},
					},
				},
			},
			SecurityContext:               &corev1.PodSecurityContext{},
			TerminationGracePeriodSeconds: ptr.To[int64](0),
			Volumes: []corev1.Volume{
				{
					Name: resources.AdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.AdminKubeconfigSecretName,
						},
					},
				},
			},
		},
	}

	return pod
}
