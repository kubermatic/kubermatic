package common

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

// UIVERSION is a magic variable containing the tag / git commit hash
// of the dashboard-v2 Docker repository to deploy. It gets fed by the
// Makefile as an ldflag.
var UIVERSION string

type Versions struct {
	Kubermatic string
	UI         string
}

func NewDefaultVersions() Versions {
	return Versions{
		Kubermatic: resources.KUBERMATICCOMMIT,
		UI:         UIVERSION,
	}
}
