package version

import (
	"testing"

	"github.com/Masterminds/semver"
)

func TestBestAutomaticUpdate(t *testing.T) {
	manager := New([]*MasterVersion{
		{
			Version: semver.MustParse("1.10.0"),
			Default: false,
		},
		{
			Version: semver.MustParse("1.10.1"),
			Default: true,
		},
	}, []*MasterUpdate{
		{
			From:      "1.10.0",
			To:        "1.10.1",
			Automatic: true,
		},
	})

	updateVersion, err := manager.AutomaticUpdate("1.10.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if updateVersion.Version.String() != "1.10.1" {
		t.Fatalf("Unexpected update version to be 1.10.1. Got=%v", updateVersion)
	}
}

func TestBestAutomaticUpdateWildCard(t *testing.T) {
	manager := New([]*MasterVersion{
		{
			Version: semver.MustParse("1.10.0"),
			Default: false,
		},
		{
			Version: semver.MustParse("1.10.1"),
			Default: true,
		},
	}, []*MasterUpdate{
		{
			From:      "1.10.*",
			To:        "1.10.1",
			Automatic: true,
		},
	})

	updateVersion, err := manager.AutomaticUpdate("1.10.0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if updateVersion.Version.String() != "1.10.1" {
		t.Fatalf("Unexpected update version to be 1.10.1. Got=%v", updateVersion)
	}
}
