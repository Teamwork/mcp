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

func TestVerboseSchema(t *testing.T) {
	t.Parallel()

	got := helpers.VerboseSchema()
	if !strings.Contains(got.Description, "id + name only") {
		t.Errorf("VerboseSchema description = %q, want it to mention 'id + name only'", got.Description)
	}
	if got.AnyOf[0].Type != "boolean" {
		t.Errorf("VerboseSchema first AnyOf type = %q, want boolean", got.AnyOf[0].Type)
	}
	if string(got.Default) != "true" {
		t.Errorf("VerboseSchema Default = %q, want true", string(got.Default))
	}
}

func TestMatchAllTagsSchema(t *testing.T) {
	t.Parallel()

	got := helpers.MatchAllTagsSchema()
	if !strings.Contains(got.Description, "match all tags") ||
		!strings.Contains(got.Description, "match any") {
		t.Errorf("MatchAllTagsSchema description missing expected phrases: %q", got.Description)
	}
	if got.AnyOf[0].Type != "boolean" {
		t.Errorf("MatchAllTagsSchema first AnyOf type = %q, want boolean", got.AnyOf[0].Type)
	}
	if string(got.Default) != "false" {
		t.Errorf("MatchAllTagsSchema Default = %q, want false", string(got.Default))
	}
}

func TestUserGroupsSchema(t *testing.T) {
	t.Parallel()

	t.Run("required", func(t *testing.T) {
		got := helpers.UserGroupsSchema("Assignees for the task.", true)
		if got.Type != "object" {
			t.Errorf("required UserGroupsSchema type = %q, want object", got.Type)
		}
		if got.Description != "Assignees for the task." {
			t.Errorf("required UserGroupsSchema description = %q", got.Description)
		}
		if got.MinProperties == nil || *got.MinProperties != 1 {
			t.Errorf("required UserGroupsSchema MinProperties = %v, want 1", got.MinProperties)
		}
		if got.MaxProperties == nil || *got.MaxProperties != 4 {
			t.Errorf("required UserGroupsSchema MaxProperties = %v, want 4", got.MaxProperties)
		}
		for _, key := range []string{"user_ids", "company_ids", "team_ids", "job_role_ids"} {
			prop, ok := got.Properties[key]
			if !ok {
				t.Errorf("required UserGroupsSchema missing %q", key)
				continue
			}
			if prop.Type != "array" {
				t.Errorf("required UserGroupsSchema %q type = %q, want array", key, prop.Type)
			}
			if prop.Items == nil || prop.Items.Type != "integer" {
				t.Errorf("required UserGroupsSchema %q items = %+v, want integer", key, prop.Items)
			}
			if prop.MinItems == nil || *prop.MinItems != 1 {
				t.Errorf("required UserGroupsSchema %q MinItems = %v, want 1", key, prop.MinItems)
			}
		}
		if len(got.AnyOf) != 4 {
			t.Fatalf("required UserGroupsSchema AnyOf len = %d, want 4", len(got.AnyOf))
		}
	})

	t.Run("optional", func(t *testing.T) {
		got := helpers.UserGroupsSchema("Followers of task changes.", false)
		if got.Type != "" {
			t.Errorf("optional UserGroupsSchema outer type = %q, want empty", got.Type)
		}
		if got.Description != "Followers of task changes." {
			t.Errorf("optional UserGroupsSchema description = %q", got.Description)
		}
		if len(got.AnyOf) != 2 {
			t.Fatalf("optional UserGroupsSchema AnyOf len = %d, want 2", len(got.AnyOf))
		}
		if got.AnyOf[0].Type != "object" {
			t.Errorf("optional UserGroupsSchema AnyOf[0].Type = %q, want object", got.AnyOf[0].Type)
		}
		if got.AnyOf[1].Type != "null" {
			t.Errorf("optional UserGroupsSchema AnyOf[1].Type = %q, want null", got.AnyOf[1].Type)
		}
	})
}

func TestDateTimeFilterSchema(t *testing.T) {
	t.Parallel()

	got := helpers.DateTimeFilterSchema("Filter tasks created after.")
	if got.Description != "Filter tasks created after." {
		t.Errorf("DateTimeFilterSchema description = %q", got.Description)
	}
	if len(got.AnyOf) != 2 {
		t.Fatalf("DateTimeFilterSchema AnyOf len = %d, want 2", len(got.AnyOf))
	}
	if got.AnyOf[0].Type != "string" || got.AnyOf[0].Format != "date-time" {
		t.Errorf("DateTimeFilterSchema AnyOf[0] = %+v, want string/date-time", got.AnyOf[0])
	}
	if got.AnyOf[1].Type != "null" {
		t.Errorf("DateTimeFilterSchema AnyOf[1].Type = %q, want null", got.AnyOf[1].Type)
	}
}

func TestDateFilterSchema(t *testing.T) {
	t.Parallel()

	got := helpers.DateFilterSchema("Start of the workload period.")
	if got.Description != "Start of the workload period." {
		t.Errorf("DateFilterSchema description = %q", got.Description)
	}
	if len(got.AnyOf) != 2 {
		t.Fatalf("DateFilterSchema AnyOf len = %d, want 2", len(got.AnyOf))
	}
	if got.AnyOf[0].Type != "string" || got.AnyOf[0].Format != "date" {
		t.Errorf("DateFilterSchema AnyOf[0] = %+v, want string/date", got.AnyOf[0])
	}
	if got.AnyOf[1].Type != "null" {
		t.Errorf("DateFilterSchema AnyOf[1].Type = %q, want null", got.AnyOf[1].Type)
	}
}
