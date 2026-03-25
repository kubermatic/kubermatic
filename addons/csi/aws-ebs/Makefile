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

# https://github.com/kubernetes-sigs/aws-ebs-csi-driver/tree/master/charts/aws-ebs-csi-driver

OUTPUT_FILE = driver.yaml
REPO_NAME = kkp-addons-csi-aws-ebs
CHART_VERSION = 2.55.0

.PHONY: default
default: setup-helm build clean-helm

.PHONY: build
build:
	helm --namespace kube-system template aws-ebs-csi-driver $(REPO_NAME)/aws-ebs-csi-driver \
	  --version $(CHART_VERSION) \
	  --set 'controller.k8sTagClusterId=\{{ "{{ .Cluster.Name }}" }}' \
	  --set 'node.securityContext.seccompProfile.type=RuntimeDefault' \
	  --set 'controller.securityContext.seccompProfile.type=RuntimeDefault' \
	  --set 'controller.podAnnotations.cluster-autoscaler\.kubernetes\.io/safe-to-evict-local-volumes=socket-dir' \
	  --api-versions 'policy/v1/PodDisruptionBudget' \
	  --skip-tests \
	  >> $(OUTPUT_FILE).tmp
	../../templatify.sh $(OUTPUT_FILE).tmp
	cat _header.txt $(OUTPUT_FILE).tmp _footer.txt > $(OUTPUT_FILE)
	rm $(OUTPUT_FILE).tmp

.PHONY: setup-helm
setup-helm:
	helm repo add --force-update $(REPO_NAME) https://kubernetes-sigs.github.io/aws-ebs-csi-driver
	helm repo update $(REPO_NAME)

.PHONY: clean-helm
clean-helm:
	helm repo remove $(REPO_NAME)
