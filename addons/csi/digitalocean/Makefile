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

$(DRIVER_OUTPUT_FILE): $(DRIVER_TEMP_FILE) assemble-csi-driver

$(DRIVER_TEMP_FILE):
	touch $(DRIVER_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/digitalocean/csi-digitalocean/master/deploy/kubernetes/releases/csi-digitalocean-v4.7.1/driver.yaml -O - >> $(DRIVER_TEMP_FILE)

.PHONY: assemble-csi-driver
assemble-csi-driver:
	cat _header.txt > $(DRIVER_OUTPUT_FILE)
	rudi -l ../../library.rudi -s csi-driver.rudi -o yamldocs --enable-funcs $(DRIVER_TEMP_FILE) >> $(DRIVER_OUTPUT_FILE)
	cat _footer.txt >> $(DRIVER_OUTPUT_FILE)

$(CTRL_OUTPUT_FILE): $(CTRL_TEMP_FILE) assemble-snapshot-controller

$(CTRL_TEMP_FILE):
	touch $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/digitalocean/csi-digitalocean/master/deploy/kubernetes/releases/csi-digitalocean-v4.7.1/crds.yaml -O - >> $(CTRL_TEMP_FILE)
	echo "---" >> $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/digitalocean/csi-digitalocean/master/deploy/kubernetes/releases/csi-digitalocean-v4.7.1/snapshot-controller.yaml -O - >> $(CTRL_TEMP_FILE)
	echo "---" >> $(CTRL_TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/digitalocean/csi-digitalocean/master/deploy/kubernetes/releases/csi-digitalocean-v4.7.1/snapshot-validation-webhook.yaml -O - >> $(CTRL_TEMP_FILE)

.PHONY: assemble-snapshot-controller
assemble-snapshot-controller:
	cat _header.txt > $(CTRL_OUTPUT_FILE)
	rudi -l ../../library.rudi -s snapshot-controller.rudi -o yamldocs --enable-funcs $(CTRL_TEMP_FILE) >> $(CTRL_OUTPUT_FILE)
	cat _footer.txt >> $(CTRL_OUTPUT_FILE)

.PHONY: cleanup
cleanup:
	rm -f $(DRIVER_TEMP_FILE) $(CTRL_TEMP_FILE)
