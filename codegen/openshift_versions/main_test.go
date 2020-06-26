/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"testing"

	testhelper "github.com/kubermatic/kubermatic/api/pkg/test"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGenerateImageTagGetters(t *testing.T) {
	t.Parallel()
	in := `
Name:      4.1.9
Digest:    sha256:27fd24c705d1107cc73cb7dda8257fe97900e130b68afc314d0ef0e31bcf9b8e
Created:   2019-08-02T19:16:27Z
OS/Arch:   linux/amd64
Manifests: 288

Pull From: quay.io/openshift-release-dev/ocp-release@sha256:27fd24c705d1107cc73cb7dda8257fe97900e130b68afc314d0ef0e31bcf9b8e

Release Metadata:
  Version:  4.1.9
  Upgrades: 4.1.0, 4.1.1, 4.1.2, 4.1.3, 4.1.4, 4.1.6, 4.1.7, 4.1.8
  Metadata:
    description:
  Metadata:
    url: https://access.redhat.com/errata/RHBA-2019:2010-02

Component Versions:
  Kubernetes 1.13.4

Images:
  NAME                                          DIGEST
  aws-machine-controllers                       sha256:a44ad7f98e23aaff9447d668ca426851902577f73cd0e5101aa01c2d62ee5426
  azure-machine-controllers                     sha256:84ce455c18694b0376c1f070c2117992cc947cf714b755c701269da2a74db1ab
  baremetal-machine-controllers                 sha256:817fceb3b2e2e796be4c1a4c803c0b916dcc2315dc11d667ac607b5a9ea5547c
  cli                                           sha256:52ef9f5ade93e32f85e13bb9f588b2e126717256789023f8eb455b1147761562
  cli-artifacts                                 sha256:a7df9d0bd3016311899e8da569f41db8b2094831c9f4c4ce616b28c0d8e72c17
  cloud-credential-operator                     sha256:1071e05a8fd4e13630be9097f4e412356af481af4568bf241f208e337665864f
  cluster-authentication-operator               sha256:eab42d21effdd966ac7198e46f2553716bd75089f0c4311b94558d69cb3ba848
  cluster-autoscaler                            sha256:23c48a5576c50214d3814e66599082593006fb273201836bb8a94951b0d4fc1d
  cluster-autoscaler-operator                   sha256:cc6315f185b7f2d7b0fa6897cd04c26709627f025caa03140d21709b40b76826
  cluster-bootstrap                             sha256:6e15073f974a11c8fba123018136899a8fab93841e7075c88a70b6a20ed5f77a
  cluster-config-operator                       sha256:5cfc5c8c35cf0497870a8c353891851d8c1fcde33a2fbc94c8d4d439a63fa267
  cluster-dns-operator                          sha256:2aca09bcf2d705c8fe457e21507319550d4363fd07012db6403f59c314ecc7e0
  cluster-image-registry-operator               sha256:2fb3e2f3eb6dbc013dcd4f7b94f9a9cff5231d1005174a030e265899160efc68
  cluster-ingress-operator                      sha256:21bbccbdb421f1c3fb8e2bf4c8d7e36257d2dca5776808f72f732e66b2f9f7a3
  cluster-kube-apiserver-operator               sha256:322887be05c4032a6f41a3afa5a11886a8d170cc3b2190a6f2b0267afa7fbee1
  cluster-kube-controller-manager-operator      sha256:829235661429eaca37910ac342f3b12afcd561d8de63f2cc660fa6f9dabbda23
  cluster-kube-scheduler-operator               sha256:03068e746c5ce7556ad92b9b534e72229c6975e9680aeb2a72fe138e67ed14d0
  cluster-machine-approver                      sha256:a1a46823be725f41441281fe8e680782f775b79b30c159c38e25e24383f88586
  cluster-monitoring-operator                   sha256:116c12861e3a6cd602b70fcc4e56155ecb965d49e89266a99c229e55fc923b91
  cluster-network-operator                      sha256:541465dbff9e28b303c533f5d86cea7a4f5ef1c920736a655380bb5b64954182
  cluster-node-tuned                            sha256:0465224eb16d364fffc1bdb991f8b1957813f491f865469b8ab34520f2c964a4
  cluster-node-tuning-operator                  sha256:b243202bbff5dd1d7b1b880689bd85b5e4bc5ed9bfb22c343bf73544a378ef37
  cluster-openshift-apiserver-operator          sha256:4480dc9c78087ddc73a8c204b164fb6796b4dcb214be991eafca8c9d76af94d9
  cluster-openshift-controller-manager-operator sha256:1b59ea6ce2637df6fe205366e1aa3559986d2e312b2e1e65c5ae1b32efc07ef6
  cluster-samples-operator                      sha256:1edfaddf56851fe07bb8e2f206f3acae80d0353d5a19cd8201d1ee8208366710
  cluster-storage-operator                      sha256:c2d654de8042e59dc3b2011b3d70a7944d8632d8da777a64368c0a393719b87b
  cluster-svcat-apiserver-operator              sha256:586019607a6ac5c32b01efbd5d298ca2efa8bb13df9d8d1b08a17be4c10eabbb
  cluster-svcat-controller-manager-operator     sha256:d8f576494617d3af1226e95ff88e5d5ead61b2ddfc2de90736dde0e206b18e91
  cluster-update-keys                           sha256:63cdcff29dd038dcf2edb839ca0624e8c99eda4d472843c6beb6df8f0c839397
  cluster-version-operator                      sha256:7143faf98889a0fa54228a706a62d900700c12b5cf80efefab47a2d3aa675c72
  configmap-reloader                            sha256:d9ea877227b095b88e8926b455c3aaddd9151ff2ed2edf34dba079450fe71564
  console                                       sha256:9e554ac4505edd925eb73fec52e33d7418e2cfaf8058b59d8246ed478337748d
  console-operator                              sha256:d85be45a7acd5a84ce2e0ccd0d4e48b4b92e7c304b66ce1a7f0f64ce001d9bd7
  container-networking-plugins-supported        sha256:473d03cbfa265d2a6def817f8ec5bd1c6536d3e39cf8c2f8223dd41ed2bd4541
  container-networking-plugins-unsupported      sha256:d7c6701150c7ad12fc6dd26f2c6b093da5e9e3b43dea89196a77da1c6ef6904b
  coredns                                       sha256:390cc1784aba986fad6315142d1d2524b2707a91eea3705d448367b51a112438
  deployer                                      sha256:8b946a142a8ba328ffe04195bb3fc4beeff26aaa4d8d0e99528340e8880eba7e
  docker-builder                                sha256:f4d2df04a0ac1b689bc275c060e5520781f48f007dabf849d92cf1519f16ea82
  docker-registry                               sha256:5c0b76746c2f86177b5a0fdce866cf41dbb752af58b96daa8fa7b033fa2c4fc9
  etcd                                          sha256:6b7e8457de9fbad1415a73eabcfe7ca1ae20eb38dddbfd2d3079715cbf527623
  grafana                                       sha256:cacb1e44ca771c876319cbfc32ce6f392ff2496ef332e2257c34677e4024bc08
  haproxy-router                                sha256:78ad72095da80242181a0af0a2ca31c026dd01edb33ffac1fe830d88dac1dd69
  hyperkube                                     sha256:155ef40a64608c946ca9ca0310bbf88f5a4664b2925502b3acac86847bc158e6
  hypershift                                    sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad
  installer                                     sha256:527527544f773f00b2b057d0d0179ce99cfeb941707e834a6d96fd2bd976b45b
  installer-artifacts                           sha256:e2594cc9f7f10b73d6a7ea7c07cb031e32dcdcc7cb06cc96e81d35c95f33d214
  jenkins                                       sha256:bbf03e51fcb4eb6fb8e51aa77384377ebeeaefc8e0b1e1b51a957042abb7b6d9
  jenkins-agent-maven                           sha256:d40e3fae4b830b4f4f13fb14909bcfa56603ff95e8b095313475e34863c62b7b
  jenkins-agent-nodejs                          sha256:d1e48b1d29e620835d7cc57a86d9ef75316bd18ba7011a84cc76c74cbbb3399c
  k8s-prometheus-adapter                        sha256:af119469f5f786517eb357fcdc78291c71b6a572426f4334397e543ab6c10367
  kube-client-agent                             sha256:59a731fdbb6c36b02cf4f4068e2ee9052e7f79b7e7a8b09d6db156fb0b84fd85
  kube-etcd-signer-server                       sha256:fa076ffb8ac4b02cf4328c4d66d0f7a28f7c7be535749eabf5051da1650cb783
  kube-rbac-proxy                               sha256:fcd5e8287064b5c7340c8dab7814e080e3630d2254bd212b80462fef1283cfbc
  kube-state-metrics                            sha256:6d9cefdc8b0355defa17173da20a43abd20fc3f6451e2dca6513650b08289032
  libvirt-machine-controllers                   sha256:e5907cc1017fa9806f92ef47e9c88cbafe396acde6a53375af6a467dfc10c4e2
  machine-api-operator                          sha256:79519cf2f4245164ac38b0444b910a62f444296b1a800888f8294ffe8f30441f
  machine-config-controller                     sha256:85a62b31c2f6f24e39cc5fd649534fc6bfc8247be85855ccce4aad36b0b318c0
  machine-config-daemon                         sha256:c1c9e1d76a89b9108607e7ecd9c44d9ba94564b79a1e4957093880be17075af4
  machine-config-operator                       sha256:533a6c674fae6c4d45538d089a9bda5b5b34f67de3b984886d5f33e035bdbc26
  machine-config-server                         sha256:e036aab3e6fe57d2cc7451c02946426adf92e6ea2c622b6c2eadcee37d252440
  machine-os-content                            sha256:60d15a500766bb720cec7900da9b00d0ba68887087ec8242158a53f41cf19bc5
  multus-cni                                    sha256:6766e62f61307e7c5a187f61d33b99ba90390b2f43351f591bb8da951915ce04
  must-gather                                   sha256:3defe6cb85ca70789da67ea7dcf1d71038a595b8a8f06b96a65baa1eadb43677
  node                                          sha256:472dd90bc413a9bcb99be23f7296763468ebbeb985c10b26d1c44c4b04f57a77
  oauth-proxy                                   sha256:1548a38e03059c6acd7eba3340b0ad3719f35a2295e5681c4051d561e52a70ed
  openstack-machine-controllers                 sha256:0bb746ebce9b725b37e28d6850b99be3d55427ae33aebe879290cab3bb1a0146
  operator-lifecycle-manager                    sha256:e9ef9b6111e5c5f430c0cb0cce9dccc41d40525aee83598cd4272899ff80316c
  operator-marketplace                          sha256:86b3915e9ff7a94fcab1f5c0353caf0cfff1d56ec104cb78b3459d94e0d431ae
  operator-registry                             sha256:6961bcb2dd0e0240638251a2a6a5b6e39da39eea9081935b738613282645da13
  ovn-kubernetes                                sha256:81088a1f27ff88e7e4a65dd3ca47513aad76bfbfc44af359887baa1d3fa60eba
  pod                                           sha256:f64a0b025e2dfbb808028c70621295578bc47c3d07f40113a278ca76f47b3443
  prom-label-proxy                              sha256:a157e9f1551a4cc7ec0773f99f50dfb23be5643e4616f9de8f2312dedc6a3c08
  prometheus                                    sha256:5cc842ce3308bf0c1f81d0b945eea3986c1e1b11ad85166e60cb9f900a0cf552
  prometheus-alertmanager                       sha256:ec681e4125e3005ceadeafa4448c4f7383c4bb07159cd655caa33ba7ff4a3708
  prometheus-config-reloader                    sha256:a34393626cab69c7bc88dc22dd5851f49b72efae6a0c18da9ce2bafdb91a63ad
  prometheus-node-exporter                      sha256:ff8982824514711d392a6f5f49bb8d28772afbb887d59405ae0441303bdd9688
  prometheus-operator                           sha256:ae6fc57a7034d61fcb5fdc9b1bfa733fbf19ef1518c865e025b2157ea1965ca4
  service-ca-operator                           sha256:32fe6f395d639fdfd4c3759f38b64e43edea66a639fab1af9e6231ddabfd6c73
  service-catalog                               sha256:51f5e49be8f57dcd177477f3f63ba133f95ad51901639c584528ac24da04689a
  setup-etcd-environment                        sha256:0ea658a0c8d05e348185b5a144b83b001403d5dfcdd5f3ad49ef125f61f802a7
  sriov-cni                                     sha256:9d332f4b42997f917fa7660d85975c579ee4abe354473acbd45fc2a093b12e3b
  sriov-network-device-plugin                   sha256:21c668c419662bf1a5c1f38d55f6ab20b4e22b807d076f927efb1ac954beed60
  telemeter                                     sha256:df115119ac9c8f246c1d99c15230267634baeeab755794bf7d5f179e1e87bd96
  tests                                         sha256:e49f3186e7ecfc9ade794baca9b271b10959dd65fff883978439f2de2ae6b714
`
	result, err := generateImageTagGetters([]string{"4.1.9"}, testResolverFactory(in))
	if err != nil {
		t.Fatalf("failed calling generateImageTagGetters: %v", err)
	}

	testhelper.CompareOutput(t, "4.1.9", string(result), *update, ".go")
}

func testResolverFactory(res string) func(string) (string, error) {
	return func(_ string) (string, error) {
		return res, nil
	}
}
