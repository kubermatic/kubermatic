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

# https://github.com/kubernetes-sigs/azurefile-csi-driver/tree/master/charts

OUTPUT_FILE = driver.yaml
REPO_NAME = kkp-addons-csi-azure-file
CHART_VERSION = 1.35.0

.PHONY: default
default: setup-helm build clean-helm

.PHONY: build
build:
	helm --namespace kube-system template azurefile-csi-driver $(REPO_NAME)/azurefile-csi-driver \
	  --version $(CHART_VERSION) \
	  --set 'windows.enabled=false' \
	  --set 'controller.replicas=1' \
	  --set 'controller.podAnnotations.cluster-autoscaler\.kubernetes\.io/safe-to-evict-local-volumes=socket-dir' \
	  --skip-tests \
	  > $(OUTPUT_FILE)
	kubectl kustomize . > $(OUTPUT_FILE).tmp
	../../templatify.sh $(OUTPUT_FILE).tmp --dynamic "mcr.microsoft.com/oss/v2/kubernetes-csi/azurefile-csi:version"
	cat _header.txt $(OUTPUT_FILE).tmp _footer.txt > $(OUTPUT_FILE)
	git apply --no-unsafe-paths --no-recount customizations.patch
	rm $(OUTPUT_FILE).tmp

.PHONY: setup-helm
setup-helm:
	helm repo add --force-update $(REPO_NAME) https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/charts
	helm repo update $(REPO_NAME)

.PHONY: clean-helm
clean-helm:
	helm repo remove $(REPO_NAME)
