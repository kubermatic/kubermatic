package version

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/kubermatic/api"
)

func TestBestAutomaticUpdate(t *testing.T) {
	version := "1.5.0"

	expected := api.MasterUpdate{
		From:      "1.5.0",
		To:        "1.5.3",
		Automatic: true,
	}
	updateVersion, err := BestAutomaticUpdate(version,
		[]api.MasterUpdate{
			expected,
			{
				From: "1.5.5",
				To:   "1.6.0",
			},
		})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if diff := deep.Equal(updateVersion, &expected); diff != nil {
		t.Fatalf("Unexpected update path: expected=%v, got=%v, diff=%v", expected, updateVersion)
	}

}

func TestBestAutomaticUpdateWildCard(t *testing.T) {
	version := "1.5.1"

	expected := api.MasterUpdate{
		From:      "1.5.*",
		To:        "1.5.3",
		Automatic: true,
	}
	updateVersion, err := BestAutomaticUpdate(version,
		[]api.MasterUpdate{
			expected,
			{
				From: "1.5.5",
				To:   "1.6.0",
			},
		})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if diff := deep.Equal(updateVersion, &expected); diff != nil {
		t.Fatalf("Unexpected update path: expected=%v, got=%v, diff=%v", expected, updateVersion)
	}

}

func TestBestAutomaticUpdateWildCardMultiValid(t *testing.T) {
	version := "1.5.1"

	expected := api.MasterUpdate{
		From:      "1.5.*",
		To:        "1.6.0",
		Automatic: true,
	}
	updateVersion, err := BestAutomaticUpdate(version,
		[]api.MasterUpdate{
			expected,
			{
				From:      "1.5.*",
				To:        "1.5.3",
				Automatic: true,
			},
			{
				From:      "1.5.3",
				To:        "1.5.5",
				Automatic: true,
			},
		})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if diff := deep.Equal(updateVersion, &expected); diff != nil {
		t.Fatalf("Unexpected update path: expected=%v, got=%v, diff=%v", expected, updateVersion)
	}
}
