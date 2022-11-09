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

package machine

import (
	"encoding/json"
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8s.io/apimachinery/pkg/api/resource"
)

func getFakeQuotaRequest(config *providerconfig.Config) (*ResourceDetails, error) {
	spec := &FakeProviderSpec{}
	if err := json.Unmarshal(config.CloudProviderSpec.Raw, spec); err != nil {
		return nil, fmt.Errorf("error unmarshalling fake raw config: %w", err)
	}
	cpu, err := resource.ParseQuantity(spec.Cpu)
	if err != nil {
		return nil, fmt.Errorf("error parsing quantity: %w", err)
	}

	mem, err := resource.ParseQuantity(spec.Memory)
	if err != nil {
		return nil, fmt.Errorf("error parsing quantity: %w", err)
	}

	storage, err := resource.ParseQuantity(spec.Storage)
	if err != nil {
		return nil, fmt.Errorf("error parsing quantity: %w", err)
	}

	return NewResourceDetails(cpu, mem, storage), nil
}

type FakeProviderSpec struct {
	Cpu     string `json:"cpu"`
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
}
