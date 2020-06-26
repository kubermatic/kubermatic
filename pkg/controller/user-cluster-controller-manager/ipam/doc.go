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
Package ipam contains a controller responsible for assigning IP addresses from a configured
pool to machines that have an annotation keyed `machine-controller.kubermatic.io/initializers`
which contains the value ipam. After that is done, the `ipam` value is removed.

This is used for environments where no DHCP is available. The aforementioned annotation will keep
the machine-controller from reconciling the machine.
*/
package ipam
