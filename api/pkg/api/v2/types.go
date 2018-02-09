package v2

import "time"

// ObjectMeta is an object storing common metadata for persistable objects.
// swagger:model ObjectMetaV2
type ObjectMeta struct {
	//The unique name
	Name string `json:"name"`
	//The name to display in the frontend
	DisplayName string `json:"displayName"`

	//Tells that the resource is being deleted
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}
