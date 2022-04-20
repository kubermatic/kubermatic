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
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/resource"
)

// ValidateQuota validates if the requested Machine resource consumption fits in the quota of the clusters project.
func ValidateQuota(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, machine *v1alpha1.Machine) error {
	config, err := types.GetConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("failed to read machine.spec.providerSpec: %v", err)
	}

	// TODO add all providers
	var quotaReq *ResourceQuota
	switch config.CloudProvider {
	// add this fake for test and so further code is reachable until more providers are implemented
	case types.CloudProviderFake:
		quotaReq, err = getFakeQuotaRequest(config)
		if err != nil {
			return fmt.Errorf("error getting fake quota req: %v", err)
		}
	default:
		// TODO skip for now, when all providers are added, throw error
		log.Debugf("provider %q not supported", config.CloudProvider)
		return nil
	}

	// TODO Get quota and usage from ResourceQuota when its implemented
	quota, currentUsage, err := getResourceQuota()
	if err != nil {
		return fmt.Errorf("failed to get resouce quota: %v", err)
	}

	// add requested resources to current usage and compare
	combinedUsage := NewResourceQuota(currentUsage.cpu, currentUsage.mem, currentUsage.storage)
	combinedUsage.Cpu().Add(*quotaReq.Cpu())
	combinedUsage.Memory().Add(*quotaReq.Memory())
	combinedUsage.Storage().Add(*quotaReq.Storage())

	if quota.Cpu().Cmp(*combinedUsage.Cpu()) < 0 {
		log.Debugf("requested CPU %q would exceed current quota (quota/used %q/%q)",
			quotaReq.Cpu(), quota.Cpu(), currentUsage.Cpu())
		return fmt.Errorf("requested CPU %q would exceed current quota (quota/used %q/%q)",
			quotaReq.Cpu(), quota.Cpu(), currentUsage.Cpu())
	}

	if quota.Memory().Cmp(*combinedUsage.Memory()) < 0 {
		log.Debugf("requested Memory %q would exceed current quota (quota/used %q/%q)",
			quotaReq.Memory(), quota.Memory(), currentUsage.Memory())
		return fmt.Errorf("requested Memory %q would exceed current quota (quota/used %q/%q)",
			quotaReq.Memory(), quota.Memory(), currentUsage.Memory())
	}

	if quota.Storage().Cmp(*combinedUsage.Storage()) < 0 {
		log.Debugf("requested Storage %q would exceed current quota (quota/used %q/%q)",
			quotaReq.Storage(), quota.Storage(), currentUsage.Storage())
		return fmt.Errorf("requested Storage %q would exceed current quota (quota/used %q/%q)",
			quotaReq.Storage(), quota.Storage(), currentUsage.Storage())
	}

	return nil
}

// TODO we should get it from the resourceQuota CRD for the project, for now just some hardcoded values for tests
func getResourceQuota() (*ResourceQuota, *ResourceQuota, error) {
	cpu, err := resource.ParseQuantity("5")
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing quantity: %v", err)
	}
	cpuUsed, err := resource.ParseQuantity("3")
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing quantity: %v", err)
	}

	mem, err := resource.ParseQuantity("5G")
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing quantity: %v", err)
	}
	memUsed, err := resource.ParseQuantity("3G")
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing quantity: %v", err)
	}

	storage, err := resource.ParseQuantity("100G")
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing quantity: %v", err)
	}
	storageUsed, err := resource.ParseQuantity("60G")
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing quantity: %v", err)
	}

	return NewResourceQuota(cpu, mem, storage),
		NewResourceQuota(cpuUsed, memUsed, storageUsed), nil
}

type ResourceQuota struct {
	cpu     resource.Quantity
	mem     resource.Quantity
	storage resource.Quantity
}

func NewResourceQuota(cpu resource.Quantity, mem resource.Quantity, storage resource.Quantity) *ResourceQuota {
	return &ResourceQuota{
		cpu:     cpu,
		mem:     mem,
		storage: storage,
	}
}

func (r *ResourceQuota) Cpu() *resource.Quantity {
	return &r.cpu
}

func (r *ResourceQuota) Memory() *resource.Quantity {
	return &r.mem
}

func (r *ResourceQuota) Storage() *resource.Quantity {
	return &r.storage
}
