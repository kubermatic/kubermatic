/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package operatingsystem

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/providerconfig/amzn2"
	"k8c.io/operating-system-manager/pkg/providerconfig/flatcar"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"
	"k8c.io/operating-system-manager/pkg/providerconfig/rockylinux"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"
)

func DefaultSpec(os providerconfig.OperatingSystem, cloudProvider kubermaticv1.ProviderType) (interface{}, error) {
	switch os {
	case providerconfig.OperatingSystemAmazonLinux2:
		return NewAmazonLinux2SpecBuilder(cloudProvider).Build(), nil
	case providerconfig.OperatingSystemFlatcar:
		return NewFlatcarSpecBuilder(cloudProvider).Build(), nil
	case providerconfig.OperatingSystemRHEL:
		return NewRHELSpecBuilder(cloudProvider).Build(), nil
	case providerconfig.OperatingSystemRockyLinux:
		return NewRockyLinuxSpecBuilder(cloudProvider).Build(), nil
	case providerconfig.OperatingSystemUbuntu:
		return NewUbuntuSpecBuilder(cloudProvider).Build(), nil
	default:
		return nil, fmt.Errorf("unknown operating system %q", os)
	}
}

type UbuntuSpecBuilder struct {
	ubuntu.Config
}

func NewUbuntuSpecBuilder(_ kubermaticv1.ProviderType) *UbuntuSpecBuilder {
	return &UbuntuSpecBuilder{}
}

func (b *UbuntuSpecBuilder) Build() ubuntu.Config {
	return b.Config
}

func (b *UbuntuSpecBuilder) WithDistUpgradeOnBoot(enable bool) *UbuntuSpecBuilder {
	b.DistUpgradeOnBoot = enable
	return b
}

type RockyLinuxSpecBuilder struct {
	rockylinux.Config
}

func (b *RockyLinuxSpecBuilder) Build() rockylinux.Config {
	return b.Config
}

func NewRockyLinuxSpecBuilder(_ kubermaticv1.ProviderType) *RockyLinuxSpecBuilder {
	return &RockyLinuxSpecBuilder{}
}

func (b *RockyLinuxSpecBuilder) WithDistUpgradeOnBoot(enable bool) *RockyLinuxSpecBuilder {
	b.DistUpgradeOnBoot = enable
	return b
}

type RHELSpecBuilder struct {
	rhel.Config
}

func (b *RHELSpecBuilder) Build() rhel.Config {
	return b.Config
}

func NewRHELSpecBuilder(_ kubermaticv1.ProviderType) *RHELSpecBuilder {
	return &RHELSpecBuilder{}
}

func (b *RHELSpecBuilder) WithDistUpgradeOnBoot(enable bool) *RHELSpecBuilder {
	b.DistUpgradeOnBoot = enable
	return b
}

func (b *RHELSpecBuilder) SetSubscriptionDetails(username, password, offlineToken string) *RHELSpecBuilder {
	b.RHELSubscriptionManagerUser = username
	b.RHELSubscriptionManagerPassword = password
	b.RHSMOfflineToken = offlineToken
	b.AttachSubscription = username != "" && password != ""
	return b
}

func (b *RHELSpecBuilder) WithPatch(patch func(*RHELSpecBuilder)) *RHELSpecBuilder {
	patch(b)
	return b
}

type AmazonLinux2SpecBuilder struct {
	amzn2.Config
}

func (b *AmazonLinux2SpecBuilder) Build() amzn2.Config {
	return b.Config
}

func NewAmazonLinux2SpecBuilder(_ kubermaticv1.ProviderType) *AmazonLinux2SpecBuilder {
	return &AmazonLinux2SpecBuilder{}
}

func (b *AmazonLinux2SpecBuilder) WithDistUpgradeOnBoot(enable bool) *AmazonLinux2SpecBuilder {
	b.DistUpgradeOnBoot = enable
	return b
}

type FlatcarSpecBuilder struct {
	flatcar.Config
}

func (b *FlatcarSpecBuilder) Build() flatcar.Config {
	return b.Config
}

func NewFlatcarSpecBuilder(provider kubermaticv1.ProviderType) *FlatcarSpecBuilder {
	builder := &FlatcarSpecBuilder{
		Config: flatcar.Config{
			// We manage Flatcar updates via the CoreOS update operator which requires locksmithd
			// to be disabled: https://github.com/coreos/container-linux-update-operator#design
			DisableLocksmithD:   true,
			DisableAutoUpdate:   true,
			ProvisioningUtility: flatcar.Ignition,
		},
	}

	if provider == kubermaticv1.AnexiaCloudProvider {
		builder.WithProvisioningUtility(flatcar.CloudInit)
	}

	return builder
}

func (b *FlatcarSpecBuilder) WithDisableAutoUpdate(disable bool) *FlatcarSpecBuilder {
	b.DisableAutoUpdate = disable
	return b
}

func (b *FlatcarSpecBuilder) WithDisableLocksmithD(disable bool) *FlatcarSpecBuilder {
	b.DisableLocksmithD = disable
	return b
}

func (b *FlatcarSpecBuilder) WithDisableUpdateEngine(disable bool) *FlatcarSpecBuilder {
	b.DisableUpdateEngine = disable
	return b
}

func (b *FlatcarSpecBuilder) WithProvisioningUtility(utility flatcar.ProvisioningUtility) *FlatcarSpecBuilder {
	b.ProvisioningUtility = utility
	return b
}
