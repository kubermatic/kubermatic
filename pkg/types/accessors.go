package types

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

// GetBringYourOwn returns the bringyourown cloud spec.
func (spec *CloudSpec) GetBringYourOwn() *BringYourOwnCloudSpec {
	if spec == nil {
		return nil
	}
	return spec.BringYourOwn
}

// GetAWS return the aws cloud spec.
func (spec *CloudSpec) GetAWS() *AWSCloudSpec {
	if spec == nil {
		return nil
	}
	return spec.AWS
}
