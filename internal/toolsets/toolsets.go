// Package toolsets provides a framework for managing collections of tools. This
// was heavily inspired by GitHub's MCP server implementation:
//
//	https://github.com/github/github-mcp-server/blob/3341e6bc461b461f0789518879f97bbd86ef7ee9/pkg/toolsets/toolsets.go
package toolsets

import (
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	registeredMethods      = make(map[Method]struct{})
	registeredMethodsMutex sync.RWMutex
)

// Method identifies the name of a logical unit of operation or action that can
// be executed as part of a pipeline step or invoked via tool calling from an
// LLM.
type Method string

// MethodAll is a special method that can be used to indicate that all Toolsets
// should be enabled. It is not a valid method for any specific Toolset, but
// rather a convenience to enable all Toolsets in a ToolsetGroup at once.
const MethodAll Method = "all"

// String returns the string representation of the Method.
func (m Method) String() string {
	return string(m)
}

// IsRegistered checks if the method is registered.
func (m Method) IsRegistered() bool {
	registeredMethodsMutex.RLock()
	defer registeredMethodsMutex.RUnlock()
	_, exists := registeredMethods[m]
	return exists || m == MethodAll
}

// RegisterMethod registers a method. This is used to validate that only known
// methods are used when enabling Toolsets.
func RegisterMethod(method Method) {
	registeredMethodsMutex.Lock()
	defer registeredMethodsMutex.Unlock()
	registeredMethods[method] = struct{}{}
}

// ToolsetDoesNotExistError is an error type that indicates a requested toolset
// does not exist in the toolset group.
type ToolsetDoesNotExistError struct {
	Method Method
}

// NewToolsetDoesNotExistError creates a new ToolsetDoesNotExistError with the
// given method.
func NewToolsetDoesNotExistError(method Method) *ToolsetDoesNotExistError {
	return &ToolsetDoesNotExistError{
		Method: method,
	}
}

// Error implements the error interface for ToolsetDoesNotExistError.
func (e *ToolsetDoesNotExistError) Error() string {
	return fmt.Sprintf("toolset %q does not exist", e.Method)
}

// Is checks if the error is of type ToolsetDoesNotExistError.
func (e *ToolsetDoesNotExistError) Is(target error) bool {
	if target == nil {
		return false
	}
	_, ok := target.(*ToolsetDoesNotExistError)
	return ok
}

// ServerResourceTemplate represents a resource template that can be registered
// with the MCP server.
type ServerResourceTemplate struct {
	resourceTemplate *mcp.ResourceTemplate
	handler          mcp.ResourceHandler
}

// NewServerResourceTemplate creates a new ServerResourceTemplate with the given
// resource template and handler function.
func NewServerResourceTemplate(
	resourceTemplate *mcp.ResourceTemplate,
	handler mcp.ResourceHandler,
) ServerResourceTemplate {
	return ServerResourceTemplate{
		resourceTemplate: resourceTemplate,
		handler:          handler,
	}
}

// ServerPrompt represents a prompt that can be registered with the MCP server.
type ServerPrompt struct {
	Prompt  *mcp.Prompt
	Handler mcp.PromptHandler
}

// NewServerPrompt creates a new ServerPrompt with the given prompt and handler
// function.
func NewServerPrompt(prompt *mcp.Prompt, handler mcp.PromptHandler) ServerPrompt {
	return ServerPrompt{
		Prompt:  prompt,
		Handler: handler,
	}
}

// ToolWrapper is a simple struct that wraps an MCP tool and its handler.
type ToolWrapper struct {
	Tool *mcp.Tool

	// Ideally we would use mcp.TooHandlerFor for easier parsing of parameters and
	// error handling, but it would require loads of changes from the existing
	// structure. So for now we just use the raw handler.
	//
	// https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk@v1.0.0/mcp#ToolHandlerFor
	Handler mcp.ToolHandler
}

// Toolset represents a collection of MCP functionality that can be enabled or
// disabled as a group.
type Toolset struct {
	Method      Method
	Description string
	Enabled     bool
	readOnly    bool
	writeTools  []ToolWrapper
	readTools   []ToolWrapper
	// resources are not tools, but the community seems to be moving towards
	// namespaces as a broader concept and in order to have multiple servers
	// running concurrently, we want to avoid overlapping resources too.
	resourceTemplates []ServerResourceTemplate
	// prompts are also not tools but are namespaced similarly
	prompts []ServerPrompt
}

// NewToolset creates a new Toolset with the given method and description. The
// Toolset is initially disabled and not in read-only mode.
func NewToolset(method Method, description string) *Toolset {
	return &Toolset{
		Method:      method,
		Description: description,
		Enabled:     false,
		readOnly:    false,
	}
}

// GetActiveTools returns the tools that are currently active in the
// Toolset. If the Toolset is enabled, it returns both read and write tools.
// If the Toolset is not enabled, it returns nil.
func (t *Toolset) GetActiveTools() []ToolWrapper {
	if t.Enabled {
		if t.readOnly {
			return t.readTools
		}
		return append(t.readTools, t.writeTools...)
	}
	return nil
}

// GetAvailableTools returns the tools that are available in the Toolset.
func (t *Toolset) GetAvailableTools() []ToolWrapper {
	if t.readOnly {
		return t.readTools
	}
	return append(t.readTools, t.writeTools...)
}

// RegisterTools registers the tools in the Toolset with the MCP server.
func (t *Toolset) RegisterTools(s *mcp.Server) {
	if !t.Enabled {
		return
	}
	for _, toolWrapper := range t.readTools {
		s.AddTool(toolWrapper.Tool, toolWrapper.Handler)
	}
	if !t.readOnly {
		for _, tool := range t.writeTools {
			s.AddTool(tool.Tool, tool.Handler)
		}
	}
}

// AddResourceTemplates adds resource templates to the Toolset. These templates
// can be used to define resources that the MCP server can manage.
func (t *Toolset) AddResourceTemplates(templates ...ServerResourceTemplate) *Toolset {
	t.resourceTemplates = append(t.resourceTemplates, templates...)
	return t
}

// AddPrompts adds prompts to the Toolset. These prompts can be used to define
// interactions that the MCP server can handle.
func (t *Toolset) AddPrompts(prompts ...ServerPrompt) *Toolset {
	t.prompts = append(t.prompts, prompts...)
	return t
}

// GetActiveResourceTemplates returns the resource templates that are currently
// active in the Toolset. If the Toolset is enabled, it returns all resource
// templates.
func (t *Toolset) GetActiveResourceTemplates() []ServerResourceTemplate {
	if !t.Enabled {
		return nil
	}
	return t.resourceTemplates
}

// GetAvailableResourceTemplates returns the resource templates that are
// available in the Toolset. This includes all resource templates regardless of
// whether the Toolset is enabled or not.
func (t *Toolset) GetAvailableResourceTemplates() []ServerResourceTemplate {
	return t.resourceTemplates
}

// RegisterResourcesTemplates registers the resource templates in the Toolset
// with the MCP server.
func (t *Toolset) RegisterResourcesTemplates(s *mcp.Server) {
	if !t.Enabled {
		return
	}
	for _, resource := range t.resourceTemplates {
		s.AddResourceTemplate(resource.resourceTemplate, resource.handler)
	}
}

// RegisterPrompts registers the prompts in the Toolset with the MCP server.
func (t *Toolset) RegisterPrompts(s *mcp.Server) {
	if !t.Enabled {
		return
	}
	for _, prompt := range t.prompts {
		s.AddPrompt(prompt.Prompt, prompt.Handler)
	}
}

// SetReadOnly sets the Toolset to read-only mode. In this mode, only read tools
// can be added, and write tools will be ignored if attempted to be added.
func (t *Toolset) SetReadOnly() {
	// Set the toolset to read-only
	t.readOnly = true
}

// AddWriteTools adds write tools to the Toolset. If the Toolset is read-only,
// this method will silently ignore the tools to avoid breaching the read-only
// contract. If a tool is incorrectly annotated as read-only, it will panic.
func (t *Toolset) AddWriteTools(tools ...ToolWrapper) *Toolset {
	// Silently ignore if the toolset is read-only to avoid any breach of that contract
	for _, tool := range tools {
		if tool.Tool.Annotations.ReadOnlyHint {
			panic(fmt.Sprintf("tool (%s) is incorrectly annotated as read-only", tool.Tool.Name))
		}
	}
	if !t.readOnly {
		t.writeTools = append(t.writeTools, tools...)
	}
	return t
}

// AddReadTools adds read tools to the Toolset. It will panic if any tool is not
// annotated as read-only.
func (t *Toolset) AddReadTools(tools ...ToolWrapper) *Toolset {
	for _, tool := range tools {
		if !tool.Tool.Annotations.ReadOnlyHint {
			panic(fmt.Sprintf("tool (%s) must be annotated as read-only", tool.Tool.Name))
		}
	}
	t.readTools = append(t.readTools, tools...)
	return t
}

// ToolsetGroup is a collection of Toolsets that can be enabled or disabled as a
// group. It allows for managing multiple Toolsets and their states
// collectively.
type ToolsetGroup struct {
	Toolsets     map[Method]*Toolset
	everythingOn bool
	readOnly     bool
}

// NewToolsetGroup creates a new ToolsetGroup. If readOnly is true, all Toolsets
// added to this group will be set to read-only mode, meaning they can only have
// read tools added to them, and write tools will be ignored.
func NewToolsetGroup(readOnly bool) *ToolsetGroup {
	return &ToolsetGroup{
		Toolsets:     make(map[Method]*Toolset),
		everythingOn: false,
		readOnly:     readOnly,
	}
}

// AddToolset adds a Toolset to the ToolsetGroup. If the ToolsetGroup is in
// read-only mode, the Toolset will also be set to read-only.
func (tg *ToolsetGroup) AddToolset(ts *Toolset) {
	if tg.readOnly {
		ts.SetReadOnly()
	}
	tg.Toolsets[ts.Method] = ts
}

// IsEnabled checks if a Toolset with the given method is enabled in the
// ToolsetGroup.
func (tg *ToolsetGroup) IsEnabled(method Method) bool {
	// If everythingOn is true, all features are enabled
	if tg.everythingOn {
		return true
	}

	feature, exists := tg.Toolsets[method]
	if !exists {
		return false
	}
	return feature.Enabled
}

// EnableToolsets enables multiple Toolsets by their methods. If "all" is
// included in the methods, it will enable all Toolsets in the group.
func (tg *ToolsetGroup) EnableToolsets(methods ...Method) error {
	// special case for "all"
	for _, method := range methods {
		if method == MethodAll {
			tg.everythingOn = true
			break
		}
		if err := tg.EnableToolset(method); err != nil {
			return err
		}
	}
	// do this after to ensure all toolsets are enabled if "all" is present
	// anywhere in list
	if tg.everythingOn {
		for method := range tg.Toolsets {
			if err := tg.EnableToolset(method); err != nil {
				return err
			}
		}
	}
	return nil
}

// EnableToolset enables a Toolset by its method. If the Toolset does not exist,
// it returns a ToolsetDoesNotExistError.
func (tg *ToolsetGroup) EnableToolset(method Method) error {
	toolset, exists := tg.Toolsets[method]
	if !exists {
		return NewToolsetDoesNotExistError(method)
	}
	toolset.Enabled = true
	tg.Toolsets[method] = toolset
	return nil
}

// RegisterAll registers all Toolsets in the ToolsetGroup with the MCP server.
func (tg *ToolsetGroup) RegisterAll(s *mcp.Server) {
	for _, toolset := range tg.Toolsets {
		toolset.RegisterTools(s)
		toolset.RegisterResourcesTemplates(s)
		toolset.RegisterPrompts(s)
	}
}

// GetToolset retrieves a Toolset by its method from the ToolsetGroup. If the
// Toolset does not exist, it returns a ToolsetDoesNotExistError.
func (tg *ToolsetGroup) GetToolset(method Method) (*Toolset, error) {
	toolset, exists := tg.Toolsets[method]
	if !exists {
		return nil, NewToolsetDoesNotExistError(method)
	}
	return toolset, nil
}

// HasTools checks if the ToolsetGroup has any enabled Toolsets with available
// tools. It returns true if at least one Toolset is enabled and has tools,
// otherwise it returns false.
func (tg *ToolsetGroup) HasTools() bool {
	for _, toolset := range tg.Toolsets {
		if toolset.Enabled && len(toolset.GetAvailableTools()) > 0 {
			return true
		}
	}
	return false
}

// HasPrompts checks if the ToolsetGroup has any enabled Toolsets with available
// prompts. It returns true if at least one Toolset is enabled and has prompts,
// otherwise it returns false.
func (tg *ToolsetGroup) HasPrompts() bool {
	for _, toolset := range tg.Toolsets {
		if toolset.Enabled && len(toolset.prompts) > 0 {
			return true
		}
	}
	return false
}
