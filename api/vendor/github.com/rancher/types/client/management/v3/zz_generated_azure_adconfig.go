package client

const (
	AzureADConfigType                     = "azureADConfig"
	AzureADConfigFieldAccessMode          = "accessMode"
	AzureADConfigFieldAllowedPrincipalIDs = "allowedPrincipalIds"
	AzureADConfigFieldAnnotations         = "annotations"
	AzureADConfigFieldApplicationID       = "applicationId"
	AzureADConfigFieldApplicationSecret   = "applicationSecret"
	AzureADConfigFieldAuthEndpoint        = "authEndpoint"
	AzureADConfigFieldCreated             = "created"
	AzureADConfigFieldCreatorID           = "creatorId"
	AzureADConfigFieldEnabled             = "enabled"
	AzureADConfigFieldEndpoint            = "endpoint"
	AzureADConfigFieldGraphEndpoint       = "graphEndpoint"
	AzureADConfigFieldLabels              = "labels"
	AzureADConfigFieldName                = "name"
	AzureADConfigFieldOwnerReferences     = "ownerReferences"
	AzureADConfigFieldRancherURL          = "rancherUrl"
	AzureADConfigFieldRemoved             = "removed"
	AzureADConfigFieldTenantID            = "tenantId"
	AzureADConfigFieldTokenEndpoint       = "tokenEndpoint"
	AzureADConfigFieldType                = "type"
	AzureADConfigFieldUUID                = "uuid"
)

type AzureADConfig struct {
	AccessMode          string            `json:"accessMode,omitempty" yaml:"accessMode,omitempty"`
	AllowedPrincipalIDs []string          `json:"allowedPrincipalIds,omitempty" yaml:"allowedPrincipalIds,omitempty"`
	Annotations         map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	ApplicationID       string            `json:"applicationId,omitempty" yaml:"applicationId,omitempty"`
	ApplicationSecret   string            `json:"applicationSecret,omitempty" yaml:"applicationSecret,omitempty"`
	AuthEndpoint        string            `json:"authEndpoint,omitempty" yaml:"authEndpoint,omitempty"`
	Created             string            `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID           string            `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	Enabled             bool              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Endpoint            string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	GraphEndpoint       string            `json:"graphEndpoint,omitempty" yaml:"graphEndpoint,omitempty"`
	Labels              map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Name                string            `json:"name,omitempty" yaml:"name,omitempty"`
	OwnerReferences     []OwnerReference  `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
	RancherURL          string            `json:"rancherUrl,omitempty" yaml:"rancherUrl,omitempty"`
	Removed             string            `json:"removed,omitempty" yaml:"removed,omitempty"`
	TenantID            string            `json:"tenantId,omitempty" yaml:"tenantId,omitempty"`
	TokenEndpoint       string            `json:"tokenEndpoint,omitempty" yaml:"tokenEndpoint,omitempty"`
	Type                string            `json:"type,omitempty" yaml:"type,omitempty"`
	UUID                string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}
