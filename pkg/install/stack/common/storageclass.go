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

package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StorageClassName = "kubermatic-fast"

	// This is a special, custom fake provider that is only used during
	// the installation and is meant to signal that the installer should
	// copy whatever default StorageClass exists.
	CopyDefaultCloudProvider kubermaticv1.ProviderType = "copy-default"
)

var (
	preferredCSIDrivers = map[string]kubermaticv1.ProviderType{
		"ebs.csi.aws.com":                          kubermaticv1.AWSCloudProvider,
		"disk.csi.azure.com":                       kubermaticv1.AzureCloudProvider,
		"dobs.csi.digitalocean.com":                kubermaticv1.DigitaloceanCloudProvider,
		"pd.csi.storage.gke.io":                    kubermaticv1.GCPCloudProvider,
		"csi.hetzner.cloud":                        kubermaticv1.HetznerCloudProvider,
		"cinder.csi.openstack.org":                 kubermaticv1.OpenstackCloudProvider,
		"named-disk.csi.cloud-director.vmware.com": kubermaticv1.VMwareCloudDirectorCloudProvider,
		"csi.vsphere.vmware.com":                   kubermaticv1.VSphereCloudProvider,
	}

	waitForFirstCustomer = storagev1.VolumeBindingWaitForFirstConsumer
)

func GetPreferredCSIDriver(ctx context.Context, kubeClient ctrlruntimeclient.Client) (string, kubermaticv1.ProviderType, error) {
	csiDrivers := storagev1.CSIDriverList{}
	if err := kubeClient.List(ctx, &csiDrivers); err != nil {
		return "", "", fmt.Errorf("failed to list CSIDrivers: %w", err)
	}

	for _, driver := range csiDrivers.Items {
		for name, provider := range preferredCSIDrivers {
			if name == driver.Name {
				return name, provider, nil
			}
		}
	}

	return "", "", nil
}

type StorageClassFactory func(context.Context, *logrus.Entry, ctrlruntimeclient.Client, *storagev1.StorageClass, string) error

var (
	storageClassFactories = map[kubermaticv1.ProviderType]StorageClassFactory{
		CopyDefaultCloudProvider: func(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, sc *storagev1.StorageClass, _ string) error {
			classes := storagev1.StorageClassList{}
			if err := kubeClient.List(ctx, &classes); err != nil {
				return fmt.Errorf("cannot list existing StorageClasses: %w", err)
			}

			for _, class := range classes.Items {
				if isDefaultStorageClass(class) {
					logger.Infof("Duplicating existing default class %q…", class.Name)

					sc.Annotations = class.Annotations
					sc.Labels = class.Labels
					sc.Provisioner = class.Provisioner
					sc.Parameters = class.Parameters
					sc.ReclaimPolicy = class.ReclaimPolicy
					sc.MountOptions = class.MountOptions
					sc.AllowVolumeExpansion = class.AllowVolumeExpansion
					sc.VolumeBindingMode = class.VolumeBindingMode
					sc.AllowedTopologies = class.AllowedTopologies

					delete(sc.Annotations, "storageclass.kubernetes.io/is-default-class")
					delete(sc.Annotations, "storageclass.beta.kubernetes.io/is-default-class")

					return nil
				}
			}

			return errors.New("cannot duplicate existing default StorageClass, because none of the classes are marked as the default")
		},
		kubermaticv1.GCPCloudProvider: func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, sc *storagev1.StorageClass, csiDriverName string) error {
			if csiDriverName == "" { // = in-tree CSI
				sc.Provisioner = "kubernetes.io/gce-pd"
			} else { // out-of-tree CSI
				sc.Provisioner = csiDriverName
			}

			sc.Parameters["type"] = "pd-ssd"
			sc.VolumeBindingMode = &waitForFirstCustomer

			return nil
		},
		kubermaticv1.AWSCloudProvider: func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, sc *storagev1.StorageClass, csiDriverName string) error {
			if csiDriverName == "" { // = in-tree CSI
				sc.Provisioner = "kubernetes.io/aws-ebs"
			} else { // out-of-tree CSI
				sc.Provisioner = csiDriverName
			}

			sc.Parameters["type"] = "io1"
			sc.Parameters["iopsPerGB"] = "25"
			sc.VolumeBindingMode = &waitForFirstCustomer

			return nil
		},
		kubermaticv1.AzureCloudProvider: func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, sc *storagev1.StorageClass, csiDriverName string) error {
			if csiDriverName == "" { // = in-tree CSI
				sc.Provisioner = "kubernetes.io/azure-disk"
			} else { // out-of-tree CSI
				sc.Provisioner = csiDriverName
				sc.AllowVolumeExpansion = ptr.To(true)
			}

			sc.VolumeBindingMode = &waitForFirstCustomer
			sc.Parameters["storageaccounttype"] = "Standard_LRS"
			sc.Parameters["kind"] = "managed"

			return nil
		},
		kubermaticv1.HetznerCloudProvider: func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, sc *storagev1.StorageClass, csiDriverName string) error {
			if csiDriverName == "" {
				return errors.New("only out-of-tree CSIDriver is supported for this provider")
			}

			sc.Provisioner = csiDriverName
			sc.VolumeBindingMode = &waitForFirstCustomer
			sc.AllowVolumeExpansion = ptr.To(true)

			return nil
		},
		kubermaticv1.DigitaloceanCloudProvider: func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, sc *storagev1.StorageClass, csiDriverName string) error {
			if csiDriverName == "" {
				return errors.New("only out-of-tree CSIDriver is supported for this provider")
			}

			sc.Provisioner = csiDriverName
			sc.AllowVolumeExpansion = ptr.To(true)
			sc.VolumeBindingMode = &waitForFirstCustomer

			return nil
		},
		kubermaticv1.OpenstackCloudProvider: func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, sc *storagev1.StorageClass, csiDriverName string) error {
			if csiDriverName == "" { // = in-tree CSI
				sc.Provisioner = "kubernetes.io/cinder"
			} else { // out-of-tree CSI
				sc.Provisioner = csiDriverName
				sc.AllowVolumeExpansion = ptr.To(true)
				sc.VolumeBindingMode = &waitForFirstCustomer
			}

			return nil
		},
		kubermaticv1.VSphereCloudProvider: func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, sc *storagev1.StorageClass, csiDriverName string) error {
			if csiDriverName == "" { // = in-tree CSI
				sc.Provisioner = "kubernetes.io/vsphere-volume"
				sc.Parameters["diskformat"] = "thin"
			} else { // out-of-tree CSI
				sc.Provisioner = csiDriverName
				sc.VolumeBindingMode = &waitForFirstCustomer
			}

			return nil
		},
		kubermaticv1.VMwareCloudDirectorCloudProvider: func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, sc *storagev1.StorageClass, csiDriverName string) error {
			if csiDriverName == "" {
				return errors.New("only out-of-tree CSIDriver is supported for this provider")
			}

			sc.Provisioner = csiDriverName
			sc.AllowVolumeExpansion = ptr.To(true)
			sc.Parameters["filesystem"] = "ext4"

			return nil
		},
	}
)

func StorageClassCreator(provider kubermaticv1.ProviderType) (StorageClassFactory, error) {
	factory, ok := storageClassFactories[provider]
	if !ok {
		return nil, fmt.Errorf("unknown StorageClass provider %q", provider)
	}

	return factory, nil
}

func SupportedStorageClassProviders() sets.Set[string] {
	all := sets.New[string]()
	for k := range storageClassFactories {
		all.Insert(string(k))
	}

	return all
}

func isDefaultStorageClass(sc storagev1.StorageClass) bool {
	if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
		return true
	}
	if sc.Annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true" {
		return true
	}

	return false
}
