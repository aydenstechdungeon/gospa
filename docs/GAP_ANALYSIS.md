# GoSPA Documentation Gap Analysis Report

## Executive Summary

This report identifies all undocumented public classes, methods, parameters, return types, and configuration options between the framework source code and existing documentation in `docs/API.md` and the `/website` directory.

**Coverage Statistics:**
- **Total Public APIs**: ~200+ exports across Go and TypeScript
- **Documented APIs**: ~120 (60%)
- **Undocumented APIs**: ~80 (40%)
- **Missing Documentation Pages**: 5 major areas

---

## 1. Go Server-Side API Gaps

### 1.1 `gospa.go` - Core App

#### Undocumented Config Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Port` | `string` | `":3000"` | Server port |
| `Host` | `string` | `""` | Server host binding |
| `RoutesFS` | `fs.FS` | `nil` | Embedded filesystem for routes (takes precedence over RoutesDir) |
| `StaticDir` | `string` | `"./static"` | Static files directory |
| `StaticPrefix` | `string` | `"/static"` | URL prefix for static files |
| `AppName` | `string` | `"GoSPA App"` | Application name for logging |
| `RuntimeScript` | `string` | `""` | Custom path to client runtime |
| `WebSocketPath` | `string` | `"/_gospa/ws"` | WebSocket endpoint path |
| `WebSocketMiddleware` | `fiber.Handler` | `nil` | Pre-WebSocket connection middleware |

#### Undocumented Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Scan()` | `error` | Scans routes directory for .templ files |
| `RegisterRoutes()` | `error` | Registers all discovered routes with Fiber |
| `Use()` | `func(middleware ...fiber.Handler)` | Adds middleware to the Fiber app |
| `Group()` | `*fiber.Group` | Creates a route group with prefix and middleware |

---

### 1.2 `state/` Package

#### `state/serialize.go` - **ENTIRELY UNDOCUMENTED**

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `StateMap.AddAny()` | `AddAny(key string, value any) error` | Adds any value as observable |
| `StateMap.ForEach()` | `ForEach(fn func(key string, value Observable))` | Iterates over all entries |
| `StateMap.ToMap()` | `ToMap() map[string]any` | Converts to plain map |
| `StateMap.MarshalJSON()` | `MarshalJSON() ([]byte, error)` | JSON serialization |
| `StateValidator` | `struct` | Validates state values |
| `NewStateValidator()` | `*StateValidator` | Creates validator instance |
| `AddValidator()` | `AddValidator(key string, fn func(any) error)` | Registers validation function |
| `Validate()` | `Validate(key string, value any) error` | Validates single value |
| `ValidateAll()` | `ValidateAll(state map[string]any) error` | Validates entire state |
| `SerializeState()` | `SerializeState(state *StateMap) ([]byte, error)` | Serializes state to JSON |

#### `state/rune.go` - Partially Documented

| Method | Signature | Description |
|--------|-----------|-------------|
| `ID()` | `ID() string` | Returns unique identifier for the rune |
| `MarshalJSON()` | `MarshalJSON() ([]byte, error)` | JSON serialization support |
| `GetAny()` | `GetAny() any` | Returns value as any (Observable interface) |
| `SubscribeAny()` | `SubscribeAny(fn func(any)) func()` | Subscribe with any callback |
| `SetAny()` | `SetAny(value any) error` | Set value as any (Settable interface) |

#### `state/derived.go` - Partially Documented

| Method | Signature | Description |
|--------|-----------|-------------|
| `DependOn()` | `DependOn(observable Observable)` | Adds dependency tracking |
| `ID()` | `ID() string` | Returns unique identifier |
| `MarshalJSON()` | `MarshalJSON() ([]byte, error)` | JSON serialization |

#### `state/effect.go` - Partially Documented

| Method | Signature | Description |
|--------|-----------|-------------|
| `DependOn()` | `DependOn(observable Observable)` | Adds dependency tracking |
| `IsActive()` | `IsActive() bool` | Returns if effect is active |
| `Pause()` | `Pause()` | Pauses effect execution |
| `Resume()` | `Resume()` | Resumes effect execution |
| `Dispose()` | `Dispose()` | Cleans up effect |

---

### 1.3 `routing/` Package

#### `routing/params.go` - **ENTIRELY UNDOCUMENTED**

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `Params` | `map[string]string` | Route parameter map |
| `GetDefault()` | `GetDefault(key, def string) string` | Get with default value |
| `Has()` | `Has(key string) bool` | Check if key exists |
| `Set()` | `Set(key, value string)` | Set parameter value |
| `Delete()` | `Delete(key string)` | Remove parameter |
| `Clone()` | `Clone() Params` | Clone parameter map |
| `Merge()` | `Merge(other Params)` | Merge two parameter maps |
| `ToMap()` | `ToMap() map[string]string` | Convert to plain map |
| `ToJSON()` | `ToJSON() ([]byte, error)` | JSON serialization |
| `Int()` | `Int(key string) (int, error)` | Parse as int |
| `IntDefault()` | `IntDefault(key string, def int) int` | Parse as int with default |
| `Int64()` | `Int64(key string) (int64, error)` | Parse as int64 |
| `Float64()` | `Float64(key string) (float64, error)` | Parse as float64 |
| `Bool()` | `Bool(key string) (bool, error)` | Parse as bool |
| `Slice()` | `Slice(key, sep string) ([]string, error)` | Parse as slice |
| `QueryParams` | `struct` | URL query parameter handling |
| `NewQueryParams()` | `NewQueryParams() *QueryParams` | Create query params instance |
| `NewQueryParamsFromValues()` | `NewQueryParamsFromValues(v url.Values) *QueryParams` | Create from url.Values |
| `GetAll()` | `GetAll(key string) []string` | Get all values for key |
| `Add()` | `Add(key, value string)` | Add query parameter |
| `Del()` | `Del(key string)` | Delete query parameter |
| `Encode()` | `Encode() string` | Encode to URL query string |
| `ParamExtractor` | `struct` | Extracts params from routes |
| `NewParamExtractor()` | `NewParamExtractor() *ParamExtractor` | Create extractor |
| `Extract()` | `Extract(path, pattern string) Params` | Extract params from path |
| `Match()` | `Match(path, pattern string) bool` | Check if path matches pattern |
| `RouteMatch` | `struct` | Route matching result |
| `NewRouteMatch()` | `NewRouteMatch() *RouteMatch` | Create route match |
| `Param()` | `Param(key string) string` | Get route parameter |
| `Query()` | `Query(key string) string` | Get query parameter |
| `PathBuilder` | `struct` | Builds URLs from routes |
| `NewPathBuilder()` | `NewPathBuilder(pattern string) *PathBuilder` | Create path builder |
| `Param()` | `Param(key, value string) *PathBuilder` | Set path parameter |
| `Query()` | `Query(key, value string) *PathBuilder` | Set query parameter |
| `QueryAdd()` | `QueryAdd(key, value string) *PathBuilder` | Add query parameter |
| `Build()` | `Build() string` | Build final URL |
| `BuildURL()` | `BuildURL(pattern string, params, query map[string]string) string` | Build URL helper |
| `ValidateParams()` | `ValidateParams(pattern, path string) error` | Validate params against pattern |

#### `routing/registry.go` - **ENTIRELY UNDOCUMENTED**

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `Registry` | `struct` | Component registry |
| `NewRegistry()` | `NewRegistry() *Registry` | Create registry |
| `RegisterPage()` | `RegisterPage(path string, fn PageFunc)` | Register page component |
| `RegisterPageWithOptions()` | `RegisterPageWithOptions(path string, fn PageFunc, opts RouteOptions)` | Register with options |
| `GetRouteOptions()` | `GetRouteOptions(path string) RouteOptions` | Get route options |
| `RegisterLayout()` | `RegisterLayout(path string, fn LayoutFunc)` | Register layout component |
| `GetPage()` | `GetPage(path string) PageFunc` | Get page component |
| `GetLayout()` | `GetLayout(path string) LayoutFunc` | Get layout component |
| `RegisterRootLayout()` | `RegisterRootLayout(fn LayoutFunc)` | Register root layout |
| `GetRootLayout()` | `GetRootLayout() LayoutFunc` | Get root layout |
| `HasPage()` | `HasPage(path string) bool` | Check if page exists |
| `HasLayout()` | `HasLayout(path string) bool` | Check if layout exists |
| `RenderStrategy` | `type` | SSR, CSR, SSG enum |
| `RouteOptions` | `struct` | Route configuration options |

#### `routing/remote.go` - **ENTIRELY UNDOCUMENTED**

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `RemoteActionFunc` | `type` | Remote action function type |
| `RemoteRegistry` | `struct` | Remote action registry |
| `RegisterRemoteAction()` | `RegisterRemoteAction(name string, fn RemoteActionFunc)` | Register remote action |
| `GetRemoteAction()` | `GetRemoteAction(name string) (RemoteActionFunc, bool)` | Get remote action |

#### `routing/auto.go` - Partially Documented

| Field/Method | Signature | Description |
|--------------|-----------|-------------|
| `Route.File` | `File string` | Source file path (documented as FilePath) |
| `Route.Params` | `[]string` | Dynamic param names |
| `Route.IsDynamic` | `bool` | Has dynamic segments |
| `Route.IsCatchAll` | `bool` | Is catch-all route |
| `Route.Priority` | `int` | Route matching priority |
| `Route.Children` | `[]*Route` | Child routes |
| `Route.Layout` | `*Route` | Associated layout |
| `Route.Middleware` | `[]fiber.Handler` | Route middleware |
| `Route.Meta` | `map[string]string` | Custom metadata |
| `GetErrorRoute()` | `GetErrorRoute() *Route` | Get error route |

---

### 1.4 `fiber/` Package

#### `fiber/errors.go` - **ENTIRELY UNDOCUMENTED**

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `ErrorCode` | `type` | Error code type |
| `ErrorCodeInternal` | `const` | Internal server error |
| `ErrorCodeNotFound` | `const` | Not found error |
| `ErrorCodeBadRequest` | `const` | Bad request error |
| `ErrorCodeUnauthorized` | `const` | Unauthorized error |
| `ErrorCodeForbidden` | `const` | Forbidden error |
| `ErrorCodeConflict` | `const` | Conflict error |
| `ErrorCodeValidation` | `const` | Validation error |
| `ErrorCodeTimeout` | `const` | Timeout error |
| `ErrorCodeUnavailable` | `const` | Service unavailable |
| `AppError` | `struct` | Application error type |
| `NewAppError()` | `NewAppError(code ErrorCode, message string) *AppError` | Create app error |
| `WithDetails()` | `WithDetails(details any) *AppError` | Add error details |
| `WithStack()` | `WithStack(stack []byte) *AppError` | Add stack trace |
| `WithRecover()` | `WithRecover(r any) *AppError` | Add recovery info |
| `ErrInternal` | `var` | Pre-defined internal error |
| `ErrNotFound` | `var` | Pre-defined not found |
| `ErrBadRequest` | `var` | Pre-defined bad request |
| `ErrUnauthorized` | `var` | Pre-defined unauthorized |
| `ErrForbidden` | `var` | Pre-defined forbidden |
| `ErrConflict` | `var` | Pre-defined conflict |
| `ErrValidation` | `var` | Pre-defined validation |
| `ErrTimeout` | `var` | Pre-defined timeout |
| `ErrUnavailable` | `var` | Pre-defined unavailable |
| `ErrorHandlerConfig` | `struct` | Error handler config |
| `ErrorHandler()` | `ErrorHandler(config ErrorHandlerConfig) fiber.Handler` | Error handler middleware |
| `NotFoundHandler()` | `NotFoundHandler() fiber.Handler` | 404 handler |
| `ValidationError()` | `ValidationError(field, message string) *AppError` | Create validation error |
| `ValidationErrors()` | `ValidationErrors(errors map[string]string) *AppError` | Multiple validation errors |
| `PanicHandler()` | `PanicHandler() fiber.Handler` | Panic recovery middleware |
| `IsAppError()` | `IsAppError(err error) bool` | Check if AppError |
| `AsAppError()` | `AsAppError(err error) (*AppError, bool)` | Convert to AppError |
| `WrapError()` | `WrapError(err error, code ErrorCode) *AppError` | Wrap error as AppError |

#### `fiber/dev.go` - **ENTIRELY UNDOCUMENTED**

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `DevConfig` | `struct` | Development config |
| `DefaultDevConfig()` | `DefaultDevConfig() DevConfig` | Default dev config |
| `FileWatcher` | `struct` | File watcher for hot reload |
| `DevTools` | `struct` | Development tools |
| `StateLogEntry` | `struct` | State log entry |
| `DebugMiddleware()` | `DebugMiddleware(config DevConfig) fiber.Handler` | Debug middleware |
| `StateInspectorMiddleware()` | `StateInspectorMiddleware(config DevConfig) fiber.Handler` | State inspector |

---

### 1.5 `component/` Package - **ENTIRELY UNDOCUMENTED**

#### `component/base.go`

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `BaseComponent` | `struct` | Base component implementation |
| `ID()` | `ID() string` | Component unique ID |
| `Name()` | `Name() string` | Component name |
| `State()` | `State() *state.StateMap` | Component state |
| `Props()` | `Props() Props` | Component props |
| `Children()` | `Children() []Component` | Child components |
| `Parent()` | `Parent() Component` | Parent component |
| `AddChild()` | `AddChild(child Component)` | Add child component |
| `RemoveChild()` | `RemoveChild(child Component)` | Remove child |
| `GetSlot()` | `GetSlot(name string) templ.Component` | Get named slot |
| `SetSlot()` | `SetSlot(name string, comp templ.Component)` | Set named slot |
| `Context()` | `Context() context.Context` | Get context |
| `SetContext()` | `SetContext(ctx context.Context)` | Set context |
| `ToJSON()` | `ToJSON() ([]byte, error)` | JSON serialization |
| `Clone()` | `Clone() Component` | Clone component |
| `Component` | `interface` | Component interface |
| `ComponentTree` | `struct` | Component tree structure |
| `NewComponentTree()` | `NewComponentTree(root Component) *ComponentTree` | Create tree |
| `Root()` | `Root() Component` | Get root component |
| `Get()` | `Get(id string) Component` | Get component by ID |
| `Add()` | `Add(parent, child Component)` | Add to tree |
| `Remove()` | `Remove(component Component)` | Remove from tree |
| `OnMount()` | `OnMount(fn func(Component))` | Mount callback |
| `OnUpdate()` | `OnUpdate(fn func(Component))` | Update callback |
| `OnDestroy()` | `OnDestroy(fn func(Component))` | Destroy callback |
| `Mount()` | `Mount()` | Trigger mount |
| `Update()` | `Update()` | Trigger update |
| `Walk()` | `Walk(fn func(Component) bool)` | Walk tree |
| `Find()` | `Find(fn func(Component) bool) Component` | Find component |
| `FindAll()` | `FindAll(fn func(Component) bool) []Component` | Find all matching |
| `FindByName()` | `FindByName(name string) Component` | Find by name |
| `FindByProp()` | `FindByProp(key string, value any) Component` | Find by prop |
| `Option` | `type` | Component option type |
| `WithProps()` | `WithProps(props Props) Option` | Props option |
| `WithState()` | `WithState(state *state.StateMap) Option` | State option |
| `WithChildren()` | `WithChildren(children ...Component) Option` | Children option |
| `WithParent()` | `WithParent(parent Component) Option` | Parent option |
| `WithContext()` | `WithContext(ctx context.Context) Option` | Context option |
| `WithSlots()` | `WithSlots(slots map[string]templ.Component) Option` | Slots option |

#### `component/lifecycle.go`

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `LifecyclePhase` | `type` | Lifecycle phase enum |
| `PhaseCreated` | `const` | Created phase |
| `PhaseMounting` | `const` | Mounting phase |
| `PhaseMounted` | `const` | Mounted phase |
| `PhaseUpdating` | `const` | Updating phase |
| `PhaseUpdated` | `const` | Updated phase |
| `PhaseDestroying` | `const` | Destroying phase |
| `PhaseDestroyed` | `const` | Destroyed phase |
| `Lifecycle` | `struct` | Lifecycle manager |
| `NewLifecycle()` | `NewLifecycle() *Lifecycle` | Create lifecycle |
| `Phase()` | `Phase() LifecyclePhase` | Current phase |
| `IsMounted()` | `IsMounted() bool` | Check if mounted |
| `OnBeforeMount()` | `OnBeforeMount(fn func())` | Before mount hook |
| `OnMount()` | `OnMount(fn func())` | Mount hook |
| `OnBeforeUpdate()` | `OnBeforeUpdate(fn func())` | Before update hook |
| `OnUpdate()` | `OnUpdate(fn func())` | Update hook |
| `OnBeforeDestroy()` | `OnBeforeDestroy(fn func())` | Before destroy hook |
| `OnDestroy()` | `OnDestroy(fn func())` | Destroy hook |
| `OnCleanup()` | `OnCleanup(fn func())` | Cleanup hook |
| `Mount()` | `Mount()` | Trigger mount |
| `Update()` | `Update()` | Trigger update |
| `Destroy()` | `Destroy()` | Trigger destroy |
| `ClearHooks()` | `ClearHooks()` | Clear all hooks |
| `LifecycleAware` | `interface` | Lifecycle interface |
| `MountComponent()` | `MountComponent(comp Component) error` | Mount helper |
| `DestroyComponent()` | `DestroyComponent(comp Component) error` | Destroy helper |
| `UpdateComponent()` | `UpdateComponent(comp Component) error` | Update helper |

#### `component/props.go`

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `Props` | `map[string]any` | Props map type |
| `Get()` | `Get(key string) any` | Get prop value |
| `Set()` | `Set(key string, value any)` | Set prop value |
| `GetDefault()` | `GetDefault(key string, def any) any` | Get with default |
| `GetString()` | `GetString(key string) string` | Get as string |
| `GetInt()` | `GetInt(key string) int` | Get as int |
| `GetInt64()` | `GetInt64(key string) int64` | Get as int64 |
| `GetFloat64()` | `GetFloat64(key string) float64` | Get as float64 |
| `GetBool()` | `GetBool(key string) bool` | Get as bool |
| `GetSlice()` | `GetSlice(key string) []any` | Get as slice |
| `GetMap()` | `GetMap(key string) map[string]any` | Get as map |
| `Has()` | `Has(key string) bool` | Check if exists |
| `Delete()` | `Delete(key string)` | Delete prop |
| `Keys()` | `Keys() []string` | Get all keys |
| `Values()` | `Values() []any` | Get all values |
| `Clone()` | `Clone() Props` | Clone props |
| `Merge()` | `Merge(other Props)` | Merge props |
| `ToJSON()` | `ToJSON() ([]byte, error)` | JSON serialization |
| `Equals()` | `Equals(other Props) bool` | Compare props |
| `PropDefinition` | `struct` | Prop definition |
| `PropSchema` | `struct` | Prop schema |
| `NewPropSchema()` | `NewPropSchema() *PropSchema` | Create schema |
| `Define()` | `Define(name string, typ reflect.Kind) *PropSchema` | Define prop |
| `DefineWithValidator()` | `DefineWithValidator(name string, fn func(any) error) *PropSchema` | Define with validator |
| `Validate()` | `Validate(props Props) error` | Validate props |
| `ApplyDefaults()` | `ApplyDefaults(props Props) Props` | Apply defaults |
| `ValidateAndApply()` | `ValidateAndApply(props Props) (Props, error)` | Validate and apply |
| `GetDefinition()` | `GetDefinition(name string) PropDefinition` | Get definition |
| `Definitions()` | `Definitions() []PropDefinition` | Get all definitions |
| `BindableProp` | `struct` | Bindable property |
| `NewBindableProp()` | `NewBindableProp(name string, value any) *BindableProp` | Create bindable |
| `Get()` | `Get() any` | Get value |
| `Set()` | `Set(value any) error` | Set value |
| `Name()` | `Name() string` | Get name |
| `OnChange()` | `OnChange(fn func(any))` | Change callback |
| `SetValidator()` | `SetValidator(fn func(any) error)` | Set validator |
| `Bind()` | `Bind(other *BindableProp)` | Two-way bind |
| `BindableProps` | `struct` | Collection of bindables |
| `NewBindableProps()` | `NewBindableProps() *BindableProps` | Create collection |
| `Add()` | `Add(prop *BindableProp)` | Add bindable |
| `Get()` | `Get(name string) *BindableProp` | Get bindable |
| `Remove()` | `Remove(name string)` | Remove bindable |
| `Names()` | `Names() []string` | Get all names |
| `ToProps()` | `ToProps() Props` | Convert to props |

---

### 1.6 `templ/` Package

#### `templ/render.go` - Partially Documented

| Function | Signature | Description |
|----------|-----------|-------------|
| `RuntimeScriptInline()` | `RuntimeScriptInline() templ.Component` | Inline runtime script |
| `CSS()` | `CSS(css string) templ.Component` | CSS component |
| `CSSInline()` | `CSSInline(css string) templ.Component` | Inline CSS |
| `Meta()` | `Meta(name, content string) templ.Component` | Meta tag |
| `MetaProperty()` | `MetaProperty(property, content string) templ.Component` | Meta property tag |
| `Title()` | `Title(title string) templ.Component` | Title tag |
| `Favicon()` | `Favicon(href string) templ.Component` | Favicon link |
| `Head()` | `Head(elements ...templ.Component) templ.Component` | Head wrapper |
| `HTMLPage()` | `HTMLPage(title string, body templ.Component) templ.Component` | Full HTML page |
| `SPAPage()` | `SPAPage(config SPAConfig) templ.Component` | SPA page wrapper |
| `MetaTag` | `struct` | Meta tag definition |
| `SPAConfig` | `struct` | SPA configuration |
| `Raw()` | `Raw(html string) templ.Component` | Raw HTML |
| `HTMLContent()` | `HTMLContent(html string) templ.Component` | HTML content |
| `TextContent()` | `TextContent(text string) templ.Component` | Text content |
| `Attrs()` | `Attrs(attrs map[string]any) templ.Attributes` | Attributes helper |
| `Class()` | `Class(classes ...string) templ.Attributes` | Class attribute |
| `ClassIf()` | `ClassIf(condition bool, class string) templ.Attributes` | Conditional class |
| `Style()` | `Style(styles string) templ.Attributes` | Style attribute |
| `DataAttrs()` | `DataAttrs(data map[string]any) templ.Attributes` | Data attributes |
| `ID()` | `ID(id string) templ.Attributes` | ID attribute |
| `Name()` | `Name(name string) templ.Attributes` | Name attribute |
| `Type()` | `Type(typ string) templ.Attributes` | Type attribute |
| `ValueAttr()` | `ValueAttr(value string) templ.Attributes` | Value attribute |
| `Placeholder()` | `Placeholder(placeholder string) templ.Attributes` | Placeholder attribute |
| `Disabled()` | `Disabled(disabled bool) templ.Attributes` | Disabled attribute |
| `Readonly()` | `Readonly(readonly bool) templ.Attributes` | Readonly attribute |
| `Required()` | `Required(required bool) templ.Attributes` | Required attribute |
| `CheckedAttr()` | `CheckedAttr(checked bool) templ.Attributes` | Checked attribute |
| `Selected()` | `Selected(selected bool) templ.Attributes` | Selected attribute |
| `Hidden()` | `Hidden(hidden bool) templ.Attributes` | Hidden attribute |
| `Href()` | `Href(href string) templ.Attributes` | Href attribute |
| `Src()` | `Src(src string) templ.Attributes` | Src attribute |
| `Alt()` | `Alt(alt string) templ.Attributes` | Alt attribute |
| `Target()` | `Target(target string) templ.Attributes` | Target attribute |
| `Rel()` | `Rel(rel string) templ.Attributes` | Rel attribute |
| `Aria()` | `Aria(attrs map[string]string) templ.Attributes` | Aria attributes |
| `Role()` | `Role(role string) templ.Attributes` | Role attribute |
| `TabIndex()` | `TabIndex(index int) templ.Attributes` | Tabindex attribute |
| `Fragment()` | `Fragment(components ...templ.Component) templ.Component` | Fragment wrapper |
| `Empty()` | `Empty() templ.Component` | Empty component |
| `When()` | `When(condition bool, comp templ.Component) templ.Component` | Conditional render |
| `WhenElse()` | `WhenElse(condition bool, ifComp, elseComp templ.Component) templ.Component` | Conditional with else |
| `For()` | `For(items []any, render func(any) templ.Component) templ.Component` | List render |
| `ForKey()` | `ForKey(items []any, key func(any) string, render func(any) templ.Component) templ.Component` | Keyed list render |
| `Switch()` | `Switch(value any, cases ...CaseComponent) templ.Component` | Switch component |
| `Case()` | `Case(value any, comp templ.Component) CaseComponent` | Case component |
| `Default()` | `Default(comp templ.Component) CaseComponent` | Default case |
| `HeadManager` | `struct` | Head element manager |
| `NewHeadManager()` | `NewHeadManager() *HeadManager` | Create manager |
| `SetHeadTitle()` | `SetHeadTitle(title string)` | Set title |
| `AddHeadMeta()` | `AddHeadMeta(name, content string)` | Add meta |
| `AddHeadMetaProperty()` | `AddHeadMetaProperty(property, content string)` | Add meta property |
| `AddHeadLink()` | `AddHeadLink(rel, href string)` | Add link |
| `AddHeadScript()` | `AddHeadScript(src string)` | Add script |
| `AddHeadInlineScript()` | `AddHeadInlineScript(script string)` | Add inline script |
| `AddHeadStyle()` | `AddHeadStyle(href string)` | Add stylesheet |
| `AddHeadInlineStyle()` | `AddHeadInlineStyle(css string)` | Add inline style |
| `AddHeadElement()` | `AddHeadElement(element templ.Component)` | Add custom element |
| `Render()` | `Render() templ.Component` | Render head |
| `HeadTitle()` | `HeadTitle(title string) templ.Component` | Title helper |
| `HeadMeta()` | `HeadMeta(name, content string) templ.Component` | Meta helper |
| `HeadMetaProp()` | `HeadMetaProp(property, content string) templ.Component` | Meta property helper |
| `HeadLink()` | `HeadLink(rel, href string) templ.Component` | Link helper |
| `HeadScript()` | `HeadScript(src string) templ.Component` | Script helper |
| `HeadStyle()` | `HeadStyle(href string) templ.Component` | Style helper |

#### `templ/events.go` - Partially Documented

| Function | Signature | Description |
|----------|-----------|-------------|
| `EventHandler` | `struct` | Event handler definition |
| `OnWithModifiers()` | `OnWithModifiers(event, componentID, handler string, mods EventModifiers) templ.Attributes` | Event with modifiers |
| `OnClickPrevent()` | `OnClickPrevent(componentID, handler string) templ.Attributes` | Click with preventDefault |
| `OnInput()` | `OnInput(componentID, handler string) templ.Attributes` | Input event |
| `OnChange()` | `OnChange(componentID, handler string) templ.Attributes` | Change event |
| `OnSubmit()` | `OnSubmit(componentID, handler string) templ.Attributes` | Submit event |
| `OnKeydown()` | `OnKeydown(componentID, handler string) templ.Attributes` | Keydown event |
| `OnKeyup()` | `OnKeyup(componentID, handler string) templ.Attributes` | Keyup event |
| `OnFocus()` | `OnFocus(componentID, handler string) templ.Attributes` | Focus event |
| `OnBlur()` | `OnBlur(componentID, handler string) templ.Attributes` | Blur event |
| `OnMouseenter()` | `OnMouseenter(componentID, handler string) templ.Attributes` | Mouseenter event |
| `OnMouseleave()` | `OnMouseleave(componentID, handler string) templ.Attributes` | Mouseleave event |
| `Debounced()` | `Debounced(event, componentID, handler string, delay int) templ.Attributes` | Debounced event |
| `Throttled()` | `Throttled(event, componentID, handler string, delay int) templ.Attributes` | Throttled event |
| `OnKey()` | `OnKey(componentID, handler, key string) templ.Attributes` | Specific key event |
| `OnKeys()` | `OnKeys(componentID, handler string, keys []string) templ.Attributes` | Multiple keys event |
| `OnCtrlKey()` | `OnCtrlKey(componentID, handler string) templ.Attributes` | Ctrl key event |
| `OnShiftKey()` | `OnShiftKey(componentID, handler string) templ.Attributes` | Shift key event |
| `OnAltKey()` | `OnAltKey(componentID, handler string) templ.Attributes` | Alt key event |
| `OnKeyCombo()` | `OnKeyCombo(componentID, handler string, keys []string) templ.Attributes` | Key combo event |
| `ServerAction()` | `ServerAction(componentID, action string) templ.Attributes` | Server action |
| `ServerActionJSON()` | `ServerActionJSON(componentID, action string) templ.Attributes` | JSON server action |
| `FormAction()` | `FormAction(componentID, action string) templ.Attributes` | Form action |
| `Navigate()` | `Navigate(path string) templ.Attributes` | Navigation action |
| `NavigateBack()` | `NavigateBack() templ.Attributes` | Back navigation |
| `ScrollTo()` | `ScrollTo(elementID string) templ.Attributes` | Scroll to element |
| `ScrollToTop()` | `ScrollToTop() templ.Attributes` | Scroll to top |
| `Toggle()` | `Toggle(componentID, key string) templ.Attributes` | Toggle state |
| `Increment()` | `Increment(componentID, key string) templ.Attributes` | Increment state |
| `Decrement()` | `Decrement(componentID, key string) templ.Attributes` | Decrement state |
| `SetState()` | `SetState(componentID, key string, value any) templ.Attributes` | Set state |
| `PreventDefault()` | `PreventDefault() EventModifier` | Prevent default modifier |
| `StopPropagation()` | `StopPropagation() EventModifier` | Stop propagation modifier |
| `Once()` | `Once() EventModifier` | Once modifier |
| `Passive()` | `Passive() EventModifier` | Passive modifier |
| `Capture()` | `Capture() EventModifier` | Capture modifier |
| `Self()` | `Self() EventModifier` | Self modifier |
| `EventModifiers()` | `EventModifiers(mods ...EventModifier) EventModifiers` | Combine modifiers |

#### `templ/bind.go` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `BindingType` | `type` | Binding type enum |
| `TextBind` | `const` | Text binding |
| `HTMLBind` | `const` | HTML binding |
| `ValueBind` | `const` | Value binding |
| `CheckedBind` | `const` | Checked binding |
| `ClassBind` | `const` | Class binding |
| `StyleBind` | `const` | Style binding |
| `AttrBind` | `const` | Attribute binding |
| `PropBind` | `const` | Property binding |
| `ShowBind` | `const` | Show binding |
| `IfBind` | `const` | If binding |
| `Binding` | `struct` | Binding definition |
| `BindWithAttr()` | `BindWithAttr(componentID, key, attr string) templ.Attributes` | Bind with attribute |
| `BindWithTransform()` | `BindWithTransform(componentID, key string, fn func(any) any) templ.Attributes` | Bind with transform |
| `TwoWayBind()` | `TwoWayBind(componentID, key string) templ.Attributes` | Two-way binding |
| `ClassBinding()` | `ClassBinding(componentID, key, className string) templ.Attributes` | Class binding |
| `ClassBindings()` | `ClassBindings(componentID string, bindings map[string]string) templ.Attributes` | Multiple class bindings |
| `StyleBinding()` | `StyleBinding(componentID, key, property string) templ.Attributes` | Style binding |
| `ShowBinding()` | `ShowBinding(componentID, key string) templ.Attributes` | Show binding |
| `IfBinding()` | `IfBinding(componentID, key string) templ.Attributes` | If binding |
| `ListBinding()` | `ListBinding(componentID, key string) templ.Attributes` | List binding |
| `ListBindingWithKey()` | `ListBindingWithKey(componentID, key, itemKey string) templ.Attributes` | Keyed list binding |
| `AttrBinding()` | `AttrBinding(componentID, key, attr string) templ.Attributes` | Attribute binding |
| `AttrBindings()` | `AttrBindings(componentID string, bindings map[string]string) templ.Attributes` | Multiple attr bindings |
| `PropBinding()` | `PropBinding(componentID, key, prop string) templ.Attributes` | Property binding |
| `Text()` | `Text(componentID, key string) templ.Attributes` | Text shorthand |
| `HTML()` | `HTML(componentID, key string) templ.Attributes` | HTML shorthand |
| `Value()` | `Value(componentID, key string) templ.Attributes` | Value shorthand |
| `Checked()` | `Checked(componentID, key string) templ.Attributes` | Checked shorthand |
| `ComponentState` | `struct` | Component state manager |
| `NewComponentState()` | `NewComponentState(componentID string) *ComponentState` | Create state manager |
| `AddRune()` | `AddRune(key string, rune *state.Rune)` | Add rune |
| `GetRune()` | `GetRune(key string) *state.Rune` | Get rune |
| `AddBinding()` | `AddBinding(binding Binding)` | Add binding |
| `ToJSON()` | `ToJSON() ([]byte, error)` | JSON serialization |
| `StateAttrs()` | `StateAttrs() templ.Attributes` | State attributes |
| `InitScript()` | `InitScript() templ.Component` | Init script |
| `RenderBindings()` | `RenderBindings() templ.Component` | Render bindings |
| `SafeHTML()` | `SafeHTML(html string) templ.SafeHTML` | Safe HTML helper |
| `SafeAttr()` | `SafeAttr(value string) templ.SafeAttribute` | Safe attribute helper |

#### `templ/layout.go` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `LayoutProps` | `struct` | Layout properties |
| `LayoutFunc` | `type` | Layout function type |
| `LayoutLoader` | `interface` | Layout loader interface |
| `LayoutLoaderFunc` | `type` | Layout loader function |
| `Layout` | `struct` | Layout definition |
| `LayoutChain` | `struct` | Layout chain |
| `NewLayoutChain()` | `NewLayoutChain() *LayoutChain` | Create chain |
| `RenderLayout()` | `RenderLayout(chain *LayoutChain, props LayoutProps) templ.Component` | Render layout chain |
| `LayoutComponent()` | `LayoutComponent(layout *Layout, content templ.Component, props LayoutProps) templ.Component` | Layout component |
| `WithData()` | `WithData(data map[string]any) LayoutProps` | Set layout data |
| `WithSlot()` | `WithSlot(name string, comp templ.Component) LayoutProps` | Add slot |
| `GetSlot()` | `GetSlot(name string) templ.Component` | Get slot |
| `HasSlot()` | `HasSlot(name string) bool` | Check slot exists |
| `GetProp()` | `GetProp(key string) any` | Get prop |
| `GetParam()` | `GetParam(key string) string` | Get route param |
| `RenderChildren()` | `RenderChildren() templ.Component` | Render children |
| `LayoutContext` | `struct` | Layout context |
| `WithLayoutContext()` | `WithLayoutContext(ctx context.Context, lc *LayoutContext) context.Context` | Set layout context |
| `GetLayoutContext()` | `GetLayoutContext(ctx context.Context) *LayoutContext` | Get layout context |
| `GetLayoutData()` | `GetLayoutData(ctx context.Context) map[string]any` | Get layout data |
| `GetLayoutParam()` | `GetLayoutParam(ctx context.Context, key string) string` | Get layout param |
| `GetLayoutPath()` | `GetLayoutPath(ctx context.Context) string` | Get layout path |
| `NestedLayout` | `struct` | Nested layout builder |
| `NewNestedLayout()` | `NewNestedLayout(layout *Layout) *NestedLayout` | Create nested layout |
| `Nest()` | `Nest(layout *Layout) *NestedLayout` | Add nested layout |
| `Build()` | `Build() *LayoutChain` | Build chain |
| `RootLayout()` | `RootLayout(layout *Layout) *LayoutChain` | Create root chain |
| `WrapLayout()` | `WrapLayout(inner, outer *Layout) *LayoutChain` | Wrap layout |

---

### 1.7 `cli/` Package - **ENTIRELY UNDOCUMENTED**

#### `cli/create.go`

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `ProjectConfig` | `struct` | Project configuration |
| `CreateProject()` | `CreateProject(name string) error` | Create new project |
| `CreateProjectWithConfig()` | `CreateProjectWithConfig(config ProjectConfig) error` | Create with config |
| `ValidateProjectName()` | `ValidateProjectName(name string) error` | Validate project name |

#### `cli/build.go`

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `BuildConfig` | `struct` | Build configuration |
| `Build()` | `Build() error` | Build project |
| `BuildWithConfig()` | `BuildWithConfig(config BuildConfig) error` | Build with config |
| `BuildAll()` | `BuildAll() error` | Build all platforms |
| `Clean()` | `Clean() error` | Clean build artifacts |
| `Watch()` | `Watch() error` | Watch and rebuild |

#### `cli/generate.go`

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `GenerateConfig` | `struct` | Generator configuration |
| `Generate()` | `Generate() error` | Generate routes |
| `GenerateWithConfig()` | `GenerateWithConfig(config GenerateConfig) error` | Generate with config |

#### `cli/dev.go`

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `DevConfig` | `struct` | Dev server configuration |
| `Dev()` | `Dev() error` | Start dev server |
| `DevWithConfig()` | `DevWithConfig(config DevConfig) error` | Dev with config |

#### `cli/output.go`

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `Output` | `struct` | Output formatter |
| `PrintSuccess()` | `PrintSuccess(msg string)` | Print success message |
| `PrintError()` | `PrintError(msg string)` | Print error message |
| `PrintWarning()` | `PrintWarning(msg string)` | Print warning message |
| `PrintInfo()` | `PrintInfo(msg string)` | Print info message |

---

## 2. TypeScript Client API Gaps

### 2.1 `client/src/state.ts` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `Unsubscribe` | `type` | Unsubscribe function type |
| `Subscriber` | `type` | Subscriber callback type |
| `EffectFn` | `type` | Effect function type |
| `ComputeFn` | `type` | Compute function type |
| `ResourceStatus` | `type` | Resource status type |
| `Rune.value` | `getter/setter` | Direct value access |
| `Rune.toJSON()` | `toJSON() any` | JSON serialization |
| `Derived.value` | `getter` | Direct value access |
| `Derived.dispose()` | `dispose()` | Cleanup derived |
| `derived()` | `derived(fn: ComputeFn<T>)` | Derived factory |
| `effect()` | `effect(fn: EffectFn)` | Effect factory |
| `watch()` | `watch(sources, callback)` | Watch factory |
| `StateMap.set()` | `set(key, value)` | Set value |
| `StateMap.get()` | `get(key)` | Get rune |
| `StateMap.has()` | `has(key)` | Check exists |
| `StateMap.delete()` | `delete(key)` | Delete entry |
| `StateMap.clear()` | `clear()` | Clear all |
| `StateMap.toJSON()` | `toJSON()` | JSON serialization |
| `StateMap.fromJSON()` | `fromJSON(data)` | Load from JSON |
| `stateMap()` | `stateMap()` | StateMap factory |
| `untrack()` | `untrack(fn)` | Untrack reads |
| `PreEffect` | `class` | Pre-DOM effect |
| `preEffect()` | `preEffect(fn)` | PreEffect factory |
| `RuneRaw` | `class` | Shallow reactive |
| `runeRaw()` | `runeRaw(value)` | RuneRaw factory |
| `snapshot()` | `snapshot(rune)` | Snapshot helper |
| `EffectRoot` | `class` | Effect root |
| `stop()` | `stop()` | Stop effect root |
| `restart()` | `restart()` | Restart effect root |
| `dispose()` | `dispose()` | Cleanup effect root |
| `effectRoot()` | `effectRoot(fn)` | EffectRoot factory |
| `tracking()` | `tracking()` | Check if tracking |
| `DerivedAsync` | `class` | Async derived |
| `DerivedAsync.value` | `getter` | Value getter |
| `DerivedAsync.error` | `getter` | Error getter |
| `DerivedAsync.status` | `getter` | Status getter |
| `DerivedAsync.isPending` | `getter` | Pending check |
| `DerivedAsync.isSuccess` | `getter` | Success check |
| `DerivedAsync.isError` | `getter` | Error check |
| `DerivedAsync.get()` | `get()` | Get value |
| `DerivedAsync.subscribe()` | `subscribe(fn)` | Subscribe |
| `DerivedAsync.dispose()` | `dispose()` | Cleanup |
| `derivedAsync()` | `derivedAsync(fn)` | DerivedAsync factory |
| `Resource` | `class` | Resource container |
| `Resource.data` | `getter` | Data getter |
| `Resource.error` | `getter` | Error getter |
| `Resource.status` | `getter` | Status getter |
| `Resource.isIdle` | `getter` | Idle check |
| `Resource.isPending` | `getter` | Pending check |
| `Resource.isSuccess` | `getter` | Success check |
| `Resource.isError` | `getter` | Error check |
| `Resource.refetch()` | `refetch()` | Refetch data |
| `Resource.reset()` | `reset()` | Reset resource |
| `resource()` | `resource(fn)` | Resource factory |
| `resourceReactive()` | `resourceReactive(sources, fn)` | Reactive resource |
| `inspect()` | `inspect(...values)` | Debug inspector |
| `inspect.trace()` | `trace(label?)` | Trace dependencies |
| `watchPath()` | `watchPath(obj, path, fn)` | Watch object path |
| `derivedPath()` | `derivedPath(obj, path)` | Derived from path |

### 2.2 `client/src/runtime-core.ts` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `ComponentDefinition` | `interface` | Component definition |
| `ComponentInstance` | `interface` | Component instance |
| `RuntimeConfig` | `interface` | Runtime configuration |
| `StateMessage` | `interface` | State message |
| `init()` | `init(config)` | Initialize runtime |
| `createComponent()` | `createComponent(id, def)` | Create component |
| `destroyComponent()` | `destroyComponent(id)` | Destroy component |
| `getComponent()` | `getComponent(id)` | Get component |
| `getState()` | `getState(id)` | Get component state |
| `setState()` | `setState(id, state)` | Set component state |
| `callAction()` | `callAction(action, payload)` | Call server action |
| `bind()` | `bind(element, rune)` | Bind element |
| `autoInit()` | `autoInit()` | Auto-initialize from DOM |
| `getWebSocket()` | `getWebSocket()` | Lazy load WebSocket |
| `getNavigation()` | `getNavigation()` | Lazy load navigation |
| `getTransitions()` | `getTransitions()` | Lazy load transitions |

### 2.3 `client/src/websocket.ts` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `ConnectionState` | `type` | Connection state type |
| `MessageType` | `type` | Message type enum |
| `StateMessage` | `interface` | State message |
| `WebSocketConfig` | `interface` | WebSocket config |
| `SyncedStateOptions` | `interface` | Synced state options |
| `WSClient` | `class` | WebSocket client |
| `WSClient.state` | `getter` | State map |
| `WSClient.isConnected` | `getter` | Connection check |
| `WSClient.connect()` | `connect()` | Connect to server |
| `WSClient.disconnect()` | `disconnect()` | Disconnect |
| `WSClient.send()` | `send(data)` | Send data |
| `WSClient.sendWithResponse()` | `sendWithResponse(data)` | Send and wait |
| `WSClient.requestSync()` | `requestSync()` | Request state sync |
| `WSClient.sendAction()` | `sendAction(action, payload)` | Send action |
| `WSClient.requestState()` | `requestState(key)` | Request state value |
| `sendAction()` | `sendAction(action, payload)` | Global send action |
| `getWebSocketClient()` | `getWebSocketClient()` | Get client instance |
| `initWebSocket()` | `initWebSocket(url, config?)` | Initialize WebSocket |
| `syncedRune()` | `syncedRune(key, initial)` | Server-synced rune |
| `syncBatch()` | `syncBatch(keys)` | Batch sync |
| `applyStateUpdate()` | `applyStateUpdate(update)` | Apply state update |

### 2.4 `client/src/navigation.ts` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `NavigationOptions` | `interface` | Navigation options |
| `NavigationCallback` | `type` | Navigation callback |
| `PageData` | `interface` | Page data |
| `onBeforeNavigate()` | `onBeforeNavigate(fn)` | Before navigate hook |
| `onAfterNavigate()` | `onAfterNavigate(fn)` | After navigate hook |
| `navigate()` | `navigate(path, options?)` | Navigate to path |
| `back()` | `back()` | Go back |
| `forward()` | `forward()` | Go forward |
| `go()` | `go(delta)` | Go delta |
| `getCurrentPath()` | `getCurrentPath()` | Get current path |
| `isNavigating()` | `isNavigating()` | Check if navigating |
| `initNavigation()` | `initNavigation()` | Initialize navigation |
| `destroyNavigation()` | `destroyNavigation()` | Cleanup navigation |
| `prefetch()` | `prefetch(path, options?)` | Prefetch page |
| `createNavigationState()` | `createNavigationState()` | Create nav state |

### 2.5 `client/src/dom.ts` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `BindingType` | `enum` | Binding type enum |
| `Binding` | `interface` | Binding definition |
| `setSanitizer()` | `setSanitizer(fn)` | Set HTML sanitizer |
| `sanitizeHtml` | `var` | Sanitizer function |
| `registerBinding()` | `registerBinding(type, fn)` | Register binding |
| `unregisterBinding()` | `unregisterBinding(type)` | Unregister binding |
| `bindElement()` | `bindElement(id, rune)` | Bind element |
| `bindDerived()` | `bindDerived(id, derived)` | Bind derived |
| `bindTwoWay()` | `bindTwoWay(id, rune)` | Two-way bind |
| `querySelector()` | `querySelector(sel)` | Query selector |
| `querySelectorAll()` | `querySelectorAll(sel)` | Query all |
| `createElement()` | `createElement(tag, attrs?)` | Create element |
| `renderIf()` | `renderIf(condition, element)` | Conditional render |
| `renderList()` | `renderList(items, render)` | List render |

### 2.6 `client/src/events.ts` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `EventHandler` | `type` | Event handler type |
| `ModifierHandler` | `type` | Modifier handler |
| `EventModifier` | `interface` | Event modifier |
| `EventConfig` | `interface` | Event config |
| `on()` | `on(el, type, handler)` | Add event listener |
| `offAll()` | `offAll(el)` | Remove all listeners |
| `debounce()` | `debounce(fn, delay)` | Debounce function |
| `throttle()` | `throttle(fn, delay)` | Throttle function |
| `bindEvent()` | `bindEvent(el, config)` | Bind event config |
| `transformers` | `object` | Event transformers |
| `transformers.value` | `fn` | Value transformer |
| `transformers.checked` | `fn` | Checked transformer |
| `transformers.numberValue` | `fn` | Number transformer |
| `transformers.files` | `fn` | Files transformer |
| `transformers.formData` | `fn` | FormData transformer |
| `delegate()` | `delegate(container, sel, type, fn)` | Event delegation |
| `onKey()` | `onKey(keys, handler)` | Key handler |
| `parseEventString()` | `parseEventString(str)` | Parse event string |
| `keys` | `object` | Key constants |
| `keys.enter` | `const` | Enter key |
| `keys.escape` | `const` | Escape key |
| `keys.tab` | `const` | Tab key |
| `keys.space` | `const` | Space key |
| `keys.arrowUp` | `const` | Arrow up |
| `keys.arrowDown` | `const` | Arrow down |
| `keys.arrowLeft` | `const` | Arrow left |
| `keys.arrowRight` | `const` | Arrow right |

### 2.7 `client/src/transition.ts` - Partially Documented

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `TransitionConfig` | `interface` | Transition config |
| `TransitionFn` | `type` | Transition function |
| `linear` | `fn` | Linear easing |
| `cubicOut` | `fn` | Cubic out easing |
| `cubicInOut` | `fn` | Cubic in-out easing |
| `elasticOut` | `fn` | Elastic out easing |
| `bounceOut` | `fn` | Bounce out easing |
| `fade()` | `fade(el, config?)` | Fade transition |
| `fly()` | `fly(el, config?)` | Fly transition |
| `slide()` | `slide(el, config?)` | Slide transition |
| `scale()` | `scale(el, config?)` | Scale transition |
| `blur()` | `blur(el, config?)` | Blur transition |
| `crossfade()` | `crossfade(a, b, config?)` | Crossfade transition |
| `transitionIn()` | `transitionIn(el, fn, config?)` | Transition in |
| `transitionOut()` | `transitionOut(el, fn, config?)` | Transition out |
| `setupTransitions()` | `setupTransitions(el, config)` | Setup transitions |

### 2.8 `client/src/sanitize.ts` - **ENTIRELY UNDOCUMENTED**

| Type/Function | Signature | Description |
|---------------|-----------|-------------|
| `domPurifySanitizer()` | `domPurifySanitizer(config?)` | DOMPurify sanitizer |

### 2.9 `client/src/state-min.ts` - **ENTIRELY UNDOCUMENTED**

Minified reactive primitives for production builds. Contains same API as `state.ts` but optimized for size.

---

## 3. Website Documentation Gaps

### 3.1 Missing Documentation Pages

| Page | Description | Priority |
|------|-------------|----------|
| `/docs/cli` | CLI commands reference | High |
| `/docs/component-system` | Component system (BaseComponent, ComponentTree, Lifecycle) | High |
| `/docs/error-handling` | Error handling (AppError, ErrorCode, ErrorHandler) | High |
| `/docs/params` | Route parameters (Params, QueryParams, PathBuilder) | Medium |
| `/docs/dev-tools` | Development tools (DevConfig, FileWatcher, DebugMiddleware) | Medium |

### 3.2 Incomplete Documentation Pages

| Page | Missing Content |
|------|-----------------|
| `/docs/reactive-primitives` | Missing: ID(), MarshalJSON(), DependOn(), Dispose(), Pause(), Resume(), IsActive() |
| `/docs/routing` | Missing: Route struct fields (Priority, Children, Middleware, Meta), Registry, RemoteRegistry |
| `/docs/client-runtime` | Missing: Full API reference, only has basic examples |
| `/docs/api` | Missing: Many Config options, CLI commands, Component system |

---

## 4. Recommended Documentation Additions

### 4.1 New Documentation Files to Create

1. **`docs/CLI.md`** - Complete CLI reference
2. **`docs/COMPONENT_SYSTEM.md`** - Component system documentation
3. **`docs/ERROR_HANDLING.md`** - Error handling guide
4. **`docs/PARAMS.md`** - Parameter handling reference
5. **`docs/DEV_TOOLS.md`** - Development tools guide

### 4.2 Documentation Updates Required

1. **`docs/API.md`** - Add missing Config options, methods, and types
2. **Website `/docs/reactive-primitives`** - Add advanced API methods
3. **Website `/docs/routing`** - Add Registry, RemoteRegistry, Route fields
4. **Website `/docs/client-runtime`** - Expand to full API reference

---

## 5. Priority Matrix

| Priority | Category | Items |
|----------|----------|-------|
| **Critical** | Core API | Config options, App methods, Rune/Derived/Effect methods |
| **High** | Component System | BaseComponent, ComponentTree, Lifecycle, Props |
| **High** | Error Handling | AppError, ErrorCode, ErrorHandler |
| **High** | CLI | All CLI commands and options |
| **Medium** | Routing | Params, QueryParams, PathBuilder, Registry |
| **Medium** | Dev Tools | DevConfig, FileWatcher, DebugMiddleware |
| **Low** | Client Advanced | RuneRaw, EffectRoot, DerivedAsync, Resource |
| **Low** | Transitions | Full transition API |
| **Low** | Sanitization | DOMPurify integration |

---

## 6. Next Steps

1. Create missing documentation files
2. Update existing documentation with missing API items
3. Add code examples for complex APIs
4. Create API reference pages for website
5. Add TypeScript type definitions to documentation
6. Create migration guides for breaking changes
