package plz

import "testing"
import "slices"

func TestNewScope(t *testing.T) {
	const (
		parentName = "parent"
		child1Name = "child 1"
		child2Name = "child 2"
	)

	p := NewScope(parentName, nil)
	c1 := NewScope(child1Name, p)
	c2 := NewScope(child2Name, p)

	for _, sc := range []*Scope{p, c1, c2} {
		if sc == nil {
			t.Fatal("should not return nil")
		}
		if sc.IsProc {
			t.Errorf("%v: should not be a proc", sc)
		}
		if sc.IsTask {
			t.Errorf("%v: should not be a task", sc)
		}
		if sc.Symbols == nil {
			t.Errorf("%v: symbols should not be nil", sc)
		}
	}

	if p.Name != parentName {
		t.Errorf("Name: expected %s got %s", parentName, p.Name)
	}
	if c1.Name != child1Name {
		t.Errorf("Name: expected %s got %s", child1Name, c1.Name)
	}
	if c2.Name != child2Name {
		t.Errorf("Name: expected %s got %s", child2Name, c2.Name)
	}

	if c1.Parent != p {
		t.Errorf("Parent: expected %v got %v", p, c1.Parent)
	}
	if c2.Parent != p {
		t.Errorf("Parent: expected %v got %v", p, c2.Parent)
	}
}

func TestNewProc(t *testing.T) {
	const (
		parentName = "parent"
		child1Name = "child 1"
		child2Name = "child 2"
	)

	p := NewProcScope(parentName, nil)
	c1 := NewProcScope(child1Name, p)
	c2 := NewProcScope(child2Name, p)

	for _, sc := range []*Scope{p, c1, c2} {
		if sc == nil {
			t.Fatal("should not return nil")
		}
		if !sc.IsProc {
			t.Errorf("%v: should be a proc", sc)
		}
		if sc.IsTask {
			t.Errorf("%v: should not be a task", sc)
		}
		if sc.Symbols == nil {
			t.Errorf("%v: symbols should not be nil", sc)
		}
	}

	if p.Name != parentName {
		t.Errorf("Name: expected %s got %s", parentName, p.Name)
	}
	if c1.Name != child1Name {
		t.Errorf("Name: expected %s got %s", child1Name, c1.Name)
	}
	if c2.Name != child2Name {
		t.Errorf("Name: expected %s got %s", child2Name, c2.Name)
	}

	if c1.Parent != p {
		t.Errorf("Parent: expected %v got %v", p, c1.Parent)
	}
	if c2.Parent != p {
		t.Errorf("Parent: expected %v got %v", p, c2.Parent)
	}
}

func TestNewTask(t *testing.T) {
	const (
		parentName = "parent"
		child1Name = "child 1"
		child2Name = "child 2"
	)

	p := NewTaskScope(parentName, nil)
	c1 := NewTaskScope(child1Name, p)
	c2 := NewTaskScope(child2Name, p)

	for _, sc := range []*Scope{p, c1, c2} {
		if sc == nil {
			t.Fatal("should not return nil")
		}
		if sc.IsProc {
			t.Errorf("%v: should not be a proc", sc)
		}
		if !sc.IsTask {
			t.Errorf("%v: should be a task", sc)
		}
		if sc.Symbols == nil {
			t.Errorf("%v: symbols should not be nil", sc)
		}
	}

	if p.Name != parentName {
		t.Errorf("Name: expected %s got %s", parentName, p.Name)
	}
	if c1.Name != child1Name {
		t.Errorf("Name: expected %s got %s", child1Name, c1.Name)
	}
	if c2.Name != child2Name {
		t.Errorf("Name: expected %s got %s", child2Name, c2.Name)
	}

	if c1.Parent != p {
		t.Errorf("Parent: expected %v got %v", p, c1.Parent)
	}
	if c2.Parent != p {
		t.Errorf("Parent: expected %v got %v", p, c2.Parent)
	}
}

func TestAddChild(t *testing.T) {
	const (
		parentName = "parent"
		child1Name = "child 1"
		child2Name = "child 2"
		child3Name = "child 3"
		childLen   = 3
	)

	p := NewScope(parentName, nil)
	c1 := NewScope(child1Name, p)
	c2 := NewTaskScope(child2Name, p)
	c3 := NewProcScope(child3Name, p)
	p.AddChild(c1)
	p.AddChild(c2)
	p.AddChild(c3)

	if len(p.Children) != childLen {
		t.Fatalf(" expected %d children got %d", childLen, len(p.Children))
	}

	c1, c2, c3 = p.Children[0], p.Children[1], p.Children[2]

	if p.Name != parentName {
		t.Errorf("Name: expected %s got %s", parentName, p.Name)
	}
	if c1.Name != child1Name {
		t.Errorf("Name: expected %s got %s", child1Name, c1.Name)
	}
	if c2.Name != child2Name {
		t.Errorf("Name: expected %s got %s", child2Name, c2.Name)
	}
	if c3.Name != child3Name {
		t.Errorf("Name: expected %s got %s", child3Name, c3.Name)
	}

	for _, sc := range []*Scope{c1, c2, c3} {
		if sc.Parent != p {
			t.Errorf("Parent: expected %v got %v", p, sc.Parent)
		}
	}

	if c1.IsProc || c1.IsTask {
		t.Errorf("First child should be plain: %v", c1)
	}
	if c2.IsProc || !c2.IsTask {
		t.Errorf("Second child should be a task: %v", c2)
	}
	if !c3.IsProc || c3.IsTask {
		t.Errorf("Thirs child should be a proc: %v", c3)
	}
}

func TestScopeLookup(t *testing.T) {
	root := NewScope("root", nil)
	proc := NewProcScope("proc", root)
	sub := NewProcScope("sub", proc)

	root.AddChild(proc)
	proc.AddChild(sub)

	root.Symbols["foo"] = Declare{}
	root.Symbols["bar"] = Declare{}
	proc.Symbols["foo"] = Declare{}
	sub.Symbols["foo"] = Declare{}

	if _, ok := sub.Lookup("foo"); !ok {
		t.Errorf("foo should be visible")
	}

	if _, ok := sub.Lookup("bar"); !ok {
		t.Errorf("bar should be visible")
	}

	if _, ok := sub.Lookup("quux"); ok {
		t.Errorf("quux is visible but should be undefined")
	}
}

func TestScopeLookupinTree(t *testing.T) {
	root := NewScope("root", nil)
	proc := NewProcScope("proc", root)
	sub := NewProcScope("sub", proc)

	root.AddChild(proc)
	proc.AddChild(sub)

	root.Symbols["faa"] = Declare{}
	root.Symbols["bar"] = Declare{}
	proc.Symbols["foo"] = Declare{}
	sub.Symbols["foo"] = Declare{}

	if _, sc, ok := sub.LookupInTree("foo"); !ok {
		t.Errorf("foo should be visible")
		if sc != root {
			t.Errorf("returned scope should be proc")
		}
	}

	if _, sc, ok := sub.LookupInTree("bar"); ok {
		t.Errorf("bar is visble but should not be visible")
		if sc != root {
			t.Errorf("returned scope should be root")
		}
	}

	if _, sc, ok := sub.LookupInTree("quux"); ok {
		t.Errorf("quux is visible but should be undefined")
		if sc != nil {
			t.Errorf("returned scope should be nil")
		}
	}

	if _, sc, ok := root.LookupInTree("foo"); !ok {
		t.Errorf("foo should be visible")
		if sc != root {
			t.Errorf("returned scope should be root")
		}
	}

	if _, sc, ok := root.LookupInTree("bar"); !ok {
		t.Errorf("bar should be visible")
		if sc != proc {
			t.Errorf("returned scope should be proc")
		}

	}

	if _, sc, ok := root.LookupInTree("quux"); ok {
		t.Errorf("quux is visible but should be undefined")
		if sc != nil {
			t.Errorf("returned scope should be nil")
		}
	}
}

func TestScopeNodes(t *testing.T) {
	root := NewScope("root", nil)
	proc1 := NewProcScope("proc 1", root)
	proc2 := NewProcScope("proc 2", root)
	sub11 := NewProcScope("sub 11", proc1)
	sub12 := NewProcScope("sub 12", proc1)
	sub21 := NewProcScope("sub 21", proc2)
	sub22 := NewProcScope("sub 21", proc2)
	root.AddChild(proc1)
	root.AddChild(proc2)
	proc1.AddChild(sub11)
	proc1.AddChild(sub12)
	proc2.AddChild(sub21)
	proc2.AddChild(sub22)

	expected := []*Scope{root, proc1, sub11, sub12, proc2, sub21, sub22}

	observed := root.Nodes()
	if !slices.Equal(expected, observed) {
		t.Fatalf("nodes expected %v, observed %v", expected, observed)
	}
}

func TestScopeDepth(t *testing.T) {
	root := NewScope("root", nil)
	proc1 := NewProcScope("proc 1", root)
	proc2 := NewProcScope("proc 2", root)
	sub11 := NewProcScope("sub 11", proc1)
	sub12 := NewProcScope("sub 12", proc1)
	sub21 := NewProcScope("sub 21", proc2)
	sub22 := NewProcScope("sub 21", proc2)
	root.AddChild(proc1)
	root.AddChild(proc2)
	proc1.AddChild(sub11)
	proc1.AddChild(sub12)
	proc2.AddChild(sub21)
	proc2.AddChild(sub22)

	tests := []*Scope{root, proc1, sub11, sub12, proc2, sub21, sub22}
	expected := []int{0, 1, 2, 2, 1, 2, 2}
	observed := make([]int, len(expected))
	for i, test := range tests {
		observed[i] = test.Depth()
	}

	if !slices.Equal(expected, observed) {
		t.Fatalf("depths expected %v, observed %v", expected, observed)
	}
}
