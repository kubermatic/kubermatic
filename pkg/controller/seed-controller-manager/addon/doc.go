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
Package addon contains a controller that applies addons based on a Addon CRD. It needs
a folder per addon that contains all manifests, then adds a label to all objects and applies
the addon via `kubectl apply --purge -l $added-label`, which result in all objects that
do have the label but are not in the on-disk manifests being removed.
*/
package addon
