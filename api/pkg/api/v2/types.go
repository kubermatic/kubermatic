package v2

import "time"

// ObjectMeta is an object storing common metadata for persistable objects.
// swagger:model ObjectMetaV2
type ObjectMeta struct {
	// The unique name
	Name string `json:"name"`
	// The name to display in the frontend
	DisplayName string `json:"displayName"`

	// DeletionTimestamp is a timestamp representing the server time when this object was deleted.
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"`

	// CreationTimestamp is a timestamp representing the server time when this object was created.
	CreationTimestamp time.Time `json:"creationTimestamp,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// NewSSHKey represents a ssh key
// swagger:model NewSSHKey
type NewSSHKey struct {
	Name              string    `json:"name"`
	DisplayName       string    `json:"displayName"`
	CreationTimestamp time.Time `json:"creationTimestamp,omitempty"`
	Fingerprint       string    `json:"fingerprint"`
	PublicKey         string    `json:"publicKey"`
}

// NewSSHKeyList represents a list of NewSSHKey
// swagger:model NewSSHKeyList
type NewSSHKeyList []NewSSHKey
