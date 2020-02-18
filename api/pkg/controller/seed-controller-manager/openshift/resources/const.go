package resources

const (
	openshiftImage       = "quay.io/openshift-release-dev/ocp-v4.0-art-dev"
	openshiftVersion419  = "4.1.9"
	openshiftVersion4118 = "4.1.18"
)

//go:generate go run ../../../../../codegen/openshift_versions/main.go
