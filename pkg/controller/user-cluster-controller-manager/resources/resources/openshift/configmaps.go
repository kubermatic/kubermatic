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

package openshift

import (
	"encoding/json"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func ControlplaneConfigCreator(platformName string) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return "cluster-config-v1", func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			installConfig := map[string]interface{}{}
			if val, exists := cm.Data["install-config"]; exists {
				if err := json.Unmarshal([]byte(val), &installConfig); err != nil {
					return nil, fmt.Errorf("failed to unmarshal install-config: %v", err)
				}
			}

			installConfig, err := getInstallConfig(installConfig, platformName)
			if err != nil {
				return nil, fmt.Errorf("failed to get install-config: %v", err)
			}

			bytes, err := json.Marshal(installConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal install-config: %v", err)
			}

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Data["install-config"] = string(bytes)
			return cm, nil
		}
	}
}

func getInstallConfig(existingData map[string]interface{}, platformName string) (map[string]interface{}, error) {
	if _, exists := existingData["apiVersion"]; !exists {
		existingData["apiVersion"] = "v1"
	}
	controlPlaneKeyValue, controlPlaneKeyExists := existingData["controlPlane"]
	if !controlPlaneKeyExists {
		controlPlaneKeyValue = map[string]interface{}{}
	}
	controlPlaneKeyValueAsserted, ok := controlPlaneKeyValue.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("the `controlPlane` key is not a map[string]interface{} but a %T", controlPlaneKeyValue)
	}

	platformKeyValue, platformKeyExists := controlPlaneKeyValueAsserted["platform"]
	if !platformKeyExists {
		platformKeyValue = map[string]interface{}{}
	}
	platformKeyValueAsserted, ok := platformKeyValue.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("the `controlplane.platform` key is not a map[string]interface{} but a %T`", platformKeyValue)
	}

	if _, exists := platformKeyValueAsserted[platformName]; !exists {
		platformKeyValueAsserted[platformName] = struct{}{}
	}

	controlPlaneKeyValueAsserted["platform"] = platformKeyValueAsserted
	existingData["controlPlane"] = controlPlaneKeyValueAsserted
	return existingData, nil
}
