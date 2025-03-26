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

	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/alibaba"
	awsdata "k8c.io/kubermatic/v2/pkg/provider/cloud/aws/data"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/digitalocean"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/hetzner"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/packet"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	alibabatypes "k8c.io/machine-controller/sdk/cloudprovider/alibaba"
	anexiatypes "k8c.io/machine-controller/sdk/cloudprovider/anexia"
	awstypes "k8c.io/machine-controller/sdk/cloudprovider/aws"
	azuretypes "k8c.io/machine-controller/sdk/cloudprovider/azure"
	digitaloceantypes "k8c.io/machine-controller/sdk/cloudprovider/digitalocean"
	equinixtypes "k8c.io/machine-controller/sdk/cloudprovider/equinixmetal"
	gcptypes "k8c.io/machine-controller/sdk/cloudprovider/gce"
	hetznertypes "k8c.io/machine-controller/sdk/cloudprovider/hetzner"
	kubevirttypes "k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
	nutanixtypes "k8c.io/machine-controller/sdk/cloudprovider/nutanix"
	openstacktypes "k8c.io/machine-controller/sdk/cloudprovider/openstack"
	vmwareclouddirectortypes "k8c.io/machine-controller/sdk/cloudprovider/vmwareclouddirector"
	vspheretypes "k8c.io/machine-controller/sdk/cloudprovider/vsphere"
	"k8c.io/machine-controller/sdk/providerconfig"

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
	// Equinix Metal credential env.
	envMetalToken     = "METAL_AUTH_TOKEN"
	envMetalProjectID = "METAL_PROJECT_ID"
	// KubeVirt credential env.
	envKubeVirtKubeconfig = "KUBEVIRT_KUBECONFIG"
)

func GetMachineResourceUsage(ctx context.Context, userClient ctrlruntimeclient.Client, machine *clusterv1alpha1.Machine, caBundle *certificates.CABundle) (*ResourceDetails, error) {
	config, err := providerconfig.GetConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read machine.spec.providerSpec: %w", err)
	}

	var quotaUsage *ResourceDetails
	switch config.CloudProvider {
	case providerconfig.CloudProviderFake, providerconfig.CloudProviderExternal:
		quotaUsage, err = getFakeQuotaRequest(config)
	case providerconfig.CloudProviderAWS:
		quotaUsage, err = getAWSResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderGoogle:
		quotaUsage, err = getGCPResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderAzure:
		quotaUsage, err = getAzureResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderKubeVirt:
		quotaUsage, err = getKubeVirtResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderVsphere:
		quotaUsage, err = getVsphereResourceRequirements(config)
	case providerconfig.CloudProviderOpenstack:
		quotaUsage, err = getOpenstackResourceRequirements(ctx, userClient, config, caBundle)
	case providerconfig.CloudProviderAlibaba:
		quotaUsage, err = getAlibabaResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderHetzner:
		quotaUsage, err = getHetznerResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderNutanix:
		quotaUsage, err = getNutanixResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderDigitalocean:
		quotaUsage, err = getDigitalOceanResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderVMwareCloudDirector:
		quotaUsage, err = getVMwareCloudDirectorResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderAnexia:
		quotaUsage, err = getAnexiaResourceRequirements(ctx, userClient, config)
	case providerconfig.CloudProviderEquinixMetal, providerconfig.CloudProviderPacket:
		// Name Packet has been replaced at some point by Equinix Metal.
		// We are in the process of migration to the new name, meaning that both names appear in our sourcecode.
		quotaUsage, err = getPacketResourceRequirements(ctx, userClient, config)
	default:
		return nil, fmt.Errorf("Provider %s not supported", config.CloudProvider)
	}

	return quotaUsage, err
}

func getAWSResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	rawConfig, err := awstypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting aws raw config: %w", err)
	}

	instanceType, err := configVarResolver.GetConfigVarStringValue(rawConfig.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS instance type from machine config: %w", err)
	}

	instanceSize, err := awsdata.GetInstanceSize(instanceType)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS instance type data: %w", err)
	}

	// overwrite the reported disk size with the configured one
	if err := instanceSize.WithStorage(int(rawConfig.DiskSize), "G"); err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}

	return NewResourceDetailsFromCapacity(instanceSize)
}

func getGCPResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	rawConfig, err := gcptypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("error getting gcp raw config: %w", err)
	}

	serviceAccount, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ServiceAccount, envGoogleServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"serviceAccount\" field: %w", err)
	}

	machineType, err := configVarResolver.GetConfigVarStringValue(rawConfig.MachineType)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP machine type from machine config: %w", err)
	}
	zone, err := configVarResolver.GetConfigVarStringValue(rawConfig.Zone)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP zone from machine config: %w", err)
	}

	capacity, err := gcp.GetMachineSize(ctx, machineType, serviceAccount, zone)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP machine size data %w", err)
	}

	// parse the GCP resource requests
	// memory is given in MB and storage in GB
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSize))
	if err != nil {
		return nil, fmt.Errorf("error parsing machine storage request to quantity: %w", err)
	}
	capacity.Storage = &storageReq

	return NewResourceDetailsFromCapacity(capacity)
}

func getAzureResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	rawConfig, err := azuretypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get azure raw config: %w", err)
	}

	subscriptionID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.SubscriptionID, envAzureSubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of azure \"subscriptionID\" field: %w", err)
	}

	tenantID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.TenantID, envAzureTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"tenantID\" field: %w", err)
	}

	clientID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ClientID, envAzureClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of azure \"clientID\" field: %w", err)
	}

	clientSecret, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ClientSecret, envAzureClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of azure \"clientSecret\" field: %w", err)
	}

	location, err := configVarResolver.GetConfigVarStringValue(rawConfig.Location)
	if err != nil {
		return nil, fmt.Errorf("error getting azure  \"location\" from machine config: %w", err)
	}
	vmSizeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.VMSize)
	if err != nil {
		return nil, fmt.Errorf("error getting azure \"vmSizeName\" from machine config, error : %w", err)
	}

	creds := azure.Credentials{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}

	vmSize, err := azure.GetVMSize(ctx, creds, location, vmSizeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure VM size: %w", err)
	}

	// Azure allows for setting os and data disk size separately
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DataDiskSize))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity: %w", err)
	}
	osDiskStorageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.OSDiskSize))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity: %w", err)
	}
	storageReq.Add(osDiskStorageReq)
	vmSize.Storage = &storageReq

	return NewResourceDetailsFromCapacity(vmSize)
}

func getKubeVirtResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := kubevirttypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubevirt raw config: %w", err)
	}

	var cpuReq, memReq resource.Quantity
	// KubeVirt machine size can be configured either directly or through instancetypes.
	// If VM templating (Instancetype) is set then read cpu and memory from it.
	if rawConfig.VirtualMachine.Instancetype != nil && len(rawConfig.VirtualMachine.Instancetype.Name) != 0 {
		kubeconfig, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Auth.Kubeconfig, envKubeVirtKubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt kubeconfig from machine config, error: %w", err)
		}
		capacity, err := kubevirt.DescribeInstanceType(ctx, kubeconfig, rawConfig.VirtualMachine.Instancetype)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt VMI instancetype: %w", err)
		}
		cpuReq = *capacity.CPUCores
		memReq = *capacity.Memory
	} else {
		cpu, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.CPUs)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt cpu request from machine config: %w", err)
		}
		memory, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.Memory)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt memory request from machine config: %w", err)
		}
		cpuReq, err = resource.ParseQuantity(cpu)
		if err != nil {
			return nil, fmt.Errorf("failed to parse machine cpu request to quantity: %w", err)
		}
		memReq, err = resource.ParseQuantity(memory)
		if err != nil {
			return nil, fmt.Errorf("failed to parse machine memory request to quantity: %w", err)
		}
	}

	storage, err := configVarResolver.GetConfigVarStringValue(rawConfig.VirtualMachine.Template.PrimaryDisk.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get KubeVirt primary disk size from machine config: %w", err)
	}
	storageReq, err := resource.ParseQuantity(storage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity: %w", err)
	}

	// Add all secondary disks
	for _, d := range rawConfig.VirtualMachine.Template.SecondaryDisks {
		secondaryStorage, err := configVarResolver.GetConfigVarStringValue(d.Size)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeVirt secondary disk size from machine config: %w", err)
		}
		secondaryStorageReq, err := resource.ParseQuantity(secondaryStorage)
		if err != nil {
			return nil, fmt.Errorf("failed to get machine storage request to quantity: %w", err)
		}
		storageReq.Add(secondaryStorageReq)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getVsphereResourceRequirements(config *providerconfig.Config) (*ResourceDetails, error) {
	// extract storage and image info from provider config
	rawConfig, err := vspheretypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere raw config: %w", err)
	}

	capacity := provider.NewNodeCapacity()
	capacity.WithCPUCount(int(rawConfig.CPUs))

	if err := capacity.WithMemory(int(rawConfig.MemoryMB), "M"); err != nil {
		return nil, fmt.Errorf("failed to parse memory size: %w", err)
	}

	if err := capacity.WithStorage(int(*rawConfig.DiskSizeGB), "G"); err != nil {
		return nil, fmt.Errorf("failed to parse disk size: %w", err)
	}

	return NewResourceDetailsFromCapacity(capacity)
}

func getOpenstackResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config, caBundle *certificates.CABundle) (*ResourceDetails, error) {
	// extract storage and image info from provider config
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	rawConfig, err := openstacktypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get openstack raw config: %w", err)
	}

	creds := &resources.OpenstackCredentials{}

	creds.Username, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Username, envOSUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"username\" field: %w", err)
	}
	creds.Password, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Password, envOSPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"password\" field: %w", err)
	}
	creds.ProjectID, err = getProjectIDOrTenantID(configVarResolver, rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"projectID\" or fallback to\"tenantID\" field: %w", err)
	}
	creds.Project, err = getProjectNameOrTenantName(configVarResolver, rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"projectName\" field or fallback to \"tenantName\" field: %w", err)
	}
	creds.ApplicationCredentialID, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ApplicationCredentialID, envOSApplicationCredentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"applicationCredentialID\" field: %w", err)
	}
	if creds.ApplicationCredentialID != "" {
		creds.ApplicationCredentialSecret, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ApplicationCredentialSecret, envOSApplicationCredentialSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get the value of openstack \"applicationCredentialSecret\" field: %w", err)
		}
	}
	creds.Domain, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.DomainName, envOSDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"domainName\" field: %w", err)
	}
	creds.Token, err = configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.TokenID, envOSToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"token\" field: %w", err)
	}

	flavorName, err := configVarResolver.GetConfigVarStringValue(rawConfig.Flavor)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"flavor\" field: %w", err)
	}
	identityEndpoint, err := configVarResolver.GetConfigVarStringValue(rawConfig.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"identityEndpoint\" field: %w", err)
	}
	region, err := configVarResolver.GetConfigVarStringValue(rawConfig.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"region\" field: %w", err)
	}

	flavor, err := openstack.DescribeFlavor(creds, identityEndpoint, region, caBundle.CertPool(), flavorName)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of openstack \"flavorSize\" field: %w", err)
	}

	if rawConfig.RootDiskSizeGB != nil {
		// Setting custom disk size
		storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", *rawConfig.RootDiskSizeGB))
		if err != nil {
			return nil, fmt.Errorf("failed to parse machine storage request to quantity: %w", err)
		}
		flavor.Storage = &storageReq
	}

	return NewResourceDetailsFromCapacity(flavor)
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

	// fallback to tenantID.
	return configVarResolver.GetConfigVarStringValue(rawConfig.TenantID)
}

func getAlibabaResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	rawConfig, err := alibabatypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get alibaba raw config: %w", err)
	}

	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)
	instanceType, err := configVarResolver.GetConfigVarStringValue(rawConfig.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of alibaba \"instanceType\" field: %w", err)
	}
	accessKeyID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.AccessKeyID, envAlibabaAccessKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of alibaba \"accessKeyID\" field: %w", err)
	}
	accessKeySecret, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.AccessKeySecret, envAlibabaAccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of alibaba \"accessKeySecret\" field: %w", err)
	}
	region, err := configVarResolver.GetConfigVarStringValue(rawConfig.RegionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of  alibaba \"region\" from machine config: %w", err)
	}
	disk, err := configVarResolver.GetConfigVarStringValue(rawConfig.DiskSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of alibaba \"disk\" from machine config: %w", err)
	}
	diskGB, err := strconv.ParseInt(disk, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid disk size %q: %w", disk, err)
	}

	capacity, err := alibaba.DescribeInstanceType(accessKeyID, accessKeySecret, region, instanceType)
	if err != nil {
		return nil, err
	}

	if err := capacity.WithStorage(int(diskGB), "G"); err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity: %w", err)
	}

	return NewResourceDetailsFromCapacity(capacity)
}

func getHetznerResourceRequirements(ctx context.Context,
	userClient ctrlruntimeclient.Client,
	config *providerconfig.Config,
) (*ResourceDetails, error) {
	rawConfig, err := hetznertypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get hetzner raw config: %w", err)
	}
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)

	serverTypeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.ServerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of \"serverType\" field: %w", err)
	}
	token, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Token, envHZToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of hetzner \"token\" field: %w", err)
	}

	capacity, err := hetzner.GetServerType(ctx, token, serverTypeName)
	if err != nil {
		return nil, err
	}

	return NewResourceDetailsFromCapacity(capacity)
}

func getNutanixResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	rawConfig, err := nutanixtypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get nutanix raw config: %w", err)
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

	capacity := provider.NewNodeCapacity()
	capacity.WithCPUCount(int(totalCPUCores))

	if err := capacity.WithMemory(int(rawConfig.MemoryMB), "M"); err != nil {
		return nil, fmt.Errorf("failed to parse memory size: %w", err)
	}

	if err := capacity.WithStorage(int(*rawConfig.DiskSize), "G"); err != nil {
		return nil, fmt.Errorf("failed to parse disk size: %w", err)
	}

	return NewResourceDetailsFromCapacity(capacity)
}

func getDigitalOceanResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	rawConfig, err := digitaloceantypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get digitalOcean raw config: %w", err)
	}
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, userClient)

	token, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Token, envDOToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of digitalOcean \"token\" field: %w", err)
	}
	sizeName, err := configVarResolver.GetConfigVarStringValue(rawConfig.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of digitalOcean \"size\" field: %w", err)
	}

	capacity, err := digitalocean.DescribeDropletSize(ctx, token, sizeName)
	if err != nil {
		return nil, err
	}

	return NewResourceDetailsFromCapacity(capacity)
}

func getVMwareCloudDirectorResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	rawConfig, err := vmwareclouddirectortypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get VMware Cloud Director raw config: %w", err)
	}

	var totalCPUCores int64
	switch {
	case rawConfig.CPUs == 0:
		return nil, fmt.Errorf("found invalid value of VMware Cloud Director \"cpus\" from machine config, %v", rawConfig.CPUs)
	case rawConfig.CPUCores != 0:
		totalCPUCores = rawConfig.CPUs * rawConfig.CPUCores
	default:
		totalCPUCores = rawConfig.CPUs
	}

	capacity := provider.NewNodeCapacity()
	capacity.WithCPUCount(int(totalCPUCores))

	if err := capacity.WithMemory(int(rawConfig.MemoryMB), "M"); err != nil {
		return nil, fmt.Errorf("failed to parse memory size: %w", err)
	}

	if rawConfig.DiskSizeGB != nil {
		if err := capacity.WithStorage(int(*rawConfig.DiskSizeGB), "G"); err != nil {
			return nil, fmt.Errorf("failed to parse disk size: %w", err)
		}
	}

	return NewResourceDetailsFromCapacity(capacity)
}

func getAnexiaResourceRequirements(ctx context.Context, userClient ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	rawConfig, err := anexiatypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get anexia raw config: %w", err)
	}
	cpuReq, err := resource.ParseQuantity(fmt.Sprintf("%d", rawConfig.CPUs))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine cpu request to quantity: %w", err)
	}
	memReq, err := resource.ParseQuantity(fmt.Sprintf("%dM", rawConfig.Memory))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine memory request to quantity: %w", err)
	}
	storageReq, err := resource.ParseQuantity(fmt.Sprintf("%dG", rawConfig.DiskSize))
	if err != nil {
		return nil, fmt.Errorf("failed to parse machine storage request to quantity: %w", err)
	}

	return NewResourceDetails(cpuReq, memReq, storageReq), nil
}

func getPacketResourceRequirements(ctx context.Context, client ctrlruntimeclient.Client, config *providerconfig.Config) (*ResourceDetails, error) {
	configVarResolver := providerconfig.NewConfigVarResolver(ctx, client)
	rawConfig, err := equinixtypes.GetConfig(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to get packet raw config: %w", err)
	}

	token, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Token, envMetalToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of packet \"token\": %w", err)
	}

	projectID, err := configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.ProjectID, envMetalProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of packet \"projectID\": %w", err)
	}

	instanceType, err := configVarResolver.GetConfigVarStringValue(rawConfig.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get the value of packet \"instanceType\": %w", err)
	}

	capacity, err := packet.DescribeSize(ctx, token, projectID, instanceType)
	if err != nil {
		return nil, err
	}

	return NewResourceDetailsFromCapacity(capacity)
}
