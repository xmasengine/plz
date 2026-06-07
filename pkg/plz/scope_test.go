package plz

import "testing"

func TestNewScope(t *testing.T) {
	s := NewScope("global", nil)
	if s.Name != "global" {
		t.Errorf("expected name 'global', got %q", s.Name)
	}
	if s.Parent != nil {
		t.Error("expected nil parent for root scope")
	}
	if len(s.Symbols) != 0 {
		t.Error("expected empty symbols map")
	}
	if s.IsProc || s.IsTask {
		t.Error("expected IsProc and IsTask both false")
	}
}

func TestNewProcScope(t *testing.T) {
	root := NewScope("global", nil)
	s := NewProcScope("main", root)
	if s.Name != "main" {
		t.Errorf("expected name 'main', got %q", s.Name)
	}
	if s.Parent != root {
		t.Error("expected parent to be root")
	}
	if !s.IsProc {
		t.Error("expected IsProc true")
	}
}

func TestNewTaskScope(t *testing.T) {
	root := NewScope("global", nil)
	s := NewTaskScope("t1", root)
	if !s.IsTask {
		t.Error("expected IsTask true")
	}
	if s.IsProc {
		t.Error("expected IsProc false for task scope")
	}
}

func TestAddChild(t *testing.T) {
	root := NewScope("global", nil)
	child := NewScope("child", nil)
	root.AddChild(child)
	if child.Parent != root {
		t.Error("expected child.Parent to be root after AddChild")
	}
	if len(root.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(root.Children))
	}
	if root.Children[0] != child {
		t.Error("expected root.Children[0] to be child")
	}
}

func TestAddChildSetsParent(t *testing.T) {
	root := NewScope("global", nil)
	child := NewScope("child", nil) // no parent set yet
	root.AddChild(child)
	if child.Parent != root {
		t.Error("AddChild should set child.Parent")
	}
}

func TestLookup(t *testing.T) {
	root := NewScope("global", nil)
	child := NewScope("child", root)
	root.Symbols["x"] = Declare{Identifier: "x"}
	child.Symbols["y"] = Declare{Identifier: "y"}

	_, ok := child.Lookup("x")
	if !ok {
		t.Error("expected Lookup from child to find 'x' in root")
	}
}

func TestLookupNotFound(t *testing.T) {
	root := NewScope("global", nil)
	_, ok := root.Lookup("nonexistent")
	if ok {
		t.Error("expected Lookup to return false for nonexistent identifier")
	}
}

func TestLookupShadowing(t *testing.T) {
	root := NewScope("global", nil)
	child := NewScope("child", root)
	root.Symbols["x"] = Declare{Identifier: "x"}
	child.Symbols["x"] = Declare{Identifier: "x"}

	d, ok := child.Lookup("x")
	if !ok {
		t.Fatal("expected Lookup to find 'x'")
	}
	if d != child.Symbols["x"] {
		t.Error("expected inner declaration to shadow outer")
	}

	// From root, should find the root declaration
	d, ok = root.Lookup("x")
	if !ok {
		t.Fatal("expected root Lookup to find 'x'")
	}
	if d != root.Symbols["x"] {
		t.Error("expected root Lookup to return root declaration")
	}
}

func TestLookupInTree(t *testing.T) {
	root := NewScope("global", nil)
	proc := NewProcScope("foo", root)
	root.AddChild(proc)
	block := NewScope("do", proc)
	proc.AddChild(block)

	proc.Symbols["a"] = Declare{Identifier: "a"}
	block.Symbols["b"] = Declare{Identifier: "b"}

	// Search from root
	d, scope, ok := root.LookupInTree("a")
	if !ok {
		t.Fatal("expected LookupInTree to find 'a'")
	}
	if scope != proc {
		t.Errorf("expected scope 'foo', got %q", scope.Name)
	}
	_ = d

	d, scope, ok = root.LookupInTree("b")
	if !ok {
		t.Fatal("expected LookupInTree to find 'b'")
	}
	if scope != block {
		t.Errorf("expected scope 'do', got %q", scope.Name)
	}
}

func TestLookupInTreeNotFound(t *testing.T) {
	root := NewScope("global", nil)
	_, _, ok := root.LookupInTree("nonexistent")
	if ok {
		t.Error("expected LookupInTree to return false")
	}
}

func TestLookupInTreeReturnsFirstDFS(t *testing.T) {
	root := NewScope("global", nil)
	proc := NewProcScope("foo", root)
	root.AddChild(proc)
	block := NewScope("do", proc)
	proc.AddChild(block)

	// Same name in both proc and block level
	proc.Symbols["x"] = Declare{Identifier: "x_proc"}
	block.Symbols["x"] = Declare{Identifier: "x_block"}

	// DFS from root: root (no), children → foo (no), children → do (yes, found)
	_, scope, ok := root.LookupInTree("x")
	if !ok {
		t.Fatal("expected LookupInTree to find 'x'")
	}
	if scope != block {
		t.Error("expected first DFS match to be in innermost scope (block)")
	}
}

func TestNodes(t *testing.T) {
	root := NewScope("global", nil)
	a := NewScope("a", root)
	root.AddChild(a)
	b := NewScope("b", a)
	a.AddChild(b)
	c := NewScope("c", root)
	root.AddChild(c)

	nodes := root.Nodes()
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}
	expected := []string{"global", "a", "b", "c"}
	for i, n := range nodes {
		if n.Name != expected[i] {
			t.Errorf("node %d: expected %q, got %q", i, expected[i], n.Name)
		}
	}
}

func TestDepth(t *testing.T) {
	root := NewScope("global", nil)
	if root.Depth() != 0 {
		t.Errorf("expected root depth 0, got %d", root.Depth())
	}

	proc := NewScope("proc", root)
	if proc.Depth() != 1 {
		t.Errorf("expected proc depth 1, got %d", proc.Depth())
	}

	block := NewScope("do", proc)
	if block.Depth() != 2 {
		t.Errorf("expected block depth 2, got %d", block.Depth())
	}
}

func TestScopeTreePersistence(t *testing.T) {
	// Simulate the checker's push/pop pattern: child scopes survive
	// after the parent pops back to the enclosing scope.
	root := NewScope("global", nil)
	current := root

	// Push procedure scope
	proc := NewProcScope("foo", current)
	current.AddChild(proc)
	current = proc

	// Push block scope
	block := NewScope("do", current)
	current.AddChild(block)
	current = block

	block.Symbols["local"] = Declare{Identifier: "local"}

	// Pop back to proc
	current = current.Parent
	proc.Symbols["param"] = Declare{Identifier: "param"}

	// Pop back to root
	current = current.Parent

	// Verify scope tree is intact
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 root child, got %d", len(root.Children))
	}
	if root.Children[0] != proc {
		t.Error("expected root child to be proc")
	}
	if len(proc.Children) != 1 {
		t.Fatalf("expected 1 proc child, got %d", len(proc.Children))
	}
	if proc.Children[0] != block {
		t.Error("expected proc child to be block")
	}

	// Verify symbols are still there
	_, ok1 := proc.Symbols["param"]
	_, ok2 := block.Symbols["local"]
	if !ok1 {
		t.Error("expected proc.Symbols['param'] to still exist")
	}
	if !ok2 {
		t.Error("expected block.Symbols['local'] to still exist")
	}
}

func TestScopeTreePersistenceWithGenerator(t *testing.T) {
	// This test verifies the generator can navigate a checker-style
	// scope tree using child-index tracking (matching pushScope/
	// popScope in Gen).
	root := NewScope("global", nil)
	current := root

	// Simulate checking phase: push procedures and blocks
	proc1 := NewProcScope("main", current)
	current.AddChild(proc1)
	current = proc1
	proc1.Symbols["x"] = Declare{Identifier: "x"}

	body1 := NewScope("do", current)
	current.AddChild(body1)
	current = body1

	inner1 := NewScope("do", current)
	current.AddChild(inner1)
	current = inner1
	inner1.Symbols["y"] = Declare{Identifier: "y"}
	current = current.Parent // back to body1
	current = current.Parent // back to proc1
	current = current.Parent // back to root

	proc2 := NewProcScope("helper", current)
	current.AddChild(proc2)
	current = proc2
	proc2.Symbols["z"] = Declare{Identifier: "z"}
	current = current.Parent // back to root

	// Now simulate generation side with child-index tracking
	scopeChildIdx := make(map[*Scope]int)

	// Enter main
	idx := scopeChildIdx[root]
	genCurrent := root.Children[idx]
	scopeChildIdx[root] = idx + 1

	if genCurrent != proc1 {
		t.Error("expected generation to enter proc1 first")
	}

	// Enter main's body
	idx = scopeChildIdx[proc1]
	genBody := proc1.Children[idx]
	scopeChildIdx[proc1] = idx + 1

	if genBody != body1 {
		t.Error("expected generation to enter body1")
	}

	// Enter inner block
	idx = scopeChildIdx[body1]
	genInner := body1.Children[idx]
	scopeChildIdx[body1] = idx + 1

	if genInner != inner1 {
		t.Error("expected generation to enter inner1")
	}

	// Verify y is only in inner1, not in body1
	if _, ok := genInner.Lookup("y"); !ok {
		t.Error("expected to find 'y' from inner1 scope")
	}
	if _, ok := genBody.Lookup("y"); ok {
		t.Error("did not expect to find 'y' from body1 scope")
	}
	if _, ok := genBody.Lookup("x"); !ok {
		t.Error("expected to find 'x' from body1 (outer scope)")
	}

	// Pop back to body1
	genCurrent = genBody

	// Pop back to proc1
	genCurrent = proc1

	// Pop back to root
	genCurrent = root

	// Enter helper
	idx = scopeChildIdx[root]
	genHelper := root.Children[idx]
	scopeChildIdx[root] = idx + 1

	if genHelper != proc2 {
		t.Error("expected generation to enter proc2 second")
	}

	if _, ok := genHelper.Lookup("z"); !ok {
		t.Error("expected to find 'z' from helper scope")
	}
	if _, ok := genHelper.Lookup("x"); ok {
		t.Error("did not expect to find 'x' from helper scope (different branch)")
	}
}
