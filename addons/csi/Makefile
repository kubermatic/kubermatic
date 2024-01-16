# Copyright 2024 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

default: aws-ebs azure-disk azure-file azure-snapshot-controller digitalocean gcp hetzner openstack vsphere

.PHONY: aws-ebs
aws-ebs:
	make -C aws-ebs

.PHONY: azure-disk
azure-disk:
	make -C azure-disk

.PHONY: azure-file
azure-file:
	make -C azure-file

.PHONY: azure-snapshot-controller
azure-snapshot-controller:
	make -C azure-snapshot-controller

.PHONY: digitalocean
digitalocean:
	make -C digitalocean

.PHONY: gcp
gcp:
	make -C gcp

.PHONY: hetzner
hetzner:
	make -C hetzner

.PHONY: openstack
openstack:
	make -C openstack

.PHONY: vsphere
vsphere:
	make -C vsphere
