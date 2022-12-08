/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package nodedeploymentmigration

import (
	"encoding/json"
	"errors"
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/shared/nodedeployment-migration/api"
	"k8c.io/kubermatic/v2/pkg/controller/shared/nodedeployment-migration/machine"
)

func ParseNodeOrMachineDeployment(cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.Datacenter, encoded string) (machineDeployment *clusterv1alpha1.MachineDeployment, migrated bool, err error) {
	if len(encoded) == 0 {
		return nil, false, nil
	}

	machineDeployment = &clusterv1alpha1.MachineDeployment{}
	if err := json.Unmarshal([]byte(encoded), machineDeployment); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal string as MachineDeployment: %w", err)
	}

	// looks like we found an already migrated machine deployment :)
	if machineDeployment.Name != "" {
		return machineDeployment, false, nil
	}

	nodeDeployment := &api.NodeDeployment{}
	if err := json.Unmarshal([]byte(encoded), nodeDeployment); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal string as NodeDeployment: %w", err)
	}

	if nodeDeployment.Name == "" {
		return nil, false, errors.New("string is neither a valid MachineDeployment nor a valid NodeDeployment")
	}

	machineDeployment, err = machine.Deployment(cluster, nodeDeployment, datacenter, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to convert NodeDeployment: %w", err)
	}

	return machineDeployment, true, nil
}
