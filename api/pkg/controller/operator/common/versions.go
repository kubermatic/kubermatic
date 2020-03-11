package common

// UIDOCKERTAG is a magic variable containing the tag / git commit hash
// of the dashboard-v2 Docker image to deploy. It gets fed by the
// Makefile as an ldflag.
var UIDOCKERTAG string

// KUBERMATICDOCKERTAG is a magic variable containing the tag / git commit hash
// of the kubermatic Docker image to deploy. It gets fed by the
// Makefile as an ldflag.
var KUBERMATICDOCKERTAG string

type Versions struct {
	Kubermatic string
	UI         string
	VPA        string
	Envoy      string
}

func NewDefaultVersions() Versions {
	return Versions{
		Kubermatic: KUBERMATICDOCKERTAG,
		UI:         UIDOCKERTAG,
		VPA:        "0.5.0",
		Envoy:      "v1.13.0",
	}
}
