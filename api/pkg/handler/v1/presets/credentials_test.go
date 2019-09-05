package presets_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/diff"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"github.com/kubermatic/kubermatic/api/pkg/presets"
)

func TestCredentialEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		provider         string
		credentials      *kubermaticv1.PresetList
		httpStatus       int
		expectedResponse string
	}{
		{
			name:             "test no credentials for AWS",
			provider:         "aws",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test list of credential names for AWS",
			provider: "aws",
			credentials: &kubermaticv1.PresetList{
				Items: []kubermaticv1.Preset{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{kubermaticv1.PresetEmailDomainLabel: test.GenDefaultUser().Spec.Email},
						},
						Spec: kubermaticv1.PresetSpec{
							AWS: kubermaticv1.AWS{
								Credentials: []kubermaticv1.AWSPresetCredentials{
									{Name: "first"},
									{Name: "second"},
								}},
						},
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["first","second"]}`,
		},
		{
			name:             "test no credentials for Azure",
			provider:         "azure",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test list of credential names for Azure",
			provider: "azure",
			credentials: &kubermaticv1.PresetList{
				Items: []kubermaticv1.Preset{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{kubermaticv1.PresetEmailDomainLabel: test.GenDefaultUser().Spec.Email},
						},
						Spec: kubermaticv1.PresetSpec{
							Azure: kubermaticv1.Azure{Credentials: []kubermaticv1.AzurePresetCredentials{
								{Name: "first", ClientID: "test-first", ClientSecret: "secret-first", SubscriptionID: "subscription-first", TenantID: "tenant-first"},
								{Name: "second", ClientID: "test-second", ClientSecret: "secret-second", SubscriptionID: "subscription-second", TenantID: "tenant-second"},
							}},
						},
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["first","second"]}`,
		},
		{
			name:             "test no credentials for DigitalOcean",
			provider:         "digitalocean",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test list of credential names for DigitalOcean",
			provider: "digitalocean",
			credentials: &kubermaticv1.PresetList{
				Items: []kubermaticv1.Preset{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{kubermaticv1.PresetEmailDomainLabel: test.GenDefaultUser().Spec.Email},
						},
						Spec: kubermaticv1.PresetSpec{
							Digitalocean: kubermaticv1.Digitalocean{Credentials: []kubermaticv1.DigitaloceanPresetCredentials{
								{Name: "digitalocean-first"},
								{Name: "digitalocean-second"},
							}},
						},
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["digitalocean-first","digitalocean-second"]}`,
		},
		{
			name:             "test no credentials for GCP",
			provider:         "gcp",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test list of credential names for GCP",
			provider: "gcp",
			credentials: &kubermaticv1.PresetList{
				Items: []kubermaticv1.Preset{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{kubermaticv1.PresetEmailDomainLabel: test.GenDefaultUser().Spec.Email},
						},
						Spec: kubermaticv1.PresetSpec{
							GCP: kubermaticv1.GCP{Credentials: []kubermaticv1.GCPPresetCredentials{
								{Name: "first"},
								{Name: "second"},
							}},
						},
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["first","second"]}`,
		},
		{
			name:             "test no credentials for Hetzner",
			provider:         "hetzner",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test list of credential names for Hetzner",
			provider: "hetzner",
			credentials: &kubermaticv1.PresetList{
				Items: []kubermaticv1.Preset{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{kubermaticv1.PresetEmailDomainLabel: test.GenDefaultUser().Spec.Email},
						},
						Spec: kubermaticv1.PresetSpec{
							Hetzner: kubermaticv1.Hetzner{Credentials: []kubermaticv1.HetznerPresetCredentials{
								{Name: "first"},
								{Name: "second"},
							}},
						},
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["first","second"]}`,
		},
		{
			name:             "test no credentials for OpenStack",
			provider:         "openstack",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test list of credential names for OpenStack",
			provider: "openstack",
			credentials: &kubermaticv1.PresetList{
				Items: []kubermaticv1.Preset{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{kubermaticv1.PresetEmailDomainLabel: test.GenDefaultUser().Spec.Email},
						},
						Spec: kubermaticv1.PresetSpec{
							Openstack: kubermaticv1.Openstack{Credentials: []kubermaticv1.OpenstackPresetCredentials{
								{Name: "first"},
								{Name: "second"},
							}},
						},
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["first","second"]}`,
		},
		{
			name:             "test no credentials for Packet",
			provider:         "packet",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test list of credential names for Packet",
			provider: "packet",
			credentials: &kubermaticv1.PresetList{
				Items: []kubermaticv1.Preset{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{kubermaticv1.PresetEmailDomainLabel: test.GenDefaultUser().Spec.Email},
						},
						Spec: kubermaticv1.PresetSpec{
							Packet: kubermaticv1.Packet{Credentials: []kubermaticv1.PacketPresetCredentials{
								{Name: "first"},
								{Name: "second"},
							}},
						},
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["first","second"]}`,
		},
		{
			name:             "test no credentials for Vsphere",
			provider:         "vsphere",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test list of credential names for Vsphere",
			provider: "vsphere",
			credentials: &kubermaticv1.PresetList{
				Items: []kubermaticv1.Preset{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{kubermaticv1.PresetEmailDomainLabel: test.GenDefaultUser().Spec.Email},
						},
						Spec: kubermaticv1.PresetSpec{
							VSphere: kubermaticv1.VSphere{Credentials: []kubermaticv1.VSpherePresetCredentials{
								{Name: "first"},
								{Name: "second"},
							}},
						},
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["first","second"]}`,
		},
		{
			name:       "test no existing provider",
			provider:   "test",
			httpStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/providers/%s/presets/credentials", tc.provider), strings.NewReader(""))

			credentialsManager := presets.New()
			if tc.credentials != nil {
				credentialsManager = presets.NewWithPresets(tc.credentials)
			}

			res := httptest.NewRecorder()
			router, err := test.CreateCredentialTestEndpoint(credentialsManager, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v\n", err)
			}
			router.ServeHTTP(res, req)

			// validate
			assert.Equal(t, tc.httpStatus, res.Code)

			if res.Code == http.StatusOK {
				compareJSON(t, res, tc.expectedResponse)
			}
		})
	}
}

func compareJSON(t *testing.T, res *httptest.ResponseRecorder, expectedResponseString string) {
	t.Helper()
	var actualResponse interface{}
	var expectedResponse interface{}

	// var err error
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}
	err = json.Unmarshal(bBytes, &actualResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(expectedResponseString), &expectedResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 2 :: %s", err.Error())
	}
	if !equality.Semantic.DeepEqual(actualResponse, expectedResponse) {
		t.Fatalf("Objects are different: %v", diff.ObjectDiff(actualResponse, expectedResponse))
	}
}
