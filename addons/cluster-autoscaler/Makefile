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

# https://github.com/kubernetes/autoscaler/tree/master/charts/cluster-autoscaler

OUTPUT_FILE = cluster-autoscaler.yaml
REPO_NAME = kkp-addons-cluster-autoscaler
CHART_VERSION = 9.46.6

.PHONY: default
default: setup-helm build clean-helm

.PHONY: build
build:
	helm --namespace kube-system template cluster-autoscaler $(REPO_NAME)/cluster-autoscaler \
	  --version $(CHART_VERSION) \
	  --set 'cloudProvider=clusterapi' \
	  --set 'autoDiscovery.labels[0].namespace=kube-system' \
	  --set 'extraEnv.CAPI_GROUP=cluster.k8s.io' \
	  --set 'securityContext.seccompProfile.type=RuntimeDefault' \
	  > $(OUTPUT_FILE).tmp
	../templatify.sh $(OUTPUT_FILE).tmp --dynamic "registry.k8s.io/autoscaling/cluster-autoscaler:version"
	cat _header.txt $(OUTPUT_FILE).tmp _footer.txt > $(OUTPUT_FILE)
	rm $(OUTPUT_FILE).tmp

.PHONY: setup-helm
setup-helm:
	helm repo add --force-update $(REPO_NAME) https://kubernetes.github.io/autoscaler
	helm repo update $(REPO_NAME)

.PHONY: clean-helm
clean-helm:
	helm repo remove $(REPO_NAME)
