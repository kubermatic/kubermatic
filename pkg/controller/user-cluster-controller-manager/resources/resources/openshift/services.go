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
	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/openshift/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func APIServicecreatorGetterFactory(clusterNS string) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return "api", func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Selector = nil
			s.Spec.Type = corev1.ServiceTypeExternalName
			s.Spec.Ports = nil
			s.Spec.ExternalName = openshiftresources.OpenshiftAPIServerServiceName + "." + clusterNS + ".svc.cluster.local"
			return s, nil
		}
	}
}
