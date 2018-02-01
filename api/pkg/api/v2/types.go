package v2

// ObjectMeta is an object storing common metadata for persistable objects.
// swagger:model ObjectMetaV2
type ObjectMeta struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`

	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}
