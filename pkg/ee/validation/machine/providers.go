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

	kubevirtv1 "kubevirt.io/api/core/v1"

	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	azuretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure/types"
	gcptypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/gce/types"
	kubevirttypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/kubevirt/types"
	openstacktypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	vspheretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"

	"k8s.io/apimachinery/pkg/api/resource"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getAWSResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := awstypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting aws raw config: %w", err)
	}

	instanceType, err := configVarResolver.GetConfigVarStringValue(rawConfig.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS instance type from machine config: %w", err)
	}

	awsSize, err := provider.GetAWSInstance(instanceType)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS instance type data: %w", err)
	}

	// parse the AWS resource requests
	// memory and storage are given in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(awsSize.VCPUs))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%fG", awsSize.Memory))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getGCPResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := gcptypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP raw config: %w", err)
	}

	serviceAccount, err := configVarResolver.GetConfigVarStringValue(rawConfig.ServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP service account from machine config: %w", err)
	}
	machineType, err := configVarResolver.GetConfigVarStringValue(rawConfig.MachineType)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP machine type from machine config: %w", err)
	}
	zone, err := configVarResolver.GetConfigVarStringValue(rawConfig.Zone)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP zone from machine config: %w", err)
	}

	machineSize, err := provider.GetGCPInstanceSize(ctx, machineType, serviceAccount, zone)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP machine size data %w", err)
	}

	// parse the GCP resource requests
	// memory is given in MB and storage in GB
	cpuReq, err := resource.ParseQuantity(strconv.FormatInt(machineSize.VCPUs, 10))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", machineSize.Memory))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getAzureResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := azuretypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure raw config: %w", err)
	}

	subId, err := configVarResolver.GetConfigVarStringValue(rawConfig.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure subscription ID from machine config: %w", err)
	}
	clientId, err := configVarResolver.GetConfigVarStringValue(rawConfig.ClientID)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure client ID from machine config: %w", err)
	}
	clientSecret, err := configVarResolver.GetConfigVarStringValue(rawConfig.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure client secret from machine config: %w", err)
	}
	tenantId, err := configVarResolver.GetConfigVarStringValue(rawConfig.TenantID)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure tenant ID from machine config: %w", err)
	}
	location, err := configVarResolver.GetConfigVarStringValue(rawConfig.Location)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure location from machine config: %w", err)
	}
	vmSizeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.VMSize)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure vm size name from machine config: %w", err)
	}

	vmSize, err := provider.GetAzureVMSize(ctx, subId, clientId, clientSecret, tenantId, location, vmSizeName)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure vm size data %w", err)
	}

	// parse the Azure resource requests
	// memory is given in MB and storage in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(int(vmSize.NumberOfCores)))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", vmSize.MemoryInMB))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %w", err)
	}

	// Azure allows for setting os and data disk size separately
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DataDiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}
	osDiskStorageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.OSDiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}
	storageReq.Add(osDiskStorageReq)

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getKubeVirtResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := kubevirttypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting kubevirt raw config: %w", err)
	}

	// KubeVirt machine size can be configured either directly or through a flavor
	flavor, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Flavor.Name)
	if err != nil {
		return nil, fmt.Errorf("error getting KubeVirt flavor from machine config: %w", err)
	}

	var cpuReq, memReq resource.Quantity
	// if flavor is set, then take the resource details from the vmi preset, otherwise take it from the config
	if len(flavor) != 0 {
		kubeconfig, err := configVarResolver.GetConfigVarStringValue(rawConfig.Auth.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("error getting KubeVirt kubeconfig from machine config: %w", err)
		}
		preset, err := provider.KubeVirtVMIPreset(ctx, kubeconfig, flavor)
		if err != nil {
			return nil, fmt.Errorf("error getting KubeVirt VMI Preset: %w", err)
		}

		cpuReq, memReq, err = getKubeVirtPresetResourceDetails(preset)
		if err != nil {
			return nil, err
		}
	} else {
		cpu, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.CPUs)
		if err != nil {
			return nil, fmt.Errorf("error getting KubeVirt cpu request from machine config: %w", err)
		}
		memory, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.Memory)
		if err != nil {
			return nil, fmt.Errorf("error getting KubeVirt memory request from machine config: %w", err)
		}
		// parse the KubeVirt resource requests
		cpuReq, err = resource.ParseQuantity(cpu)
		if err != nil {
			return nil, fmt.Errorf("error parsing machine cpu request to quantity: %w", err)
		}
		memReq, err = resource.ParseQuantity(memory)
		if err != nil {
			return nil, fmt.Errorf("error parsing machine memory request to quantity: %w", err)
		}
	}

	storage, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.PrimaryDisk.Size)
	if err != nil {
		return nil, fmt.Errorf("error getting KubeVirt primary disk size from machine config: %w", err)
	}
	storageReq, err := resource.ParseQuantity(storage)
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}

	// Add all secondary disks
	for _, d := range rawConfig.VirtualMachine.Template.SecondaryDisks {
		secondaryStorage, err := configVarResolver.GetConfigVarStringValue(d.Size)
		if err != nil {
			return nil, fmt.Errorf("error getting KubeVirt secondary disk size from machine config: %w", err)
		}
		secondaryStorageReq, err := resource.ParseQuantity(secondaryStorage)
		if err != nil {
			return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
		}
		storageReq.Add(secondaryStorageReq)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

// getKubeVirtPresetResourceDetails extracts cpu and mem resource requests from the kubevirt preset
// for CPU, take the value by priority:
// - if resource limit is set, use that
// - check if resource request is set, use that
// - check if spec.cpu is set, if socket and threads are set then do the calculation, use that
// - if nothing from above, consider its 1
// for memory, take the value by priority:
// - if resource limit is set, use that
// - if resource request is set, use that
// - if nothing from above, consider it 1Gi.
func getKubeVirtPresetResourceDetails(preset *kubevirtv1.VirtualMachineInstancePreset) (resource.Quantity, resource.Quantity, error) {
	var err error
	// Get CPU
	cpuReq := resource.MustParse("1")
	if preset.Spec.Domain.CPU != nil && preset.Spec.Domain.CPU.Cores != 0 {
		cores := preset.Spec.Domain.CPU.Cores
		// if threads and sockets are set, calculate VCPU
		threads := preset.Spec.Domain.CPU.Threads
		if threads == 0 {
			threads = 1
		}
		sockets := preset.Spec.Domain.CPU.Sockets
		if sockets == 0 {
			sockets = 1
		}

		cpuReq, err = resource.ParseQuantity(strconv.Itoa(int(cores * threads * sockets)))
		if err != nil {
			return resource.Quantity{}, resource.Quantity{}, fmt.Errorf("error parsing calculated KubeVirt VCPU: %w", err)
		}
	}
	if !preset.Spec.Domain.Resources.Requests.Cpu().IsZero() {
		cpuReq = *preset.Spec.Domain.Resources.Requests.Cpu()
	}
	if !preset.Spec.Domain.Resources.Limits.Cpu().IsZero() {
		cpuReq = *preset.Spec.Domain.Resources.Limits.Cpu()
	}

	// get MEM
	memReq := resource.MustParse("1Gi")
	if !preset.Spec.Domain.Resources.Requests.Memory().IsZero() {
		memReq = *preset.Spec.Domain.Resources.Requests.Memory()
	}
	if !preset.Spec.Domain.Resources.Limits.Memory().IsZero() {
		memReq = *preset.Spec.Domain.Resources.Limits.Memory()
	}

	return cpuReq, memReq, nil
}

func getVsphereResourceRequirements(config *types.Config) (*ResourceDetails, error) {
	// extract storage and image info from provider config
	rawConfig, err := vspheretypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting vsphere raw config: %w", err)
	}

	// parse the vSphere resource requests
	// memory is in MB and storage is given in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(int(rawConfig.CPUs)))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", rawConfig.MemoryMB))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSizeGB))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getOpenstackResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *types.Config, caBundle *certificates.CABundle) (*ResourceDetails, error) {
	// extract storage and image info from provider config
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := openstacktypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting Openstack raw config: %w", err)
	}

	identityEndpoint, err := configVarResolver.GetConfigVarStringValue(rawConfig.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack identity endpoint from machine config: %w", err)
	}
	region, err := configVarResolver.GetConfigVarStringValue(rawConfig.Region)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack region from machine config: %w", err)
	}
	username, err := configVarResolver.GetConfigVarStringValue(rawConfig.Username)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack username from machine config: %w", err)
	}
	password, err := configVarResolver.GetConfigVarStringValue(rawConfig.Password)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack password from machine config: %w", err)
	}
	tenantId, err := configVarResolver.GetConfigVarStringValue(rawConfig.TenantID)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack tenant ID from machine config: %w", err)
	}
	projectId, err := configVarResolver.GetConfigVarStringValue(rawConfig.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack project ID from machine config: %w", err)
	}
	tenantName, err := configVarResolver.GetConfigVarStringValue(rawConfig.TenantName)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack tenant name from machine config: %w", err)
	}
	projectName, err := configVarResolver.GetConfigVarStringValue(rawConfig.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack tenant name from machine config: %w", err)
	}
	appCredId, err := configVarResolver.GetConfigVarStringValue(rawConfig.ApplicationCredentialID)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack application credential ID from machine config: %w", err)
	}
	appCredSecret, err := configVarResolver.GetConfigVarStringValue(rawConfig.ApplicationCredentialSecret)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack application credential secret from machine config: %w", err)
	}
	tokenId, err := configVarResolver.GetConfigVarStringValue(rawConfig.TokenID)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack token ID from machine config: %w", err)
	}
	domainName, err := configVarResolver.GetConfigVarStringValue(rawConfig.DomainName)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack domain name from machine config: %w", err)
	}
	flavor, err := configVarResolver.GetConfigVarStringValue(rawConfig.Flavor)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack flavor from machine config: %w", err)
	}

	creds := &resources.OpenstackCredentials{
		Username:                    username,
		Password:                    password,
		Domain:                      domainName,
		ProjectID:                   projectId,
		Project:                     projectName,
		ApplicationCredentialID:     appCredId,
		ApplicationCredentialSecret: appCredSecret,
		Token:                       tokenId,
	}

	// if projectName and projectId are empty, fallback to tenantName and tenantId
	if len(projectId) == 0 {
		creds.Project = tenantId
	}
	if len(projectName) == 0 {
		creds.Project = tenantName
	}

	flavorSize, err := provider.GetOpenStackFlavorSize(creds, identityEndpoint, region, caBundle.CertPool(), flavor)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack flavor size data %w", err)
	}

	// parse the Openstack resource requests
	// memory is in MB and storage is in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(flavorSize.VCPUs))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", flavorSize.Memory))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine memory request to quantity: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", flavorSize.Disk))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}
