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

package etcdbackup

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func caBundleConfigMapName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s-ca-bundle", cluster.Name)
}

func caBundleConfigMapCreator(name string, caBundle resources.CABundle) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return name, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			c.Data = map[string]string{
				resources.CABundleConfigMapKey: caBundle.String(),
			}

			return c, nil
		}
	}
}
