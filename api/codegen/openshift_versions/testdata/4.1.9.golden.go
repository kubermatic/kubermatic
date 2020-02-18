package resources

import (
	"fmt"
)

func cliImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:52ef9f5ade93e32f85e13bb9f588b2e126717256789023f8eb455b1147761562", openshiftVersion, "cli", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func cloudCredentialOperatorImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:1071e05a8fd4e13630be9097f4e412356af481af4568bf241f208e337665864f", openshiftVersion, "cloud-credential-operator", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterDnsOperatorImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:2aca09bcf2d705c8fe457e21507319550d4363fd07012db6403f59c314ecc7e0", openshiftVersion, "cluster-dns-operator", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterImageRegistryOperatorImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:2fb3e2f3eb6dbc013dcd4f7b94f9a9cff5231d1005174a030e265899160efc68", openshiftVersion, "cluster-image-registry-operator", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterNetworkOperatorImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:541465dbff9e28b303c533f5d86cea7a4f5ef1c920736a655380bb5b64954182", openshiftVersion, "cluster-network-operator", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func consoleImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:9e554ac4505edd925eb73fec52e33d7418e2cfaf8058b59d8246ed478337748d", openshiftVersion, "console", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func containerNetworkingPluginsSupportedImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:473d03cbfa265d2a6def817f8ec5bd1c6536d3e39cf8c2f8223dd41ed2bd4541", openshiftVersion, "container-networking-plugins-supported", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func containerNetworkingPluginsUnsupportedImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:d7c6701150c7ad12fc6dd26f2c6b093da5e9e3b43dea89196a77da1c6ef6904b", openshiftVersion, "container-networking-plugins-unsupported", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func corednsImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:390cc1784aba986fad6315142d1d2524b2707a91eea3705d448367b51a112438", openshiftVersion, "coredns", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func deployerImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:8b946a142a8ba328ffe04195bb3fc4beeff26aaa4d8d0e99528340e8880eba7e", openshiftVersion, "deployer", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func dockerBuilderImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:f4d2df04a0ac1b689bc275c060e5520781f48f007dabf849d92cf1519f16ea82", openshiftVersion, "docker-builder", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func dockerRegistryImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:5c0b76746c2f86177b5a0fdce866cf41dbb752af58b96daa8fa7b033fa2c4fc9", openshiftVersion, "docker-registry", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func hyperkubeImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:155ef40a64608c946ca9ca0310bbf88f5a4664b2925502b3acac86847bc158e6", openshiftVersion, "hyperkube", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func hypershiftImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad", openshiftVersion, "hypershift", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func multusCniImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:6766e62f61307e7c5a187f61d33b99ba90390b2f43351f591bb8da951915ce04", openshiftVersion, "multus-cni", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func nodeImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:472dd90bc413a9bcb99be23f7296763468ebbeb985c10b26d1c44c4b04f57a77", openshiftVersion, "node", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func ovnKubernetesImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:81088a1f27ff88e7e4a65dd3ca47513aad76bfbfc44af359887baa1d3fa60eba", openshiftVersion, "ovn-kubernetes", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func sriovCniImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:9d332f4b42997f917fa7660d85975c579ee4abe354473acbd45fc2a093b12e3b", openshiftVersion, "sriov-cni", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func sriovNetworkDevicePluginImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:21c668c419662bf1a5c1f38d55f6ab20b4e22b807d076f927efb1ac954beed60", openshiftVersion, "sriov-network-device-plugin", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}
