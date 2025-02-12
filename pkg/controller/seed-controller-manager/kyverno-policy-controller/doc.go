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
Package kyvernopolicycontroller contains a controller that is responsible for managing Kyverno policies
across user clusters. It watches PolicyBinding resources in the seed cluster and reconciles them by:

- Managing the lifecycle of PolicyBinding resources
- Managing ClusterPolicy resources in target user clusters based on the binding's selector

The controller supports both global and project-scoped policy bindings, and handles various target
selectors including cluster names and label selectors.
*/
package kyvernopolicycontroller
