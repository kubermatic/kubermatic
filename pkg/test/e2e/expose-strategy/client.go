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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/kubectl"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	openvpnImage        = "quay.io/kubermatic/openvpn:v2.5.2-r0"
	clientPodName       = "client-pod"
	clientContainerName = "openvpn"
	kubeconfigPath      = "/etc/kubernetes"
)

// KubeVersions is used to unmarshal the output of 'kubectl version -ojson'
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

func (cj *clientJig) QueryApiserverVersion(kasHostPort string, insecure bool, expectServerVersion semver.Semver, retries, minSuccess int) bool {
	binary, err := kubectl.BinaryForClusterVersion(expectServerVersion.Version)
	if err != nil {
		cj.Log.Errorf("Failed to determine kubectl binary to use: %v", err)
		return false
	}

	c := []string{
		binary,
		"version",
		"-o",
		"json",
	}
	// If kasHostPort is provided we patch the kubeconfig
	if kasHostPort != "" {
		c = append([]string{
			"sed",
			fmt.Sprintf(`'s/\sserver: .*$/ server: https:\/\/%s/'`, kasHostPort),
			"${KUBECONFIG}",
			"|",
		}, c...)
		c = append(c, "--kubeconfig", "/dev/stdin")
	}
	if insecure {
		c = append(c, "--insecure-skip-tls-verify=true")
	}
	cmd := strings.Join(c, " ")

	filterCmd := fmt.Sprintf("%s | grep -v '^\\s*$'", cmd)

	s := 0
	for i := 0; i < retries; i++ {
		stdout, stderr, err := cj.Exec(clientContainerName, "/bin/sh", "-c", filterCmd)
		if err != nil || len(stderr) > 0 {
			cj.Log.Infof("Failed to execute %q: %v, stdout: %q, stderr: %q", filterCmd, err, stdout, stderr)
		} else {
			rawVersion := strings.TrimSpace(stdout)
			cj.Log.Debugf("Got response: %q", rawVersion)
			v := KubeVersions{}
			if err := json.Unmarshal([]byte(rawVersion), &v); err != nil {
				cj.Log.Errorf("Failed to unmarshal output of kubeclt version command: %v", err)
				continue
			}
			if v.ServerVersion.MajorVersion() == expectServerVersion.Major() &&
				v.ServerVersion.MinorVersion() == expectServerVersion.Minor() {
				s++
			}
		}

	}

	return s >= minSuccess
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
			TerminationGracePeriodSeconds: pointer.Int64Ptr(0),
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
