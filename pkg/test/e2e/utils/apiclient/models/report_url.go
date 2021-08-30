// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// ReportURL ReportURL represent an S3 pre signed URL to download a report
//
// swagger:model ReportURL
type ReportURL string

// Validate validates this report URL
func (m ReportURL) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this report URL based on context it is used
func (m ReportURL) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
