package resources

import (
	"fmt"
)

func cliImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:528f2ead3d1605bdf818579976d97df5dd86df0a2a5d80df9aa8209c82333a86", openshiftVersion, "cli", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:52ef9f5ade93e32f85e13bb9f588b2e126717256789023f8eb455b1147761562", openshiftVersion, "cli", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func cloudCredentialOperatorImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:0a0e07e408baea29d05812441657155086b7883863ce9084e4c5fcc6dbcef844", openshiftVersion, "cloud-credential-operator", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:1071e05a8fd4e13630be9097f4e412356af481af4568bf241f208e337665864f", openshiftVersion, "cloud-credential-operator", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterDnsOperatorImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:6a57cf4d52f44b3087fc91157186037975160e8e1aae8247b728eb8e744ef834", openshiftVersion, "cluster-dns-operator", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:2aca09bcf2d705c8fe457e21507319550d4363fd07012db6403f59c314ecc7e0", openshiftVersion, "cluster-dns-operator", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterImageRegistryOperatorImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:a46baddc04f823f4c89b6ec36ca71bd3b934b57d00feaf9de85750d9f6bdd51d", openshiftVersion, "cluster-image-registry-operator", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:2fb3e2f3eb6dbc013dcd4f7b94f9a9cff5231d1005174a030e265899160efc68", openshiftVersion, "cluster-image-registry-operator", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func clusterNetworkOperatorImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:ddb3514fa03aa820fc0dd60875317c0f964884f5b06b0436dcefa22dce6c77d0", openshiftVersion, "cluster-network-operator", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:541465dbff9e28b303c533f5d86cea7a4f5ef1c920736a655380bb5b64954182", openshiftVersion, "cluster-network-operator", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func consoleImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:aca4d430a0f53388614f46ce1f857f73f1c50337d8d96088356e2adfbacb6be1", openshiftVersion, "console", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:9e554ac4505edd925eb73fec52e33d7418e2cfaf8058b59d8246ed478337748d", openshiftVersion, "console", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func containerNetworkingPluginsSupportedImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:caf943952c2fac2718cd705ca6b682d6716add0355c07cb629b6cde0c0fa0aed", openshiftVersion, "container-networking-plugins-supported", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:473d03cbfa265d2a6def817f8ec5bd1c6536d3e39cf8c2f8223dd41ed2bd4541", openshiftVersion, "container-networking-plugins-supported", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func containerNetworkingPluginsUnsupportedImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:6da503d2c0cd4c57699b6539f7bc5a1ec111a76ce966c04fac88a1026367d673", openshiftVersion, "container-networking-plugins-unsupported", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:d7c6701150c7ad12fc6dd26f2c6b093da5e9e3b43dea89196a77da1c6ef6904b", openshiftVersion, "container-networking-plugins-unsupported", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func corednsImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:c9e82ff7f5744062c4d251ff89ef04f45d7b9dfa17ea01a1c4092c4e1fd2b541", openshiftVersion, "coredns", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:390cc1784aba986fad6315142d1d2524b2707a91eea3705d448367b51a112438", openshiftVersion, "coredns", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func deployerImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:0213a23caa39d575db4b2cb7d5adc5c907c71cf76f58ab537732509d160f80a9", openshiftVersion, "deployer", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:8b946a142a8ba328ffe04195bb3fc4beeff26aaa4d8d0e99528340e8880eba7e", openshiftVersion, "deployer", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func dockerBuilderImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:decbd4a5d1a1660f5cc541bfb2664a73689d6ba8215e2729111ce3d94315ccf2", openshiftVersion, "docker-builder", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:f4d2df04a0ac1b689bc275c060e5520781f48f007dabf849d92cf1519f16ea82", openshiftVersion, "docker-builder", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func dockerRegistryImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:171d5a868e31ebb3265dff9ca24ba5f447ac357037e781630eb21fe4aee7de27", openshiftVersion, "docker-registry", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:5c0b76746c2f86177b5a0fdce866cf41dbb752af58b96daa8fa7b033fa2c4fc9", openshiftVersion, "docker-registry", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func hyperkubeImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:0d6ba37d0613ea9b0ee4a1b5b8e083f070fc8651eab09e266a50f6c66148dd72", openshiftVersion, "hyperkube", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:155ef40a64608c946ca9ca0310bbf88f5a4664b2925502b3acac86847bc158e6", openshiftVersion, "hyperkube", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func hypershiftImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:cc170b944f9a308cee401d75ecd0e047807d66e5ada8075d7b60f1b4b6106ff7", openshiftVersion, "hypershift", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad", openshiftVersion, "hypershift", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func multusCniImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:6f4ffe4653d7c2b4f04146cc34e2d2b23499361570de053f15d6f25a11e09a1e", openshiftVersion, "multus-cni", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:6766e62f61307e7c5a187f61d33b99ba90390b2f43351f591bb8da951915ce04", openshiftVersion, "multus-cni", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func nodeImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:a92e3ae780e2eb2b69f57285611143e5c44d99bafbdfa0220e2780a7e1bc46a2", openshiftVersion, "node", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:472dd90bc413a9bcb99be23f7296763468ebbeb985c10b26d1c44c4b04f57a77", openshiftVersion, "node", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func ovnKubernetesImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:4dd49bb4921b802dfaf0ebee96682872e21c72895d57db8c3e94b03b63053942", openshiftVersion, "ovn-kubernetes", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:81088a1f27ff88e7e4a65dd3ca47513aad76bfbfc44af359887baa1d3fa60eba", openshiftVersion, "ovn-kubernetes", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func sriovCniImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:1b151df2ab3f9058ea3c006101a2a28468c6d8d39e3e4b78453ae96c51c404cf", openshiftVersion, "sriov-cni", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:9d332f4b42997f917fa7660d85975c579ee4abe354473acbd45fc2a093b12e3b", openshiftVersion, "sriov-cni", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}

func sriovNetworkDevicePluginImage(openshiftVersion, registry string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion4118:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:4493f9c75d46ecfcee81bbb0ca9c2579bac4ffb150b2255f8385eac50a885ca6", openshiftVersion, "sriov-network-device-plugin", registry)
	case openshiftVersion419:
		return OpenshiftImageWithRegistry(openshiftImage+"@sha256:21c668c419662bf1a5c1f38d55f6ab20b4e22b807d076f927efb1ac954beed60", openshiftVersion, "sriov-network-device-plugin", registry)
	default:
		return "", fmt.Errorf("no tag for openshiftVersion %q available", openshiftVersion)
	}
}
