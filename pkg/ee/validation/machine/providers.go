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
	"strconv"

	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	azuretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure/types"
	gcptypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/gce/types"
	kubevirttypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/kubevirt/types"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/pkg/handler/common/provider"

	"k8s.io/apimachinery/pkg/api/resource"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getAWSResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config) (*ResourceQuota, error) {
	// extract storage and image info from provider config
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := awstypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting aws raw config: %w", err)
	}

	instanceType, err := configVarResolver.GetConfigVarStringValue(rawConfig.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS instance type from machine config: %v", err)
	}

	awsSize, err := provider.GetAWSInstance(instanceType)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS instance type data: %v", err)
	}

	// parse the AWS resource requests
	// memory and storage are given in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(awsSize.VCPUs))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %v", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%fG", awsSize.Memory))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %v", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %v", err)
	}

	return NewResourceQuota(cpuReq, memReq, storageReq), nil
}

func getGCPResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config) (*ResourceQuota, error) {
	// extract storage and image info from provider config
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := gcptypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP raw config: %w", err)
	}

	serviceAccount, err := configVarResolver.GetConfigVarStringValue(rawConfig.ServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP service account from machine config: %v", err)
	}
	machineType, err := configVarResolver.GetConfigVarStringValue(rawConfig.MachineType)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP machine type from machine config: %v", err)
	}
	zone, err := configVarResolver.GetConfigVarStringValue(rawConfig.Zone)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP zone from machine config: %v", err)
	}

	machineSize, err := provider.GetGCPInstanceSize(ctx, machineType, serviceAccount, zone)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP machine size data %v", err)
	}

	// parse the GCP resource requests
	// memory is given in MB and storage in GB
	cpuReq, err := resource.ParseQuantity(strconv.FormatInt(machineSize.VCPUs, 10))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %v", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%fM", machineSize.Memory))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %v", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %v", err)
	}

	return NewResourceQuota(cpuReq, memReq, storageReq), nil
}

func getAzureResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config) (*ResourceQuota, error) {
	// extract storage and image info from provider config
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := azuretypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure raw config: %w", err)
	}

	subId, err := configVarResolver.GetConfigVarStringValue(rawConfig.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure subscription ID from machine config: %v", err)
	}
	clientId, err := configVarResolver.GetConfigVarStringValue(rawConfig.ClientID)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure client ID from machine config: %v", err)
	}
	clientSecret, err := configVarResolver.GetConfigVarStringValue(rawConfig.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure client secret from machine config: %v", err)
	}
	tenantId, err := configVarResolver.GetConfigVarStringValue(rawConfig.TenantID)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure tenant ID from machine config: %v", err)
	}
	location, err := configVarResolver.GetConfigVarStringValue(rawConfig.Location)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure location from machine config: %v", err)
	}
	vmSizeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.VMSize)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure vm size name from machine config: %v", err)
	}

	vmSize, err := provider.GetAzureVMSize(ctx, subId, clientId, clientSecret, tenantId, location, vmSizeName)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure vm size data %v", err)
	}

	// parse the Azure resource requests
	// memory is given in MB and storage in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(int(vmSize.NumberOfCores)))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %v", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%fM", vmSize.MemoryInMB))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %v", err)
	}

	// Azure allows for setting os and data disk size separately
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DataDiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %v", err)
	}
	osDiskStorageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.OSDiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %v", err)
	}
	storageReq.Add(osDiskStorageReq)

	return NewResourceQuota(cpuReq, memReq, storageReq), nil
}

func getKubeVirtResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config) (*ResourceQuota, error) {
	// extract storage and image info from provider config
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := kubevirttypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting kubevirt raw config: %w", err)
	}

	cpu, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.CPUs)
	if err != nil {
		return nil, fmt.Errorf("error getting Kubevirt cpu request from machine config: %v", err)
	}
	memory, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.Memory)
	if err != nil {
		return nil, fmt.Errorf("error getting Kubevirt memory request from machine config: %v", err)
	}

	storage, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.PrimaryDisk.Size)
	if err != nil {
		return nil, fmt.Errorf("error getting Kubevirt primary disk size from machine config: %v", err)
	}

	// parse the KubeVirt resource requests
	// memory is in MB and storage is given in GB
	cpuReq, err := resource.ParseQuantity(cpu)
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %v", err)
	}
	memReq, err := resource.ParseQuantity(memory)
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %v", err)
	}
	storageReq, err := resource.ParseQuantity(storage)
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %v", err)
	}

	// Add all secondary disks
	for _, d := range rawConfig.VirtualMachine.Template.SecondaryDisks {
		secondaryStorage, err := configVarResolver.GetConfigVarStringValue(d.Size)
		if err != nil {
			return nil, fmt.Errorf("error getting Kubevirt secondary disk size from machine config: %v", err)
		}
		secondaryStorageReq, err := resource.ParseQuantity(secondaryStorage)
		if err != nil {
			return nil, fmt.Errorf("error parsing machine storage request to quantity: %v", err)
		}
		storageReq.Add(secondaryStorageReq)
	}

	return NewResourceQuota(cpuReq, memReq, storageReq), nil
}
