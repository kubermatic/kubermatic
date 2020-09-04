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

package common

import (
	"fmt"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func ConvertNodeMetrics(metrics []v1beta1.NodeMetrics, availableResources map[string]corev1.ResourceList) ([]apiv1.NodeMetric, error) {
	nodeMetrics := make([]apiv1.NodeMetric, 0)

	if metrics == nil {
		return nil, fmt.Errorf("metric list can not be nil")
	}

	for _, m := range metrics {
		nodeMetric := apiv1.NodeMetric{
			Name: m.Name,
		}

		resourceMetricsInfo := common.ResourceMetricsInfo{
			Name:      m.Name,
			Metrics:   m.Usage.DeepCopy(),
			Available: availableResources[m.Name],
		}

		if available, found := resourceMetricsInfo.Available[corev1.ResourceCPU]; found {
			quantityCPU := resourceMetricsInfo.Metrics[corev1.ResourceCPU]
			// cpu in mili cores
			nodeMetric.CPUTotalMillicores = quantityCPU.MilliValue()
			nodeMetric.CPUAvailableMillicores = available.MilliValue()
			fraction := float64(quantityCPU.MilliValue()) / float64(available.MilliValue()) * 100
			nodeMetric.CPUUsedPercentage = int64(fraction)
		}

		if available, found := resourceMetricsInfo.Available[corev1.ResourceMemory]; found {
			quantityM := resourceMetricsInfo.Metrics[corev1.ResourceMemory]
			// memory in bytes
			nodeMetric.MemoryTotalBytes = quantityM.Value() / (1024 * 1024)
			nodeMetric.MemoryAvailableBytes = available.Value() / (1024 * 1024)
			fraction := float64(quantityM.MilliValue()) / float64(available.MilliValue()) * 100
			nodeMetric.MemoryUsedPercentage = int64(fraction)
		}

		nodeMetrics = append(nodeMetrics, nodeMetric)
	}

	return nodeMetrics, nil
}
