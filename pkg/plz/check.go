package plz

import "fmt"

// Scope represents a lexical scope in the PL/Z program.
// Each Scope holds a name, a pointer to its parent scope, and a mapping of
// identifiers to their declarations. Scopes are chained together to form the
// scope hierarchy used during semantic analysis.
type Scope struct {
	Name    string                 // Name of the scope (e.g. "global", procedure name)
	Parent  *Scope                 // Parent scope
	Symbols map[Identifier]Declare // Symbols in this scope
}

// Checker performs semantic analysis on a parsed PL/Z program.
// It maintains the current scope chain, a registry of all declared procedures
// and tasks, and provides helper methods for scope management and error reporting.
type Checker struct {
	current    *Scope                 // innermost scope during checking
	root       *Scope                 // global scope
	Procedures map[string]Procedure   // procedure name → definition
	Tasks      map[string]int         // task name → task index
	TaskDefs   []Task                 // task definitions in order
	Constants  map[Identifier]Literal // named constant values
	Datas      map[Identifier]Data    // named data values
}

// NewChecker returns a new Checker with an initialized global scope.
func NewChecker() *Checker {
	c := &Checker{
		Procedures: make(map[string]Procedure),
		Tasks:      make(map[string]int),
		Constants:  make(map[Identifier]Literal),
		Datas:      make(map[Identifier]Data),
	}
	c.root = &Scope{Name: "global", Symbols: make(map[Identifier]Declare)}
	c.current = c.root
	return c
}

// pushScope creates a new scope with the given name, sets its parent to the
// current innermost scope, and makes it the new current scope.
func (c *Checker) pushScope(name string) {
	c.current = &Scope{Name: name, Parent: c.current, Symbols: make(map[Identifier]Declare)}
}

// popScope restores the parent scope as the current innermost scope.
// It is a no-op if the current scope has no parent.
func (c *Checker) popScope() {
	if c.current.Parent != nil {
		c.current = c.current.Parent
	}
}

// lookup walks the scope chain from innermost to outermost when searching for a
// given identifier. It returns the declaration and true if found, or the zero
// value and false otherwise.
func (c *Checker) lookup(id Identifier) (Declare, bool) {
	for s := c.current; s != nil; s = s.Parent {
		if d, ok := s.Symbols[id]; ok {
			return d, true
		}
	}
	return Declare{}, false
}

// Lookup searches for an identifier starting from the root (global) scope and
// walking toward inner scopes. It is the public counterpart of lookup,
// intended for use by the code generator after semantic analysis is complete.
func (c *Checker) Lookup(id Identifier) (Declare, bool) {
	for s := c.root; s != nil; s = s.Parent {
		if d, ok := s.Symbols[id]; ok {
			return d, true
		}
	}
	return Declare{}, false
}

// Errorf formats a semantic error message with the given position and format
// string. The returned error is prefixed with "check:".
func (c *Checker) Errorf(pos string, form string, args ...any) error {
	return fmt.Errorf("check: %s: %s", pos, fmt.Sprintf(form, args...))
}

// EvalConstExpr evaluates an Expression to an integer at compile time.
// It resolves references to previously defined constants, evaluates all
// standard arithmetic, bitwise, comparison, shift, and logical operators,
// and rejects non-constant sub-expressions (variables, CALL, INPUT, etc.).
func (c *Checker) EvalConstExpr(e Expression) (int, error) {
	switch {
	case e.Operand() != nil:
		op := e.Operand()
		switch {
		case op.Literal() != nil:
			lit := op.Literal()
			if n := lit.Number(); n != nil {
				return n.Value, nil
			}
			if r := lit.Reference(); r != nil {
				id := r.Value.Identifier
				if lit2, ok := c.Constants[id]; ok {
					if n := lit2.Number(); n != nil {
						return n.Value, nil
					}
					return 0, fmt.Errorf("constant %q is not a number", id)
				}
				return 0, fmt.Errorf("undefined constant %q", id)
			}
			return 0, fmt.Errorf("text literal cannot be used in a numeric constant expression")
		case op.Expr() != nil:
			return c.EvalConstExpr(*op.Expr())
		case op.Reference() != nil:
			ref := op.Reference()
			if len(ref.Subscripts) > 0 || len(ref.Fields) > 0 {
				return 0, fmt.Errorf("cannot evaluate reference %q as constant expression", ref.Identifier)
			}
			if lit, ok := c.Constants[ref.Identifier]; ok {
				if n := lit.Number(); n != nil {
					return n.Value, nil
				}
				return 0, fmt.Errorf("constant %q is not a number", ref.Identifier)
			}
			return 0, fmt.Errorf("undefined identifier %q in constant expression", ref.Identifier)
		default:
			return 0, fmt.Errorf("CALL and INPUT cannot be used in constant expressions")
		}

	case e.Prefix() != nil:
		p := e.Prefix()
		v, err := c.EvalConstExpr(Expression{Expr: &p.Operand})
		if err != nil {
			return 0, err
		}
		switch p.Operator {
		case OperatorNEG:
			return -v, nil
		case OperatorNOT:
			if v == 0 {
				return 1, nil
			}
			return 0, nil
		default:
			return 0, fmt.Errorf("unknown prefix operator in constant expression")
		}

	case e.Infix() != nil:
		i := e.Infix()
		l, err := c.EvalConstExpr(Expression{Expr: &i.Operands[0]})
		if err != nil {
			return 0, err
		}
		r, err := c.EvalConstExpr(Expression{Expr: &i.Operands[1]})
		if err != nil {
			return 0, err
		}
		switch i.Operator {
		case OperatorADD:
			return l + r, nil
		case OperatorSUB:
			return l - r, nil
		case OperatorMUL:
			return l * r, nil
		case OperatorDIV:
			if r == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return l / r, nil
		case OperatorMOD:
			if r == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			return l % r, nil
		case OperatorAND:
			return l & r, nil
		case OperatorOR:
			return l | r, nil
		case OperatorXOR:
			return l ^ r, nil
		case OperatorShiftLeft:
			return l << uint(r), nil
		case OperatorShiftRight:
			return l >> uint(r), nil
		case OperatorEQU:
			if l == r {
				return 1, nil
			}
			return 0, nil
		case OperatorNEQ:
			if l != r {
				return 1, nil
			}
			return 0, nil
		case OperatorLT:
			if l < r {
				return 1, nil
			}
			return 0, nil
		case OperatorGT:
			if l > r {
				return 1, nil
			}
			return 0, nil
		case OperatorLTE:
			if l <= r {
				return 1, nil
			}
			return 0, nil
		case OperatorGTE:
			if l >= r {
				return 1, nil
			}
			return 0, nil
		default:
			return 0, fmt.Errorf("unknown operator in constant expression")
		}

	case e.Suffix() != nil:
		return 0, fmt.Errorf("suffix operators (index, call, field) cannot be used in constant expressions")
	}
	return 0, fmt.Errorf("empty expression")
}

// Check runs semantic analysis on the program using a two-pass approach.
//
// Pass 1 collects procedure and task signatures into the Checker's
// procedure and task registries. It also registers procedure names in the
// global scope so that CALL expressions can resolve them. Duplicate procedure
// or task names are reported as errors.
//
// Pass 2 walks each top-level statement with proper scope push/pop,
// validating that all referenced identifiers have been declared and that field
// accesses, subscripts, and other constructs are semantically correct.
func (p Program) Check(c *Checker) error {
	// First pass: collect procedure and task signatures.
	for _, stmt := range p.Statements {
		switch cmd := stmt.Command.(type) {
		case Task:
			name := cmd.Name.Name
			if _, ok := c.Tasks[name]; ok {
				return c.Errorf("", "duplicate task %q", name)
			}
			idx := len(c.TaskDefs)
			c.Tasks[name] = idx
			c.TaskDefs = append(c.TaskDefs, cmd)

		case Procedure:
			name := cmd.Name.Name
			if _, ok := c.Procedures[name]; ok {
				return c.Errorf("", "duplicate procedure %q", name)
			}
			c.Procedures[name] = cmd
			// Register procedure name in global scope so CALL expressions resolve it.
			if _, ok := c.current.Symbols[Identifier(name)]; !ok {
				c.current.Symbols[Identifier(name)] = Declare{
					Identifier: Identifier(name),
					Type:       Type{Typ: &PredeclaredType{Kind: PredeclaredWord}},
				}
			}
		}
	}

	// Second pass: walk statements with scope push/pop.
	for _, stmt := range p.Statements {
		if err := stmt.Check(c); err != nil {
			return err
		}
	}
	return nil
}

// Check delegates semantic analysis to the underlying command's Checklet
// interface, or does nothing if the command does not implement it.
func (s Statement) Check(c *Checker) error {
	checklet, ok := s.Command.(Checklet)
	if ok {
		return checklet.Check(c)
	}
	return nil // no check implemented, not needed
}

// Check validates a type alias definition. Define aliases are always valid,
// so this is a no-op.
func (s Define) Check(c *Checker) error {
	return nil
}

// Check validates an AT directive. This is a no-op because AT only
// sets the assembly address and has no semantic constraints.
func (s At) Check(c *Checker) error {
	return nil
}

// Check validates an INTERRUPT or NMI install statement. It verifies
// that the target identifier names a declared procedure.
func (s InterruptStmt) Check(c *Checker) error {
	if _, ok := c.Procedures[string(s.Target)]; !ok {
		return fmt.Errorf("INTERRUPT/NMI: undefined procedure %q", s.Target)
	}
	return nil
}

// Check registers a variable declaration in the current scope.
// It returns an error if the identifier has already been declared in the
// same scope, or if both an AT address and an initializer are present.
func (s Declare) Check(c *Checker) error {
	if _, ok := c.current.Symbols[s.Identifier]; ok {
		return c.Errorf("", "duplicate declaration of %s", s.Identifier)
	}
	if s.At != nil && s.Initializer != nil {
		return c.Errorf("", "%s: AT and initializer are mutually exclusive", s.Identifier)
	}
	c.current.Symbols[s.Identifier] = s
	return nil
}

// Check evaluates the constant expression and registers the result in the
// checker's Constants map. Numeric values are stored as NumberLit for use
// by later constant and code generation passes.
func (s Constant) Check(c *Checker) error {
	if s.Expr.Expr == nil {
		return nil
	}
	if v, err := c.EvalConstExpr(s.Expr); err == nil {
		c.Constants[Identifier(s.Name)] = Literal{Lit: &NumberLit{Value: v}}
	} else if op := s.Expr.Operand(); op != nil {
		if lit := op.Literal(); lit != nil {
			if t := lit.Text(); t != nil {
				c.Constants[Identifier(s.Name)] = *lit
				return nil
			}
		}
		return err
	} else {
		return err
	}
	return nil
}

// Check registers a named data value in the checker's Data map so
// that references to the data name can be resolved during code generation.
func (d Data) Check(c *Checker) error {
	c.Datas[Identifier(d.Name)] = d
	return nil
}

// Check validates an if-then-else statement. It checks the condition
// expression, the "then" branch, and, if present, the "else" branch.
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

// Check validates a compound statement group (DO/WHILE/FOR/CASE block).
// It checks loop or case expressions if present, pushes a new scope for the
// block body (except CASE, which uses the enclosing scope), and validates
// each statement within it.
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
		for _, branch := range s.Case.Branches {
			for _, cv := range branch.Values {
				if cv.Name != "" {
					lit, ok := c.Constants[Identifier(cv.Name)]
					if !ok {
						return fmt.Errorf("CASE: undefined constant %q", cv.Name)
					}
					if lit.Number() == nil {
						return fmt.Errorf("CASE: constant %q is not a number", cv.Name)
					}
				}
			}
			if err := branch.Statement.Check(c); err != nil {
				return err
			}
		}
		if s.Case.Default != nil {
			return s.Case.Default.Check(c)
		}
		return nil
	}
	c.pushScope("do")
	defer c.popScope()
	for _, stmt := range s.Statements {
		if err := stmt.Check(c); err != nil {
			return err
		}
	}
	return nil
}

// Check validates a FOR loop header. It checks the loop variable reference,
// the start expression, the end (TO) expression, and, if present, the step
// (BY) expression.
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

// Check validates an OUTPUT statement by checking its value expression.
func (s Output) Check(c *Checker) error {
	return s.Value.Check(c)
}

// Check validates a LET assignment by checking the right-hand side expression
// and validating any subscripts or field accesses on the left-hand side.
// The left-hand side variable is implicitly declared if it does not exist.
// Constants cannot be assigned to.
func (s Let) Check(c *Checker) error {
	// Reject assignment to constants.
	if _, ok := c.Constants[s.Reference.Identifier]; ok {
		return c.Errorf("", "cannot assign to constant %q", s.Reference.Identifier)
	}
	// Validate subscripts and fields if the base variable is already declared.
	if r := s.Reference; r.Identifier != "" {
		if d, ok := c.lookup(r.Identifier); ok {
			for _, sub := range r.Subscripts {
				if err := sub.Check(c); err != nil {
					return err
				}
			}
			for _, fname := range r.Fields {
				rec := d.Type.Record()
				if rec == nil {
					return c.Errorf("", "%q is not a record, cannot access field %q", r.Identifier, fname)
				}
				found := false
				for _, f := range rec.Fields {
					if f.Identifier == fname {
						found = true
						break
					}
				}
				if !found {
					return c.Errorf("", "struct %q has no field %q", r.Identifier, fname)
				}
			}
		}
	}
	return s.Expression.Check(c)
}

// Check validates a procedure definition. It pushes a new scope named after
// the procedure, registers the parameters in that scope (marking record and
// DATA parameters as pass-by-reference), and then validates each statement in
// the procedure body.
func (s Procedure) Check(c *Checker) error {
	c.pushScope(s.Name.Name)
	defer c.popScope()
	// Register parameters in the procedure scope.
	for i, param := range s.Parameters {
		ptype := Type{Typ: &PredeclaredType{Kind: PredeclaredWord}}
		if i < len(s.ParamTypes) {
			ptype = s.ParamTypes[i]
		}
		isRef := ptype.Record() != nil || ptype.Predeclared() == PredeclaredData
		c.current.Symbols[param] = Declare{
			Identifier: param,
			Type:       ptype,
			ParamRef:   isRef,
		}
	}
	// Process body — Declare.Check registers locals in the procedure scope.
	for i := range s.Statements {
		if err := s.Statements[i].Check(c); err != nil {
			return err
		}
	}
	return nil
}

// Check validates a task definition. It pushes a new scope for the task body
// and validates each statement within it.
func (t Task) Check(c *Checker) error {
	c.pushScope(t.Name.Name)
	defer c.popScope()
	for i := range t.Body {
		if err := t.Body[i].Check(c); err != nil {
			return err
		}
	}
	return nil
}

// Check validates a SUSPEND statement by verifying that the named task has
// been declared.
func (s Suspend) Check(c *Checker) error {
	if _, ok := c.Tasks[string(s.Name)]; !ok {
		return c.Errorf("", "undeclared task %q", s.Name)
	}
	return nil
}

// Check validates a RESUME statement by verifying that the named task has
// been declared.
func (r Resume) Check(c *Checker) error {
	if _, ok := c.Tasks[string(r.Name)]; !ok {
		return c.Errorf("", "undeclared task %q", r.Name)
	}
	return nil
}

// Check validates a SLEEP statement by checking its duration expression.
func (s Sleep) Check(c *Checker) error {
	return s.Duration.Check(c)
}

// Check validates a RETURN statement by checking each return expression.
func (s Return) Check(c *Checker) error {
	for i := range s.Expressions {
		if err := s.Expressions[i].Check(c); err != nil {
			return err
		}
	}
	return nil
}

// Check validates a CALL statement by checking each argument expression.
func (s Call) Check(c *Checker) error {
	for i := range s.Arguments {
		if err := s.Arguments[i].Check(c); err != nil {
			return err
		}
	}
	return nil
}

// Check validates an INPUT expression by checking its port expression.
func (i Input) Check(c *Checker) error {
	return i.Port.Check(c)
}

// Check validates an expression by dispatching to the appropriate sub-check
// based on whether it is an operand, a prefix operation, an infix operation,
// or a suffix operation.
func (e Expression) Check(c *Checker) error {
	switch {
	case e.Operand() != nil:
		return e.Operand().Check(c)
	case e.Prefix() != nil:
		return e.Prefix().Check(c)
	case e.Infix() != nil:
		return e.Infix().Check(c)
	case e.Suffix() != nil:
		return e.Suffix().Check(c)
	}
	return nil
}

// Check validates an operand by dispatching to the appropriate sub-check
// based on whether it is a reference, a parenthesized expression, a function
// call, or a literal. Literals are always valid so no check is performed.
func (o Operand) Check(c *Checker) error {
	switch {
	case o.Reference() != nil:
		return o.Reference().Check(c)
	case o.Expr() != nil:
		return o.Expr().Check(c)
	case o.Call() != nil:
		return o.Call().Check(c)
	case o.Input() != nil:
		return o.Input().Check(c)
	case o.Literal() != nil:
	}
	return nil
}

// Check validates a prefix expression by checking its single operand.
func (p Prefix) Check(c *Checker) error {
	return p.Operand.Check(c)
}

// Check validates an infix expression by checking each operand in sequence.
func (i Infix) Check(c *Checker) error {
	for _, op := range i.Operands {
		if err := op.Check(c); err != nil {
			return err
		}
	}
	return nil
}

// Check validates a suffix (postfix) expression by checking each operand.
// For the FIELD operator, the second operand (the field name) is skipped
// since it is not a variable reference but a struct field identifier.
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

// Check validates a variable reference. It looks up the identifier in the
// current scope chain and returns an error if the variable is undeclared. It
// also validates any array subscripts and verifies that field accesses are
// valid for record types.
func (r *Reference) Check(c *Checker) error {
	if r.Identifier == "" {
		return nil
	}
	// Constants are resolved during code generation; no declaration needed.
	if _, ok := c.Constants[r.Identifier]; ok {
		if len(r.Fields) > 0 || len(r.Subscripts) > 0 {
			return c.Errorf("", "constant %q cannot be subscripted or used with field access", r.Identifier)
		}
		return nil
	}
	// Data is resolved during code generation; no declaration needed.
	if _, ok := c.Datas[r.Identifier]; ok {
		if len(r.Fields) > 0 {
			return c.Errorf("", "data %q cannot be used with field access", r.Identifier)
		}
		return nil
	}

	d, ok := c.lookup(r.Identifier)
	if !ok {
		return c.Errorf("", "undeclared variable %q", r.Identifier)
	}
	for _, sub := range r.Subscripts {
		if err := sub.Check(c); err != nil {
			return err
		}
	}
	for _, fname := range r.Fields {
		rec := d.Type.Record()
		if rec == nil {
			return c.Errorf("", "%q is not a record, cannot access field %q", r.Identifier, fname)
		}
		found := false
		for _, f := range rec.Fields {
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
