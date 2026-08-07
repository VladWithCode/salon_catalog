package db

import "testing"

// TestCatalogProductIdentifierKind fixes, in code and tests, the exact
// contract 06-product-page.md documented by reading
// FindCatalogProductDetail: a path segment that parses as a UUID resolves
// against products.id; anything else resolves against products.slug. This
// is the single resolver both /catalogo/producto/{id} and /productos/{id}
// (the QR-compat route) depend on.
func TestCatalogProductIdentifierKind(t *testing.T) {
	for _, testCase := range []struct {
		name string
		id   string
		want catalogProductIdentifierKindT
	}{
		{name: "canonical UUID", id: "01890f3a-dc02-7cb5-a4cc-451231879f0b", want: catalogProductIdentifierUUID},
		{name: "nil UUID is still a valid UUID", id: "00000000-0000-0000-0000-000000000000", want: catalogProductIdentifierUUID},
		{name: "uppercase UUID", id: "01890F3A-DC02-7CB5-A4CC-451231879F0B", want: catalogProductIdentifierUUID},
		{name: "plain slug", id: "mesa-redonda", want: catalogProductIdentifierSlug},
		{name: "slug with digits", id: "silla-tiffany-2026", want: catalogProductIdentifierSlug},
		{name: "malformed uuid-shaped string falls back to slug", id: "01890f3a-dc02-7cb5-a4cc-45123187", want: catalogProductIdentifierSlug},
		{name: "empty string is a slug, not a UUID", id: "", want: catalogProductIdentifierSlug},
		{name: "uuid with stray characters is a slug", id: "01890f3a-dc02-7cb5-a4cc-451231879f0b-extra", want: catalogProductIdentifierSlug},
		{name: "unicode slug", id: "cojín-decorativo", want: catalogProductIdentifierSlug},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := catalogProductIdentifierKind(testCase.id); got != testCase.want {
				t.Fatalf("catalogProductIdentifierKind(%q) = %v, want %v", testCase.id, got, testCase.want)
			}
		})
	}
}
