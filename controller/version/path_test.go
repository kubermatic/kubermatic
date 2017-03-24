package version

import (
	"reflect"
	"testing"

	"github.com/kubermatic/api"
)

func TestPathSearch_Search(t *testing.T) {
	search := NewUpdatePathSearch(
		map[string]*api.MasterVersion{
			"1.5.1": {
				ID: "1.5.1",
			},
			"1.5.2": {
				ID: "1.5.2",
			},
			"1.5.3": {
				ID: "1.5.3",
			},
			"1.5.4": {
				ID: "1.5.4",
			},
		}, []api.MasterUpdate{
			{
				From: "1.5.1",
				To:   "1.5.2",
			},
			{
				From: "1.5.2",
				To:   "1.5.3",
			},
			{
				From: "1.5.2",
				To:   "1.5.3",
			},
			{
				From: "1.5.3",
				To:   "1.5.4",
			},
		}, EqualityMatcher{},
	)

	p, err := search.Search("1.5.2", "1.5.4")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	versions := []string{"1.5.2"}
	for _, vu := range p {
		versions = append(versions, vu.To)
	}

	expected := []string{"1.5.2", "1.5.3", "1.5.4"}
	if !reflect.DeepEqual(versions, expected) {
		t.Fatalf("Unexpected update path: expected=%v, got=%v", expected, versions)
	}
}

func TestPathSearch_SemverSearch(t *testing.T) {
	search := NewUpdatePathSearch(
		map[string]*api.MasterVersion{
			"1.5.1": {
				ID: "1.5.1",
			},
			"1.5.2": {
				ID: "1.5.2",
			},
			"1.5.3": {
				ID: "1.5.3",
			},
			"1.5.4": {
				ID: "1.5.4",
			},
		}, []api.MasterUpdate{
			{
				From: "1.5.1",
				To:   "1.5.2",
			},
			{
				From: "1.5.2",
				To:   "1.5.3",
			},
			{
				From: "1.5.2",
				To:   "1.5.3",
			},
			{
				From: "~1.5.x",
				To:   "1.5.4",
			},
			{
				From: "1.5.3",
				To:   "1.5.4",
			},
		}, SemverMatcher{},
	)

	p, err := search.Search("1.5.2", "1.5.4")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	versions := []string{"1.5.2"}
	for _, vu := range p {
		versions = append(versions, vu.To)
	}

	expected := []string{"1.5.2", "1.5.4"}
	if !reflect.DeepEqual(versions, expected) {
		t.Fatalf("Unexpected update path: expected=%v, got=%v", expected, versions)
	}
}
