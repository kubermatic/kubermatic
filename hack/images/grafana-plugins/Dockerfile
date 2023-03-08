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

FROM docker.io/alpine:3.17
LABEL maintainer="support@kubermatic.com"

RUN mkdir -p /plugins
WORKDIR /plugins

# Not all plugin files support stream decoding, so we cannot just
# pipe `wget -O- <url> | unzip`, even if it might work with _some_
# of the plugins.

RUN wget -O plugin.zip https://grafana.com/api/plugins/grafana-piechart-panel/versions/1.6.4/download && unzip plugin.zip && rm plugin.zip
RUN wget -O plugin.zip https://grafana.com/api/plugins/farski-blendstat-panel/versions/1.0.3/download && unzip plugin.zip && rm plugin.zip
RUN wget -O plugin.zip https://grafana.com/api/plugins/michaeldmoore-multistat-panel/versions/1.7.2/download && unzip plugin.zip && rm plugin.zip
RUN wget -O plugin.zip https://grafana.com/api/plugins/vonage-status-panel/versions/1.0.11/download && unzip plugin.zip && rm plugin.zip
