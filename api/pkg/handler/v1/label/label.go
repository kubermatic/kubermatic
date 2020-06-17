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

package label

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticcrdv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

const (
	ProjectResourceType        apiv1.ResourceType = "project"
	ClusterResourceType        apiv1.ResourceType = "cluster"
	NodeDeploymentResourceType apiv1.ResourceType = "nodedeployment"
)

// List of labels restricted by the Kubermatic, that should not be used by the users.
// Each resource type has its' own list.
var systemLabels apiv1.ResourceLabelMap = map[apiv1.ResourceType]apiv1.LabelKeyList{
	ProjectResourceType: {},
	ClusterResourceType: {
		kubermaticcrdv1.WorkerNameLabelKey,
		kubermaticcrdv1.ProjectIDLabelKey,
	},
	NodeDeploymentResourceType: {},
}

// ListSystemLabels defines an endpoint to get list of system labels.
func ListSystemLabels() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return &systemLabels, nil
	}
}

// GetSystemLabels returns restricted system labels object. We do not want anyone to modify original object outside of
// this package. That is why only getter is exposed.
func GetSystemLabels() apiv1.ResourceLabelMap {
	return systemLabels
}

// FilterLabels removes system labels from the provided labels map
func FilterLabels(resource apiv1.ResourceType, labels map[string]string) map[string]string {
	for _, label := range systemLabels[resource] {
		delete(labels, label)
	}

	return labels
}
