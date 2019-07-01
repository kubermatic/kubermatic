package presets_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"github.com/kubermatic/kubermatic/api/pkg/presets"
	"github.com/stretchr/testify/assert"
)

func TestProviderNetworkEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		provider         string
		credentials      *presets.Presets
		httpStatus       int
		expectedResponse string
	}{
		{
			name:             "test no default network for AWS",
			provider:         "aws",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test network name for AWS",
			provider: "aws",
			credentials: &presets.Presets{AWS: presets.AWS{
				Network: presets.Network{Name: "aws-test"},
			}},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"name":"aws-test"}`,
		},
		{
			name:             "test no default network for Azure",
			provider:         "azure",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test network name for Azure",
			provider: "azure",
			credentials: &presets.Presets{Azure: presets.Azure{
				Network: presets.Network{Name: "azure-test"},
			}},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"name":"azure-test"}`,
		},
		{
			name:             "test no default network for Vsphere",
			provider:         "vsphere",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test network name for Vsphere",
			provider: "vsphere",
			credentials: &presets.Presets{VSphere: presets.VSphere{
				Network: presets.Network{Name: "vsphere-test"},
			}},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"name":"vsphere-test"}`,
		},
		{
			name:             "test no default network for OpenStack",
			provider:         "openstack",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test network name for OpenStack",
			provider: "openstack",
			credentials: &presets.Presets{Openstack: presets.Openstack{
				Network: presets.Network{Name: "openstack-test"},
			}},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"name":"openstack-test"}`,
		},
		{
			name:             "test no default network for GCP",
			provider:         "gcp",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name:     "test network name for GCP",
			provider: "gcp",
			credentials: &presets.Presets{GCP: presets.GCP{
				Network: presets.Network{Name: "gcp-test"},
			}},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"name":"gcp-test"}`,
		},
		{
			name:             "test not supported provider - Packet",
			provider:         "packet",
			httpStatus:       http.StatusBadRequest,
			expectedResponse: `{"error":{"code":400,"message":"invalid provider name packet"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/providers/%s/presets/network", tc.provider), strings.NewReader(""))

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

			compareJSON(t, res, tc.expectedResponse)
		})
	}
}
