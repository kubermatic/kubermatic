package api

// GetToken returns digital ocean's token.
func (spec *DigitaloceanCloudSpec) GetToken() string {
	if spec == nil {
		return ""
	}
	return spec.Token
}

// GetDigitalocean returns the digitalocean cloud spec.
func (spec *CloudSpec) GetDigitalocean() *DigitaloceanCloudSpec {
	if spec == nil {
		return nil
	}
	return spec.Digitalocean
}

// GetRegion returns the region for the active cloud spec or empty string.
func (spec *CloudSpec) GetRegion() string {
	switch {
	case spec.Digitalocean != nil:
		return spec.Digitalocean.Region
	case spec.Fake != nil:
		return spec.Fake.Region
	case spec.Linode != nil:
		return "linode"
	}

	return ""
}
