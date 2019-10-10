package resources

import (
	"fmt"
)

func cloudCredentialOperatorImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:1071e05a8fd4e13630be9097f4e412356af481af4568bf241f208e337665864f", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterDnsOperatorImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:2aca09bcf2d705c8fe457e21507319550d4363fd07012db6403f59c314ecc7e0", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterImageRegistryOperatorImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:2fb3e2f3eb6dbc013dcd4f7b94f9a9cff5231d1005174a030e265899160efc68", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterNetworkOperatorImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:541465dbff9e28b303c533f5d86cea7a4f5ef1c920736a655380bb5b64954182", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func consoleImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:d85be45a7acd5a84ce2e0ccd0d4e48b4b92e7c304b66ce1a7f0f64ce001d9bd7", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func deployerImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:8b946a142a8ba328ffe04195bb3fc4beeff26aaa4d8d0e99528340e8880eba7e", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func dockerBuilderImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:f4d2df04a0ac1b689bc275c060e5520781f48f007dabf849d92cf1519f16ea82", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func hyperkubeImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:155ef40a64608c946ca9ca0310bbf88f5a4664b2925502b3acac86847bc158e6", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func hypershiftImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func oauthProxyImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return openshiftImage + "@sha256:1548a38e03059c6acd7eba3340b0ad3719f35a2295e5681c4051d561e52a70ed", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}
