package client

const (
	RancherKubernetesEngineConfigType                     = "rancherKubernetesEngineConfig"
	RancherKubernetesEngineConfigFieldAddonJobTimeout     = "addonJobTimeout"
	RancherKubernetesEngineConfigFieldAddons              = "addons"
	RancherKubernetesEngineConfigFieldAddonsInclude       = "addonsInclude"
	RancherKubernetesEngineConfigFieldAuthentication      = "authentication"
	RancherKubernetesEngineConfigFieldAuthorization       = "authorization"
	RancherKubernetesEngineConfigFieldBastionHost         = "bastionHost"
	RancherKubernetesEngineConfigFieldCloudProvider       = "cloudProvider"
	RancherKubernetesEngineConfigFieldClusterName         = "clusterName"
	RancherKubernetesEngineConfigFieldDNS                 = "dns"
	RancherKubernetesEngineConfigFieldIgnoreDockerVersion = "ignoreDockerVersion"
	RancherKubernetesEngineConfigFieldIngress             = "ingress"
	RancherKubernetesEngineConfigFieldMonitoring          = "monitoring"
	RancherKubernetesEngineConfigFieldNetwork             = "network"
	RancherKubernetesEngineConfigFieldNodes               = "nodes"
	RancherKubernetesEngineConfigFieldPrefixPath          = "prefixPath"
	RancherKubernetesEngineConfigFieldPrivateRegistries   = "privateRegistries"
	RancherKubernetesEngineConfigFieldRestore             = "restore"
	RancherKubernetesEngineConfigFieldRotateCertificates  = "rotateCertificates"
	RancherKubernetesEngineConfigFieldSSHAgentAuth        = "sshAgentAuth"
	RancherKubernetesEngineConfigFieldSSHCertPath         = "sshCertPath"
	RancherKubernetesEngineConfigFieldSSHKeyPath          = "sshKeyPath"
	RancherKubernetesEngineConfigFieldServices            = "services"
	RancherKubernetesEngineConfigFieldVersion             = "kubernetesVersion"
)

type RancherKubernetesEngineConfig struct {
	AddonJobTimeout     int64               `json:"addonJobTimeout,omitempty" yaml:"addonJobTimeout,omitempty"`
	Addons              string              `json:"addons,omitempty" yaml:"addons,omitempty"`
	AddonsInclude       []string            `json:"addonsInclude,omitempty" yaml:"addonsInclude,omitempty"`
	Authentication      *AuthnConfig        `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	Authorization       *AuthzConfig        `json:"authorization,omitempty" yaml:"authorization,omitempty"`
	BastionHost         *BastionHost        `json:"bastionHost,omitempty" yaml:"bastionHost,omitempty"`
	CloudProvider       *CloudProvider      `json:"cloudProvider,omitempty" yaml:"cloudProvider,omitempty"`
	ClusterName         string              `json:"clusterName,omitempty" yaml:"clusterName,omitempty"`
	DNS                 *DNSConfig          `json:"dns,omitempty" yaml:"dns,omitempty"`
	IgnoreDockerVersion bool                `json:"ignoreDockerVersion,omitempty" yaml:"ignoreDockerVersion,omitempty"`
	Ingress             *IngressConfig      `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	Monitoring          *MonitoringConfig   `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`
	Network             *NetworkConfig      `json:"network,omitempty" yaml:"network,omitempty"`
	Nodes               []RKEConfigNode     `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	PrefixPath          string              `json:"prefixPath,omitempty" yaml:"prefixPath,omitempty"`
	PrivateRegistries   []PrivateRegistry   `json:"privateRegistries,omitempty" yaml:"privateRegistries,omitempty"`
	Restore             *RestoreConfig      `json:"restore,omitempty" yaml:"restore,omitempty"`
	RotateCertificates  *RotateCertificates `json:"rotateCertificates,omitempty" yaml:"rotateCertificates,omitempty"`
	SSHAgentAuth        bool                `json:"sshAgentAuth,omitempty" yaml:"sshAgentAuth,omitempty"`
	SSHCertPath         string              `json:"sshCertPath,omitempty" yaml:"sshCertPath,omitempty"`
	SSHKeyPath          string              `json:"sshKeyPath,omitempty" yaml:"sshKeyPath,omitempty"`
	Services            *RKEConfigServices  `json:"services,omitempty" yaml:"services,omitempty"`
	Version             string              `json:"kubernetesVersion,omitempty" yaml:"kubernetesVersion,omitempty"`
}
