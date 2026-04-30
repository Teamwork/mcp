package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
)

func init() {
	toolsets.RegisterProfile("project-manager", []toolsets.Method{
		twprojects.ToolsetProjects,
		twprojects.ToolsetTasks,
		twprojects.ToolsetPeople,
		twprojects.ToolsetContent,
	})
	toolsets.RegisterProfile("support", []toolsets.Method{
		twdesk.ToolsetTickets,
		twdesk.ToolsetCustomers,
	})
	toolsets.RegisterProfile("analyst", []toolsets.Method{
		twprojects.ToolsetProjects,
		twprojects.ToolsetTasks,
		twprojects.ToolsetPeople,
		twprojects.ToolsetTime,
		twprojects.ToolsetContent,
		twdesk.ToolsetTickets,
		twdesk.ToolsetCustomers,
		twdesk.ToolsetAdmin,
	})
	toolsets.RegisterProfile("knowledge-manager", []toolsets.Method{
		twspaces.ToolsetSpaces,
		twspaces.ToolsetPages,
		twspaces.ToolsetContent,
	})
	toolsets.RegisterProfile("ops", []toolsets.Method{
		toolsets.MethodAll,
	})
}

// Methods is a slice of toolsets.Method that implements the flag.Value
// interface, allowing it to be used as a command-line flag type.
type Methods []toolsets.Method

// String returns a comma-separated string representation of the Methods slice.
func (m Methods) String() string {
	methods := make([]string, len(m))
	for i, method := range m {
		methods[i] = method.String()
	}
	return strings.Join(methods, ", ")
}

// Set parses a comma-separated string of method names and updates the Methods
// slice. It supports individual method names, as well as profile names that
// expand into multiple methods. If an invalid method or profile name is
// encountered, it returns an error.
func (m *Methods) Set(value string) error {
	if value == "" || m == nil {
		return nil
	}
	*m = (*m)[:0] // reset slice

	var errs error
	for token := range strings.SplitSeq(value, ",") {
		token = strings.TrimSpace(token)
		// expand named profiles into their constituent methods
		if profileMethods, ok := toolsets.LookupProfile(token); ok {
			*m = append(*m, profileMethods...)
			continue
		}
		if method := toolsets.Method(token); method.IsRegistered() {
			*m = append(*m, method)
		} else {
			errs = errors.Join(errs, fmt.Errorf(`invalid toolset: %q (use a sub-toolset key like "twprojects-tasks", a `+
				`profile like "project-manager", or "all")`, token))
		}
	}
	return errs
}
