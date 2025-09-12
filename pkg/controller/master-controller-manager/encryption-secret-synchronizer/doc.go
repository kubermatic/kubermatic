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
Package encryptionsecretsynchonizer contains a controller that is responsible for synchronizing
encryption-at-rest secrets from the kubermatic namespace to user cluster namespaces on seed clusters.

This controller monitors secrets with the pattern "encryption-key-cluster-*" that have the
"kubermatic.io/cluster-name" annotation and ensures they are available in the corresponding
cluster namespace where the encryption-at-rest-controller can use them.
*/
package encryptionsecretsynchonizer
