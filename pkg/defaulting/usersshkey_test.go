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

package defaulting

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	"k8s.io/apimachinery/pkg/runtime"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = kubermaticv1.AddToScheme(testScheme)
}

func TestDefaultUserSSHKey(t *testing.T) {
	tests := []struct {
		name        string
		key         *kubermaticv1.UserSSHKey
		oldKey      *kubermaticv1.UserSSHKey
		expectedKey *kubermaticv1.UserSSHKey
		wantError   bool
	}{
		{
			name: "Create fully speced out UserSSHKey",
			key: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
				},
			},
			expectedKey: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
				},
			},
			wantError: false,
		},
		{
			name: "Add missing fingerprint",
			key: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:      "UserSSHKey",
					PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
				},
			},
			expectedKey: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
				},
			},
		},
		{
			name: "Fix wrong fingerprint",
			key: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "i am not a fingerprint",
				},
			},
			expectedKey: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
				},
			},
		},
		{
			name: "Implicitly lowercase the fingerprint",
			key: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1C:07:99:4F:C8:4B:08:48:2A:95:51:14:AC:5C:AA:11",
				},
			},
			expectedKey: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
				},
			},
		},
		{
			name: "Update fingerprint when key changes",
			oldKey: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1C:07:99:4F:C8:4B:08:48:2A:95:51:14:AC:5C:AA:11",
				},
			},
			key: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCzaYpq344ryKwyl1Mqvo8NMQ+HWQyzEpMklwmgRJM9H2YSBJYax0AldaDEDT4JUGXixQt5oJ+7RnpvoGK3m/9OFiaWWZHk+vOBTDPn5e69kjjnNqBr9r42wJMZaQ5s3/R7rKeCzlhXJkjY5fpyfxETIfG1Oj/ShUrWbECQGB95/4KpHt91yIKeLp7omGawkfF5Nc3oZia3XTKTDiK3FcVWCqj6IPXkSUdH5XeX3uz7D4lYVlv2kz2sKOrdppeHtmqGWL2gfEy18GAzqjghVNJVrfWkHLM54XriEVz9/KBRsZjyo/bvbwCRBXmn5rfxijA7K5iXMoDJxT1qOLf5TaAzzQRX6tDF7J1OCJHWyWerc7DL7O1gtF9+CtayzzPeEk1+4R4E3l1vK6589auEfBjrVwLL2vUd0egKQbPvw6ey9X+cRL6gi25z9YnmndhJlsvvuOYnH9DtFkUIJ8/IzsY/BDtYYhPfzXpXqe264ubimtcVWyDSVF0iaVaUP2LtY5U= test@example.com",
					Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
				},
			},
			expectedKey: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCzaYpq344ryKwyl1Mqvo8NMQ+HWQyzEpMklwmgRJM9H2YSBJYax0AldaDEDT4JUGXixQt5oJ+7RnpvoGK3m/9OFiaWWZHk+vOBTDPn5e69kjjnNqBr9r42wJMZaQ5s3/R7rKeCzlhXJkjY5fpyfxETIfG1Oj/ShUrWbECQGB95/4KpHt91yIKeLp7omGawkfF5Nc3oZia3XTKTDiK3FcVWCqj6IPXkSUdH5XeX3uz7D4lYVlv2kz2sKOrdppeHtmqGWL2gfEy18GAzqjghVNJVrfWkHLM54XriEVz9/KBRsZjyo/bvbwCRBXmn5rfxijA7K5iXMoDJxT1qOLf5TaAzzQRX6tDF7J1OCJHWyWerc7DL7O1gtF9+CtayzzPeEk1+4R4E3l1vK6589auEfBjrVwLL2vUd0egKQbPvw6ey9X+cRL6gi25z9YnmndhJlsvvuOYnH9DtFkUIJ8/IzsY/BDtYYhPfzXpXqe264ubimtcVWyDSVF0iaVaUP2LtY5U= test@example.com",
					Fingerprint: "21:58:87:54:0d:e3:db:1e:7f:1d:c5:b6:19:cb:3b:d9",
				},
			},
		},
		{
			name: "Reject breaking the fingerprint",
			oldKey: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
				},
			},
			key: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX:XX",
				},
			},
			expectedKey: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:        "UserSSHKey",
					PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoimncCXKFYqIvfwZcOpv416VGLPPZsbpx1pvycYVCMomBDz0t8WOk8jziGAi+NWfTI23OuPGgCf9AGKWqW2zoPLuu2lr0JM4RLtWIbG2SL2j8+eTtmGulbtThWjlletega5Wr87zOU5g9YTzmX1TVQijP2S+6tgJiUZG1AMUvv0R1ub+HA6eMykBZ0/eTydlIrNSUqfzc9Bg/FYf3qXoI8BRSNfNf2FNjjGOW3UnoGSuxQt8sjSRLYrI9twl3LgkHVpI5Q+WBbye9OZJUeK2QqL2C5z9pjXUtKMDe6Bbzl5uGwDw+3MgR3D6SDHRXpCupr9pNETarwEXyF1xaV3cyKm2D+UQCQL63vIpgiV40Eka15F+VjaSAqNqYA9oWUs4y2Bbajnl6ywtZFRobQE8G4k4rFa4XHUIRj6O8F3XzOVsdVehOM9W/+hcJCwryRqCkWqYuHHwmMMdF825M+jo/trFpQLaobR1eA9MFq4rU4Dx+aCbfxZeV86D36LABY8k= test@example.com",
					Fingerprint: "1c:07:99:4f:c8:4b:08:48:2a:95:51:14:ac:5c:aa:11",
				},
			},
		},
		{
			name: "Error out on invalid keys",
			key: &kubermaticv1.UserSSHKey{
				Spec: kubermaticv1.SSHKeySpec{
					Name:      "UserSSHKey",
					PublicKey: "i am not a public key",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutatedKey, err := DefaultUserSSHKey(tt.key, tt.oldKey)
			if tt.wantError && err == nil {
				t.Fatal("Expected error, but got no error.")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}
			if err != nil {
				return
			}

			if !diff.SemanticallyEqual(tt.expectedKey, mutatedKey) {
				t.Fatalf("Diff found between expected and actual key:\n%v", diff.ObjectDiff(tt.expectedKey, mutatedKey))
			}
		})
	}
}
