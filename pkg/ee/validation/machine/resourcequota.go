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
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/api/resource"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateQuota validates if the requested Machine resource consumption fits in the quota of the clusters project.
func ValidateQuota(ctx context.Context,
	log *zap.SugaredLogger,
	userClient ctrlruntimeclient.Client,
	machine *clusterv1alpha1.Machine,
	caBundle *certificates.CABundle,
	resourceQuota *kubermaticv1.ResourceQuota,
) error {
	machineResourceUsage, err := GetMachineResourceUsage(ctx, userClient, machine, caBundle)
	if err != nil {
		return fmt.Errorf("error getting machine resource request: %w", err)
	}

	var currentCPU = resource.Quantity{}
	if resourceQuota.Status.GlobalUsage.CPU != nil {
		currentCPU = *resourceQuota.Status.GlobalUsage.CPU
	}

	var currentMem = resource.Quantity{}
	if resourceQuota.Status.GlobalUsage.Memory != nil {
		currentMem = *resourceQuota.Status.GlobalUsage.Memory
	}

	var currentStorage = resource.Quantity{}
	if resourceQuota.Status.GlobalUsage.Storage != nil {
		currentStorage = *resourceQuota.Status.GlobalUsage.Storage
	}

	// add requested resources to current usage and compare
	combinedUsage := NewResourceDetails(currentCPU, currentMem, currentStorage)
	combinedUsage.CPU().Add(*machineResourceUsage.CPU())
	combinedUsage.Memory().Add(*machineResourceUsage.Memory())
	combinedUsage.Storage().Add(*machineResourceUsage.Storage())

	quota := resourceQuota.Spec.Quota
	if quota.CPU != nil && quota.CPU.Cmp(*combinedUsage.CPU()) < 0 {
		log.Debugw("requested CPU would exceed current quota", "request",
			machineResourceUsage.CPU(), "quota", quota.CPU, "used", currentCPU.String())
		return fmt.Errorf("requested CPU %q would exceed current quota (quota/used %q/%q)",
			machineResourceUsage.CPU(), quota.CPU, currentCPU.String())
	}

	if quota.Memory != nil && quota.Memory.Cmp(*combinedUsage.Memory()) < 0 {
		log.Debugw("requested Memory would exceed current quota", "request",
			machineResourceUsage.Memory(), "quota", quota.Memory, "used", currentMem.String())
		return fmt.Errorf("requested Memory %q would exceed current quota (quota/used %q/%q)",
			machineResourceUsage.Memory(), quota.Memory, currentMem.String())
	}

	if quota.Storage != nil && quota.Storage.Cmp(*combinedUsage.Storage()) < 0 {
		log.Debugw("requested disk size would exceed current quota", "request",
			machineResourceUsage.Storage(), "quota", quota.Storage, "used", currentStorage.String())
		return fmt.Errorf("requested disk size %q would exceed current quota (quota/used %q/%q)",
			machineResourceUsage.Storage(), quota.Storage, currentStorage.String())
	}

	return nil
}

type ResourceDetails struct {
	cpu     resource.Quantity
	mem     resource.Quantity
	storage resource.Quantity
}

func NewResourceDetails(cpu resource.Quantity, mem resource.Quantity, storage resource.Quantity) *ResourceDetails {
	return &ResourceDetails{
		cpu:     cpu,
		mem:     mem,
		storage: storage,
	}
}

func NewResourceDetailsFromCapacity(capacity *provider.NodeCapacity) (*ResourceDetails, error) {
	if capacity.CPUCores == nil {
		return nil, errors.New("CPUs must not be nil")
	}

	if capacity.Memory == nil {
		return nil, errors.New("memory must not be nil")
	}

	if capacity.Storage == nil {
		return nil, errors.New("storage must not be nil")
	}

	return &ResourceDetails{
		cpu:     *capacity.CPUCores,
		mem:     *capacity.Memory,
		storage: *capacity.Storage,
	}, nil
}

func (r *ResourceDetails) CPU() *resource.Quantity {
	return &r.cpu
}

func (r *ResourceDetails) Memory() *resource.Quantity {
	return &r.mem
}

func (r *ResourceDetails) Storage() *resource.Quantity {
	return &r.storage
}
