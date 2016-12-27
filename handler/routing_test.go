package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"github.com/kubermatic/api/provider/kubernetes"
)

var jwtSecret = "super secret auth key nobody will guess"

func createTestEndpoint() http.Handler {
	ctx := context.Background()

	dcs, err := provider.DatacentersMeta("./fixtures/datacenters.yaml")
	if err != nil {
		log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", "./fixtures/datacenters.yaml", err))
	}

	// create CloudProviders
	cps := cloud.Providers(dcs)

	kps, err := kubernetes.Providers("./fixtures/kubecfg.yaml", dcs, cps, "./fixtures/secrets.yaml", true)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	routing := NewRouting(ctx, dcs, kps, cps, true, base64.URLEncoding.EncodeToString([]byte(jwtSecret)))
	routing.Register(router)

	return router
}

func getTokenStr(admin bool) string {
	roles := []interface{}{}
	if admin {
		roles = append(roles, "admin")
	}

	data := jwt.MapClaims{
		"sub": "Thomas Tester",
		"app_metadata": map[string]interface{}{
			"roles": roles,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(data))

	tokenStr, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		panic(err)
	}

	return tokenStr
}

func authenticateHeader(req *http.Request, admin bool) {
	tokenStr := getTokenStr(admin)
	req.Header.Add("Authorization", "bearer "+tokenStr)
}

func authenticateQuery(req *http.Request, admin bool) {
	tokenStr := getTokenStr(admin)
	q := req.URL.Query()
	q.Add("token", tokenStr)
	req.URL.RawQuery = q.Encode()
}

func compareWithResult(t *testing.T, res *httptest.ResponseRecorder, file string) {
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}

	rBytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("Cannot read response file %s", file)
	}

	r := strings.TrimSpace(string(rBytes))
	b := strings.TrimSpace(string(bBytes))

	if r != b {
		t.Fatalf("Expected response body to be '%s', got '%s'", r, b)
	}
}

func TestUpRoute(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Errorf("Expected route to return code 200, got %d", res.Code)
	}
}
