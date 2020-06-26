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
The usersshkeyssynchronizer controller is responsible for synchronizing usersshkeys into
a secret in the cluster namespace. From there, the usercluster controller synchronizes them
into the usercluster and then a DaemonSet that runs on all nodes synchronizes them onto the
.ssh/authorized_keys file.
*/
package usersshkeyssynchronizer
