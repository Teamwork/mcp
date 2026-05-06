package helpers_test

import (
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/helpers"
)

func TestPaginationSchemas(t *testing.T) {
	t.Parallel()

	page := helpers.PageSchema()
	if page.Description == "" {
		t.Error("PageSchema description must not be empty")
	}
	if len(page.AnyOf) != 2 || page.AnyOf[0].Type != "integer" || page.AnyOf[1].Type != "null" {
		t.Errorf("PageSchema AnyOf = %+v, want integer/null", page.AnyOf)
	}

	size := helpers.PageSizeSchema()
	if size.Description == "" {
		t.Error("PageSizeSchema description must not be empty")
	}
	if len(size.AnyOf) != 2 || size.AnyOf[0].Type != "integer" || size.AnyOf[1].Type != "null" {
		t.Errorf("PageSizeSchema AnyOf = %+v, want integer/null", size.AnyOf)
	}

	offset := helpers.PageOffsetSchema()
	if !strings.Contains(offset.Description, "index") {
		t.Errorf("PageOffsetSchema description = %q, want it to mention an index", offset.Description)
	}
}

func TestOrderingSchemas(t *testing.T) {
	t.Parallel()

	by := helpers.OrderBySchema()
	if by.AnyOf[0].Type != "string" {
		t.Errorf("OrderBySchema first AnyOf type = %q, want string", by.AnyOf[0].Type)
	}

	dir := helpers.OrderDirectionSchema()
	if !strings.Contains(dir.Description, "asc") || !strings.Contains(dir.Description, "desc") {
		t.Errorf("OrderDirectionSchema description = %q, want asc/desc", dir.Description)
	}
}

func TestSearchTermSchema(t *testing.T) {
	t.Parallel()

	got := helpers.SearchTermSchema("companies", "name")
	want := "A search term to filter companies by name."
	if got.Description != want {
		t.Errorf("SearchTermSchema description = %q, want %q", got.Description, want)
	}
	if got.AnyOf[0].Type != "string" {
		t.Errorf("SearchTermSchema first AnyOf type = %q, want string", got.AnyOf[0].Type)
	}
}

func TestTagIDsSchemas(t *testing.T) {
	t.Parallel()

	filter := helpers.TagIDsFilterSchema("projects")
	if filter.Description != "A list of tag IDs to filter projects by tags." {
		t.Errorf("TagIDsFilterSchema description = %q", filter.Description)
	}
	if filter.AnyOf[0].Type != "array" || filter.AnyOf[0].Items.Type != "integer" {
		t.Errorf("TagIDsFilterSchema items shape unexpected: %+v", filter.AnyOf[0])
	}

	assoc := helpers.TagIDsAssociateSchema("project")
	if assoc.Description != "A list of tag IDs to associate with the project." {
		t.Errorf("TagIDsAssociateSchema description = %q", assoc.Description)
	}
}

func TestMatchAllTagsSchema(t *testing.T) {
	t.Parallel()

	got := helpers.MatchAllTagsSchema("tasks")
	if !strings.Contains(got.Description, "tasks that have all the specified tags") ||
		!strings.Contains(got.Description, "tasks that have any of the specified tags") ||
		!strings.Contains(got.Description, "Defaults to false") {
		t.Errorf("MatchAllTagsSchema description missing expected phrases: %q", got.Description)
	}
	if got.AnyOf[0].Type != "boolean" {
		t.Errorf("MatchAllTagsSchema first AnyOf type = %q, want boolean", got.AnyOf[0].Type)
	}
}
