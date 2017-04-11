package extensions

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/testapi"
	"k8s.io/client-go/pkg/apimachinery/registered"
	uapi "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
)

func objBody(object interface{}) io.ReadCloser {
	output, err := json.MarshalIndent(object, "", "")
	if err != nil {
		panic(err)
	}
	return ioutil.NopCloser(bytes.NewReader([]byte(output)))
}

// FakeWrapClientsetWithExtensions returns a fake clientset which should only be used for testing
func FakeWrapClientsetWithExtensions(config *rest.Config) Clientset {
	fakeClient := &fake.RESTClient{
		NegotiatedSerializer: testapi.Default.NegotiatedSerializer(),
		Resp: &http.Response{
			StatusCode: 200,
			Body:       objBody(&uapi.APIVersions{Versions: []string{"version1", registered.GroupOrDie(api.GroupName).GroupVersion.String()}}),
		},
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			header := http.Header{}
			header.Set("Content-Type", runtime.ContentTypeJSON)
			return &http.Response{StatusCode: 200, Header: header, Body: objBody(&uapi.APIVersions{Versions: []string{"version1", registered.GroupOrDie(api.GroupName).GroupVersion.String()}})}, nil
		}),
	}
	return &WrappedClientset{
		Client: fakeClient,
	}
}
