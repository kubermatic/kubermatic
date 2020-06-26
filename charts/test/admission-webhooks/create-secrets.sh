#!/bin/bash

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

name=demo-validator.default.svc

if [ ! -s ./snakeoil.key -o ! -s ./snakeoil.crt ]; then
	echo "Creating new snakeoil-secrets..."
	openssl req -newkey rsa:1024 -nodes \
		-days 30 -x509 \
		-subj "/O=Snakeoil Inc/CN=$name" \
		-keyout snakeoil.key -out snakeoil.crt
fi

echo "Injecting snakeoil-secrets into manifests..."
sed -i "s/snakeoil.crt:.*$/snakeoil.crt: `openssl base64 -A < snakeoil.crt`/" \
	admissionvalidator-svc.yaml
sed -i "s/snakeoil.key:.*$/snakeoil.key: `openssl base64 -A < snakeoil.key`/" \
	admissionvalidator-svc.yaml
sed -i "s/caBundle:.*$/caBundle: `openssl base64 -A < snakeoil.crt`/" \
	master-admission-config.yaml

echo "Done."
