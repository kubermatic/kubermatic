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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
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

const (
	// GCP credential env.
	envServiceAccount = "GOOGLE_SERVICE_ACCOUNT"
	// Azure credential env.
	envClientID       = "AZURE_CLIENT_ID"
	envClientSecret   = "AZURE_CLIENT_SECRET"
	envTenantID       = "AZURE_TENANT_ID"
	envSubscriptionID = "AZURE_SUBSCRIPTION_ID"
	// Openstack credential env.
	envOSUsername                    = "OS_USER_NAME"
	envOSPassword                    = "OS_PASSWORD"
	envOSDomain                      = "OS_DOMAIN_NAME"
	envOSApplicationCredentialID     = "OS_APPLICATION_CREDENTIAL_ID"
	envOSApplicationCredentialSecret = "OS_APPLICATION_CREDENTIAL_SECRET"
	envOSToken                       = "OS_TOKEN"
	envOSProjectName                 = "OS_PROJECT_NAME"
	envOSProjectID                   = "OS_PROJECT_ID"
)

func GetMachineResourceUsage(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	machine *clusterv1alpha1.Machine,
	caBundle *certificates.CABundle,
) (*ResourceDetails, error) {
	config, err := types.GetConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read machine.spec.providerSpec: %w", err)
	}

	var quotaUsage *ResourceDetails
	// TODO add all providers
	switch config.CloudProvider {
	case types.CloudProviderFake:
		quotaUsage, err = getFakeQuotaRequest(config)
	case types.CloudProviderAWS:
		quotaUsage, err = getAWSResourceRequirements(ctx, userClient, config)
	case types.CloudProviderGoogle:
		quotaUsage, err = getGCPResourceRequirements(ctx, userClient, config)
	case types.CloudProviderAzure:
		quotaUsage, err = getAzureResourceRequirements(ctx, userClient, config)
	case types.CloudProviderKubeVirt:
		quotaUsage, err = getKubeVirtResourceRequirements(ctx, userClient, config)
	case types.CloudProviderVsphere:
		quotaUsage, err = getVsphereResourceRequirements(config)
	case types.CloudProviderOpenstack:
		quotaUsage, err = getOpenstackResourceRequirements(ctx, userClient, config, caBundle)
	default:
		// TODO skip for now, when all providers are added, throw error
		return NewResourceDetails(resource.Quantity{}, resource.Quantity{}, resource.Quantity{}), nil
	}

	return quotaUsage, err
}

func getAWSResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *types.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
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

func getGCPResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	rawConfig, err := gcptypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP raw config: %w", err)
	}

	serviceAccount, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ServiceAccount, envServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"serviceAccount\" field, error = %w", err)
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

func getAzureResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	rawConfig, err := azuretypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure raw config: %w", err)
	}

	subscriptionID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.SubscriptionID, envSubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"subscriptionID\" field, error = %w", err)
	}

	tenantID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.TenantID, envTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"tenantID\" field, error = %w", err)
	}

	clientID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ClientID, envClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"clientID\" field, error = %w", err)
	}

	clientSecret, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ClientSecret, envClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"clientSecret\" field, error = %w", err)
	}

	location, err := configVarResolver.GetConfigVarStringValue(rawConfig.Location)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure location from machine config: %w", err)
	}
	vmSizeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.VMSize)
	if err != nil {
		return nil, fmt.Errorf("error getting Azure vm size name from machine config: %w", err)
	}

	vmSize, err := provider.GetAzureVMSize(ctx, subscriptionID, clientID, clientSecret, tenantID, location, vmSizeName)
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

func getKubeVirtResourceRequirements(ctx context.Context,
	client ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
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

		cpuReq, memReq, err = provider.GetKubeVirtPresetResourceDetails(preset.Spec)
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

func getOpenstackResourceRequirements(ctx context.Context,
	client ctrlruntimeclient.Client,
	config *types.Config,
	caBundle *certificates.CABundle,
) (*ResourceDetails, error) {
	// extract storage and image info from provider config
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := openstacktypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting Openstack raw config: %w", err)
	}

	creds := &resources.OpenstackCredentials{}

	creds.Username, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Username, envOSUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"username\" field, error = %w", err)
	}
	creds.Password, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Password, envOSPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"password\" field, error = %w", err)
	}
	creds.ProjectID, err = getProjectIDOrTenantID(configVarResolver, rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"projectID\" or fallback to\"tenantID\" field, error = %w", err)
	}
	creds.Project, err = getProjectNameOrTenantName(configVarResolver, rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"projectName\" field or fallback to \"tenantName\" field, error = %w", err)
	}
	creds.ApplicationCredentialID, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ApplicationCredentialID, envOSApplicationCredentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"applicationCredentialID\" field, error = %w", err)
	}
	if creds.ApplicationCredentialID != "" {
		creds.ApplicationCredentialSecret, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ApplicationCredentialSecret, envOSApplicationCredentialSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get the value of \"applicationCredentialSecret\" field, error = %w", err)
		}
		return nil, nil
	}
	creds.Domain, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.DomainName, envOSDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"domainName\" field, error = %w", err)
	}
	creds.Token, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.TokenID, envOSToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"token\" field, error = %w", err)
	}

	flavor, err := configVarResolver.GetConfigVarStringValue(rawConfig.Flavor)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack flavor from machine config: %w", err)
	}
	identityEndpoint, err := configVarResolver.GetConfigVarStringValue(rawConfig.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack identity endpoint from machine config: %w", err)
	}
	region, err := configVarResolver.GetConfigVarStringValue(rawConfig.Region)
	if err != nil {
		return nil, fmt.Errorf("error getting OpenStack region from machine config: %w", err)
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

// Get the Project name from config or env var. If not defined fallback to tenant name.
func getProjectNameOrTenantName(configVarResolver *providerconfig.ConfigVarResolver, rawConfig *openstacktypes.RawConfig) (string, error) {
	projectName, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ProjectName, envOSProjectName)
	if err == nil && len(projectName) > 0 {
		return projectName, nil
	}

	// fallback to tenantName.
	return configVarResolver.GetConfigVarStringValue(rawConfig.TenantName)
}

// Get the Project id from config or env var. If not defined fallback to tenant id.
func getProjectIDOrTenantID(configVarResolver *providerconfig.ConfigVarResolver, rawConfig *openstacktypes.RawConfig) (string, error) {
	projectID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ProjectID, envOSProjectID)
	if err == nil && len(projectID) > 0 {
		return projectID, nil
	}

	// fallback to tenantName.
	return configVarResolver.GetConfigVarStringValue(rawConfig.TenantID)
}
