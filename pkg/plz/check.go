package plz

import "fmt"

type Checker struct {
	Symbols    map[Identifier]*Declare
	Procedures map[string]*Procedure // procedure name → definition
}

func NewChecker() *Checker {
	return &Checker{
		Symbols:    make(map[Identifier]*Declare),
		Procedures: make(map[string]*Procedure),
	}
}

func (c *Checker) Errorf(pos string, form string, args ...any) error {
	return fmt.Errorf("check: %s: %s", pos, fmt.Sprintf(form, args...))
}

// Check runs semantic analysis on the program.
func (p Program) Check(c *Checker) error {
	// First pass: collect declarations and procedure parameters.
	for _, stmt := range p.Statements {
		if stmt.Declare != nil {
			if existing, ok := c.Symbols[stmt.Declare.Identifier]; ok {
				return c.Errorf("", "duplicate declaration of %s (first at %s)",
					stmt.Declare.Identifier, existing.Identifier)
			}
			c.Symbols[stmt.Declare.Identifier] = stmt.Declare
		}
		if stmt.Procedure != nil {
			// Store procedure definition for call-site lookup.
			c.Procedures[stmt.Procedure.Name.Name] = stmt.Procedure
			// Register procedure name so it can be used in call expressions.
			if _, ok := c.Symbols[Identifier(stmt.Procedure.Name.Name)]; !ok {
				c.Symbols[Identifier(stmt.Procedure.Name.Name)] = &Declare{
					Identifier: Identifier(stmt.Procedure.Name.Name),
					Type:       Type{Predeclared: PredeclaredWord},
				}
			}
			for i, param := range stmt.Procedure.Parameters {
				if _, ok := c.Symbols[param]; !ok {
					ptype := Type{Predeclared: PredeclaredWord}
					if i < len(stmt.Procedure.ParamTypes) {
						ptype = stmt.Procedure.ParamTypes[i]
					}
					isRef := ptype.Record != nil // records passed by reference
					c.Symbols[param] = &Declare{
						Identifier: param,
						Type:       ptype,
						ParamRef:   isRef,
					}
				}
			}
			for _, local := range stmt.Procedure.Locals {
				if existing, ok := c.Symbols[local.Identifier]; ok {
					return c.Errorf("", "duplicate declaration of %s (first at %s)",
						local.Identifier, existing.Identifier)
				}
				c.Symbols[local.Identifier] = &local
			}
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
	case s.Data != nil:
	case s.Return != nil:
		return s.Return.Check(c)
	case s.Halt != nil:
	case s.Enable != nil:
	case s.Disable != nil:
	}
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
	for i := range s.Statements {
		if err := s.Statements[i].Check(c); err != nil {
			return err
		}
	}
	return nil
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
	d, ok := c.Symbols[r.Identifier]
	if !ok {
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
