package main

import (
	"strings"
	"testing"
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
			name: "non-matching single block",
			path: "/my/single/non-matching-block.yaml",
			content: `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    k8s-app: kubernetes-dashboard
  name: kubernetes-dashboard
  namespace: kube-system`,
			expected: "",
		}, {
			name: "matching single block",
			path: "/my/single/matching-block.yaml",
			content: `apiVersion: apps/v1
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
        effect: NoSchedule`,
			expected: `

#### Addon: single / File: matching-block.yaml

##### Container: kubernetes-dashboard

` + "```yaml" +
				`
limits:
  cpu: "75m"
  memory: "50Mi"
requests:
  cpu: "75m"
  memory: "50Mi"
` + "```",
		},
	}

	for i, test := range tests {
		t.Logf("#%d: %s", i, test.name)
		docr := newDocumenter(test.path, []byte(test.content))
		if test.shallFail {
			err := docr.scanAll()
			if err == nil || err.Error() != test.expected {
				t.Errorf("did not fail like expected: %v <> %s", err, test.expected)
			}
			continue
		}
		err := docr.scanAll()
		if err != nil {
			t.Errorf("scanning failed: %v", err)
		}
		var builder strings.Builder
		err = docr.document().writeAll(&builder)
		if err != nil {
			t.Errorf("writing failed: %v", err)
		}
		if builder.String() != test.expected {
			t.Errorf("documenter result doesn't match expected: %s <> %s", builder.String(), test.expected)
		}
	}
}
