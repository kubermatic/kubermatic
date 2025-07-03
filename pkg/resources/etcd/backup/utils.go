/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package etcd

import (
	"fmt"
	"net/url"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
)

func GenSecretEnvVar(name, key string, destination *kubermaticv1.BackupDestination) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: destination.Credentials.Name},
				Key:                  key,
			},
		},
	}
}

func GetEtcdBackupSecretName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s-etcd-client-certificate", cluster.Name)
}

func isInsecureURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	// a hostname like "foo.com:9000" is parsed as {scheme: "foo.com", host: ""},
	// so we must make sure to not mis-interpret "http:9000" ({scheme: "http", host: ""}) as
	// an HTTP url

	return strings.ToLower(parsed.Scheme) == "http" && parsed.Host != ""
}
