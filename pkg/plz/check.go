package plz

import "fmt"

type Scope struct {
	Name     string                  // Name of the scope or procedure we are in
	Parent   *Scope                  // Parent scope
	Symbols  map[Identifier]*Declare // Symbols in the scope
	Children []*Scope                // Child scopes
}

type Checker struct {
	Symbols     map[Identifier]*Declare            // global symbols
	procSymbols map[string]map[Identifier]*Declare // procedure → local symbols
	Procedures  map[string]*Procedure              // procedure name → definition
	Tasks       map[string]int                     // task name → task index
	TaskDefs    []*Task                            // task definitions in order
	currentProc string                             // procedure currently being checked
}

func NewChecker() *Checker {
	c := &Checker{
		Symbols:     make(map[Identifier]*Declare),
		procSymbols: make(map[string]map[Identifier]*Declare),
		Procedures:  make(map[string]*Procedure),
		Tasks:       make(map[string]int),
	}
	c.Symbols["INPUT"] = &Declare{
		Identifier: "INPUT",
		Type:       Type{Predeclared: PredeclaredWord},
	}
	return c
}

// lookup resolves an identifier within the current procedure scope.
func (c *Checker) lookup(id Identifier) *Declare {
	if c.currentProc != "" {
		if m, ok := c.procSymbols[c.currentProc]; ok {
			if d, ok := m[id]; ok {
				return d
			}
		}
	}
	return c.Symbols[id]
}

func (c *Checker) Errorf(pos string, form string, args ...any) error {
	return fmt.Errorf("check: %s: %s", pos, fmt.Sprintf(form, args...))
}

// ProcSymbols returns the per-procedure symbol map for the given procedure.
func (c *Checker) ProcSymbols(procName string) (map[Identifier]*Declare, bool) {
	m, ok := c.procSymbols[procName]
	return m, ok
}

// Check runs semantic analysis on the program.
func (p Program) Check(c *Checker) error {
	// First pass: collect declarations, labels, and procedure parameters.
	for _, stmt := range p.Statements {
		if stmt.Label != nil && stmt.Label.Name != "" {
			if _, ok := c.Symbols[Identifier(stmt.Label.Name)]; !ok {
				kind := PredeclaredWord
				if stmt.Data != nil {
					kind = PredeclaredByte
				}
				c.Symbols[Identifier(stmt.Label.Name)] = &Declare{
					Identifier: Identifier(stmt.Label.Name),
					Type:       Type{Predeclared: kind},
				}
			}
		}
		if stmt.Declare != nil {
			if existing, ok := c.Symbols[stmt.Declare.Identifier]; ok && existing.ProcName == "" {
				return c.Errorf("", "duplicate declaration of %s (first at %s)",
					stmt.Declare.Identifier, existing.Identifier)
			}
			c.Symbols[stmt.Declare.Identifier] = stmt.Declare
		}
		if stmt.Task != nil {
			name := stmt.Task.Name.Name
			if _, ok := c.Tasks[name]; ok {
				return c.Errorf("", "duplicate task %q", name)
			}
			idx := len(c.TaskDefs)
			c.Tasks[name] = idx
			c.TaskDefs = append(c.TaskDefs, stmt.Task)
		}
		if stmt.Procedure != nil {
			proc := stmt.Procedure
			// Store procedure definition for call-site lookup.
			c.Procedures[proc.Name.Name] = proc
			// Register procedure name so it can be used in call expressions.
			if _, ok := c.Symbols[Identifier(proc.Name.Name)]; !ok {
				c.Symbols[Identifier(proc.Name.Name)] = &Declare{
					Identifier: Identifier(proc.Name.Name),
					Type:       Type{Predeclared: PredeclaredWord},
				}
			}
			// Create per-procedure symbol map.
			pm := make(map[Identifier]*Declare)
			dup := func(name Identifier) error {
				if _, ok := pm[name]; ok {
					return c.Errorf("", "duplicate parameter %q in procedure %s",
						name, proc.Name.Name)
				}
				return nil
			}
			for i, param := range proc.Parameters {
				if err := dup(param); err != nil {
					return err
				}
				ptype := Type{Predeclared: PredeclaredWord}
				if i < len(proc.ParamTypes) {
					ptype = proc.ParamTypes[i]
				}
				isRef := ptype.Record != nil || ptype.Predeclared == PredeclaredData
				d := &Declare{
					Identifier: param,
					Type:       ptype,
					ParamRef:   isRef,
					ProcName:   proc.Name.Name,
				}
				pm[param] = d
				c.Symbols[param] = d
			}
			for _, local := range proc.Locals {
				if _, ok := pm[local.Identifier]; ok {
					return c.Errorf("", "duplicate declaration of %s in procedure %s",
						local.Identifier, proc.Name.Name)
				}
				local.ProcName = proc.Name.Name
				pm[local.Identifier] = &local
				c.Symbols[local.Identifier] = &local
			}
			c.procSymbols[proc.Name.Name] = pm
		}
	}
	// Second pass: check all statements.
	for _, stmt := range p.Statements {
		if err := stmt.Check(c); err != nil {
			return err
		}
	}
	return nil
}

func (s Statement) Check(c *Checker) error {
	switch {
	case s.If != nil:
		return s.If.Check(c)
	case s.Let != nil:
		return s.Let.Check(c)
	case s.Group != nil:
		return s.Group.Check(c)
	case s.Procedure != nil:
		return s.Procedure.Check(c)
	case s.Output != nil:
		return s.Output.Check(c)
	case s.Call != nil:
		return s.Call.Check(c)
	case s.GoTo != nil:
	case s.Constant != nil:
	case s.Declare != nil:
	case s.Define != nil:
		return s.Define.Check(c)
	case s.Data != nil:
	case s.Return != nil:
		return s.Return.Check(c)
	case s.Halt != nil:
	case s.Enable != nil:
	case s.Disable != nil:
	case s.Task != nil:
		return s.Task.Check(c)
	case s.Suspend != nil:
		return s.Suspend.Check(c)
	case s.Resume != nil:
		return s.Resume.Check(c)
	case s.Sleep != nil:
		return s.Sleep.Check(c)
	case s.Yield != nil:
	case s.Label != nil:
	}
	return nil
}

func (s Define) Check(c *Checker) error {
	return nil
}

func (s If) Check(c *Checker) error {
	if err := s.Condition.Check(c); err != nil {
		return err
	}
	if err := s.Then.Check(c); err != nil {
		return err
	}
	if s.Else != nil {
		return s.Else.Check(c)
	}
	return nil
}

func (s Group) Check(c *Checker) error {
	if s.While != nil {
		if err := s.While.Expression.Check(c); err != nil {
			return err
		}
	}
	if s.For != nil {
		if err := s.For.Check(c); err != nil {
			return err
		}
	}
	if s.Case != nil {
		if err := s.Case.Expression.Check(c); err != nil {
			return err
		}
	}
	for _, stmt := range s.Statements {
		if err := stmt.Check(c); err != nil {
			return err
		}
	}
	return nil
}

func (s For) Check(c *Checker) error {
	if err := s.Reference.Check(c); err != nil {
		return err
	}
	if err := s.Start.Check(c); err != nil {
		return err
	}
	if err := s.To.Check(c); err != nil {
		return err
	}
	if s.By != nil {
		return s.By.Check(c)
	}
	return nil
}

func (s Output) Check(c *Checker) error {
	return s.Value.Check(c)
}

func (s Let) Check(c *Checker) error {
	return s.Expression.Check(c)
}

func (s Procedure) Check(c *Checker) error {
	save := c.currentProc
	c.currentProc = s.Name.Name
	for i := range s.Statements {
		if err := s.Statements[i].Check(c); err != nil {
			c.currentProc = save
			return err
		}
	}
	c.currentProc = save
	return nil
}

func (t Task) Check(c *Checker) error {
	for i := range t.Body {
		if err := t.Body[i].Check(c); err != nil {
			return err
		}
	}
	return nil
}

func (s Suspend) Check(c *Checker) error {
	if _, ok := c.Tasks[string(s.Name)]; !ok {
		return c.Errorf("", "undeclared task %q", s.Name)
	}
	return nil
}

func (r Resume) Check(c *Checker) error {
	if _, ok := c.Tasks[string(r.Name)]; !ok {
		return c.Errorf("", "undeclared task %q", r.Name)
	}
	return nil
}

func (s Sleep) Check(c *Checker) error {
	return s.Duration.Check(c)
}

func (s Return) Check(c *Checker) error {
	for i := range s.Expressions {
		if err := s.Expressions[i].Check(c); err != nil {
			return err
		}
	}
	return nil
}

func (s Call) Check(c *Checker) error {
	for i := range s.Arguments {
		if err := s.Arguments[i].Check(c); err != nil {
			return err
		}
	}
	return nil
}

func (e Expression) Check(c *Checker) error {
	switch {
	case e.Operand != nil:
		return e.Operand.Check(c)
	case e.Prefix != nil:
		return e.Prefix.Check(c)
	case e.Infix != nil:
		return e.Infix.Check(c)
	case e.Suffix != nil:
		return e.Suffix.Check(c)
	}
	return nil
}

func (o Operand) Check(c *Checker) error {
	switch {
	case o.Reference != nil:
		return o.Reference.Check(c)
	case o.Expression != nil:
		return o.Expression.Check(c)
	case o.Call != nil:
		return o.Call.Check(c)
	case o.Literal != nil:
	}
	return nil
}

func (p Prefix) Check(c *Checker) error {
	return p.Operand.Check(c)
}

func (i Infix) Check(c *Checker) error {
	for _, op := range i.Operands {
		if err := op.Check(c); err != nil {
			return err
		}
	}
	return nil
}

func (s Suffix) Check(c *Checker) error {
	for i, op := range s.Operands {
		// The second operand of OperatorFIELD is a field name, not a variable.
		if s.Operator == OperatorFIELD && i == 1 {
			continue
		}
		if err := op.Check(c); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reference) Check(c *Checker) error {
	if r.Identifier == "" {
		return nil
	}
	d := c.lookup(r.Identifier)
	if d == nil {
		return c.Errorf("", "undeclared variable %q", r.Identifier)
	}
	for _, sub := range r.Subscripts {
		if err := sub.Check(c); err != nil {
			return err
		}
	}
	for _, fname := range r.Fields {
		if d.Type.Record == nil {
			return c.Errorf("", "%q is not a record, cannot access field %q", r.Identifier, fname)
		}
		found := false
		for _, f := range d.Type.Record.Fields {
			if f.Identifier == fname {
				found = true
				break
			}
		}
		if !found {
			return c.Errorf("", "struct %q has no field %q", r.Identifier, fname)
		}
	}
	return nil
}
