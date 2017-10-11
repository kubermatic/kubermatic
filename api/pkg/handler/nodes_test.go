package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubermatic/kubermatic/api"
)

func TestCreateNodesEndpointNotExistingDC(t *testing.T) {
	reqObj := createNodesReq{
		Instances: 1,
		Spec: api.NodeSpec{
			Fake: &api.FakeNodeSpec{
				OS:   "any",
				Type: "any",
			},
		},
	}

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(false))

	req := httptest.NewRequest("POST", "/api/v1/dc/testtest/cluster/234jkh24234g/node", encodeReq(t, reqObj))

	e.ServeHTTP(res, req)
	checkStatusCode(http.StatusBadRequest, res, t)
}

func TestKubernetesNodeInfoEndpoint(t *testing.T) {
	t.Skip("Cannot execute test due to client calls in handler method.")
}

func TestKubernetesNodesEndpoint(t *testing.T) {
	t.Skip("Cannot execute test due to client calls in handler method.")
}
