package handler

/*
func TestCreateNodesEndpoint(t *testing.T) {
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
	e := createTestEndpoint()

	req := httptest.NewRequest("POST", "/api/v1/dc/fake-1/cluster/234jkh24234g/node", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("Expected status code to be 200, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

	var ns []*api.Node
	if err := json.Unmarshal(res.Body.Bytes(), &ns); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if len(ns) != 1 {
		t.Errorf("Expected nodes length 1, got %d", len(ns))
		return
	}

	if ns[0].Metadata.UID == "" {
		t.Error("Expected node UID to be filled, got nil.")
	}
	if ns[0].Metadata.Name == "" {
		t.Error("Expected node name to be filled, got nil.")
	}
}

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
	e := createTestEndpoint()

	req := httptest.NewRequest("POST", "/api/v1/dc/testtest/cluster/234jkh24234g/node", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	e.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Errorf("Expected status code to be 400, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

	exp := "unknown kubernetes datacenter \"testtest\"\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestNodesEndpointEmpty(t *testing.T) {
	res := httptest.NewRecorder()
	e := createTestEndpoint()

	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster/234jkh24234g/node", nil)
	authenticateHeader(req, false)

	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("Expected status code to be 200, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

	var ns []*api.Node
	if err := json.Unmarshal(res.Body.Bytes(), &ns); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if len(ns) != 0 {
		t.Errorf("Expected nodes length 0, got %d", len(ns))
	}
}

func TestNodesEndpoint(t *testing.T) {
	res := httptest.NewRecorder()
	e := createTestEndpoint()

	_, err := createTestNode(t, e)
	if err != nil {
		t.Error(err)
		return
	}

	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster/234jkh24234g/node", nil)
	authenticateHeader(req, false)
	e.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Errorf("Expected status code to be 200, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

	var ns []*api.Node
	if err := json.Unmarshal(res.Body.Bytes(), &ns); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if len(ns) != 1 {
		t.Errorf("Expected nodes length 1, got %d", len(ns))
	}
}

func TestDeleteNodesEndpoint(t *testing.T) {
	res := httptest.NewRecorder()
	e := createTestEndpoint()

	tn, err := createTestNode(t, e)
	if err != nil {
		t.Error(err)
		return
	}

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/dc/fake-1/cluster/234jkh24234g/node/%s", tn.Metadata.UID), nil)
	authenticateHeader(req, false)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("Expected status code to be 200, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}
}

func TestDeleteNodesEndpointNotExistingDC(t *testing.T) {
	res := httptest.NewRecorder()
	e := createTestEndpoint()

	tn, err := createTestNode(t, e)
	if err != nil {
		t.Error(err)
		return
	}

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/dc/testtest/cluster/234jkh24234g/node/%s", tn.Metadata.UID), nil)
	authenticateHeader(req, false)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Errorf("Expected status code to be 400, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

	exp := "unknown kubernetes datacenter \"testtest\"\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestKubernetesNodeInfoEndpoint(t *testing.T) {
	t.Skip("Cannot execute test due to client calls in handler method.")
}

func TestKubernetesNodesEndpoint(t *testing.T) {
	t.Skip("Cannot execute test due to client calls in handler method.")
}

func createTestNode(t *testing.T, e http.Handler) (*api.Node, error) {
	reqObj := createNodesReq{
		Instances: 1,
		Spec: api.NodeSpec{
			Fake: &api.FakeNodeSpec{
				OS:   "any",
				Type: "any",
			},
		},
	}

	req := httptest.NewRequest("POST", "/api/v1/dc/fake-1/cluster/234jkh24234g/node", encodeReq(t, &reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("Expected status code to be 200, got %d", res.Code)
		t.Error(res.Body.String())
		return nil, fmt.Errorf("Expected status code to be 200, got %d", res.Code)
	}

	var ns []*api.Node
	if err := json.Unmarshal(res.Body.Bytes(), &ns); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return nil, err
	}

	if len(ns) != 1 {
		err := fmt.Errorf("Expected nodes length 1, got %d", len(ns))
		t.Error(err)
		return nil, err
	}

	return ns[0], nil
}
*/
