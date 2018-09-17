package ubuntu

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

type installCandidate struct {
	versions   []string
	pkgVersion string
	pkg        string
}

var dockerInstallCandidates = []installCandidate{
	{
		versions:   []string{"17.12", "17.12.1"},
		pkg:        "docker.io",
		pkgVersion: "17.12.1-0ubuntu1",
	},
	{
		versions:   []string{"18.03", "18.03.1"},
		pkg:        "docker-ce",
		pkgVersion: "18.03.1~ce~3-0~ubuntu",
	},
	{
		versions:   []string{"18.06.0"},
		pkg:        "docker-ce",
		pkgVersion: "18.06.0~ce~3-0~ubuntu",
	},
	{
		versions:   []string{"18.06", "18.06.1"},
		pkg:        "docker-ce",
		pkgVersion: "18.06.1~ce~3-0~ubuntu",
	},
}

func getDockerInstallCandidate(desiredVersion string) (pkg string, version string, err error) {
	for _, ic := range dockerInstallCandidates {
		if sets.NewString(ic.versions...).Has(desiredVersion) {
			return ic.pkg, ic.pkgVersion, nil
		}
	}

	return "", "", errNoInstallCandidateAvailable
}
