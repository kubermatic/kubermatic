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

package azure

import (
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

func ignoreNotFound(err error) error {
	if isNotFound(err) {
		return nil
	}

	return err
}

func isNotFound(err error) bool {
	var aerr *azcore.ResponseError
	if err != nil && errors.As(err, &aerr) {
		return aerr.StatusCode == http.StatusNotFound
	}

	return false
}

func getResourceGroup(cloud kubermaticv1.CloudSpec) string {
	if cloud.Azure.VNetResourceGroup != "" {
		return cloud.Azure.VNetResourceGroup
	}

	return cloud.Azure.ResourceGroup
}

func hasOwnershipTag(tags map[string]*string, cluster *kubermaticv1.Cluster) bool {
	if value, ok := tags[clusterTagKey]; ok {
		return *value == cluster.Name
	}

	return false
}
