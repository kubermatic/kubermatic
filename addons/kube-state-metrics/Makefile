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

OUTPUT_FILE = kube-state-metrics.yaml
REPO_NAME = kkp-addons-kube-state-metrics
CHART_VERSION = 5.25.1

.PHONY: default
default: setup-helm build clean-helm

.PHONY: build
build:
	helm --namespace kube-system template kube-state-metrics $(REPO_NAME)/kube-state-metrics \
	  --version $(CHART_VERSION) \
	  --values helm-values \
	  --api-versions 'policy/v1/PodDisruptionBudget' \
	  > $(OUTPUT_FILE).tmp
	../templatify.sh $(OUTPUT_FILE).tmp
	cat _header.txt $(OUTPUT_FILE).tmp > $(OUTPUT_FILE)
	rm $(OUTPUT_FILE).tmp
	git apply --no-unsafe-paths --no-recount customizations.patch

.PHONY: setup-helm
setup-helm:
	helm repo add --force-update $(REPO_NAME) https://prometheus-community.github.io/helm-charts
	helm repo update $(REPO_NAME)

.PHONY: clean-helm
clean-helm:
	helm repo remove $(REPO_NAME)
