// Package toolsets provides a framework for managing collections of tools. This
// was heavily inspired by GitHub's MCP server implementation:
//
//	https://github.com/github/github-mcp-server/blob/3341e6bc461b461f0789518879f97bbd86ef7ee9/pkg/toolsets/toolsets.go
package toolsets

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	registeredMethods      = make(map[Method]struct{})
	registeredMethodsMutex sync.RWMutex

	registeredProfiles      = make(map[string][]Method)
	registeredProfilesMutex sync.RWMutex

	registeredToolOrder      []Method
	registeredToolOrderMutex sync.RWMutex
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

// RegisterProfile registers a named profile that maps to a set of methods.
// Profiles are a convenience for enabling a predefined collection of toolsets
// with a single name (e.g. "project-manager", "support"). Use RegisterProfiles
// to register the same set of methods under multiple aliases.
func RegisterProfile(name string, methods []Method) {
	registeredProfilesMutex.Lock()
	defer registeredProfilesMutex.Unlock()
	registeredProfiles[name] = methods
}

// RegisterProfiles registers multiple named profiles that map to the same set
// of methods. This is useful for declaring aliases — for example "support" and
// "desk" both expand to the Teamwork Desk toolsets.
func RegisterProfiles(names []string, methods []Method) {
	registeredProfilesMutex.Lock()
	defer registeredProfilesMutex.Unlock()
	for _, name := range names {
		registeredProfiles[name] = methods
	}
}

// RegisterToolOrder sets the preferred display order for tools. Methods that
// appear earlier in the list are presented first to MCP clients; this helps
// clients that truncate the tool list at a fixed size keep the most useful
// tools. Methods not listed here fall back to alphabetical order.
func RegisterToolOrder(methods []Method) {
	registeredToolOrderMutex.Lock()
	defer registeredToolOrderMutex.Unlock()
	registeredToolOrder = methods
}

// ToolOrder returns the registered preferred tool order. The returned slice
// must not be modified.
func ToolOrder() []Method {
	registeredToolOrderMutex.RLock()
	defer registeredToolOrderMutex.RUnlock()
	return registeredToolOrder
}

// LookupProfile returns the methods associated with a named profile, and
// whether the profile exists.
func LookupProfile(name string) ([]Method, bool) {
	registeredProfilesMutex.RLock()
	defer registeredProfilesMutex.RUnlock()
	methods, exists := registeredProfiles[name]
	return methods, exists
}

// IsProfile reports whether name is a registered profile.
func IsProfile(name string) bool {
	registeredProfilesMutex.RLock()
	defer registeredProfilesMutex.RUnlock()
	_, exists := registeredProfiles[name]
	return exists
}

// ListProfiles returns all registered profile names.
func ListProfiles() []string {
	registeredProfilesMutex.RLock()
	defer registeredProfilesMutex.RUnlock()
	names := make([]string, 0, len(registeredProfiles))
	for name := range registeredProfiles {
		names = append(names, name)
	}
	return names
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

// ServerResource represents a plain resource that can be registered with the
// MCP server and will appear in resources/list.
type ServerResource struct {
	resource *mcp.Resource
	handler  mcp.ResourceHandler
}

// NewServerResource creates a new ServerResource with the given resource and
// handler function.
func NewServerResource(resource *mcp.Resource, handler mcp.ResourceHandler) ServerResource {
	return ServerResource{resource: resource, handler: handler}
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
	resources         []ServerResource
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
		s.AddTool(toolWrapper.Tool, withInputValidation(toolWrapper.Tool, toolWrapper.Handler))
	}
	if !t.readOnly {
		for _, tool := range t.writeTools {
			s.AddTool(tool.Tool, withInputValidation(tool.Tool, tool.Handler))
		}
	}
}

// withInputValidation wraps a tool handler so that incoming arguments are
// validated against the tool's InputSchema before the handler runs. The MCP
// go-sdk (as of v1.6.0) does not validate arguments on the client or the
// server when tools are registered via the untyped Server.AddTool method —
// see https://github.com/modelcontextprotocol/go-sdk/issues/648. Without this
// wrapper, tools receive whatever JSON the caller sends, regardless of the
// schema published via tools/list.
func withInputValidation(tool *mcp.Tool, handler mcp.ToolHandler) mcp.ToolHandler {
	schema, ok := tool.InputSchema.(*jsonschema.Schema)
	if !ok || schema == nil {
		return handler
	}
	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		panic(fmt.Sprintf("toolsets: failed to resolve input schema for %q: %v", tool.Name, err))
	}
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := map[string]any{}
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return newInputValidationError("invalid arguments JSON: %s", err.Error()), nil
			}
		}
		// Repair arguments from clients that serialize scalars as strings before
		// validating (see coerceStringScalars). When anything changes, re-marshal
		// so the handler receives the coerced values too.
		if coerceStringScalars(schema, args) {
			raw, err := json.Marshal(args)
			if err != nil {
				return newInputValidationError("invalid arguments: %s", err.Error()), nil
			}
			req.Params.Arguments = raw
		}
		if err := resolved.Validate(args); err != nil {
			return newInputValidationError("invalid arguments: %s", err.Error()), nil
		}
		return handler(ctx, req)
	}
}

// coerceStringScalars repairs arguments from MCP clients that serialize scalar
// values as strings (e.g. "911218" instead of 911218, "false" instead of false)
// before they are validated against the tool's InputSchema. Some clients coerce
// arguments to the declared JSON type but do not look inside anyOf/oneOf
// branches, so nullable/optional parameters — which we express as
// anyOf: [{type: <scalar>}, {type: "null"}] to satisfy OpenAI strict mode —
// arrive as strings and fail validation before the handler ever runs (issue
// #383). Coercion is schema-directed and conservative: a string is converted
// only when the schema accepts the target scalar type but not string, so
// genuine string parameters (e.g. search_term) are left untouched. It returns
// true if any value was changed. The value map is mutated in place.
func coerceStringScalars(schema *jsonschema.Schema, value any) bool {
	if schema == nil {
		return false
	}
	switch v := value.(type) {
	case map[string]any:
		obj := objectBranch(schema)
		if obj == nil {
			return false
		}
		var changed bool
		for key, sub := range v {
			propSchema, ok := obj.Properties[key]
			if !ok {
				continue
			}
			if s, isStr := sub.(string); isStr {
				if coerced, did := coerceScalarString(propSchema, s); did {
					v[key] = coerced
					changed = true
					continue
				}
			}
			if coerceStringScalars(propSchema, sub) {
				changed = true
			}
		}
		return changed
	case []any:
		items := arrayItems(schema)
		if items == nil {
			return false
		}
		var changed bool
		for i, sub := range v {
			if s, isStr := sub.(string); isStr {
				if coerced, did := coerceScalarString(items, s); did {
					v[i] = coerced
					changed = true
					continue
				}
			}
			if coerceStringScalars(items, sub) {
				changed = true
			}
		}
		return changed
	default:
		return false
	}
}

// coerceScalarString converts a string to the scalar type declared by schema,
// returning the converted value and true when a conversion was applied. It
// leaves the string unchanged (returning false) when the schema accepts string,
// declares no compatible scalar type, or the string does not parse as the
// target type — in which case validation surfaces the appropriate error.
func coerceScalarString(schema *jsonschema.Schema, s string) (any, bool) {
	types := make(map[string]bool)
	collectTypes(schema, types)
	if types["string"] {
		return s, false
	}
	switch {
	case types["integer"] || types["number"]:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
	case types["boolean"]:
		if b, err := strconv.ParseBool(s); err == nil {
			return b, true
		}
	}
	return s, false
}

// collectTypes gathers the set of JSON types accepted by schema, descending
// into anyOf/oneOf/allOf branches (the shape used for nullable parameters).
func collectTypes(schema *jsonschema.Schema, out map[string]bool) {
	if schema == nil {
		return
	}
	if schema.Type != "" {
		out[schema.Type] = true
	}
	for _, t := range schema.Types {
		out[t] = true
	}
	for _, branch := range schema.AnyOf {
		collectTypes(branch, out)
	}
	for _, branch := range schema.OneOf {
		collectTypes(branch, out)
	}
	for _, branch := range schema.AllOf {
		collectTypes(branch, out)
	}
}

// objectBranch returns the object sub-schema of schema (schema itself when it is
// an object, otherwise the first object branch of an anyOf/oneOf/allOf), or nil
// when schema describes no object.
func objectBranch(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil {
		return nil
	}
	if len(schema.Properties) > 0 {
		return schema
	}
	for _, branches := range [][]*jsonschema.Schema{schema.AnyOf, schema.OneOf, schema.AllOf} {
		for _, branch := range branches {
			if obj := objectBranch(branch); obj != nil {
				return obj
			}
		}
	}
	return nil
}

// arrayItems returns the item schema for the array described by schema (schema
// itself when it is an array, otherwise the first array branch of an
// anyOf/oneOf/allOf), or nil when schema describes no array.
func arrayItems(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil {
		return nil
	}
	if schema.Items != nil {
		return schema.Items
	}
	for _, branches := range [][]*jsonschema.Schema{schema.AnyOf, schema.OneOf, schema.AllOf} {
		for _, branch := range branches {
			if items := arrayItems(branch); items != nil {
				return items
			}
		}
	}
	return nil
}

func newInputValidationError(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf(format, args...)},
		},
	}
}

// AddResources adds plain resources to the Toolset. These will appear in
// resources/list responses.
func (t *Toolset) AddResources(resources ...ServerResource) *Toolset {
	t.resources = append(t.resources, resources...)
	return t
}

// GetActiveResources returns the plain resources that are currently active in
// the Toolset.
func (t *Toolset) GetActiveResources() []ServerResource {
	if !t.Enabled {
		return nil
	}
	return t.resources
}

// RegisterResources registers the plain resources in the Toolset with the MCP
// server.
func (t *Toolset) RegisterResources(s *mcp.Server) {
	if !t.Enabled {
		return
	}
	for _, r := range t.resources {
		s.AddResource(r.resource, r.handler)
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
// included in the methods, it will enable all Toolsets in the group. Methods
// that do not belong to this group are silently ignored, allowing the same
// method list to be passed to multiple groups without error.
func (tg *ToolsetGroup) EnableToolsets(methods ...Method) error {
	// special case for "all"
	for _, method := range methods {
		if method == MethodAll {
			tg.everythingOn = true
			break
		}
		// silently skip methods that belong to other groups
		if _, exists := tg.Toolsets[method]; !exists {
			continue
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
		toolset.RegisterResources(s)
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

// HasResources checks if the ToolsetGroup has any enabled Toolsets with
// available resources. It returns true if at least one Toolset is enabled and
// has resources, otherwise it returns false.
func (tg *ToolsetGroup) HasResources() bool {
	for _, toolset := range tg.Toolsets {
		if toolset.Enabled && (len(toolset.resources) > 0 || len(toolset.resourceTemplates) > 0) {
			return true
		}
	}
	return false
}
