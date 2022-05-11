//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package machine_test

import (
	"context"
	"fmt"
	"testing"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/ee/validation/machine"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
)

func TestResourceQuotaValidation(t *testing.T) {
	l := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()

	testCases := []struct {
		name        string
		machine     *clusterv1alpha1.Machine
		expectedErr bool
	}{
		{
			name:        "quota that fits should succeed",
			machine:     genFakeMachine("2", "2G", "10G"),
			expectedErr: false,
		},
		{
			name:        "should fail with CPU quota exceeded",
			machine:     genFakeMachine("50", "2G", "10G"),
			expectedErr: true,
		},
		{
			name:        "should fail with Memory quota exceeded",
			machine:     genFakeMachine("2", "50G", "10G"),
			expectedErr: true,
		},
		{
			name:        "should fail with Storage quota exceeded",
			machine:     genFakeMachine("2", "2G", "5000G"),
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := machine.ValidateQuota(context.Background(), l, nil, nil, tc.machine, nil)
			if err != nil {
				if !tc.expectedErr {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if err == nil && tc.expectedErr {
				t.Fatal("expected error, got none")
			}
		})
	}
}

func genFakeMachine(cpu, memory, storage string) *clusterv1alpha1.Machine {
	return test.GenTestMachine("fake",
		fmt.Sprintf(`{"cloudProvider":"fake", "cloudProviderSpec":{"cpu":"%s","memory":"%s","storage":"%s"}}`, cpu, memory, storage),
		nil, nil)
}
