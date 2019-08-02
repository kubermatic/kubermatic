package models

import (
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// KubevirtCloudSpec KubevirtCloudSpec specifies access data to a Kubevirt cloud.
// swagger:model KubevirtCloudSpec
type KubevirtCloudSpec struct {
	Config string `json:"config,omitempty"`
}

// Validate validates this kubevirt cloud spec
func (m *KubevirtCloudSpec) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *KubevirtCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *KubevirtCloudSpec) UnmarshalBinary(b []byte) error {
	var res KubevirtCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
