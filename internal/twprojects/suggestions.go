package twprojects

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

const (
	// maxSuggestions caps how many near-miss candidates a list tool reports. The
	// point is to name the entity the caller probably meant, not to hand back a
	// second result set.
	maxSuggestions = 5

	// suggestionSearchLimit is how many hits the near-miss lookup asks the search
	// endpoint for. It is above maxSuggestions because a hit the lookup cannot
	// name is skipped, and a term can match several of those before it matches
	// something nameable.
	suggestionSearchLimit = 20

	// minSuggestionSearchTerm is the shortest term the search endpoint accepts. A
	// shorter one is answered with a 400, so the lookup does not send it.
	minSuggestionSearchTerm = 3
)

// suggestionEntity describes one entity type the near-miss lookup can report.
// The sideload section, the reported type name, the fields to request and the
// label extraction live in one entry so a type cannot be half-wired: a section
// added to the search without a label would return records the lookup has to
// skip.
//
// Comments and timelogs are deliberately absent. Neither has a name — the
// search sideloads a comment's body and a timelog's description as their label —
// so neither answers "the thing you searched for is actually a …".
type suggestionEntity struct {
	// section is the sideload the search response carries these records in. It is
	// also the value a hit reports as its type, which is how a hit is matched back
	// to an entry.
	section projects.SearchRequestSideload

	// typeName is the singular type reported in a suggestion, spelled to match
	// the entity's own tools: a "project" candidate is read with
	// twprojects-get_project.
	typeName string

	// fields narrows the sideloaded record to what a suggestion needs, keeping
	// the discarded remainder of the response off the wire.
	fields func(*projects.SearchFields)

	// label returns the record's display name for the given sideload key, or an
	// empty string when the response does not carry it.
	label func(*projects.SearchResponse, string) string
}

var suggestionEntities = []suggestionEntity{{
	section:  projects.SearchRequestSideloadProjects,
	typeName: "project",
	fields: func(f *projects.SearchFields) {
		f.Projects = []projects.ProjectField{projects.ProjectFieldID, projects.ProjectFieldName}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Projects[key].Name },
}, {
	section:  projects.SearchRequestSideloadTasks,
	typeName: "task",
	fields: func(f *projects.SearchFields) {
		f.Tasks = []projects.TaskField{projects.TaskFieldID, projects.TaskFieldName}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Tasks[key].Name },
}, {
	section:  projects.SearchRequestSideloadTasklists,
	typeName: "tasklist",
	fields: func(f *projects.SearchFields) {
		f.Tasklists = []projects.TasklistField{projects.TasklistFieldID, projects.TasklistFieldName}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Tasklists[key].Name },
}, {
	section:  projects.SearchRequestSideloadMilestones,
	typeName: "milestone",
	fields: func(f *projects.SearchFields) {
		f.Milestones = []projects.MilestoneField{projects.MilestoneFieldID, projects.MilestoneFieldName}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Milestones[key].Name },
}, {
	section:  projects.SearchRequestSideloadNotebooks,
	typeName: "notebook",
	fields: func(f *projects.SearchFields) {
		f.Notebooks = []projects.NotebookField{projects.NotebookFieldID, projects.NotebookFieldName}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Notebooks[key].Name },
}, {
	section:  projects.SearchRequestSideloadMessages,
	typeName: "message",
	fields: func(f *projects.SearchFields) {
		f.Messages = []projects.MessageField{projects.MessageFieldID, projects.MessageFieldTitle}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Messages[key].Title },
}, {
	section:  projects.SearchRequestSideloadLinks,
	typeName: "link",
	fields: func(f *projects.SearchFields) {
		f.Links = []projects.LinkField{projects.LinkFieldID, projects.LinkFieldTitle}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Links[key].Title },
}, {
	section:  projects.SearchRequestSideloadCompanies,
	typeName: "company",
	fields: func(f *projects.SearchFields) {
		f.Companies = []projects.CompanyField{projects.CompanyFieldID, projects.CompanyFieldName}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Companies[key].Name },
}, {
	section:  projects.SearchRequestSideloadTeams,
	typeName: "team",
	fields: func(f *projects.SearchFields) {
		f.Teams = []projects.TeamField{projects.TeamFieldID, projects.TeamFieldName}
	},
	label: func(r *projects.SearchResponse, key string) string { return r.Included.Teams[key].Name },
}, {
	section:  projects.SearchRequestSideloadUsers,
	typeName: "user",
	fields: func(f *projects.SearchFields) {
		f.Users = []projects.UserField{projects.UserFieldID, projects.UserFieldFirstName, projects.UserFieldLastName}
	},
	label: func(r *projects.SearchResponse, key string) string {
		user := r.Included.Users[key]
		return strings.TrimSpace(user.FirstName + " " + user.LastName)
	},
}}

// Everything below is derived from suggestionEntities so the table stays the
// single source of truth for which types the lookup covers.
var (
	suggestionSideloads  []projects.SearchRequestSideload
	suggestionFields     projects.SearchFields
	suggestionsBySection map[string]suggestionEntity
	suggestionTypeEnum   []any
)

func init() {
	suggestionsBySection = make(map[string]suggestionEntity, len(suggestionEntities))
	for _, entity := range suggestionEntities {
		suggestionSideloads = append(suggestionSideloads, entity.section)
		entity.fields(&suggestionFields)
		suggestionsBySection[string(entity.section)] = entity
		suggestionTypeEnum = append(suggestionTypeEnum, entity.typeName)
	}
}

// searchSuggestion is a near-miss candidate: an entity whose name matches a
// search term that returned nothing in the entity type the caller listed.
type searchSuggestion struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// withSuggestionsSchema publishes the suggestions a list tool can attach to an
// empty result. It is applied at the OutputSchema line rather than in an init
// block because the schema var is shared with the matching get_* tool, which
// never returns suggestions.
func withSuggestionsSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil || schema.Properties == nil {
		return schema
	}
	schema.Properties["suggestions"] = &jsonschema.Schema{
		Type: "array",
		Description: "Present only when search_term was supplied and the result list came back empty. " +
			"Entities whose name matches the term, most relevant first, so the term can be recognised instead of " +
			"treated as unknown. A candidate may be of another type than the one listed, or of the same type but " +
			"excluded by this tool's filters — completed items are searched regardless of show_completed, so a " +
			"candidate says nothing about completion. Read it with the entity type's own get tool.",
		Items: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"type": {
					Type:        "string",
					Description: "The entity type of the candidate.",
					Enum:        suggestionTypeEnum,
				},
				"id":   {Type: "integer", Description: "The ID of the candidate."},
				"name": {Type: "string", Description: "The name of the candidate."},
			},
		},
	}
	return schema
}

// withNearMissSuggestions attaches up to maxSuggestions near-miss candidates to
// a raw list response whose result array is empty, and returns it otherwise
// untouched. listKey is the top-level attribute holding the list ("tasks").
//
// It must run after helpers.WebLinker: the linker walks every top-level array
// and would stamp each suggestion with a link built from the calling tool's own
// path prefix, pointing a project candidate at /app/tasks.
//
// A term the search endpoint would reject (under three characters, its floor)
// skips the lookup and returns the response as it stands. Any other failure —
// the search call, a body that does not decode or re-encode — is returned to
// the caller alongside the original body, to be reported like any other API
// failure: an empty list without suggestions then always means the lookup ran
// and found nothing, never that it silently failed.
func withNearMissSuggestions(
	ctx context.Context,
	engine *twapi.Engine,
	body []byte,
	listKey string,
	searchTerm string,
) ([]byte, error) {
	if utf8.RuneCountInString(searchTerm) < minSuggestionSearchTerm {
		return body, nil
	}

	var decoded map[string]any
	err := json.Unmarshal(body, &decoded)
	if err != nil {
		return body, err
	}
	// An absent key is not an empty result: it means this is not the list
	// response shape the lookup understands, and guessing would fire a search
	// against every unrelated payload.
	list, ok := decoded[listKey]
	if !ok {
		return body, nil
	}
	switch list := list.(type) {
	case nil:
	case []any:
		if len(list) > 0 {
			return body, nil
		}
	default:
		return body, nil
	}

	suggestions, err := nearMissSuggestions(ctx, engine, searchTerm)
	if err != nil {
		return body, err
	}
	if len(suggestions) == 0 {
		return body, nil
	}
	decoded["suggestions"] = suggestions

	encoded, err := json.Marshal(decoded)
	if err != nil {
		return body, err
	}

	return encoded, nil
}

// nearMissSuggestions runs one search for the term and returns the hits it can
// name, most relevant first, or the search's own error.
func nearMissSuggestions(ctx context.Context, engine *twapi.Engine, searchTerm string) ([]searchSuggestion, error) {
	var searchRequest projects.SearchRequest
	searchRequest.Filters.SearchTerm = searchTerm
	searchRequest.Filters.Limit = suggestionSearchLimit
	searchRequest.Filters.Include = suggestionSideloads
	searchRequest.Filters.Fields = suggestionFields
	// Filters.Types is deliberately left unset. The type the caller listed is the
	// one that just came back empty, so the lookup wants every other type the
	// term matches, and each hit reports its own.
	//
	// Completed items are asked for rather than inherited from the caller's own
	// filters. "It exists but it is finished" explains an empty result as well as
	// "it is a different kind of record" does, and it is the explanation the
	// caller's filters just hid: a list that excluded completed work is exactly
	// the one that came back empty. This is why a suggestion can name the type
	// the caller listed, and why it says nothing about completion — the caller
	// reads the record with the entity's own get tool.
	searchRequest.Filters.IncludeCompletedItems = new(true)

	response, err := projects.Search(ctx, engine, searchRequest)
	if err != nil {
		return nil, err
	}

	var suggestions []searchSuggestion
	for _, item := range response.Items {
		entity, ok := suggestionsBySection[item.Type]
		if !ok {
			continue
		}
		name := entity.label(response, strconv.FormatInt(item.ID, 10))
		if name == "" {
			continue
		}
		suggestions = append(suggestions, searchSuggestion{
			Type: entity.typeName,
			ID:   item.ID,
			Name: name,
		})
		if len(suggestions) == maxSuggestions {
			break
		}
	}
	return suggestions, nil
}
