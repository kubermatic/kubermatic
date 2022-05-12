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
Package encryption contains a controller that is responsible for monitoring and
executing [encryption-at-rest](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/)
related tasks.

While updating the configuration and the kube-apiserver Pods happens in the
`kubernetes_controller`, the `encryption_controller` will update the Cluster
status according to changes observed in kube-apiserver and launch a re-encryption
job based on the observed phase of the encryption process.
*/

package encryptionatrestcontroller
