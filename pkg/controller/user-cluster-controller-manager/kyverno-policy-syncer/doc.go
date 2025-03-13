/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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
Package kyvernopolicysyncer contains the primary controller responsible for managing Kyverno policies
in Kubermatic user clusters. The controller:

- Watches and reconciles PolicyBinding resources to determine which policies should be active
- Maintains ClusterPolicy resources in the user cluster, ensuring they match PolicyBinding specifications
- Recreates ClusterPolicy resources if manually deleted

The controller runs in the user cluster and directly manages the lifecycle of Kyverno policies,
providing a robust and reliable policy management system.
*/
package kyvernopolicysyncer
