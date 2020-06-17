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
Package seedcontrollermanager contains a package for each controller that runs within the seed
controller manager binary.

Controllers running in here:

  * Must not access master resources like userprojectbindings or usersshkeys
  * Must need to access seed resources like the cluster object or the controlplane deployments
  * Must not need to access resources within the usercluster like nodes or machines except for cleanup

*/
package seedcontrollermanager
