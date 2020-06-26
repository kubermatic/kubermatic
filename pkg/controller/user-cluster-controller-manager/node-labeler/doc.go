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

/*
Package nodelabeler contains a controller that ensures Nodes have various labels present at all times:

	* A `x-kubernetes.io/distribution` label with a value of `centos`, `ubuntu`, `container-linux`, `rhel` or `sles`
	* A set of labels configured on the controller via a flag that are inherited from the cluster object
*/
package nodelabeler
