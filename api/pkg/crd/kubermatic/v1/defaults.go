package v1

// DefaultNodeAccessNetwork is the default CIDR used for the VPNs
// transit network trough which we route the ControlPlane -> Node/Pod traffic
const DefaultNodeAccessNetwork = "10.254.0.0/16"
