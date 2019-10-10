package resources

import (
	"fmt"
)

func cloudCredentialOperatorImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:0a0e07e408baea29d05812441657155086b7883863ce9084e4c5fcc6dbcef844", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:1071e05a8fd4e13630be9097f4e412356af481af4568bf241f208e337665864f", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterDnsOperatorImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:6a57cf4d52f44b3087fc91157186037975160e8e1aae8247b728eb8e744ef834", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:2aca09bcf2d705c8fe457e21507319550d4363fd07012db6403f59c314ecc7e0", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterImageRegistryOperatorImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:a46baddc04f823f4c89b6ec36ca71bd3b934b57d00feaf9de85750d9f6bdd51d", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:2fb3e2f3eb6dbc013dcd4f7b94f9a9cff5231d1005174a030e265899160efc68", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterNetworkOperatorImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:ddb3514fa03aa820fc0dd60875317c0f964884f5b06b0436dcefa22dce6c77d0", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:541465dbff9e28b303c533f5d86cea7a4f5ef1c920736a655380bb5b64954182", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func consoleImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:7e7ea54d76e07556e13c58d9e97f275a6b85ceba3a011f62e150118c77c3e645", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:d85be45a7acd5a84ce2e0ccd0d4e48b4b92e7c304b66ce1a7f0f64ce001d9bd7", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func deployerImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:0213a23caa39d575db4b2cb7d5adc5c907c71cf76f58ab537732509d160f80a9", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:8b946a142a8ba328ffe04195bb3fc4beeff26aaa4d8d0e99528340e8880eba7e", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func dockerBuilderImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:decbd4a5d1a1660f5cc541bfb2664a73689d6ba8215e2729111ce3d94315ccf2", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:f4d2df04a0ac1b689bc275c060e5520781f48f007dabf849d92cf1519f16ea82", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func hyperkubeImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:0d6ba37d0613ea9b0ee4a1b5b8e083f070fc8651eab09e266a50f6c66148dd72", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:155ef40a64608c946ca9ca0310bbf88f5a4664b2925502b3acac86847bc158e6", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func hypershiftImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:cc170b944f9a308cee401d75ecd0e047807d66e5ada8075d7b60f1b4b6106ff7", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func oauthProxyImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return openshiftImage + "@sha256:64279a4762f987de90db0310f285079247730c582e7af01b5cb2f0d70c1e8e60", nil
	case openshiftVersion419:
		return openshiftImage + "@sha256:1548a38e03059c6acd7eba3340b0ad3719f35a2295e5681c4051d561e52a70ed", nil
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}
