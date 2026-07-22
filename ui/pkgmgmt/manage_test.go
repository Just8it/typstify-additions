package pkgmgmt

import (
	"testing"

	"github.com/typstify/tpix-cli/api"
	"looz.ws/typstify/typst/pkg"
)

func TestFilterLocalPackages(t *testing.T) {
	packages := []pkg.TypstPkg{
		{SearchResult: api.SearchResult{Name: "zeta", IsTemplate: true}},
		{SearchResult: api.SearchResult{Name: "alpha", Description: "Lecture notes", IsTemplate: true}},
		{SearchResult: api.SearchResult{Name: "package", IsTemplate: false}},
	}

	all := filterLocalPackages(packages, "")
	if len(all) != 3 || all[0].Name != "alpha" || all[1].Name != "package" || all[2].Name != "zeta" {
		t.Fatalf("unexpected local packages: %#v", all)
	}

	matched := filterLocalPackages(packages, "NOTES")
	if len(matched) != 1 || matched[0].Name != "alpha" {
		t.Fatalf("unexpected filtered templates: %#v", matched)
	}
}
