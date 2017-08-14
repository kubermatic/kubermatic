# Copyright 2016 The Kubernetes Authors.
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

FROM debian:jessie

ADD https://storage.googleapis.com/kubernetes-release/release/v1.6.6/bin/linux/amd64/kubectl /usr/local/bin/
RUN chmod 755 /usr/local/bin/kubectl

ADD kube-addons.sh /opt/
ADD namespace.yaml /opt/
RUN chmod +x /opt/kube-addons.sh
ADD addons /etc/kubernetes/addons

CMD ["/opt/kube-addons.sh"]
