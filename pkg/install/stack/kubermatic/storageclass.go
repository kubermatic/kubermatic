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

package kubermatic

import (
	storagev1 "k8s.io/api/storage/v1"
)

func newEKSStorageClass() storagev1.StorageClass {
	s := storagev1.StorageClass{
		Parameters: map[string]string{},
	}
	s.Provisioner = "kubernetes.io/aws-ebs"
	s.Parameters["type"] = "gp2"

	return s
}

func newGKEStorageClass() storagev1.StorageClass {
	s := storagev1.StorageClass{
		Parameters: map[string]string{},
	}
	s.Provisioner = "kubernetes.io/gce-pd"
	s.Parameters["type"] = "pd-ssd"

	return s
}

func newAKSStorageClass() storagev1.StorageClass {
	s := storagev1.StorageClass{
		Parameters: map[string]string{},
	}
	s.Provisioner = "kubernetes.io/azure-disk"
	s.Parameters["storageaccounttype"] = "Standard_LRS"
	s.Parameters["kind"] = "managed"

	return s
}

func storageClassForProvider(name string, p string) *storagev1.StorageClass {
	var sc storagev1.StorageClass

	switch p {
	case "aks":
		sc = newAKSStorageClass()
	case "eks":
		sc = newEKSStorageClass()
	case "gke":
		sc = newGKEStorageClass()
	default:
		return nil
	}

	sc.Name = name

	return &sc
}
