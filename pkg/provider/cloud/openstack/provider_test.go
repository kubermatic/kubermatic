package openstack

import (
	"testing"

	"github.com/gophercloud/gophercloud"
)

func TestIgnoreRouterAlreadyHasPortInSubnetError(t *testing.T) {
	const subnetID = "123"
	testCases := []struct {
		name            string
		inErr           error
		expectReturnErr bool
	}{
		{
			name: "Matches",
			inErr: gophercloud.ErrDefault400{
				ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
					Body: []byte("Router already has a port on subnet " + subnetID),
				},
			},
			expectReturnErr: false,
		},
		{
			name: "Doesn't match",
			inErr: gophercloud.ErrDefault400{
				ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
					Body: []byte("Need moar permissions"),
				},
			},
			expectReturnErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ignoreRouterAlreadyHasPortInSubnetError(tc.inErr, subnetID); (err != nil) != tc.expectReturnErr {
				t.Errorf("expect return err: %t, but got err: %v", tc.expectReturnErr, err)
			}
		})
	}
}
