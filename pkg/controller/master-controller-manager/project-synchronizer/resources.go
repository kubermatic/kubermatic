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

package projectsynchronizer

import (
	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

func projectReconcilerFactory(project *kubermaticv1.Project) reconciling.NamedProjectReconcilerFactory {
	return func() (string, reconciling.ProjectReconciler) {
		return project.Name, func(p *kubermaticv1.Project) (*kubermaticv1.Project, error) {
			p.Spec = project.Spec
			return p, nil
		}
	}
}
