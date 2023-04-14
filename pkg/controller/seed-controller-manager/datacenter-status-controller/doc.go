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

/*
Package datacenterstatuscontroller contains a controller that is responsible
for managing the basic parts of the SeedStatus:

  - status.versions.kubermatic
  - status.versions.cluster
  - status.conditions.SeedConditionKubeconfigValid
  - status.phase
  - status.clusters

It does so by checking the kubeconfig for a seed and combining the other
conditions set by other controllers to compute the current phase.
*/
package datacenterstatuscontroller
