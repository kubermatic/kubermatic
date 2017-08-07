package cluster

import (
	"fmt"
	"github.com/kubermatic/api"
	"testing"
)

func TestPendingCreateAddressesSuccessfully(t *testing.T) {
	_, cc := newTestController()
	c := &api.Cluster{
		Metadata: api.Metadata{
			Name: "testcluster",
		},
		Address: &api.ClusterAddress{},
	}

	changedC, err := cc.pendingCreateAddresses(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedExternalName := fmt.Sprintf("%s.%s.%s", c.Metadata.Name, TestDC, TestExternalURL)
	if changedC.Address.ExternalName != fmt.Sprintf("%s.%s.%s", c.Metadata.Name, TestDC, TestExternalURL) {
		t.Fatalf("external name is wrong. Expected=%s Got=%s", expectedExternalName, changedC.Address.ExternalName)
	}

	if changedC.Address.ExternalPort != TestExternalPort {
		t.Fatalf("external port is wrong. Expected=%d Got=%d", TestExternalPort, changedC.Address.ExternalPort)
	}

	expectedURL := fmt.Sprintf("https://%s:%d", changedC.Address.ExternalName, TestExternalPort)
	if changedC.Address.URL != expectedURL {
		t.Fatalf("url is wrong. Expected=%s Got=%s", expectedURL, changedC.Address.URL)
	}
}

func TestPendingCreateAddressesPartially(t *testing.T) {
	_, cc := newTestController()
	c := &api.Cluster{
		Metadata: api.Metadata{
			Name: "testcluster",
		},
		Address: &api.ClusterAddress{
			ExternalName: "foo.bar",
		},
	}

	changedC, err := cc.pendingCreateAddresses(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if changedC.Address.ExternalName != "foo.bar" {
		t.Fatalf("external got overwritten")
	}

	if changedC.Address.ExternalPort != TestExternalPort {
		t.Fatalf("external port is wrong. Expected=%d Got=%d", TestExternalPort, changedC.Address.ExternalPort)
	}

	expectedURL := fmt.Sprintf("https://%s:%d", changedC.Address.ExternalName, TestExternalPort)
	if changedC.Address.URL != expectedURL {
		t.Fatalf("url is wrong. Expected=%s Got=%s", expectedURL, changedC.Address.URL)
	}
}

func TestPendingCreateAddressesAlreadyExists(t *testing.T) {
	_, cc := newTestController()
	c := &api.Cluster{
		Metadata: api.Metadata{
			Name: "testcluster",
		},
		Address: &api.ClusterAddress{
			ExternalName: "foo.bar",
			URL:          "https://foo.bar:8443",
			ExternalPort: 8443,
		},
	}

	changedC, err := cc.pendingCreateAddresses(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if changedC != nil {
		t.Fatalf("returned cluster pointer to trigger update instead of nil")
	}

	if c.Address.ExternalName != "foo.bar" || c.Address.URL != "https://foo.bar:8443" || c.Address.ExternalPort != 8443 {
		t.Fatalf("address fields were overwritten")
	}
}
