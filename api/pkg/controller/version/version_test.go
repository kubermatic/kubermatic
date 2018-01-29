package version

import (
	"testing"

	"github.com/go-test/deep"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

func TestBestAutomaticUpdate(t *testing.T) {
	version := "1.5.0"

	expected := apiv1.MasterUpdate{
		From:      "1.5.0",
		To:        "1.5.3",
		Automatic: true,
	}
	updateVersion, err := BestAutomaticUpdate(version,
		[]apiv1.MasterUpdate{
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
		t.Fatalf("Unexpected update path: expected=%v, got=%v, diff=%v", expected, updateVersion, diff)
	}

}

func TestBestAutomaticUpdateWildCard(t *testing.T) {
	version := "1.5.1"

	expected := apiv1.MasterUpdate{
		From:      "1.5.*",
		To:        "1.5.3",
		Automatic: true,
	}
	updateVersion, err := BestAutomaticUpdate(version,
		[]apiv1.MasterUpdate{
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
		t.Fatalf("Unexpected update path: expected=%v, got=%v, diff=%v", expected, updateVersion, diff)
	}

}

func TestBestAutomaticUpdateWildCardMultiValid(t *testing.T) {
	version := "1.5.1"

	expected := apiv1.MasterUpdate{
		From:      "1.5.*",
		To:        "1.6.0",
		Automatic: true,
	}
	updateVersion, err := BestAutomaticUpdate(version,
		[]apiv1.MasterUpdate{
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
		t.Fatalf("Unexpected update path: expected=%v, got=%v, diff=%v", expected, updateVersion, diff)
	}
}
