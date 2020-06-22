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
Package usersshkeysagent contains the usersshkeysagent controller, which is deployed as a DaemonSet
on all usercluster nodes and responsible for synchronizing the `$HOME/.ssh/authorized_keys` file
for all users we know about (root, core, ubuntu, centos) and that exist with the content of a
secret.

This secret in turn is synchronized based on a secret in the seed namespace via a controller running
in the usercluster controller manager and that seed namespace secret is synchronized based on the
usersshkeys custom resources in the master cluster via a controller running in the master controller
manager.
*/
package usersshkeysagent
