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

package certificates

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GlobalCABundle(ctx context.Context, client ctrlruntimeclient.Client, config *operatorv1alpha1.KubermaticConfiguration) (*corev1.ConfigMap, error) {
	caBundle := &corev1.ConfigMap{}
	key := types.NamespacedName{Name: config.Spec.CABundle.Name, Namespace: config.Namespace}

	if err := client.Get(ctx, key, caBundle); err != nil {
		return nil, fmt.Errorf("failed to fetch CA bundle: %v", err)
	}

	if err := ValidateCABundleConfigMap(caBundle); err != nil {
		return nil, fmt.Errorf("CA bundle is invalid: %v", err)
	}

	return caBundle, nil
}

func ValidateCABundleConfigMap(cm *corev1.ConfigMap) error {
	bundle, ok := cm.Data[resources.CABundleConfigMapKey]
	if !ok {
		return fmt.Errorf("ConfigMap does not contain key %q", resources.CABundleConfigMapKey)
	}

	return ValidateCABundle(bundle)
}

func ValidateCABundle(bundle string) error {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(bundle)) {
		return errors.New("bundle does not contain any valid certificates")
	}

	return nil
}
