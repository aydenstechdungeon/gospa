package component

import (
	"context"
	"testing"
)

func TestNewBaseComponent(t *testing.T) {
	c := NewBaseComponent("comp-1", "test-component")

	if c.ID() != "comp-1" {
		t.Errorf("Expected ID 'comp-1', got %s", c.ID())
	}

	if c.Name() != "test-component" {
		t.Errorf("Expected name 'test-component', got %s", c.Name())
	}

	if c.State() == nil {
		t.Error("Expected state to be initialized")
	}

	if c.Props() == nil {
		t.Error("Expected props to be initialized")
	}

	if len(c.Children()) != 0 {
		t.Errorf("Expected no children initially, got %d", len(c.Children()))
	}
}

func TestBaseComponentWithProps(t *testing.T) {
	props := Props{"title": "Test", "count": 42}
	c := NewBaseComponent("comp-1", "test", WithProps(props))

	if c.Props().Get("title") != "Test" {
		t.Errorf("Expected title 'Test', got %v", c.Props().Get("title"))
	}

	if c.Props().Get("count") != 42 {
		t.Errorf("Expected count 42, got %v", c.Props().Get("count"))
	}
}

func TestBaseComponentWithState(t *testing.T) {
	initialState := map[string]interface{}{
		"count": 0,
		"name":  "test",
	}
	c := NewBaseComponent("comp-1", "test", WithState(initialState))

	stateMap := c.State()
	if obs, ok := stateMap.Get("count"); !ok {
		t.Error("Expected to find 'count' in state")
	} else if obs.GetAny() != 0 {
		t.Errorf("Expected count 0, got %v", obs.GetAny())
	}
}

func TestBaseComponentAddChild(t *testing.T) {
	parent := NewBaseComponent("parent", "parent")
	child := NewBaseComponent("child", "child")

	parent.AddChild(child)

	if len(parent.Children()) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children()))
	}

	if parent.Children()[0].ID() != "child" {
		t.Errorf("Expected child ID 'child', got %s", parent.Children()[0].ID())
	}

	if child.Parent() != parent {
		t.Error("Expected child's parent to be set correctly")
	}
}

func TestBaseComponentRemoveChild(t *testing.T) {
	parent := NewBaseComponent("parent", "parent")
	child1 := NewBaseComponent("child1", "child")
	child2 := NewBaseComponent("child2", "child")

	parent.AddChild(child1)
	parent.AddChild(child2)
	parent.RemoveChild("child1")

	if len(parent.Children()) != 1 {
		t.Errorf("Expected 1 child after removal, got %d", len(parent.Children()))
	}

	if parent.Children()[0].ID() != "child2" {
		t.Errorf("Expected remaining child to be 'child2', got %s", parent.Children()[0].ID())
	}
}

func TestBaseComponentSetSlot(t *testing.T) {
	c := NewBaseComponent("comp-1", "test")
	c.SetSlot("header", func() string { return "Header Content" })

	slot := c.GetSlot("header")
	if slot == nil {
		t.Error("Expected slot to be set")
	} else if content := slot(); content != "Header Content" {
		t.Errorf("Expected 'Header Content', got %s", content)
	}

	if c.GetSlot("nonexistent") != nil {
		t.Error("Expected nil for nonexistent slot")
	}
}

func TestBaseComponentContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	c := NewBaseComponent("comp-1", "test", WithContext(ctx))

	if c.Context() == nil {
		t.Error("Expected context to be set")
	}

	if c.Context().Value("key") != "value" {
		t.Errorf("Expected context value 'value', got %v", c.Context().Value("key"))
	}

	// Test SetContext
	newCtx := context.WithValue(context.Background(), "key", "new-value")
	c.SetContext(newCtx)

	if c.Context().Value("key") != "new-value" {
		t.Errorf("Expected updated context value 'new-value', got %v", c.Context().Value("key"))
	}
}

func TestBaseComponentToJSON(t *testing.T) {
	initialState := map[string]interface{}{"count": 42}
	c := NewBaseComponent("comp-1", "test", WithState(initialState))

	json, err := c.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if json == "" {
		t.Error("Expected non-empty JSON")
	}
}

func TestBaseComponentClone(t *testing.T) {
	parent := NewBaseComponent("parent", "parent")
	child := NewBaseComponent("child", "child")
	parent.AddChild(child)

	clone := parent.Clone()

	if clone.ID() != "parent_clone" {
		t.Errorf("Expected cloned ID 'parent_clone', got %s", clone.ID())
	}

	if clone.Name() != parent.Name() {
		t.Errorf("Expected cloned name to match original")
	}

	if len(clone.Children()) != len(parent.Children()) {
		t.Errorf("Expected %d cloned children, got %d", len(parent.Children()), len(clone.Children()))
	}
}

func TestNewComponentTree(t *testing.T) {
	root := NewBaseComponent("root", "root")
	child1 := NewBaseComponent("child1", "child")
	child2 := NewBaseComponent("child2", "child")
	grandchild := NewBaseComponent("grandchild", "grandchild")

	root.AddChild(child1)
	root.AddChild(child2)
	child1.AddChild(grandchild)

	tree := NewComponentTree(root)

	if tree.Root() != root {
		t.Error("Expected root to be set correctly")
	}

	if tree.Get("root") != root {
		t.Error("Expected to get root by ID")
	}

	if tree.Get("child1") != child1 {
		t.Error("Expected to get child1 by ID")
	}

	if tree.Get("child2") != child2 {
		t.Error("Expected to get child2 by ID")
	}

	if tree.Get("grandchild") != grandchild {
		t.Error("Expected to get grandchild by ID")
	}

	if tree.Get("nonexistent") != nil {
		t.Error("Expected nil for nonexistent component")
	}
}

func TestComponentTreeAdd(t *testing.T) {
	root := NewBaseComponent("root", "root")
	tree := NewComponentTree(root)

	child := NewBaseComponent("child", "child")
	err := tree.Add("root", child)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if tree.Get("child") != child {
		t.Error("Expected child to be added to tree")
	}

	// Test adding to nonexistent parent
	err = tree.Add("nonexistent", NewBaseComponent("orphan", "orphan"))
	if err == nil {
		t.Error("Expected error when adding to nonexistent parent")
	}
}

func TestComponentTreeRemove(t *testing.T) {
	root := NewBaseComponent("root", "root")
	child := NewBaseComponent("child", "child")
	root.AddChild(child)

	tree := NewComponentTree(root)

	err := tree.Remove("child")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if tree.Get("child") != nil {
		t.Error("Expected child to be removed from tree")
	}

	// Test removing nonexistent component
	err = tree.Remove("nonexistent")
	if err == nil {
		t.Error("Expected error when removing nonexistent component")
	}
}

func TestComponentTreeLifecycleHooks(t *testing.T) {
	root := NewBaseComponent("root", "root")
	tree := NewComponentTree(root)

	mountCalled := false
	updateCalled := false
	tree.OnMount("root", func(c Component) {
		mountCalled = true
	})

	tree.OnUpdate("root", func(c Component) {
		updateCalled = true
	})

	tree.Mount("root")
	if !mountCalled {
		t.Error("Expected mount hook to be called")
	}

	tree.Update("root")
	if !updateCalled {
		t.Error("Expected update hook to be called")
	}

	// Create a child to test destroy
	child := NewBaseComponent("child", "child")
	root.AddChild(child)
	tree.Remove("child") // This should trigger any destroy hooks
}

func TestComponentTreeWalk(t *testing.T) {
	root := NewBaseComponent("root", "root")
	child1 := NewBaseComponent("child1", "child")
	child2 := NewBaseComponent("child2", "child")
	root.AddChild(child1)
	root.AddChild(child2)

	tree := NewComponentTree(root)

	var visited []string
	tree.Walk(func(c Component) bool {
		visited = append(visited, string(c.ID()))
		return true
	})

	if len(visited) != 3 {
		t.Errorf("Expected 3 components visited, got %d", len(visited))
	}

	// Test early termination
	visited = nil
	tree.Walk(func(c Component) bool {
		visited = append(visited, string(c.ID()))
		return false // Stop after first
	})

	if len(visited) != 1 {
		t.Errorf("Expected 1 component visited after early termination, got %d", len(visited))
	}
}

func TestComponentTreeFind(t *testing.T) {
	root := NewBaseComponent("root", "root")
	child1 := NewBaseComponent("child1", "child")
	child2 := NewBaseComponent("child2", "child")
	root.AddChild(child1)
	root.AddChild(child2)

	tree := NewComponentTree(root)

	found := tree.Find(func(c Component) bool {
		return c.ID() == "child1"
	})

	if found == nil || found.ID() != "child1" {
		t.Error("Expected to find child1")
	}

	notFound := tree.Find(func(c Component) bool {
		return c.ID() == "nonexistent"
	})

	if notFound != nil {
		t.Error("Expected nil when no component matches")
	}
}

func TestComponentTreeFindAll(t *testing.T) {
	root := NewBaseComponent("root", "root")
	child1 := NewBaseComponent("child1", "child")
	child2 := NewBaseComponent("child2", "child")
	grandchild := NewBaseComponent("grandchild", "grandchild")
	root.AddChild(child1)
	root.AddChild(child2)
	child1.AddChild(grandchild)

	tree := NewComponentTree(root)

	found := tree.FindAll(func(c Component) bool {
		return c.Name() == "child"
	})

	if len(found) != 2 {
		t.Errorf("Expected 2 children with name 'child', got %d", len(found))
	}
}

func TestComponentTreeFindByName(t *testing.T) {
	root := NewBaseComponent("root", "root")
	child := NewBaseComponent("child", "child")
	root.AddChild(child)

	tree := NewComponentTree(root)

	found := tree.FindByName("child")
	if len(found) != 1 || found[0].ID() != "child" {
		t.Errorf("Expected to find 1 child with name 'child', got %v", found)
	}
}

func TestComponentTreeFindByProp(t *testing.T) {
	root := NewBaseComponent("root", "root")
	child := NewBaseComponent("child", "child", WithProps(Props{"type": "special"}))
	root.AddChild(child)

	tree := NewComponentTree(root)

	found := tree.FindByProp("type", "special")
	if len(found) != 1 || found[0].ID() != "child" {
		t.Errorf("Expected to find 1 child with type='special', got %v", found)
	}
}

func TestComponentTreeToJSON(t *testing.T) {
	root := NewBaseComponent("root", "root", WithState(map[string]interface{}{"count": 42}))
	tree := NewComponentTree(root)

	json, err := tree.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if json == "" {
		t.Error("Expected non-empty JSON")
	}
}
