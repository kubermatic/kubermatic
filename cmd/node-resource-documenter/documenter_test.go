/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"strings"
	"testing"
)

const (
	serviceAccountManifest = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    k8s-app: kubernetes-dashboard
  name: kubernetes-dashboard
  namespace: kube-system`

	deploymentManifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: kubernetes-dashboard
  name: kubernetes-dashboard
  namespace: kube-system
spec:
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      k8s-app: kubernetes-dashboard
  template:
    metadata:
      labels:
        k8s-app: kubernetes-dashboard
    spec:
      containers:
      - name: kubernetes-dashboard
        image: 'gcr.io/google_containers/kubernetes-dashboard-amd64:v1.10.1'
        ports:
        - containerPort: 8443
          protocol: TCP
        args:
        - --auto-generate-certificates
        volumeMounts:
        - name: kubernetes-dashboard-certs
          mountPath: /certs
          # Create on-disk volume to store exec logs
        - mountPath: /tmp
          name: tmp-volume
        livenessProbe:
          httpGet:
            scheme: HTTPS
            path: /
            port: 8443
          initialDelaySeconds: 30
          timeoutSeconds: 30
        resources:
          requests:
            cpu: "75m"
            memory: "50Mi"
          limits:
            cpu: "75m"
            memory: "50Mi"
      volumes:
      - name: kubernetes-dashboard-certs
        secret:
          secretName: kubernetes-dashboard-certs
      - name: tmp-volume
        emptyDir: {}
      serviceAccountName: kubernetes-dashboard
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule`

	daemonSetManifest = `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-exporter
  namespace: kube-system
  labels:
    addonmanager.kubernetes.io/mode: Reconcile
    app.kubernetes.io/name: node-exporter
    app.kubernetes.io/version: v0.18.0
spec:
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      name: node-exporter
      labels:
        app.kubernetes.io/name: node-exporter
    spec:
      hostNetwork: true
      hostPID: true
      serviceAccountName: node-exporter
      containers:
      - name: node-exporter
        image: '{{ Registry "quay.io" }}/prometheus/node-exporter:v0.18.0'
        args:
        - '--path.procfs=/host/proc'
        - '--path.sysfs=/host/sys'
        - '--path.rootfs=/host/root'
        - '--web.listen-address=127.0.0.1:9100'
        resources:
          requests:
            cpu: 10m
            memory: 24Mi
          limits:
            cpu: 25m
            memory: 48Mi
        volumeMounts:
        - name: proc
          readOnly:  true
          mountPath: /host/proc
        - name: sys
          readOnly: true
          mountPath: /host/sys
        - name: root
          readOnly: true
          mountPath: /host/root
          mountPropagation: HostToContainer

      - name: kube-rbac-proxy
        image: 'quay.io/coreos/kube-rbac-proxy:v0.4.1'
        args:
        - '--logtostderr'
        - '--secure-listen-address=$(IP):9100'
        - '--tls-cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256'
        - '--upstream=http://127.0.0.1:9100/'
        env:
        - name: IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        ports:
        - containerPort: 9100
          hostPort: 9100
          name: https
        resources:
          requests:
            cpu: 10m
            memory: 24Mi
          limits:
            cpu: 20m
            memory: 48Mi

      tolerations:
      - effect: NoExecute
        operator: Exists
      - effect: NoSchedule
        operator: Exists
      volumes:
      - name: proc
        hostPath:
          path: /proc
      - name: sys
        hostPath:
          path: /sys
      - name: root
        hostPath:
          path: /
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534`

	defunctDeploymentManifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: bad-deployment
  namespace: kube-system
spec:
  illegal v - a - l - u - e
  has-to: crash`
)

func TestDocumenter(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		content   string
		shallFail bool
		expected  string
	}{
		{
			name:     "empty file",
			path:     "/my/empty.yaml",
			expected: "",
		}, {
			name:     "non-matching single block",
			path:     "/my/single/non-matching-block.yaml",
			content:  serviceAccountManifest,
			expected: "",
		}, {
			name:    "matching single block",
			path:    "/my/single/matching-block.yaml",
			content: deploymentManifest,
			expected: `

#### Addon: single / File: matching-block.yaml

##### Container: kubernetes-dashboard

` + "```yaml" + `
limits:
  cpu: "75m"
  memory: "50Mi"
requests:
  cpu: "75m"
  memory: "50Mi"
` + "```",
		}, {
			name:    "matching multiple blocks",
			path:    "/my/multiple/matching-blocks.yaml",
			content: serviceAccountManifest + "\n---\n" + daemonSetManifest,
			expected: `

#### Addon: multiple / File: matching-blocks.yaml

##### Container: node-exporter

` + "```yaml" + `
limits:
  cpu: "25m"
  memory: "48Mi"
requests:
  cpu: "10m"
  memory: "24Mi"
` + "```" + `

##### Container: kube-rbac-proxy

` + "```yaml" + `
limits:
  cpu: "20m"
  memory: "48Mi"
requests:
  cpu: "10m"
  memory: "24Mi"
` + "```",
		}, {
			name:      "failing blocks",
			path:      "/my/multiple/failing-blocks.yaml",
			content:   serviceAccountManifest + "\n---\n" + defunctDeploymentManifest,
			shallFail: true,
			expected:  "error converting YAML to JSON",
		},
	}

	for i, test := range tests {
		t.Logf("#%d: %s", i, test.name)
		docr := newDocumenter(test.path, []byte(test.content))
		err := docr.scanAll()
		if err != nil {
			if !test.shallFail {
				// Scanning shall not fail.
				t.Fatalf("scanning did not expect error %v", err)
			}
			if !strings.Contains(err.Error(), test.expected) {
				// Scanning shall fail different.
				t.Fatalf("scanning failed %s but shall %s", err.Error(), test.expected)
			}
			// Failed as expected.
			continue
		}
		var builder strings.Builder
		err = docr.document().writeAll(&builder)
		if err != nil {
			t.Errorf("writing failed: %v", err)
		}
		if builder.String() != test.expected {
			t.Errorf("documenter result doesn't match expected: %q <> %q", builder.String(), test.expected)
		}
	}
}
