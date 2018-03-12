package ubuntu

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

var crioInstallCandidates = []installCandidate{
	{
		versions:   []string{"1.9"},
		pkg:        "cri-o-1.9",
		pkgVersion: "",
	},
}

func getCRIOInstallCandidate(desiredVersion string) (pkg string, version string, err error) {
	for _, ic := range crioInstallCandidates {
		if sets.NewString(ic.versions...).Has(desiredVersion) {
			return ic.pkg, ic.pkgVersion, nil
		}
	}

	return "", "", NoInstallCandidateAvailableErr
}
