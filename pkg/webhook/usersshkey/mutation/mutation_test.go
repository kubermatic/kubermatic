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

package mutation

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-test/deep"
	jsonpatch "gomodules.xyz/jsonpatch/v2"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = kubermaticv1.AddToScheme(testScheme)
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name        string
		req         webhook.AdmissionRequest
		wantError   bool
		wantPatches []jsonpatch.JsonPatchOperation
	}{
		{
			name: "Create fully speced out UserSSHKey",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "UserSSHKey",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:        "UserSSHKey",
							PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
							Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
						}.Do(),
					},
				},
			},
			wantError: false,
		},
		{
			name: "Add missing fingerprint",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "UserSSHKey",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:        "UserSSHKey",
							PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
							Fingerprint: "",
						}.Do(),
					},
				},
			},
			wantError: false,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("replace", "/spec/fingerprint", "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11"),
			},
		},
		{
			name: "Fix wrong fingerprint",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "UserSSHKey",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:        "UserSSHKey",
							PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
							Fingerprint: "this is not a valid fingerprint",
						}.Do(),
					},
				},
			},
			wantError: false,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("replace", "/spec/fingerprint", "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11"),
			},
		},
		{
			name: "Implicitly lowercase the fingerprint",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "UserSSHKey",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:        "UserSSHKey",
							PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
							Fingerprint: "1C:07:99:4F:C8:4B:08:48:2A:95:51:14:AC:5C:AA:11",
						}.Do(),
					},
				},
			},
			wantError: false,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("replace", "/spec/fingerprint", "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11"),
			},
		},
		{
			name: "Update fingerprint when key changes",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "UserSSHKey",
					},
					Name: "foo",
					OldObject: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:        "UserSSHKey",
							PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
							Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
						}.Do(),
					},
					Object: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:        "UserSSHKey",
							PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCzaYpq344ryKwyl1Mqvo8NMQ+HWQyzEpMklwmgRJM9H2YSBJYax0AldaDEDT4JUGXixQt5oJ+7RnpvoGK3m/9OFiaWWZHk+vOBTDPn5e69kjjnNqBr9r42wJMZaQ5s3/R7rKeCzlhXJkjY5fpyfxETIfG1Oj/ShUrWbECQGB95/4KpHt91yIKeLp7omGawkfF5Nc3oZia3XTKTDiK3FcVWCqj6IPXkSUdH5XeX3uz7D4lYVlv2kz2sKOrdppeHtmqGWL2gfEy18GAzqjghVNJVrfWkHLM54XriEVz9/KBRsZjyo/bvbwCRBXmn5rfxijA7K5iXMoDJxT1qOLf5TaAzzQRX6tDF7J1OCJHWyWerc7DL7O1gtF9+CtayzzPeEk1+4R4E3l1vK6589auEfBjrVwLL2vUd0egKQbPvw6ey9X+cRL6gi25z9YnmndhJlsvvuOYnH9DtFkUIJ8/IzsY/BDtYYhPfzXpXqe264ubimtcVWyDSVF0iaVaUP2LtY5U= test@example.com",
							Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11", // this field should be updated by the webhook
						}.Do(),
					},
				},
			},
			wantError: false,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("replace", "/spec/fingerprint", "21:58:87:54:0d:e3:db:1e:7f:1d:c5:b6:19:cb:3b:d9"),
			},
		},
		{
			name: "Reject breaking the fingerprint",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "UserSSHKey",
					},
					Name: "foo",
					OldObject: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:        "UserSSHKey",
							PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
							Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
						}.Do(),
					},
					Object: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:        "UserSSHKey",
							PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
							Fingerprint: "XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX",
						}.Do(),
					},
				},
			},
			wantError: false,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("replace", "/spec/fingerprint", "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11"),
			},
		},
		{
			name: "Error out on invalid keys",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "UserSSHKey",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawKeyGen{
							Name:      "UserSSHKey",
							PublicKey: "i am not a valid SSH public key",
						}.Do(),
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		d, err := admission.NewDecoder(testScheme)
		if err != nil {
			t.Fatalf("error occurred while creating decoder: %v", err)
		}

		t.Run(tt.name, func(t *testing.T) {
			handler := AdmissionHandler{
				log:     logr.Discard(),
				decoder: d,
			}
			res := handler.Handle(context.Background(), tt.req)
			if res.AdmissionResponse.Result != nil && res.AdmissionResponse.Result.Code == http.StatusInternalServerError {
				if tt.wantError {
					return
				}

				t.Fatalf("Request failed: %v", res.AdmissionResponse.Result.Message)
			}

			a := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range res.Patches {
				a[p.Path] = p
			}
			w := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range tt.wantPatches {
				w[p.Path] = p
			}
			if diff := deep.Equal(a, w); len(diff) > 0 {
				t.Errorf("Diff found between wanted and actual patches: %+v", diff)
			}
		})
	}
}

type rawKeyGen struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
}

func (r rawKeyGen) Do() []byte {
	key := kubermaticv1.UserSSHKey{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "UserSSHKey",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Name,
		},
		Spec: kubermaticv1.SSHKeySpec{
			Owner:       r.Owner,
			Name:        r.Name,
			Fingerprint: r.Fingerprint,
			PublicKey:   r.PublicKey,
		},
	}

	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, testScheme, testScheme, json.SerializerOptions{Pretty: true})
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&key, buff)

	return buff.Bytes()
}
