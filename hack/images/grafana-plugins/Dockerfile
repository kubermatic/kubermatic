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

FROM alpine:3.13
LABEL maintainer="support@kubermatic.com"

RUN mkdir -p /plugins
WORKDIR /plugins

RUN wget -O- https://grafana.com/api/plugins/grafana-piechart-panel/versions/1.6.1/download | unzip -

RUN wget -O- https://grafana.com/api/plugins/farski-blendstat-panel/versions/1.0.2/download | unzip - && \
    mv farski-blendstat-grafana-* farski-blendstat-panel

RUN wget -O- https://grafana.com/api/plugins/michaeldmoore-multistat-panel/versions/1.7.1/download | unzip -

RUN wget -O- https://grafana.com/api/plugins/vonage-status-panel/versions/1.0.10/download | unzip - && \
    mv Vonage-Grafana_Status_panel-* vonage-status-panel
