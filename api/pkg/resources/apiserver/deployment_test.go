package apiserver

import (
	"reflect"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

func TestGetApiserverFlags(t *testing.T) {
	tests := []struct {
		name              string
		kubernetesVersion string
		expected          []string
	}{
		{
			name:              "Ensure no admission webhooks pre 1.9",
			kubernetesVersion: "1.8.0",
			expected: []string{
				"--advertise-address", "",
				"--secure-port", "0",
				"--kubernetes-service-node-port", "0",
				"--insecure-bind-address", "0.0.0.0",
				"--insecure-port", "8080",
				"--etcd-servers", "etcd-0",
				"--storage-backend", "etcd3",
				"--admission-control", "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota,NodeRestriction",
				"--authorization-mode", "Node,RBAC",
				"--external-hostname", "",
				"--token-auth-file", "/etc/kubernetes/tokens/tokens.csv",
				"--enable-bootstrap-token-auth", "true",
				"--service-account-key-file", "/etc/kubernetes/service-account-key/sa.key",
				"--service-cluster-ip-range", "10.0.0.0",
				"--service-node-port-range", "30000-32767",
				"--allow-privileged",
				"--audit-log-maxage", "30",
				"--audit-log-maxbackup", "3",
				"--audit-log-maxsize", "100",
				"--audit-log-path", "/var/log/audit.log",
				"--tls-cert-file", "/etc/kubernetes/tls/apiserver-tls.crt",
				"--tls-private-key-file", "/etc/kubernetes/tls/apiserver-tls.key",
				"--proxy-client-cert-file", "/etc/kubernetes/tls/apiserver-tls.crt",
				"--proxy-client-key-file", "/etc/kubernetes/tls/apiserver-tls.key",
				"--client-ca-file", "/etc/kubernetes/ca-cert/ca.crt",
				"--kubelet-client-certificate", "/etc/kubernetes/kubelet/kubelet-client.crt",
				"--kubelet-client-key", "/etc/kubernetes/kubelet/kubelet-client.key",
				"--v", "4",
				"--kubelet-preferred-address-types", "ExternalIP,InternalIP"},
		},
		{
			name:              "Ensure admission webhooks 1.9+",
			kubernetesVersion: "1.9.0",
			expected: []string{
				"--advertise-address", "",
				"--secure-port", "0",
				"--kubernetes-service-node-port", "0",
				"--insecure-bind-address", "0.0.0.0",
				"--insecure-port", "8080",
				"--etcd-servers", "etcd-0",
				"--storage-backend", "etcd3",
				"--admission-control", "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook",
				"--authorization-mode", "Node,RBAC",
				"--external-hostname", "",
				"--token-auth-file", "/etc/kubernetes/tokens/tokens.csv",
				"--enable-bootstrap-token-auth", "true",
				"--service-account-key-file", "/etc/kubernetes/service-account-key/sa.key",
				"--service-cluster-ip-range", "10.0.0.0",
				"--service-node-port-range", "30000-32767",
				"--allow-privileged",
				"--audit-log-maxage", "30",
				"--audit-log-maxbackup", "3",
				"--audit-log-maxsize", "100",
				"--audit-log-path", "/var/log/audit.log",
				"--tls-cert-file", "/etc/kubernetes/tls/apiserver-tls.crt",
				"--tls-private-key-file", "/etc/kubernetes/tls/apiserver-tls.key",
				"--proxy-client-cert-file", "/etc/kubernetes/tls/apiserver-tls.crt",
				"--proxy-client-key-file", "/etc/kubernetes/tls/apiserver-tls.key",
				"--client-ca-file", "/etc/kubernetes/ca-cert/ca.crt",
				"--kubelet-client-certificate", "/etc/kubernetes/kubelet/kubelet-client.crt",
				"--kubelet-client-key", "/etc/kubernetes/kubelet/kubelet-client.key",
				"--v", "4",
				"--kubelet-preferred-address-types", "ExternalIP,InternalIP"},
		},
	}

	for _, test := range tests {
		templateData := resources.TemplateData{}
		templateData.Cluster = &kubermaticv1.Cluster{}
		templateData.Cluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.0.0.0"}
		templateData.Cluster.Spec.Cloud = &kubermaticv1.CloudSpec{}
		templateData.Cluster.Spec.Version = test.kubernetesVersion

		if flags := getApiserverFlags(&templateData, 0, []string{"etcd-0"}); !reflect.DeepEqual(flags, test.expected) {
			t.Errorf("Result flags \n%v\n do not match expected \n%v\n!", flags, test.expected)
		}
	}

}
