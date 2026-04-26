package kit

import (
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type parentDataProvider interface {
	GospaParentData() map[string]interface{}
}

type executionScopeState struct {
	depends         map[string]struct{}
	dependencyMuted bool
	parentData      map[string]interface{}
}

var scopeByGoID sync.Map

// ExecutionScope controls helper semantics during a single load/action execution flow.
type ExecutionScope struct {
	state *executionScopeState
}

// NewExecutionScope creates a fresh helper execution scope.
func NewExecutionScope() *ExecutionScope {
	return &ExecutionScope{
		state: &executionScopeState{
			depends: make(map[string]struct{}),
		},
	}
}

// Run executes fn inside this helper scope.
func (s *ExecutionScope) Run(fn func() error) error {
	if s == nil || s.state == nil {
		return fn()
	}

	goid := currentGoID()
	prev, hadPrev := scopeByGoID.Load(goid)
	scopeByGoID.Store(goid, s.state)
	defer func() {
		if hadPrev {
			scopeByGoID.Store(goid, prev)
			return
		}
		scopeByGoID.Delete(goid)
	}()

	return fn()
}

// SetParentData sets parent data visible to kit.Parent.
func (s *ExecutionScope) SetParentData(parent map[string]interface{}) {
	if s == nil || s.state == nil {
		return
	}
	s.state.parentData = cloneStringAnyMap(parent)
}

// DependsKeys returns captured dependency keys in deterministic order.
func (s *ExecutionScope) DependsKeys() []string {
	if s == nil || s.state == nil || len(s.state.depends) == 0 {
		return nil
	}
	keys := make([]string, 0, len(s.state.depends))
	for key := range s.state.depends {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Depends captures dependency affinity keys for the current scope.
func Depends(keys ...string) {
	state := currentScopeState()
	if state == nil || state.dependencyMuted {
		return
	}
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		state.depends[trimmed] = struct{}{}
	}
}

// Untrack executes fn with dependency capture disabled.
func Untrack(fn func() error) error {
	if fn == nil {
		return nil
	}

	state := currentScopeState()
	if state == nil {
		return fn()
	}

	prev := state.dependencyMuted
	state.dependencyMuted = true
	defer func() {
		state.dependencyMuted = prev
	}()

	return fn()
}

func parentDataFromContext(c interface{}) (map[string]interface{}, bool) {
	if provider, ok := c.(parentDataProvider); ok {
		if parent := provider.GospaParentData(); parent != nil {
			return cloneStringAnyMap(parent), true
		}
	}
	if state := currentScopeState(); state != nil && state.parentData != nil {
		return cloneStringAnyMap(state.parentData), true
	}
	return nil, false
}

func currentScopeState() *executionScopeState {
	goid := currentGoID()
	if goid == 0 {
		return nil
	}
	value, ok := scopeByGoID.Load(goid)
	if !ok {
		return nil
	}
	state, _ := value.(*executionScopeState)
	return state
}

func currentGoID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	line := strings.TrimPrefix(string(buf[:n]), "goroutine ")
	if idx := strings.IndexByte(line, ' '); idx != -1 {
		line = line[:idx]
	}
	id, err := strconv.ParseUint(line, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func cloneStringAnyMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
