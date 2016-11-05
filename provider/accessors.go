package provider

// Region gets the Region from the AvailabilityZone.
func (a AWSSpec) Region() string {
	// Later add API query?
	return a.AvailabilityZone[:len(a.AvailabilityZone)-1]
}
