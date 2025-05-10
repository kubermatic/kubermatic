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

# https://github.com/kubernetes/cloud-provider-openstack/tree/master/charts/cinder-csi-plugin

OUTPUT_FILE = driver.yaml
REPO_NAME = kkp-addons-csi-openstack

# alpha.1 allows to specify extraEnv in Helm values
CHART_VERSION = 2.33.0-alpha.0

.PHONY: default
default: setup-helm build clean-helm

.PHONY: build
build:
	helm --namespace kube-system template openstack-cinder-csi $(REPO_NAME)/openstack-cinder-csi \
	  --version $(CHART_VERSION) \
	  --values helm-values \
	  --skip-tests \
	  > $(OUTPUT_FILE)
	kubectl kustomize . > $(OUTPUT_FILE).tmp
	sed -i 's/feature-gates=Topology=true/feature-gates=Topology={{ .Cluster.CSI.CinderTopologyEnabled }}/g' $(OUTPUT_FILE).tmp
	../../templatify.sh $(OUTPUT_FILE).tmp --dynamic "registry.k8s.io/provider-os/cinder-csi-plugin:version"
	cat _header.txt $(OUTPUT_FILE).tmp _footer.txt > $(OUTPUT_FILE)
	rm $(OUTPUT_FILE).tmp

.PHONY: setup-helm
setup-helm:
	helm repo add --force-update $(REPO_NAME) https://kubernetes.github.io/cloud-provider-openstack
	helm repo update $(REPO_NAME)

.PHONY: clean-helm
clean-helm:
	helm repo remove $(REPO_NAME)
