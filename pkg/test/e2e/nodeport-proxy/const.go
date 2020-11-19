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

package nodeport_proxy

// TODO(irozzo) make registries configurable
const (
	AgnosImage   = "k8s.gcr.io/e2e-test-images/agnhost:2.21"
	NetexecImage = "gcr.io/kubernetes-e2e-test-images/netexec:1.1"
)
