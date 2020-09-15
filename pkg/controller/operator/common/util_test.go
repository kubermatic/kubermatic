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

package common

import (
	"testing"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestImagePullSecretModifierFactory(t *testing.T) {
	tests := []struct {
		name                string
		cfg                 *operatorv1alpha1.KubermaticConfiguration
		inputObj            runtime.Object
		wantErrStr          string
		wantImagePullSecret bool
	}{
		{
			name:                "Empty imagePullSecret",
			cfg:                 &operatorv1alpha1.KubermaticConfiguration{},
			inputObj:            &appsv1.Deployment{},
			wantImagePullSecret: false,
		},
		{
			name: "Non empty imagePullSecret",
			cfg: &operatorv1alpha1.KubermaticConfiguration{
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					ImagePullSecret: "{}",
				},
			},
			inputObj:            &appsv1.Deployment{},
			wantImagePullSecret: true,
		},
		{
			name: "Not a Deployment",
			cfg: &operatorv1alpha1.KubermaticConfiguration{
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					ImagePullSecret: "{}",
				},
			},
			wantErrStr:          `type "apps/v1, Kind=StatefulSet" is not supported by ImagePullSecretModifier`,
			inputObj:            &appsv1.StatefulSet{TypeMeta: metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"}},
			wantImagePullSecret: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ImagePullSecretModifierFactory(tt.cfg)
			create := got(identityCreator)
			_, err := create(tt.inputObj)
			if err != nil {
				if err.Error() != tt.wantErrStr {
					t.Fatalf("wanted error message %q, got %q", tt.wantErrStr, err)
				}
			} else {
				if tt.wantErrStr != "" {
					t.Error("wanted error but none was returned")
				}
				if d, ok := tt.inputObj.(*appsv1.Deployment); ok {
					var foundImagePullSecret bool
					for _, ips := range d.Spec.Template.Spec.ImagePullSecrets {
						if ips.Name == DockercfgSecretName {
							foundImagePullSecret = true
						}
					}
					if foundImagePullSecret != tt.wantImagePullSecret {
						t.Errorf("wantImagePullSecret = %v, but got %v", tt.wantImagePullSecret, d.Spec.Template.Spec.ImagePullSecrets)
					}
				} else {
					t.Fatalf("this is an unexpected condition by the test that today only supports Deployments, if support for other resource types has been added please update this test accordingly")
				}
			}
		})
	}
}

// identityCreator is an ObjectModifier that returns the input object
// untouched.
// TODO(irozzo) May be usefule to move this in a test package?
func identityCreator(obj runtime.Object) (runtime.Object, error) {
	return obj, nil
}
