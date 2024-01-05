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

OUTPUT_FILE=csi-azurefile.yaml
TEMP_FILE=upstream.yaml

default: $(TEMP_FILE) assemble cleanup

$(TEMP_FILE):
	touch $(TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v1.29.2/deploy/v1.29.2/csi-azurefile-controller.yaml -O - >> $(TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v1.29.2/deploy/v1.29.2/csi-azurefile-driver.yaml -O - >> $(TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v1.29.2/deploy/v1.29.2/csi-azurefile-node.yaml -O - >> $(TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v1.29.2/deploy/v1.29.2/rbac-csi-azurefile-controller.yaml -O - >> $(TEMP_FILE)
	wget -nv https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v1.29.2/deploy/v1.29.2/rbac-csi-azurefile-node.yaml -O - >> $(TEMP_FILE)

.PHONY: process
process:
	rudi -s process.rudi -o yamldocs --enable-funcs $(TEMP_FILE)

.PHONY: assemble
assemble:
	cat _header.txt > $(OUTPUT_FILE)
	rudi -s process.rudi -o yamldocs --enable-funcs $(TEMP_FILE) >> $(OUTPUT_FILE)
	cat _footer.txt >> $(OUTPUT_FILE)

.PHONY: cleanup
cleanup:
	rm -f $(TEMP_FILE)
