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
		versions:   []string{"1.10", "1.10.3"},
		pkg:        "docker.io",
		pkgVersion: "1.10.3-0ubuntu6",
	},
	{
		versions:   []string{"1.13", "1.13.1"},
		pkg:        "docker.io",
		pkgVersion: "1.13.1-0ubuntu1~16.04.2",
	},
	{
		versions:   []string{"17.03.0"},
		pkg:        "docker-ce",
		pkgVersion: "17.03.0~ce-0~ubuntu-xenial",
	},
	{
		versions:   []string{"17.03.1"},
		pkg:        "docker-ce",
		pkgVersion: "17.03.1~ce-0~ubuntu-xenial",
	},
	{
		versions:   []string{"17.03", "17.03.2"},
		pkg:        "docker-ce",
		pkgVersion: "17.03.2~ce-0~ubuntu-xenial",
	},
	{
		versions:   []string{"17.06.0"},
		pkg:        "docker-ce",
		pkgVersion: "17.06.0~ce-0~ubuntu",
	},
	{
		versions:   []string{"17.06.1"},
		pkg:        "docker-ce",
		pkgVersion: "17.06.1~ce-0~ubuntu",
	},
	{
		versions:   []string{"17.06", "17.06.2"},
		pkg:        "docker-ce",
		pkgVersion: "17.06.2~ce-0~ubuntu",
	},
	{
		versions:   []string{"17.09.0"},
		pkg:        "docker-ce",
		pkgVersion: "17.09.0~ce-0~ubuntu",
	},
	{
		versions:   []string{"17.09", "17.09.1"},
		pkg:        "docker-ce",
		pkgVersion: "17.09.1~ce-0~ubuntu",
	},
	{
		versions:   []string{"17.12", "17.12.0"},
		pkg:        "docker-ce",
		pkgVersion: "17.12.0~ce-0~ubuntu",
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
