package command

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

const (
	maximumCommandNameLength        = 32
	maximumCommandDescriptionLength = 100
	maximumCommandOptions           = 25
	maximumOptionChoices            = 25
)

var commandNamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

var (
	ErrRegistryFrozen    = errors.New("command registry is frozen")
	ErrRegistryNotFrozen = errors.New("command registry is not frozen")
	ErrUnknownCommand    = errors.New("unknown command")
	ErrNilRequest        = errors.New("command request is nil")
)

// ValidationError reports every invalid registry declaration found by Freeze.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "invalid command registry: " + strings.Join(e.Problems, "; ")
}

type routeDeclaration struct {
	name        string
	description string
	handler     HandlerFunc
	options     []OptionDefinition
	middleware  []Middleware
}

type groupDeclaration struct {
	name        string
	description string
	subcommands []*routeDeclaration
	middleware  []Middleware
}

type rootDeclaration struct {
	name        string
	description string
	leaf        bool
	handler     HandlerFunc
	options     []OptionDefinition
	middleware  []Middleware
	subcommands []*routeDeclaration
	groups      []*groupDeclaration
}

// Registry is the unique startup-time command declaration tree and the
// runtime command dispatcher.
type Registry struct {
	mu          sync.RWMutex
	frozen      bool
	middleware  []Middleware
	roots       []*rootDeclaration
	definitions []*Definition
	dispatch    map[string]HandlerFunc
}

// NewRegistry creates an empty command registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Use appends registry-wide middleware.
func (r *Registry) Use(middleware ...Middleware) {
	r.lockForMutation()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware...)
}

// Slash registers a top-level leaf slash command.
func (r *Registry) Slash(
	name string,
	description string,
	handler HandlerFunc,
	options ...Option,
) *Route {
	declaration := &rootDeclaration{
		name:        name,
		description: description,
		leaf:        true,
		handler:     handler,
		options:     snapshotOptions(options),
	}
	r.lockForMutation()
	defer r.mu.Unlock()
	r.roots = append(r.roots, declaration)
	return &Route{
		registry:   r,
		middleware: &declaration.middleware,
	}
}

// Group registers a top-level slash command that owns subcommands.
func (r *Registry) Group(name, description string) *CommandGroup {
	declaration := &rootDeclaration{
		name:        name,
		description: description,
	}
	r.lockForMutation()
	defer r.mu.Unlock()
	r.roots = append(r.roots, declaration)
	return &CommandGroup{
		registry:    r,
		declaration: declaration,
	}
}

// Route is a registered leaf command.
type Route struct {
	registry   *Registry
	middleware *[]Middleware
}

// Use appends command-level middleware.
func (r *Route) Use(middleware ...Middleware) *Route {
	r.registry.lockForMutation()
	defer r.registry.mu.Unlock()
	*r.middleware = append(*r.middleware, middleware...)
	return r
}

// CommandGroup is a top-level command that can contain direct subcommands and
// one level of Discord subcommand groups.
type CommandGroup struct {
	registry    *Registry
	declaration *rootDeclaration
}

// Use appends middleware inherited by every route in the command group.
func (g *CommandGroup) Use(middleware ...Middleware) *CommandGroup {
	g.registry.lockForMutation()
	defer g.registry.mu.Unlock()
	g.declaration.middleware = append(g.declaration.middleware, middleware...)
	return g
}

// Sub registers a direct subcommand.
func (g *CommandGroup) Sub(
	name string,
	description string,
	handler HandlerFunc,
	options ...Option,
) *Route {
	declaration := &routeDeclaration{
		name:        name,
		description: description,
		handler:     handler,
		options:     snapshotOptions(options),
	}
	g.registry.lockForMutation()
	defer g.registry.mu.Unlock()
	g.declaration.subcommands = append(g.declaration.subcommands, declaration)
	return &Route{
		registry:   g.registry,
		middleware: &declaration.middleware,
	}
}

// Group registers a Discord subcommand group. The returned type deliberately
// has no Group method because Discord does not support deeper nesting.
func (g *CommandGroup) Group(name, description string) *SubcommandGroup {
	declaration := &groupDeclaration{
		name:        name,
		description: description,
	}
	g.registry.lockForMutation()
	defer g.registry.mu.Unlock()
	g.declaration.groups = append(g.declaration.groups, declaration)
	return &SubcommandGroup{
		registry:    g.registry,
		root:        g.declaration,
		declaration: declaration,
	}
}

// SubcommandGroup is Discord's single supported subcommand-group level.
type SubcommandGroup struct {
	registry    *Registry
	root        *rootDeclaration
	declaration *groupDeclaration
}

// Use appends middleware inherited by every subcommand in this subgroup.
func (g *SubcommandGroup) Use(middleware ...Middleware) *SubcommandGroup {
	g.registry.lockForMutation()
	defer g.registry.mu.Unlock()
	g.declaration.middleware = append(g.declaration.middleware, middleware...)
	return g
}

// Sub registers a subcommand inside the subgroup.
func (g *SubcommandGroup) Sub(
	name string,
	description string,
	handler HandlerFunc,
	options ...Option,
) *Route {
	declaration := &routeDeclaration{
		name:        name,
		description: description,
		handler:     handler,
		options:     snapshotOptions(options),
	}
	g.registry.lockForMutation()
	defer g.registry.mu.Unlock()
	g.declaration.subcommands = append(g.declaration.subcommands, declaration)
	return &Route{
		registry:   g.registry,
		middleware: &declaration.middleware,
	}
}

// lockForMutation acquires the declaration lock and rejects post-freeze changes.
func (r *Registry) lockForMutation() {
	r.mu.Lock()
	if r.frozen {
		r.mu.Unlock()
		panic(ErrRegistryFrozen)
	}
}

// snapshotOptions converts mutable option builders into independent definitions.
func snapshotOptions(options []Option) []OptionDefinition {
	result := make([]OptionDefinition, len(options))
	for index, option := range options {
		if option != nil {
			result[index] = option.snapshot()
		}
	}
	return result
}

// Freeze validates declarations, builds immutable Discord definitions and
// composes the runtime dispatch table.
func (r *Registry) Freeze() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		return nil
	}

	problems := r.validate()
	if len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}

	r.definitions = make([]*Definition, 0, len(r.roots))
	r.dispatch = make(map[string]HandlerFunc)
	for _, root := range r.roots {
		definition, err := r.compileRoot(root)
		if err != nil {
			return err
		}
		r.definitions = append(r.definitions, definition)
	}
	r.frozen = true
	return nil
}

func (r *Registry) compileRoot(root *rootDeclaration) (*Definition, error) {
	definition := &Definition{
		Name:        root.name,
		Description: root.description,
	}
	if root.leaf {
		definition.Options = cloneOptionDefinitions(root.options)
		path := CommandPath{Command: root.name}
		return definition, r.compileRoute(
			path,
			root.handler,
			r.middleware,
			root.middleware,
		)
	}

	if err := r.compileDirectSubcommands(definition, root); err != nil {
		return nil, err
	}
	if err := r.compileSubcommandGroups(definition, root); err != nil {
		return nil, err
	}
	return definition, nil
}

func (r *Registry) compileDirectSubcommands(
	definition *Definition,
	root *rootDeclaration,
) error {
	for _, subcommand := range root.subcommands {
		definition.Subcommands = append(
			definition.Subcommands,
			newSubcommandDefinition(subcommand),
		)
		path := CommandPath{
			Command:    root.name,
			Subcommand: subcommand.name,
		}
		if err := r.compileRoute(
			path,
			subcommand.handler,
			r.middleware,
			root.middleware,
			subcommand.middleware,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) compileSubcommandGroups(
	definition *Definition,
	root *rootDeclaration,
) error {
	for _, group := range root.groups {
		groupDefinition, err := r.compileSubcommandGroup(root, group)
		if err != nil {
			return err
		}
		definition.Groups = append(definition.Groups, groupDefinition)
	}
	return nil
}

func (r *Registry) compileSubcommandGroup(
	root *rootDeclaration,
	group *groupDeclaration,
) (SubcommandGroupDefinition, error) {
	definition := SubcommandGroupDefinition{
		Name:        group.name,
		Description: group.description,
	}
	for _, subcommand := range group.subcommands {
		definition.Subcommands = append(
			definition.Subcommands,
			newSubcommandDefinition(subcommand),
		)
		path := CommandPath{
			Command:    root.name,
			Group:      group.name,
			Subcommand: subcommand.name,
		}
		if err := r.compileRoute(
			path,
			subcommand.handler,
			r.middleware,
			root.middleware,
			group.middleware,
			subcommand.middleware,
		); err != nil {
			return SubcommandGroupDefinition{}, err
		}
	}
	return definition, nil
}

func newSubcommandDefinition(declaration *routeDeclaration) SubcommandDefinition {
	return SubcommandDefinition{
		Name:        declaration.name,
		Description: declaration.description,
		Options:     cloneOptionDefinitions(declaration.options),
	}
}

func (r *Registry) compileRoute(
	path CommandPath,
	handler HandlerFunc,
	middleware ...[]Middleware,
) error {
	compiled := compose(handler, appendMiddleware(middleware...)...)
	if compiled == nil {
		return nilMiddlewareHandlerError(path)
	}
	r.dispatch[path.key()] = compiled
	return nil
}

func nilMiddlewareHandlerError(path CommandPath) error {
	return &ValidationError{Problems: []string{fmt.Sprintf(
		"/%s middleware returned a nil handler", strings.Join(path.Segments(), " "),
	)}}
}

func appendMiddleware(groups ...[]Middleware) []Middleware {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	result := make([]Middleware, 0, total)
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

func (r *Registry) validate() []string {
	rootNames := make(map[string]struct{}, len(r.roots))
	problems := validateMiddleware("registry", r.middleware)
	for _, root := range r.roots {
		problems = append(problems, validateRoot(root, rootNames)...)
	}
	return problems
}

func validateRoot(root *rootDeclaration, rootNames map[string]struct{}) []string {
	location := "/" + root.name
	problems := validateMetadata(location, root.name, root.description)
	if _, exists := rootNames[root.name]; exists {
		problems = append(problems, fmt.Sprintf("duplicate command %s", location))
	}
	rootNames[root.name] = struct{}{}
	problems = append(problems, validateMiddleware(location, root.middleware)...)

	if root.leaf {
		problems = append(problems, validateHandler(location, root.handler)...)
		return append(problems, validateOptions(location, root.options)...)
	}
	return append(problems, validateCommandGroup(location, root)...)
}

func validateCommandGroup(location string, root *rootDeclaration) []string {
	var problems []string
	childCount := len(root.subcommands) + len(root.groups)
	if childCount == 0 {
		problems = append(problems, fmt.Sprintf("command group %s is empty", location))
	}
	if childCount > maximumCommandOptions {
		problems = append(problems, fmt.Sprintf(
			"command group %s has more than %d children",
			location,
			maximumCommandOptions,
		))
	}

	childNames := make(map[string]struct{}, childCount)
	for _, subcommand := range root.subcommands {
		childLocation := location + " " + subcommand.name
		problems = append(problems, validateRoute(childLocation, subcommand)...)
		if _, exists := childNames[subcommand.name]; exists {
			problems = append(problems, fmt.Sprintf("duplicate child %s", childLocation))
		}
		childNames[subcommand.name] = struct{}{}
	}
	for _, group := range root.groups {
		problems = append(problems, validateSubcommandGroup(location, group, childNames)...)
	}
	return problems
}

func validateSubcommandGroup(
	rootLocation string,
	group *groupDeclaration,
	childNames map[string]struct{},
) []string {
	location := rootLocation + " " + group.name
	problems := validateMetadata(location, group.name, group.description)
	problems = append(problems, validateMiddleware(location, group.middleware)...)
	if _, exists := childNames[group.name]; exists {
		problems = append(problems, fmt.Sprintf("duplicate child %s", location))
	}
	childNames[group.name] = struct{}{}

	if len(group.subcommands) == 0 {
		problems = append(problems, fmt.Sprintf("subcommand group %s is empty", location))
	}
	if len(group.subcommands) > maximumCommandOptions {
		problems = append(problems, fmt.Sprintf(
			"subcommand group %s has more than %d subcommands",
			location,
			maximumCommandOptions,
		))
	}

	subcommandNames := make(map[string]struct{}, len(group.subcommands))
	for _, subcommand := range group.subcommands {
		subcommandLocation := location + " " + subcommand.name
		problems = append(problems, validateRoute(subcommandLocation, subcommand)...)
		if _, exists := subcommandNames[subcommand.name]; exists {
			problems = append(
				problems,
				fmt.Sprintf("duplicate subcommand %s", subcommandLocation),
			)
		}
		subcommandNames[subcommand.name] = struct{}{}
	}
	return problems
}

func validateRoute(location string, route *routeDeclaration) []string {
	problems := validateMetadata(location, route.name, route.description)
	problems = append(problems, validateHandler(location, route.handler)...)
	problems = append(problems, validateMiddleware(location, route.middleware)...)
	problems = append(problems, validateOptions(location, route.options)...)
	return problems
}

func validateMetadata(location, name, description string) []string {
	var problems []string
	if len(name) == 0 || len(name) > maximumCommandNameLength || !commandNamePattern.MatchString(name) {
		problems = append(problems, fmt.Sprintf("%s has invalid name %q", location, name))
	}
	if len(description) == 0 || len(description) > maximumCommandDescriptionLength {
		problems = append(problems, fmt.Sprintf("%s has invalid description", location))
	}
	return problems
}

func validateHandler(location string, handler HandlerFunc) []string {
	if handler == nil {
		return []string{fmt.Sprintf("%s has no handler", location)}
	}
	return nil
}

func validateMiddleware(location string, middleware []Middleware) []string {
	for _, current := range middleware {
		if current == nil {
			return []string{fmt.Sprintf("%s has nil middleware", location)}
		}
	}
	return nil
}

func validateOptions(location string, options []OptionDefinition) []string {
	var problems []string
	if len(options) > maximumCommandOptions {
		problems = append(problems, fmt.Sprintf("%s has more than %d options", location, maximumCommandOptions))
	}
	names := make(map[string]struct{}, len(options))
	foundOptional := false
	for _, option := range options {
		optionLocation := location + " option " + option.Name
		problems = append(problems, validateMetadata(
			optionLocation, option.Name, option.Description,
		)...)
		if option.Type < OptionTypeString || option.Type > OptionTypeChannel {
			problems = append(problems, fmt.Sprintf("%s has invalid type", optionLocation))
		}
		if _, exists := names[option.Name]; exists {
			problems = append(problems, fmt.Sprintf("duplicate %s", optionLocation))
		}
		names[option.Name] = struct{}{}
		if !option.Required {
			foundOptional = true
		} else if foundOptional {
			problems = append(problems, fmt.Sprintf("required %s follows an optional option", optionLocation))
		}
		if option.Autocomplete && option.Type != OptionTypeString && option.Type != OptionTypeInteger {
			problems = append(problems, fmt.Sprintf("%s cannot use autocomplete", optionLocation))
		}
		if len(option.Choices) > maximumOptionChoices {
			problems = append(problems, fmt.Sprintf("%s has more than %d choices", optionLocation, maximumOptionChoices))
		}
		choiceValues := make(map[string]struct{}, len(option.Choices))
		for _, choice := range option.Choices {
			if option.Type != OptionTypeString {
				problems = append(problems, fmt.Sprintf("%s choices require a string option", optionLocation))
				break
			}
			if choice.Name == "" || choice.Value == "" {
				problems = append(problems, fmt.Sprintf("%s has an empty choice name or value", optionLocation))
			}
			if _, exists := choiceValues[choice.Value]; exists {
				problems = append(problems, fmt.Sprintf("%s has duplicate choice value %q", optionLocation, choice.Value))
			}
			choiceValues[choice.Value] = struct{}{}
		}
	}
	return problems
}

// Definitions returns a deep copy of the frozen application-command metadata.
func (r *Registry) Definitions() ([]*Definition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.frozen {
		return nil, ErrRegistryNotFrozen
	}
	return cloneDefinitions(r.definitions), nil
}

// Dispatch executes the middleware chain and handler registered for a request.
func (r *Registry) Dispatch(ctx context.Context, request *CommandRequest) error {
	if request == nil {
		return ErrNilRequest
	}
	r.mu.RLock()
	if !r.frozen {
		r.mu.RUnlock()
		return ErrRegistryNotFrozen
	}
	handler, exists := r.dispatch[request.Path.key()]
	r.mu.RUnlock()
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownCommand, strings.Join(request.Path.Segments(), " "))
	}
	return handler(ctx, request)
}
