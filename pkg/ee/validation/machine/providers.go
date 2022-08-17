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
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	alibabatypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/alibaba/types"
	anexiatypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/anexia/types"
	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	azuretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure/types"
	digitaloceantypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/digitalocean/types"
	packetypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/equinixmetal/types"
	gcptypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/gce/types"
	hetznertypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/hetzner/types"
	kubevirttypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/kubevirt/types"
	nutanixtypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/nutanix/types"
	openstacktypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	vmwareclouddirectortypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vmwareclouddirector/types"
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
	envGoogleServiceAccount = "GOOGLE_SERVICE_ACCOUNT"
	// Azure credential env.
	envAzureClientID       = "AZURE_CLIENT_ID"
	envAzureClientSecret   = "AZURE_CLIENT_SECRET"
	envAzureTenantID       = "AZURE_TENANT_ID"
	envAzureSubscriptionID = "AZURE_SUBSCRIPTION_ID"
	// Openstack credential env.
	envOSUsername                    = "OS_USER_NAME"
	envOSPassword                    = "OS_PASSWORD"
	envOSDomain                      = "OS_DOMAIN_NAME"
	envOSApplicationCredentialID     = "OS_APPLICATION_CREDENTIAL_ID"
	envOSApplicationCredentialSecret = "OS_APPLICATION_CREDENTIAL_SECRET"
	envOSToken                       = "OS_TOKEN"
	envOSProjectName                 = "OS_PROJECT_NAME"
	envOSProjectID                   = "OS_PROJECT_ID"
	// DigitalOcean credential env.
	envDOToken = "DO_TOKEN"
	// Hetzner credential env.
	envHZToken = "HZ_TOKEN"
	// Alibaba credential env.
	envAlibabaAccessKeyID     = "ALIBABA_ACCESS_KEY_ID"
	envAlibabaAccessKeySecret = "ALIBABA_ACCESS_KEY_SECRET"
	// Packet credential env.
	envPacketToken     = "PACKET_API_KEY"
	envPacketProjectID = "PACKET_PROJECT_ID"
	// KubeVirt credential env.
	envKubeVirtKubeConfig = "KUBEVIRT_KUBECONFIG"
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
	case types.CloudProviderAlibaba:
		quotaUsage, err = getAlibabaResourceRequirements(ctx, userClient, config)
	case types.CloudProviderHetzner:
		quotaUsage, err = getHetznerResourceRequirements(ctx, userClient, config)
	case types.CloudProviderNutanix:
		quotaUsage, err = getNutanixResourceRequirements(ctx, userClient, config)
	case types.CloudProviderDigitalocean:
		quotaUsage, err = getDigitalOceanResourceRequirements(ctx, userClient, config)
	case types.CloudProviderVMwareCloudDirector:
		quotaUsage, err = getVMwareCloudDirectorResourceRequirements(ctx, userClient, config)
	case types.CloudProviderAnexia:
		quotaUsage, err = getAnexiaResourceRequirements(ctx, userClient, config)
	case types.CloudProviderPacket:
		quotaUsage, err = getPacketResourceRequirements(ctx, userClient, config)
	default:
		return nil, fmt.Errorf("Provider %s not supported", config.CloudProvider)
	}

	return quotaUsage, err
}

func getAWSResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
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
		return nil, fmt.Errorf("error getting gcp raw config: %w", err)
	}

	serviceAccount, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ServiceAccount, envGoogleServiceAccount)
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
		return nil, fmt.Errorf("failed to get azure raw config, error: %w", err)
	}

	subscriptionID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.SubscriptionID, envAzureSubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of azure \"subscriptionID\" field, error: %w", err)
	}

	tenantID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.TenantID, envAzureTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"tenantID\" field, error: %w", err)
	}

	clientID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ClientID, envAzureClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of azure \"clientID\" field, error: %w", err)
	}

	clientSecret, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ClientSecret, envAzureClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of azure \"clientSecret\" field, error: %w", err)
	}

	location, err := configVarResolver.GetConfigVarStringValue(rawConfig.Location)
	if err != nil {
		return nil, fmt.Errorf("error getting azure  \"location\" from machine config, error: %w", err)
	}
	vmSizeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.VMSize)
	if err != nil {
		return nil, fmt.Errorf("error getting azure \"vmSizeName\" from machine config, error : %w", err)
	}

	vmSize, err := provider.GetAzureVMSize(ctx, subscriptionID, clientID, clientSecret, tenantID, location, vmSizeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure \"vmSize\" data, error: %w", err)
	}

	// parse the Azure resource requests
	// memory is given in MB and storage in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(int(vmSize.NumberOfCores)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", vmSize.MemoryInMB))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
	}

	// Azure allows for setting os and data disk size separately
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DataDiskSize))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}
	osDiskStorageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.OSDiskSize))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
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
		return nil, fmt.Errorf("failed to get kubevirt raw config, error: %w", err)
	}

	// KubeVirt machine size can be configured either directly or through a flavor
	flavor, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Flavor.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get KubeVirt \"flavor\" from machine config, error: %w", err)
	}

	var cpuReq, memReq resource.Quantity
	// if flavor is set, then take the resource details from the vmi preset, otherwise take it from the config
	if len(flavor) != 0 {
		kubeconfig, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Auth.Kubeconfig, envKubeVirtKubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt kubeconfig from machine config, error: %w", err)
		}
		preset, err := provider.KubeVirtVMIPreset(ctx, base64.StdEncoding.EncodeToString([]byte(kubeconfig)), flavor)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt VMI Preset, error: %w", err)
		}

		cpuReq, memReq, err = provider.GetKubeVirtPresetResourceDetails(preset.Spec)
		if err != nil {
			return nil, err
		}
	} else {
		cpu, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.CPUs)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt cpu request from machine config, error: %w", err)
		}
		memory, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.Memory)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt memory request from machine config, error: %w", err)
		}
		// parse the KubeVirt resource requests
		cpuReq, err = resource.ParseQuantity(cpu)
		if err != nil {
			return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
		}
		memReq, err = resource.ParseQuantity(memory)
		if err != nil {
			return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
		}
	}

	storage, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.PrimaryDisk.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get KubeVirt primary disk size from machine config, error: %w", err)
	}
	storageReq, err := resource.ParseQuantity(storage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}

	// Add all secondary disks
	for _, d := range rawConfig.VirtualMachine.Template.SecondaryDisks {
		secondaryStorage, err := configVarResolver.GetConfigVarStringValue(d.Size)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt secondary disk size from machine config, error: %w", err)
		}
		secondaryStorageReq, err := resource.ParseQuantity(secondaryStorage)
		if err != nil {
			return nil, fmt.Errorf("failed to get machine storage request to quantity, error: %w", err)
		}
		storageReq.Add(secondaryStorageReq)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getVsphereResourceRequirements(config *types.Config) (*ResourceDetails, error) {
	// extract storage and image info from provider config
	rawConfig, err := vspheretypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere raw config: %w", err)
	}

	// parse the vSphere resource requests
	// memory is in MB and storage is given in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(int(rawConfig.CPUs)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", rawConfig.MemoryMB))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSizeGB))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getOpenstackResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
	caBundle *certificates.CABundle,
) (*ResourceDetails, error) {
	// extract storage and image info from provider config
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	rawConfig, err := openstacktypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get openstack raw config, error: %w", err)
	}

	creds := &resources.OpenstackCredentials{}

	creds.Username, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Username, envOSUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"username\" field, error: %w", err)
	}
	creds.Password, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Password, envOSPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"password\" field, error: %w", err)
	}
	creds.ProjectID, err = getProjectIDOrTenantID(configVarResolver, rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"projectID\" or fallback to\"tenantID\" field, error: %w", err)
	}
	creds.Project, err = getProjectNameOrTenantName(configVarResolver, rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"projectName\" field or fallback to \"tenantName\" field, error: %w", err)
	}
	creds.ApplicationCredentialID, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ApplicationCredentialID, envOSApplicationCredentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"applicationCredentialID\" field, error: %w", err)
	}
	if creds.ApplicationCredentialID != "" {
		creds.ApplicationCredentialSecret, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ApplicationCredentialSecret, envOSApplicationCredentialSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get the value of openstack \"applicationCredentialSecret\" field, error: %w", err)
		}
	}
	creds.Domain, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.DomainName, envOSDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"domainName\" field, error: %w", err)
	}
	creds.Token, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.TokenID, envOSToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"token\" field, error: %w", err)
	}

	flavor, err := configVarResolver.GetConfigVarStringValue(rawConfig.Flavor)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"flavor\" field, error: %w", err)
	}
	identityEndpoint, err := configVarResolver.GetConfigVarStringValue(rawConfig.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"identityEndpoint\" field, error: %w", err)
	}
	region, err := configVarResolver.GetConfigVarStringValue(rawConfig.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"region\" field, error: %w", err)
	}
	flavorSize, err := provider.GetOpenStackFlavorSize(creds, identityEndpoint, region, caBundle.CertPool(), flavor)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"flavorSize\" field, error: %w", err)
	}

	// parse the Openstack resource requests
	// memory is in MB and storage is in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(flavorSize.VCPUs))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", flavorSize.Memory))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", flavorSize.Disk))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

// Get the Project name from config or env var. If not defined fallback to tenant name.
func getProjectNameOrTenantName(configVarResolver *providerconfig.ConfigVarResolver,
	rawConfig *openstacktypes.RawConfig,
) (string, error) {
	projectName, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ProjectName, envOSProjectName)
	if err == nil && len(projectName) > 0 {
		return projectName, nil
	}

	// fallback to tenantName.
	return configVarResolver.GetConfigVarStringValue(rawConfig.TenantName)
}

// Get the Project id from config or env var. If not defined fallback to tenant id.
func getProjectIDOrTenantID(configVarResolver *providerconfig.ConfigVarResolver,
	rawConfig *openstacktypes.RawConfig,
) (string, error) {
	projectID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ProjectID, envOSProjectID)
	if err == nil && len(projectID) > 0 {
		return projectID, nil
	}

	// fallback to tenantName.
	return configVarResolver.GetConfigVarStringValue(rawConfig.TenantID)
}

func getAlibabaResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	rawConfig, err := alibabatypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get alibaba raw config: %w", err)
	}

	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	instanceType, err := configVarResolver.GetConfigVarStringValue(rawConfig.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of alibaba \"instanceType\" field, error: %w", err)
	}
	accessKeyID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.AccessKeyID, envAlibabaAccessKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of alibaba \"accessKeyID\" field, error: %w", err)
	}
	accessKeySecret, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.AccessKeySecret, envAlibabaAccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of alibaba \"accessKeySecret\" field, error: %w", err)
	}
	region, err := configVarResolver.GetConfigVarStringValue(rawConfig.RegionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of  alibaba \"region\" from machine config, error: %w", err)
	}
	disk, err := configVarResolver.GetConfigVarStringValue(rawConfig.DiskSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of alibaba \"disk\" from machine config, error: %w", err)
	}

	instTypes, err := provider.DescribeAlibabaInstanceTypes(accessKeyID, accessKeySecret, region, instanceType)
	if err != nil {
		return nil, err
	}
	ecsInstanceType := instTypes.InstanceTypes.InstanceType
	if len(ecsInstanceType) > 0 {
		// parse the Alibaba resource requests
		// memory is in GB and storage is in GB
		cpuReq, err := resource.ParseQuantity(strconv.Itoa(ecsInstanceType[0].CpuCoreCount))
		if err != nil {
			return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
		}
		memReq, err := resource.ParseQuantity(fmt.Sprintf("%fG", ecsInstanceType[0].MemorySize))
		if err != nil {
			return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
		}
		storageReq, err := resource.ParseQuantity(fmt.Sprintf("%sG", disk))
		if err != nil {
			return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
		}
		return NewResourceDetails(cpuReq, memReq, storageReq), nil
	}

	return nil, nil
}

func getHetznerResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	rawConfig, err := hetznertypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get hetzner raw config, error: %w", err)
	}
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)

	serverTypeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.ServerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"serverType\" field, error: %w", err)
	}
	token, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Token, envHZToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of hetzner \"token\" field, error: %w", err)
	}

	serverType, err := provider.GetHetznerServerType(ctx, token, serverTypeName)
	if err != nil {
		return nil, err
	}

	// parse the Hetzner resource requests
	// memory is in GB and storage is in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(serverType.Cores))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%fG", serverType.Memory))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", serverType.Disk))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getNutanixResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	rawConfig, err := nutanixtypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get nutanix raw config, error: %w", err)
	}

	var totalCPUCores int64
	switch {
	case rawConfig.CPUs == 0:
		return nil, fmt.Errorf("found invalid value of nutanix \"cpus\" from machine config, %v", rawConfig.CPUs)
	case rawConfig.CPUCores != nil:
		totalCPUCores = rawConfig.CPUs * (*rawConfig.CPUCores)
	default:
		totalCPUCores = rawConfig.CPUs
	}

	cpuReq, err := resource.ParseQuantity(strconv.FormatInt(totalCPUCores, 10))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", rawConfig.MemoryMB))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSize))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getDigitalOceanResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	rawConfig, err := digitaloceantypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get digitalOcean raw config, error: %w", err)
	}
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)

	token, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Token, envDOToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of digitalOcean \"token\" field, error: %w", err)
	}
	sizeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of digitalOcean \"size\" field, error: %w", err)
	}

	godosize, err := provider.DescribeDigitaloceanSize(ctx, token, sizeName)
	if err != nil {
		return nil, err
	}

	// parse the DigitalOcean resource requests
	// memory is in MB and storage is in GB
	cpuReq, err := resource.ParseQuantity(strconv.Itoa(godosize.Vcpus))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", godosize.Memory))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", godosize.Disk))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}
	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getVMwareCloudDirectorResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	rawConfig, err := vmwareclouddirectortypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get vmwareclouddirector raw config, error: %w", err)
	}

	var totalCPUCores int64
	switch {
	case rawConfig.CPUs == 0:
		return nil, fmt.Errorf("found invalid value of vmwareclouddirector \"cpus\" from machine config, %v", rawConfig.CPUs)
	case rawConfig.CPUCores != 0:
		totalCPUCores = rawConfig.CPUs * rawConfig.CPUCores
	default:
		totalCPUCores = rawConfig.CPUs
	}

	cpuReq, err := resource.ParseQuantity(fmt.Sprintf("%d", totalCPUCores))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", rawConfig.MemoryMB))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", *rawConfig.DiskSizeGB))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getAnexiaResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	rawConfig, err := anexiatypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get anexia raw config, error: %w", err)
	}
	cpuReq, err := resource.ParseQuantity(fmt.Sprintf("%d", rawConfig.CPUs))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity, error: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", rawConfig.Memory))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity, error: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSize))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity, error: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getPacketResourceRequirements(ctx context.Context,
	client ctrlruntimeclient.Client,
	config *types.Config,
) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := packetypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get packet raw config, error: %w", err)
	}

	token, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Token, envPacketToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of packet \"apiKey\", error: %w", err)
	}

	projectID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ProjectID, envPacketProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of packet \"projectID\", error: %w", err)
	}

	instanceType, err := configVarResolver.GetConfigVarStringValue(rawConfig.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of packet \"instanceType\", error: %w", err)
	}

	plan, err := provider.DescribePacketSize(token, projectID, instanceType)
	if err != nil {
		return nil, err
	}

	var totalCPUs int
	for _, cpu := range plan.Specs.Cpus {
		totalCPUs += cpu.Count
	}
	cpuReq, err := resource.ParseQuantity(fmt.Sprintf("%d", totalCPUs))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine cpu request to quantity: %w", err)
	}

	var storageReq, memReq resource.Quantity
	for _, drive := range plan.Specs.Drives {
		if drive.Size != "" && drive.Count != 0 {
			// trimming "B" as quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'.
			storage, err := resource.ParseQuantity(strings.TrimSuffix(drive.Size, "B"))
			if err != nil {
				fmt.Println("error parsing machine storage request to quantity: %w", err)
			}
			// total storage for each types = drive count *drive Size.
			strDrive := strconv.FormatInt(storage.Value()*int64(drive.Count), 10)
			totalStorage, err := resource.ParseQuantity(strDrive)
			if err != nil {
				return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
			}
			// Adding storage value for all storage types like "SSD", "NVME".
			storageReq.Add(totalStorage)
		}
	}

	if plan.Specs.Memory.Total != "" {
		memReq, err = resource.ParseQuantity(plan.Specs.Memory.Total)
		if err != nil {
			return nil, fmt.Errorf("error parsing machine memory request to quantity: %w", err)
		}
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}
