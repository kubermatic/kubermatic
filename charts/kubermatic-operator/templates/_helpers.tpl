{{/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/}}

{{/*
Aborts rendering if the given content does not look like a PEM-encoded certificate.
This is a shape check only; the certificates are parsed and validated by the
kubermatic-operator when it reads the resulting ConfigMap. Catching the obvious
mistakes here turns a silently broken CA bundle into an installation-time error.

Usage: {{ include "kubermatic-operator.validatePEM" (dict "content" $pem "source" "caBundle.certificates") }}
*/}}
{{- define "kubermatic-operator.validatePEM" -}}
{{- if not (contains "-----BEGIN CERTIFICATE-----" .content) -}}
{{- fail (printf "%s does not contain a PEM-encoded certificate (expected a \"-----BEGIN CERTIFICATE-----\" block)" .source) -}}
{{- end -}}
{{- end -}}
