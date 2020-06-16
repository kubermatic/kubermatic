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
Package mastercontrollermanager contains all controllers that run within the master-controller-manager
binary.

Controllers that run in here:

  * Must need to access resources in the master cluster like userprojectbindings or usersshkeys
  * May need to access resources in seed clusters like clusters or secrets
  * Must consider that that master cluster may or may not be a seed as well
  * Must not access anything that is inside the usercluster like nodes or machines

*/
package mastercontrollermanager
