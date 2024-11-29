# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
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

.PHONY: aws-node-termination-handler
aws-node-termination-handler: OUTPUT_FILE=aws-node-termination-handler/aws-node-termination-handler.yaml
aws-node-termination-handler:
	cat aws-node-termination-handler/_header.txt > $(OUTPUT_FILE)
	helm template aws-node-termination-handler oci://public.ecr.aws/aws-ec2/helm/aws-node-termination-handler \
	  --version 0.26.0 \
	  --namespace kube-system \
	  --values values-aws-node-termination-handler.yaml \
	  >> $(OUTPUT_FILE)
	cat aws-node-termination-handler/_footer.txt >> $(OUTPUT_FILE)
	./templatify-images.sh $(OUTPUT_FILE)

.PHONY: nutanix-csi
nutanix-csi: nutanix-csi-storage nutanix-csi-snapshotter

.PHONY: nutanix-csi-storage
nutanix-csi-storage: OUTPUT_FILE=csi/nutanix/csi-driver.yaml
nutanix-csi-storage:
	helm repo add nutanix https://nutanix.github.io/helm/ || true
	helm repo update
	mkdir -p csi/nutanix
	cat csi/nutanix/_header.txt > $(OUTPUT_FILE)
	helm --namespace kube-system template nutanix-csi-storage nutanix/nutanix-csi-storage \
	  --version 2.6.6 \
	  --set 'createSecret=false' \
	  >> $(OUTPUT_FILE)
	cat csi/nutanix/_footer.txt >> $(OUTPUT_FILE)
	./templatify-images.sh $(OUTPUT_FILE)

nutanix-csi-snapshotter: OUTPUT_FILE=csi/nutanix/snapshot-controller.yaml
nutanix-csi-snapshotter:
	helm repo add nutanix https://nutanix.github.io/helm/ || true
	helm repo update
	mkdir -p csi/nutanix
	cat csi/nutanix/_header.txt > $(OUTPUT_FILE)
	helm --namespace kube-system template nutanix-csi-snapshot nutanix/nutanix-csi-snapshot \
	  --version 6.3.2 \
	  >> $(OUTPUT_FILE)
	cat csi/nutanix/_footer.txt >> $(OUTPUT_FILE)
	./templatify-images.sh $(OUTPUT_FILE)

.PHONY: metallb
metallb: OUTPUT_FILE=metallb/00_metallb.yaml
metallb:
	cat metallb/_header.txt > $(OUTPUT_FILE)
	wget -q -O- https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml >> $(OUTPUT_FILE)
	./templatify-images.sh $(OUTPUT_FILE)
