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

package defaulting

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

// DefaultCreateClusterSpec defaults the cluster spec when creating a new cluster
func DefaultCreateClusterSpec(
	spec *kubermaticv1.ClusterSpec,
	cloudProvider provider.CloudProvider) error {

	if err := cloudProvider.DefaultCloudSpec(&spec.Cloud); err != nil {
		return fmt.Errorf("failed to default cloud spec: %v", err)
	}
	if spec.ComponentsOverride.Etcd.ClusterSize == nil {
		n := int32(kubermaticv1.DefaultEtcdClusterSize)
		spec.ComponentsOverride.Etcd.ClusterSize = &n
	}
	return nil
}
