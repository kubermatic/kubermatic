package nanny

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderGetUrl(t *testing.T) {
	p := &provider{
		endpoint: "http://example.com/endpoint",
		client:   http.DefaultClient,
	}

	u := p.getURL("/nodes/1234")
	ex := "http://example.com/endpoint/nodes/1234"
	if u != ex {
		t.Errorf("Expected result url to be %q, got %q", ex, u)
	}
}

func TestProvider_HasBasicAuth(t *testing.T) {
	e := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "test" || pass != "123" {
			t.Errorf("Request did not contain basic auth. Expected user=%q pass=%q", user, pass)
		}
		t.Logf("Received request with user=%q pass=%q", user, pass)
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer e.Close()

	p := &provider{
		endpoint: e.URL,
		client:   http.DefaultClient,
		authUser: "test",
		authPass: "123",
	}
	_, err := p.GetNode("1234")
	_ = err
}

func TestProvider_HasNoBasicAuth(t *testing.T) {
	e := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if ok {
			t.Errorf("Request contains basic auth information but no were specified. user=%q pass=%q", user, pass)
		}
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer e.Close()

	p := &provider{
		endpoint: e.URL,
		client:   http.DefaultClient,
	}
	_, err := p.GetNode("1234")
	_ = err
}

func TestProvider_GetNode(t *testing.T) {
	e := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write([]byte("{\"id\": \"1234\", \"memory\": 1024, \"space\": 2048, \"cpus\": []}"))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer e.Close()

	p := &provider{
		endpoint: e.URL,
		client:   http.DefaultClient,
	}

	n, err := p.GetNode("1234")
	if err != nil {
		t.Fatal(err)
	}

	if n.UID != "1234" {
		t.Errorf("Expected node UID to be %q, got %q", 1234, n.UID)
	}

	if n.Memory != 1024 {
		t.Errorf("Expected node Memory to be %v, got %v", 1024, n.Memory)
	}

	if n.Space != 2048 {
		t.Errorf("Expected node Space to be %v, got %v", 2048, n.Space)
	}
}

func TestProvider_GetNodeNotFound(t *testing.T) {
	e := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer e.Close()

	p := &provider{
		endpoint: e.URL,
		client:   http.DefaultClient,
	}

	n, err := p.GetNode("1234")
	if err != nil {
		t.Fatal(err)
	}

	if n != nil {
		t.Fatal("Expected GetNode to return nil, got a node")
	}
}

func TestProvider_CreateNode(t *testing.T) {
	e := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusCreated)
		_, err := rw.Write([]byte("{\"id\": \"1234\", \"memory\": 1337, \"space\": 1337000, \"cpus\": []}"))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer e.Close()

	p := &provider{
		endpoint: e.URL,
		client:   http.DefaultClient,
	}

	n := &Node{
		UID:    "1234",
		Memory: 1337,
		Space:  1337000,
		CPUs:   []*CPU{},
	}

	n2, err := p.CreateNode(n)
	if err != nil {
		t.Fatal(err)
	}

	if n2.UID != n.UID {
		t.Errorf("Expected node UID to be %q, got %q", n.UID, n2.UID)
	}

	if n2.Memory != n.Memory {
		t.Errorf("Expected node Memory to be %v, got %v", n.Memory, n2.Memory)
	}

	if n2.Space != n.Space {
		t.Errorf("Expected node Space to be %v, got %v", n.Space, n2.Space)
	}
}

func TestProvider_GetClusterConfig(t *testing.T) {
	e := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write([]byte("{\"name\": \"test\", \"apiserver_url\": \"https://example.tld\", \"kubeconfig\": \"dmVyc2lvbjogMg==\"}"))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer e.Close()

	p := &provider{
		endpoint: e.URL,
		client:   http.DefaultClient,
	}

	c, err := p.GetClusterConfig(&Node{UID: "1234"})
	if err != nil {
		t.Fatal(err)
	}

	if c.Name != "test" {
		t.Errorf("Expected cluster name to be %q, got %q", "test", c.Name)
	}

	if c.APIServerURL != "https://example.tld" {
		t.Errorf("Expected cluster apiserver url to be %q, got %q", "https://example.tld", c.APIServerURL)
	}

	if c.KubeConfig != "version: 2" {
		t.Errorf("Expected cluster kubeconfig to be %q, got %q", "version: 2", c.Name)
	}
}
