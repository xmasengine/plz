package plz

// Scope represents a lexical scope in the PL/Z program.
// Each Scope holds a name, a pointer to its parent scope, a mapping of
// identifiers to their declarations, and a list of child scopes. Scopes
// form a persistent tree rooted at Checker.root that survives after the
// checker phase completes — the code generator reads type information
// directly from this tree instead of duplicating it.
type Scope struct {
	Name     string                 // Name of the scope (e.g. "global", procedure name)
	Parent   *Scope                 // Parent scope
	IsProc   bool                   // IsProc is true when this scope is a procedure body
	IsTask   bool                   // IsTask is true when this scope is a task body
	Symbols  map[Identifier]Declare // Symbols in this scope
	Children []*Scope               // Child scopes forming the persistent scope tree
	ProcData *ProcData              // Procedure metadata (set for procedure scopes)
}

// ProcData holds procedure metadata extracted from the AST for use by the
// code generator. It is stored on the procedure's Scope so the generator
// can find parameter types and the reentrant flag via scope tree navigation
// instead of a flat map on Checker.
type ProcData struct {
	Params     []Identifier
	ParamTypes []Type
	Reentrant  bool
}

// NewScope creates a new Scope with the given name and parent.
func NewScope(name string, parent *Scope) *Scope {
	return &Scope{
		Name:    name,
		Parent:  parent,
		Symbols: make(map[Identifier]Declare),
	}
}

// NewProcScope creates a new procedure scope with the given name and parent.
func NewProcScope(name string, parent *Scope) *Scope {
	s := NewScope(name, parent)
	s.IsProc = true
	return s
}

// NewProcWithData creates a new procedure scope with parameter metadata
// (parameter names, types, and reentrant flag) stored for the generator.
func NewProcWithData(name string, parent *Scope, params []Identifier, paramTypes []Type, reentrant bool) *Scope {
	s := NewProcScope(name, parent)
	s.ProcData = &ProcData{
		Params:     params,
		ParamTypes: paramTypes,
		Reentrant:  reentrant,
	}
	return s
}

// NewTaskScope creates a new task body scope with the given name and parent.
func NewTaskScope(name string, parent *Scope) *Scope {
	s := NewScope(name, parent)
	s.IsTask = true
	return s
}

// AddChild registers a scope as a child of s and sets the child's parent.
func (s *Scope) AddChild(child *Scope) {
	child.Parent = s
	s.Children = append(s.Children, child)
}

// Lookup searches for an identifier by walking the scope chain from
// the innermost (s) outward to the root, returning the first declaration
// found. This respects PL/Z block scoping — inner declarations shadow
// outer ones.
func (s *Scope) Lookup(id Identifier) (Declare, bool) {
	for cur := s; cur != nil; cur = cur.Parent {
		if d, ok := cur.Symbols[id]; ok {
			return d, true
		}
	}
	return Declare{}, false
}

// LookupInTree searches the entire subtree rooted at s (including s and
// all descendants) for an identifier. It returns the match from the
// deepest (innermost) scope first (post-order DFS) — useful for finding
// declarations within a procedure's scope hierarchy where inner scopes
// shadow outer ones.
func (s *Scope) LookupInTree(id Identifier) (Declare, *Scope, bool) {
	var best Declare
	var bestScope *Scope
	for _, child := range s.Children {
		if d, scope, ok := child.LookupInTree(id); ok {
			best, bestScope = d, scope
		}
	}
	if bestScope != nil {
		return best, bestScope, true
	}
	if d, ok := s.Symbols[id]; ok {
		return d, s, true
	}
	return Declare{}, nil, false
}

// Nodes returns this scope and all descendant scopes in DFS pre-order.
func (s *Scope) Nodes() []*Scope {
	var nodes []*Scope
	s.collectNodes(&nodes)
	return nodes
}

func (s *Scope) collectNodes(nodes *[]*Scope) {
	*nodes = append(*nodes, s)
	for _, child := range s.Children {
		child.collectNodes(nodes)
	}
}

// Depth returns the nesting depth of this scope (0 = root scope).
func (s *Scope) Depth() int {
	d := 0
	for cur := s.Parent; cur != nil; cur = cur.Parent {
		d++
	}
	return d
}
