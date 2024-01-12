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

DRIVER_OUTPUT_FILE=csi-driver.yaml
DRIVER_TEMP_FILE=csi-driver-upstream.yaml

CTRL_OUTPUT_FILE=snapshot-controller.yaml
CTRL_TEMP_FILE=snapshot-controller-upstream.yaml

default: $(DRIVER_OUTPUT_FILE) $(CTRL_OUTPUT_FILE) cleanup

.PHONY: cleanup
cleanup:
	rm -f $(DRIVER_TEMP_FILE) $(CTRL_TEMP_FILE)

##
## CSI Driver
##

$(DRIVER_OUTPUT_FILE): $(DRIVER_TEMP_FILE) assemble-csi-driver

$(DRIVER_TEMP_FILE):
	touch $(DRIVER_TEMP_FILE)
	wget -nv https://github.com/kubernetes-sigs/vsphere-csi-driver/raw/v3.0.2/manifests/vanilla/vsphere-csi-driver.yaml -O - >> $(DRIVER_TEMP_FILE)

.PHONY: assemble-csi-driver
assemble-csi-driver:
	cat _header.txt > $(DRIVER_OUTPUT_FILE)
	rudi -l ../../library.rudi -s csi-driver.rudi -o yamldocs --enable-funcs $(DRIVER_TEMP_FILE) >> $(DRIVER_OUTPUT_FILE)
	cat _footer.txt >> $(DRIVER_OUTPUT_FILE)

##
## Snapshot Controller
##

$(CTRL_OUTPUT_FILE): assemble-snapshot-controller

$(CTRL_TEMP_FILE):
	wget -nv https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/client/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml -O - >> $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/client/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml -O - >> $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/client/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml -O - >> $(CTRL_TEMP_FILE)
	echo "---" >> $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/deploy/kubernetes/snapshot-controller/rbac-snapshot-controller.yaml -O - >> $(CTRL_TEMP_FILE)
	echo "---" >> $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml -O - >> $(CTRL_TEMP_FILE)
	echo "---" >> $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/deploy/kubernetes/webhook-example/webhook.yaml -O - >> $(CTRL_TEMP_FILE)
	echo "---" >> $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/deploy/kubernetes/webhook-example/rbac-snapshot-webhook.yaml -O - >> $(CTRL_TEMP_FILE)

.PHONY: assemble-snapshot-controller
assemble-snapshot-controller: $(CTRL_TEMP_FILE)
	cat _header.txt > $(CTRL_OUTPUT_FILE)
	rudi -l ../../library.rudi -s snapshot-controller.rudi -o yamldocs --enable-funcs $(CTRL_TEMP_FILE) >> $(CTRL_OUTPUT_FILE)
	cat _footer.txt >> $(CTRL_OUTPUT_FILE)
