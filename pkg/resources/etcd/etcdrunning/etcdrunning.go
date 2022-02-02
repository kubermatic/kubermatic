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
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"

	corev1 "k8s.io/api/core/v1"
)

type etcdRunningData interface {
	ImageRegistry(defaultRegistry string) string
	Cluster() *kubermaticv1.Cluster
}

func Container(etcdEndpoints []string, data etcdRunningData) corev1.Container {
	return corev1.Container{
		Name:  "etcd-running",
		Image: data.ImageRegistry(resources.RegistryGCR) + "/etcd-development/etcd:" + etcd.ImageTag(data.Cluster()),
		Command: []string{
			"/bin/sh",
			"-ec",
			// Write a key to etcd. If we have quorum it will succeed.
			fmt.Sprintf("until ETCDCTL_API=3 /usr/local/bin/etcdctl --cacert=/etc/etcd/pki/client/ca.crt --cert=/etc/etcd/pki/client/apiserver-etcd-client.crt --key=/etc/etcd/pki/client/apiserver-etcd-client.key --dial-timeout=2s --endpoints='%s' put kubermatic/quorum-check something; do echo waiting for etcd; sleep 2; done;", strings.Join(etcdEndpoints, ",")),
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
