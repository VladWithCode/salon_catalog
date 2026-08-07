package db

import (
	"strings"
	"testing"
)

// The text search configuration must be an SQL string literal. Double quotes
// make PostgreSQL read "spanish" as an identifier, which fails at execution
// time for every non-empty search term.
const (
	catalogSearchConfigLiteral    = "'spanish'"
	catalogSearchConfigIdentifier = `"spanish"`
)

func TestBuildCatalogProductQueryConditionsQuotesSearchConfigAsLiteral(t *testing.T) {
	conditions, namedArgs := buildCatalogProductQueryConditions(CatalogProductFilterParams{
		Search:     "mesa",
		SearchMode: SearchModeFullText,
	})

	condition := strings.Join(conditions, " AND ")
	if !strings.Contains(condition, "plainto_tsquery("+catalogSearchConfigLiteral+", @search_query)") {
		t.Fatalf("expected plainto_tsquery with a literal search config, got: %s", condition)
	}
	if strings.Contains(condition, catalogSearchConfigIdentifier) {
		t.Fatalf("search config must not be quoted as an identifier, got: %s", condition)
	}
	if namedArgs["search_query"] != "mesa" {
		t.Fatalf("expected search_query named argument to carry the raw term, got: %v", namedArgs["search_query"])
	}
}

func TestBuildCatalogProductSearchRankSelectQuotesSearchConfigAsLiteral(t *testing.T) {
	rankSelect := buildCatalogProductSearchRankSelect(CatalogProductFilterParams{
		Search:     "mesa",
		SearchMode: SearchModeFullText,
	})

	if !strings.Contains(rankSelect, "plainto_tsquery("+catalogSearchConfigLiteral+", @search_query)") {
		t.Fatalf("expected ranking to use a literal search config, got: %s", rankSelect)
	}
	if strings.Contains(rankSelect, catalogSearchConfigIdentifier) {
		t.Fatalf("search config must not be quoted as an identifier, got: %s", rankSelect)
	}
}

func TestBuildCatalogProductQueryConditionsWithoutSearchIsUnchanged(t *testing.T) {
	conditions, namedArgs := buildCatalogProductQueryConditions(CatalogProductFilterParams{
		SearchMode: SearchModeFullText,
	})

	if len(conditions) != 0 {
		t.Fatalf("expected no conditions for an empty search, got: %v", conditions)
	}
	if _, ok := namedArgs["search_query"]; ok {
		t.Fatal("expected no search_query named argument for an empty search")
	}

	rankSelect := buildCatalogProductSearchRankSelect(CatalogProductFilterParams{
		SearchMode: SearchModeFullText,
	})
	if rankSelect != "0 as search_rank" {
		t.Fatalf("expected a constant rank column for an empty search, got: %s", rankSelect)
	}
}

func TestBuildCatalogProductQueryConditionsKeepsCategoryFilter(t *testing.T) {
	conditions, namedArgs := buildCatalogProductQueryConditions(CatalogProductFilterParams{
		Categories: []string{"11111111-1111-1111-1111-111111111111"},
		SearchMode: SearchModeFullText,
	})

	condition := strings.Join(conditions, " AND ")
	if !strings.Contains(condition, "category_id = ANY(@categories)") {
		t.Fatalf("expected the category filter to be preserved, got: %s", condition)
	}
	if _, ok := namedArgs["categories"]; !ok {
		t.Fatal("expected the categories named argument to be present")
	}
}

func TestBuildCatalogProductOrderByClauseIsUnchanged(t *testing.T) {
	withSearch := buildCatalogProductOrderByClause(CatalogProductFilterParams{
		Search:     "mesa",
		SearchMode: SearchModeFullText,
	})
	if withSearch != "ORDER BY search_rank DESC, name ASC" {
		t.Fatalf("unexpected ordering for a full-text search: %s", withSearch)
	}

	withoutSearch := buildCatalogProductOrderByClause(CatalogProductFilterParams{
		SearchMode: SearchModeFullText,
	})
	if withoutSearch != "ORDER BY name ASC" {
		t.Fatalf("unexpected ordering without a search term: %s", withoutSearch)
	}
}
