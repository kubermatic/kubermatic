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

package v1

import "k8s.io/apimachinery/pkg/runtime"

// isEmptyRawExtension checks if a RawExtension is empty.
// In the context of Applications, we consider a RawExtension to be
// empty, if it is nil, zero characters, or "{}", which is the
// default empty value from kube-api.
func isEmptyRawExtension(re *runtime.RawExtension) bool {
	if re == nil {
		return true
	}
	if len(re.Raw) == 0 || string(re.Raw) == "{}" {
		return true
	}
	return false
}
