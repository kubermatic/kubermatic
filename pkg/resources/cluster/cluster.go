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

package cluster

import (
	"crypto/x509"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
)

func CloudProviderForCluster(spec *kubermaticv1.ClusterSpec, dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (provider.CloudProvider, error) {
	providerName, err := kubermaticv1helper.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("invalid cloud spec: %w", err)
	}
	if providerName == "" {
		return nil, errors.New("cluster has no cloud provider")
	}

	return cloud.Provider(dc, secretKeyGetter, caBundle)
}
