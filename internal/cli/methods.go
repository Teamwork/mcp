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
	toolsets.RegisterProfiles([]string{"support", "desk"}, []toolsets.Method{
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
type Methods struct {
	profiles []string
	toolsets []toolsets.Method
}

// NewMethods creates a new Methods instance with the specified initial methods.
func NewMethods(initial ...toolsets.Method) *Methods {
	return &Methods{
		toolsets: initial,
	}
}

// String returns a comma-separated string representation of the Methods slice.
func (m Methods) String() string {
	methods := make([]string, len(m.toolsets))
	for i, method := range m.toolsets {
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
	m.toolsets = m.toolsets[:0] // reset slice

	var errs error
	for token := range strings.SplitSeq(value, ",") {
		token = strings.TrimSpace(token)
		// expand named profiles into their constituent methods
		if profileMethods, ok := toolsets.LookupProfile(token); ok {
			m.toolsets = append(m.toolsets, profileMethods...)
			m.profiles = append(m.profiles, token)
			continue
		}
		if method := toolsets.Method(token); method.IsRegistered() {
			m.toolsets = append(m.toolsets, method)
		} else {
			errs = errors.Join(errs, fmt.Errorf(`invalid toolset: %q (use a sub-toolset key like "twprojects-tasks", a `+
				`profile like "project-manager", or "all")`, token))
		}
	}
	return errs
}

// Toolsets returns the list of enabled toolset methods.
func (m *Methods) Toolsets() []toolsets.Method {
	return m.toolsets
}

// Profiles returns the list of enabled profiles.
func (m *Methods) Profiles() []string {
	return m.profiles
}
