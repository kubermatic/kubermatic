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

package etcdrunning

import (
	"fmt"
	"path/filepath"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
)

type etcdRunningData interface {
	RewriteImage(string) (string, error)
	Cluster() *kubermaticv1.Cluster
	EtcdLauncherImage() string
	EtcdLauncherTag() string
}

func Container(etcdEndpoints []string, data etcdRunningData) corev1.Container {
	return corev1.Container{
		Name:            "etcd-running",
		Image:           fmt.Sprintf("%s:%s", data.EtcdLauncherImage(), data.EtcdLauncherTag()),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/etcd-launcher",
			"is-running",
			"--etcd-ca-file=/etc/etcd/pki/client/ca.crt",
			fmt.Sprintf("--etcd-client-cert-file=%s", filepath.Join("/etc/etcd/pki/client", resources.ApiserverEtcdClientCertificateCertSecretKey)),
			fmt.Sprintf("--etcd-client-key-file=%s", filepath.Join("/etc/etcd/pki/client", resources.ApiserverEtcdClientCertificateKeySecretKey)),
			fmt.Sprintf("--cluster=%s", data.Cluster().Name),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      resources.ApiserverEtcdClientCertificateSecretName,
				MountPath: "/etc/etcd/pki/client",
				ReadOnly:  true,
			},
		},
	}
}
