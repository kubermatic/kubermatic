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

package reconciling

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-test/deep"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func init() {
	// Kubernetes Objects can be deeper than the default 10 levels.
	deep.MaxDepth = 20
	deep.LogErrors = true
}

// DeepEqual compares both objects for equality
func DeepEqual(a, b metav1.Object) bool {
	// The ClusterTemplate consist with ClusterSpec which has nested Semver type.
	// The semverlib.Version doesn't implement Equality function. Using equality.Semantic.DeepEqual for this object cause
	// an informative panic.
	if _, ok := a.(*kubermaticv1.ClusterTemplate); ok {
		if reflect.DeepEqual(a, b) {
			return true
		}
	} else {
		if equality.Semantic.DeepEqual(a, b) {
			return true
		}
	}

	// For some reason unstructured objects returned from the api have types for their fields
	// that are not map[string]interface{} and don't even exist in our codebase like
	// `openshift.infrastructureStatus`, so we have to compare the wire format here.
	// We only do this for unstrucutred as this comparison is pretty expensive.
	if _, isUnstructured := a.(*unstructured.Unstructured); isUnstructured && jsonEqual(a, b) {
		return true
	}

	// For informational purpose we use deep.equal as it tells us what the difference is.
	// We need to calculate the difference in both ways as deep.equal only does a one-way comparison
	diff := deep.Equal(a, b)
	if diff == nil {
		diff = deep.Equal(b, a)
	}

	kubermaticlog.Logger.Debugw("Object differs from generated one", "type", fmt.Sprintf("%T", a), "namespace", a.GetNamespace(), "name", a.GetName(), "diff", diff)
	return false
}

func jsonEqual(a, b interface{}) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		kubermaticlog.Logger.Errorw("Failed to marshal aJSON", zap.Error(err))
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		kubermaticlog.Logger.Errorw("Failed to marshal bJSON", zap.Error(err))
		return false
	}
	return string(aJSON) == string(bJSON)
}
