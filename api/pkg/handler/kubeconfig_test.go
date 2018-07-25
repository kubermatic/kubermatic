package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghodss/yaml"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

var config = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURDekNDQWZPZ0F3SUJBZ0lRVXd4d09OY2FER3M4NHVZNXZLbWsxekFOQmdrcWhraUc5dzBCQVFzRkFEQXYKTVMwd0t3WURWUVFERXlSaFpXVXlPR05oT1MwNU9EWTVMVFF6TVRRdFlqVmxNUzB4WWpNNE56VXpPR1l6Wm1JdwpIaGNOTVRjeE1ERTVNVFF6T0RNNFdoY05Nakl4TURFNE1UVXpPRE00V2pBdk1TMHdLd1lEVlFRREV5UmhaV1V5Ck9HTmhPUzA1T0RZNUxUUXpNVFF0WWpWbE1TMHhZak00TnpVek9HWXpabUl3Z2dFaU1BMEdDU3FHU0liM0RRRUIKQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUUNmMmVNeU9iNzc1cWdJdG80SlNJcXRqa2RKMDI1bTk3ZFI1R0wraC9iRQo1SGdaZUo2YWhHNDBFZHl5TGdlTWZMT0RPRjRFcndwRFZ0Zk91U3FCRGI4NVdsVHVaUjJZdFdqVWk0eUxJNTdqCk95SmtHUTZPb041REJuVjVZRWl4STMvYjBSSnJjVlFiK0JtMENwRk1LTDlTMkRaQkFUQm1qMW5xY1JrVjRSMlgKOHM3YXI0RVg5OHRkVjgwbkpleGlNUENBTGdyTE54TGdCZmM4b25xNjFFQjJsS1AwOW5aSStkYkpHUjNMWW0wRApmc0NhdWxlN1k0VkwzSGRzWEFaZnFxbXhCZnREeVhTNEttb1ZuV0RjVHRMYVY4OE51SW9FV0YyWTdjR21DRzhNClpMSWJUU3FwNWRyOWx3emt3TlQ1d3d3RmRsbFF6cy9TLzdUWk93Q0FtcXh4QWdNQkFBR2pJekFoTUE0R0ExVWQKRHdFQi93UUVBd0lDQkRBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFBMQpWY2RwVkNrM29GRCtGQmh0TTJSTCswWHRLRkhkeTd6cEJNVFpFN0IyUkhQVnRtUXUvNUFiNGp0UU5xL05oTk1VCnUvM0hTYUJZM1l5ckNkUWx5QWl3S0EwM1ErK0xLY01tUUdFUHdLYlgwVzdGcEJ6OGpxMDRLcnVkRm9oeGZuazMKVkFBVXYxU2NPRzVFUTdpTkI2MG5LREtVRHhxYnRJOG5xTXRKaHFZZWJtTWhhdGdQMkFEeFg5aFB6VEJmRURSeQp5Zmw4MEpYM2N2UVZDMkQwZ1RqbUJReWNkTVVva3NIblZUcHo5cnRYc2htNmlpS0tSRDVrc0J2VEFXWkhiREw4Cmp6SFJQRmJoREU4VDR3MGVPTURkRjhDU1RUWXRGb0NsOGRvM2dQbEw2MzViUkhTR01KT3FpbFltVHdhSFBFSG8KcFVwU1pEemFTMDAxNnFTVHdHWjYKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    server: https://35.198.89.109
  name: europe-west3-c
contexts:
- context:
    cluster: europe-west3-c
    user: europe-west3-c
  name: europe-west3-c
current-context: europe-west3-c
kind: Config
preferences: {}
users:
- name: europe-west3-c
  user:
    password: FOOOOO
    username: admin
`

func TestKubeConfigEndpoint(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v3/projects/foo_project/dc/us-central1/cluster/foo/kubeconfig", nil)

	res := httptest.NewRecorder()
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "foo",
			Labels: map[string]string{"user": testUsername},
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-foo",
		},
		Address: kubermaticv1.ClusterAddress{
			AdminToken: "admintoken",
			URL:        "https://foo.bar:8443",
		},
		Spec: kubermaticv1.ClusterSpec{},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.AdminKubeconfigSecretName,
			Namespace: "cluster-foo",
		},
		Data: map[string][]byte{
			resources.AdminKubeconfigSecretKey: []byte(config),
		},
	}

	ep, err := createTestEndpoint(getUser(testUsername, false), []runtime.Object{secret}, []runtime.Object{cluster}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}

	ep.ServeHTTP(res, req)
	checkStatusCode(http.StatusOK, res, t)

	b, err := yaml.YAMLToJSON(res.Body.Bytes())
	if err != nil {
		t.Error(err)
		return
	}

	var c *cmdv1.Config
	if err := json.Unmarshal(b, &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if res.Body.String() != config {
		t.Error("invalid kubeconfig returned")
	}
}
