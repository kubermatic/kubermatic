# Copyright 2022 The Kubermatic Kubernetes Platform contributors.
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

SHELL := /bin/bash

.PHONY: default
default: build

.PHONY: build
build:
	CGO_ENABLED=0 go build -v

.PHONY: docker
docker: build
	docker build -t quay.io/kubermatic/ldap-group-syncer:latest .

.PHONY: integration-tests
integration-tests: docker-env
	sleep 10
	go test -tags ldaptest -v ./...
	docker stop openldap-test

.PHONY: docker-env
docker-env:
	. testdata/pawnee-grouped/env.sh; \
	docker run \
		--rm \
		--detach \
		--name openldap-test \
		-p 10389:10389 \
		-p 10636:10636 \
		-e "DATA_DIR=/bootstrap" \
		-e "LDAP_DOMAIN=$${LDAP_DOMAIN}" \
		-e "LDAP_BASEDN=$${LDAP_BASEDN}" \
		-e "LDAP_ORGANISATION=$${LDAP_ORGANISATION}" \
		-e "LDAP_BINDDN=$${LDAP_BINDDN}" \
		-e "LDAP_SECRET=$${LDAP_SECRET}" \
		-v $(realpath testdata/pawnee-grouped):/bootstrap \
		quay.io/kubermatic-labs/openldap-test:latest || true
