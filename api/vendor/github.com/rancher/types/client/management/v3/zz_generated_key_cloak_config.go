package client

const (
	KeyCloakConfigType                     = "keyCloakConfig"
	KeyCloakConfigFieldAccessMode          = "accessMode"
	KeyCloakConfigFieldAllowedPrincipalIDs = "allowedPrincipalIds"
	KeyCloakConfigFieldAnnotations         = "annotations"
	KeyCloakConfigFieldCreated             = "created"
	KeyCloakConfigFieldCreatorID           = "creatorId"
	KeyCloakConfigFieldDisplayNameField    = "displayNameField"
	KeyCloakConfigFieldEnabled             = "enabled"
	KeyCloakConfigFieldGroupsField         = "groupsField"
	KeyCloakConfigFieldIDPMetadataContent  = "idpMetadataContent"
	KeyCloakConfigFieldLabels              = "labels"
	KeyCloakConfigFieldName                = "name"
	KeyCloakConfigFieldOwnerReferences     = "ownerReferences"
	KeyCloakConfigFieldRancherAPIHost      = "rancherApiHost"
	KeyCloakConfigFieldRemoved             = "removed"
	KeyCloakConfigFieldSpCert              = "spCert"
	KeyCloakConfigFieldSpKey               = "spKey"
	KeyCloakConfigFieldType                = "type"
	KeyCloakConfigFieldUIDField            = "uidField"
	KeyCloakConfigFieldUUID                = "uuid"
	KeyCloakConfigFieldUserNameField       = "userNameField"
)

type KeyCloakConfig struct {
	AccessMode          string            `json:"accessMode,omitempty" yaml:"accessMode,omitempty"`
	AllowedPrincipalIDs []string          `json:"allowedPrincipalIds,omitempty" yaml:"allowedPrincipalIds,omitempty"`
	Annotations         map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Created             string            `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID           string            `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	DisplayNameField    string            `json:"displayNameField,omitempty" yaml:"displayNameField,omitempty"`
	Enabled             bool              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	GroupsField         string            `json:"groupsField,omitempty" yaml:"groupsField,omitempty"`
	IDPMetadataContent  string            `json:"idpMetadataContent,omitempty" yaml:"idpMetadataContent,omitempty"`
	Labels              map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Name                string            `json:"name,omitempty" yaml:"name,omitempty"`
	OwnerReferences     []OwnerReference  `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
	RancherAPIHost      string            `json:"rancherApiHost,omitempty" yaml:"rancherApiHost,omitempty"`
	Removed             string            `json:"removed,omitempty" yaml:"removed,omitempty"`
	SpCert              string            `json:"spCert,omitempty" yaml:"spCert,omitempty"`
	SpKey               string            `json:"spKey,omitempty" yaml:"spKey,omitempty"`
	Type                string            `json:"type,omitempty" yaml:"type,omitempty"`
	UIDField            string            `json:"uidField,omitempty" yaml:"uidField,omitempty"`
	UUID                string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	UserNameField       string            `json:"userNameField,omitempty" yaml:"userNameField,omitempty"`
}
