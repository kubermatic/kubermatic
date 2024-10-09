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

package modifier

import (
	"errors"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/reconciler/pkg/reconciling"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// ManagedByLabel is the label used to identify the resources
	// created by this controller.
	ManagedByLabel = "app.kubernetes.io/managed-by"

	// helmReleaseAnnotation is the indicator for the ownership modifier to
	// not touch the object.
	helmReleaseAnnotation = "meta.helm.sh/release-name"
)

// Ownership is generating a new ObjectModifier that wraps an ObjectReconciler
// and takes care of applying the ownership and other labels for all managed objects.
func Ownership(owner metav1.Object, managedBy string, scheme *runtime.Scheme) reconciling.ObjectModifier {
	return func(reconciler reconciling.ObjectReconciler) reconciling.ObjectReconciler {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := reconciler(existing)
			if err != nil {
				return obj, err
			}

			o, ok := obj.(metav1.Object)
			if !ok {
				return obj, nil
			}

			// Sometimes, the KKP operator needs to deal with objects that are owned by Helm
			// and then re-appropriated by KKP. This will however interfere with Helm's own
			// ownership concept. Also, reconciling resources owned by Helm will just lead to
			// increased resourceVersions, which might then trigger Deployments to be reconciled
			// due to the VolumeVersion annotations.
			// To prevent this, if an object is already owned by Helm, we never touch it.
			if _, exists := o.GetAnnotations()[helmReleaseAnnotation]; exists {
				return obj, nil
			}

			// try to set an owner reference; on shared resources this would fail to set
			// the second owner ref, we ignore this error and rely on the existing
			// KubermaticConfiguration ownership
			err = controllerutil.SetControllerReference(owner, o, scheme)
			if err != nil {
				var cerr *controllerutil.AlreadyOwnedError // do not use errors.Is() on this error type
				if !errors.As(err, &cerr) {
					return obj, fmt.Errorf("failed to set owner reference: %w", err)
				}
			}

			if managedBy != "" {
				kubernetes.EnsureLabels(o, map[string]string{
					ManagedByLabel: managedBy,
				})
			}

			return obj, nil
		}
	}
}
