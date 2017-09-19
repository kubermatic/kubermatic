package handler

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func init() {
	flag.String("swagger-path", "wrong_path_to_index.json", "OpenID client ID")
}

func TestAPIDescriptionHandlerStatusNotFound(t *testing.T) {
	os.Args[1] = "-swagger-path=wrong_path_to_index.json"
	flag.Parse()
	req, err := http.NewRequest("GET", "/api/", nil)
	if err != nil {
		t.Fatal(err)
	}
	res := httptest.NewRecorder()
	e := http.HandlerFunc(APIDescriptionHandler)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			res.Code, http.StatusNotFound)
	}
}

func TestAPIDescriptionHandlerStatusOK(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "index.json")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte("{'apiVersion': '1.4.0'}")); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	os.Args[1] = fmt.Sprintf("-swagger-path=%s", tmpfile.Name())
	flag.Parse()
	req, err := http.NewRequest("GET", "/api/", nil)
	if err != nil {
		t.Fatal(err)
	}
	res := httptest.NewRecorder()
	e := http.HandlerFunc(APIDescriptionHandler)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			res.Code, http.StatusOK)
	}
}
