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
Package clustercredentialscontroller contains a controller that watches Cluster
objects and is responsible for moving inline credentials (from the CloudSpec)
into dedicated Kubernetes Secrets.

In a perfect future, we would not even ever create a Cluster with inline credentials,
but for historical reasons it's the safest method to handle credentials for now.
It is also super convenient that users do not have to manually create a Secret
somewhere themselves.
*/
package clustercredentialscontroller
