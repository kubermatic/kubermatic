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

package kubermaticmaster

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type storageClassFactory func(context.Context, *logrus.Entry, ctrlruntimeclient.Client, string) (storagev1.StorageClass, error)

var (
	storageClassFactories = map[string]storageClassFactory{
		"copy-default": func(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, name string) (storagev1.StorageClass, error) {
			s := &storagev1.StorageClass{
				Parameters: map[string]string{},
			}
			s.Name = name

			classes := storagev1.StorageClassList{}
			if err := kubeClient.List(ctx, &classes); err != nil {
				return *s, fmt.Errorf("cannot list existing StorageClasses: %v", err)
			}

			for _, class := range classes.Items {
				if isDefaultStorageClass(class) {
					logger.Infof("Duplicating existing default class %q…", class.Name)

					s = class.DeepCopy()
					s.Annotations = map[string]string{}
					s.Labels = map[string]string{}
					s.ResourceVersion = ""
					s.Name = name

					return *s, nil
				}
			}

			return *s, errors.New("cannot duplicate existing default StorageClass, because none of the classes are marked as the default")
		},
		"gce": func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, name string) (storagev1.StorageClass, error) {
			s := storagev1.StorageClass{
				Parameters: map[string]string{},
			}
			s.Name = name
			s.Provisioner = "kubernetes.io/gce-pd"
			s.Parameters["type"] = "pd-ssd"

			return s, nil
		},
		"aws": func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, name string) (storagev1.StorageClass, error) {
			s := storagev1.StorageClass{
				Parameters: map[string]string{},
			}
			s.Name = name
			s.Provisioner = "kubernetes.io/aws-ebs"
			s.Parameters["type"] = "gp2"

			return s, nil
		},
		"azure": func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, name string) (storagev1.StorageClass, error) {
			s := storagev1.StorageClass{
				Parameters: map[string]string{},
			}
			s.Name = name
			s.Provisioner = "kubernetes.io/azure-disk"
			s.Parameters["storageaccounttype"] = "Standard_LRS"
			s.Parameters["kind"] = "managed"

			return s, nil
		},
		"hetzner": func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, name string) (storagev1.StorageClass, error) {
			s := storagev1.StorageClass{
				Parameters: map[string]string{},
			}
			s.Name = name
			s.Provisioner = "csi.hetzner.cloud"

			return s, nil
		},
		"digitalocean": func(_ context.Context, _ *logrus.Entry, _ ctrlruntimeclient.Client, name string) (storagev1.StorageClass, error) {
			s := storagev1.StorageClass{
				Parameters: map[string]string{},
			}
			s.Name = name

			// see https://github.com/digitalocean/csi-digitalocean/blob/master/deploy/kubernetes/releases/csi-digitalocean-v1.3.0.yaml
			s.Provisioner = "dobs.csi.digitalocean.com"
			s.AllowVolumeExpansion = pointer.BoolPtr(true)

			return s, nil
		},
	}
)

func SupportedStorageClassProviders() sets.String {
	return sets.StringKeySet(storageClassFactories)
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
