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

package types

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

var AllProviders = sets.New(
	string(kubermaticv1.AWSCloudProvider),
	string(kubermaticv1.AlibabaCloudProvider),
	string(kubermaticv1.AnexiaCloudProvider),
	string(kubermaticv1.AzureCloudProvider),
	string(kubermaticv1.DigitaloceanCloudProvider),
	string(kubermaticv1.GCPCloudProvider),
	string(kubermaticv1.HetznerCloudProvider),
	string(kubermaticv1.KubevirtCloudProvider),
	string(kubermaticv1.NutanixCloudProvider),
	string(kubermaticv1.OpenstackCloudProvider),
	string(kubermaticv1.PacketCloudProvider),
	string(kubermaticv1.VMwareCloudDirectorCloudProvider),
	string(kubermaticv1.VSphereCloudProvider),
)
