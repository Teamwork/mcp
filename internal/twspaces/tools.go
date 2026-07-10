package twspaces

import (
	"net/http"

	"github.com/teamwork/mcp/internal/toolsets"
)

const (
	spacesDescription        = "Space CRUD and collaborators in Teamwork Spaces."
	pagesDescription         = "Page CRUD, homepage, and duplication in Teamwork Spaces."
	spacesContentDescription = "Comments, tags, categories, and search in Teamwork Spaces."
)

// Sub-toolset keys for twspaces. These are the valid values for the
// -toolsets flag when selecting Teamwork Spaces functionality.
const (
	// ToolsetSpaces covers space CRUD and collaborators.
	ToolsetSpaces toolsets.Method = "twspaces-spaces"
	// ToolsetPages covers page CRUD, home, and duplication.
	ToolsetPages toolsets.Method = "twspaces-pages"
	// ToolsetContent covers comments, tags, categories, and search.
	ToolsetContent toolsets.Method = "twspaces-content"
)

func init() {
	toolsets.RegisterMethod(ToolsetSpaces)
	toolsets.RegisterMethod(ToolsetPages)
	toolsets.RegisterMethod(ToolsetContent)
}

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Spaces.
func DefaultToolsetGroup(readOnly, allowDelete bool, httpClient *http.Client) *toolsets.ToolsetGroup {
	group := toolsets.NewToolsetGroup(readOnly)

	// --- spaces sub-toolset ---
	spacesWriteTools := []toolsets.ToolWrapper{
		SpaceCreate(httpClient),
		SpaceUpdate(httpClient),
	}
	if allowDelete {
		spacesWriteTools = append(spacesWriteTools,
			SpaceDelete(httpClient),
		)
	}
	group.AddToolset(toolsets.NewToolset(ToolsetSpaces, spacesDescription).
		AddWriteTools(spacesWriteTools...).
		AddReadTools(
			SpaceGet(httpClient),
			SpaceList(httpClient),
			SpaceCollaborators(httpClient),
		))

	// --- pages sub-toolset ---
	pagesWriteTools := []toolsets.ToolWrapper{
		PageCreate(httpClient),
		PageDuplicate(httpClient),
		PageUpdate(httpClient),
	}
	if allowDelete {
		pagesWriteTools = append(pagesWriteTools,
			PageDelete(httpClient),
		)
	}
	group.AddToolset(toolsets.NewToolset(ToolsetPages, pagesDescription).
		AddWriteTools(pagesWriteTools...).
		AddReadTools(
			PageGet(httpClient),
			PageList(httpClient),
			PageHome(httpClient),
		))

	// --- content sub-toolset ---
	contentWriteTools := []toolsets.ToolWrapper{
		CommentCreate(httpClient),
		CommentUpdate(httpClient),
		TagCreateBatch(httpClient),
		TagUpdate(httpClient),
		CategoryCreate(httpClient),
		CategoryUpdate(httpClient),
	}
	if allowDelete {
		contentWriteTools = append(contentWriteTools,
			CommentDelete(httpClient),
			TagDelete(httpClient),
			CategoryDelete(httpClient),
		)
	}
	group.AddToolset(toolsets.NewToolset(ToolsetContent, spacesContentDescription).
		AddWriteTools(contentWriteTools...).
		AddReadTools(
			CommentGet(httpClient),
			CommentList(httpClient),
			TagGet(httpClient),
			TagList(httpClient),
			CategoryGet(httpClient),
			CategoryList(httpClient),
			Search(httpClient),
		))

	return group
}
