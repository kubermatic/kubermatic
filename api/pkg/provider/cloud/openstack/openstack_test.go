package openstack

import (
	"fmt"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
	fakegopherc "github.com/gophercloud/gophercloud/testhelper/client"
)

func TestGetSubnetIDs(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/subnets", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fakegopherc.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, GetSubnets)
	})

	serviceClient := fakegopherc.ServiceClient()
	subnets, err := getAllSubnets(serviceClient)
	if err != nil {
		t.Fatalf("failed to get a list of all subnets: %v", err)
	}
	for i, subnet := range subnets {
		th.CheckDeepEquals(t, expectedSubnets[i], subnet)
	}
}

func TestGetAllNetworks(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/networks", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fakegopherc.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, GetNetworks)
	})

	serviceClient := fakegopherc.ServiceClient()
	networks, err := getAllNetworks(serviceClient)
	if err != nil {
		t.Fatalf("failed to get a list of all networks: %v", err)
	}
	for i, network := range networks {
		th.CheckDeepEquals(t, expectedNetworks[i], network.Network)
	}
}
