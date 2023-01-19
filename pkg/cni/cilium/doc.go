/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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
Package cilium contains Cilium CNI related helpers for managing CNI using Applications infra.
Cilium is managed via Applications infra starting from version 1.13.0. For details see
pkg/controller/seed-controller-manager/cni-application-installation-controller.

When introducing a new CNI version, make sure it is:
  - introduced in pkg/cni/version.go with the version string exactly matching the ApplicationDefinition's Spec.Versions.Version
  - Helm chart is mirrored in Kubermatic OCI registry, use the script cilium-mirror-chart.sh
*/
package cilium
