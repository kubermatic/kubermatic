/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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
Package defaultapplicationcontroller contains a controller that is responsible for installing default/enforced
applications for a cluster. It watches cluster resource and creates ApplicationInstallation resources for the default/installed
applications. Some features of the controller are:
1. Enforced applications are always installed and cannot be overridden by the user.
2. Default applications are only installed for fresh clusters and can be overridden by the user. If a `Cluster` has initial-applications annotation and the
default application doesn't exist in the values, the controller will not install the default application. It assumes that it was intentionally disabled during
cluster creation.
3. If an enforced ApplicationDefinition is updated, the controller will update all the existing ApplicationInstallation resources. For this functionality, the
controller watches for ApplicationDefinition resources as well and reconciles the affected Clusters against them.
4. TODO: `force-delete` and `force-install` annotations that can be used to delete and install applications if they were defaulted/enforced by KKP.
*/

package defaultapplicationcontroller
