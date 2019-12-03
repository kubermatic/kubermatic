package client

const (
	LocalConfigType                     = "localConfig"
	LocalConfigFieldAccessMode          = "accessMode"
	LocalConfigFieldAllowedPrincipalIDs = "allowedPrincipalIds"
	LocalConfigFieldAnnotations         = "annotations"
	LocalConfigFieldCreated             = "created"
	LocalConfigFieldCreatorID           = "creatorId"
	LocalConfigFieldEnabled             = "enabled"
	LocalConfigFieldLabels              = "labels"
	LocalConfigFieldName                = "name"
	LocalConfigFieldOwnerReferences     = "ownerReferences"
	LocalConfigFieldRemoved             = "removed"
	LocalConfigFieldType                = "type"
	LocalConfigFieldUUID                = "uuid"
)

type LocalConfig struct {
	AccessMode          string            `json:"accessMode,omitempty" yaml:"accessMode,omitempty"`
	AllowedPrincipalIDs []string          `json:"allowedPrincipalIds,omitempty" yaml:"allowedPrincipalIds,omitempty"`
	Annotations         map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Created             string            `json:"created,omitempty" yaml:"created,omitempty"`
	CreatorID           string            `json:"creatorId,omitempty" yaml:"creatorId,omitempty"`
	Enabled             bool              `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Labels              map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Name                string            `json:"name,omitempty" yaml:"name,omitempty"`
	OwnerReferences     []OwnerReference  `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
	Removed             string            `json:"removed,omitempty" yaml:"removed,omitempty"`
	Type                string            `json:"type,omitempty" yaml:"type,omitempty"`
	UUID                string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}
