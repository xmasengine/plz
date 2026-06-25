package plz

import (
	"fmt"

	"github.com/xmasengine/plz/pkg/pir"
)

// GenPIR is the PL/Z-to-PIR code generator. It walks a checked AST and
// emits PIR instructions. The design mirrors Gen (the Z80 generator)
// so that both backends share the same semantic analysis and the outputs
// can be compared.
//
// Expression evaluation follows the PIR data-stack convention:
// every expression leaves one value (or two for multi-value RETURN)
// on the data stack. Binary infix ops pop two values (TOS = right
// operand) and push one result.
type GenPIR struct {
	checker *Checker
	prog    *pir.Program

	// scope tracking (parallels Gen)
	currentScope  *Scope
	scopeStack    []*Scope
	scopeChildIdx map[*Scope]int

	// procedure / task / label state
	procName        string
	procReturnType  Type
	inFrame         bool
	inTask          bool

	// unique numeric label suffix
	label int

	// constant folding cache
	constValues map[Identifier]*Literal

	// loop label stack for BREAK/CONTINUE
	loopStack []struct{ start, end string }
}

// NewGenPIR creates a GenPIR attached to a checker.
func NewGenPIR(c *Checker) *GenPIR {
	return &GenPIR{
		checker:       c,
		prog:          &pir.Program{},
		scopeChildIdx: make(map[*Scope]int),
		constValues:   make(map[Identifier]*Literal),
	}
}

// nextLabel returns a unique numeric suffix for local tags.
func (g *GenPIR) nextLabel() int {
	g.label++
	return g.label
}

// pushScope pushes a new scope and advances the checker scope pointer.
func (g *GenPIR) pushScope() {
	if g.currentScope != nil {
		idx := g.scopeChildIdx[g.currentScope]
		if idx < len(g.currentScope.Children) {
			g.scopeChildIdx[g.currentScope] = idx + 1
			g.scopeStack = append(g.scopeStack, g.currentScope)
			g.currentScope = g.currentScope.Children[idx]
		}
	}
}

// popScope restores the parent scope.
func (g *GenPIR) popScope() {
	if len(g.scopeStack) > 0 {
		g.currentScope = g.scopeStack[len(g.scopeStack)-1]
		g.scopeStack = g.scopeStack[:len(g.scopeStack)-1]
	}
}

// localSym resolves an identifier to its assembly-level symbol,
// falling back to the plain identifier.
func (g *GenPIR) localSym(id Identifier) Identifier {
	if g.procName == "" {
		return id
	}
	// In PIR mode we just use the plain identifier — the PIR
	// backend handles uniqueness via the TAG/ROUTE/JOB names.
	return id
}

// localType returns the type of an identifier by walking the scope tree.
func (g *GenPIR) localType(id Identifier) (Type, bool) {
	for s := g.currentScope; s != nil; s = s.Parent {
		if d, ok := s.Symbols[id]; ok {
			return d.Type, ok
		}
	}
	return Type{}, false
}

// isParamRef returns true when id is a record or data parameter passed by reference.
func (g *GenPIR) isParamRef(id Identifier) bool {
	for s := g.currentScope; s != nil; s = s.Parent {
		if d, ok := s.Symbols[id]; ok {
			return d.ParamRef
		}
	}
	return false
}

// isByteRef reports whether the reference's resolved type is BYTE.
func (g *GenPIR) isByteRef(r *Reference) bool {
	if g.checker == nil || r == nil {
		return false
	}
	if len(r.Fields) > 0 {
		t, ok := g.localType(r.Identifier)
		if !ok || t.Record() == nil {
			return false
		}
		fname := r.Fields[0]
		for _, f := range t.Record().Fields {
			if f.Identifier == fname {
				ft := f.Type
				// If the field type is an array, check the element type
				if a := ft.Array(); a != nil {
					// If there are subscripts, we're indexing into the array
					if len(r.Subscripts) > 0 {
						return a.ElemType.Predeclared() == PredeclaredByte
					}
					// Otherwise the field itself is an array type (not byte)
					return false
				}
				// If the field has subscripts, check the subscripted type
				if len(r.Subscripts) > 0 {
					// Subscripts on the field would be handled by the Array case above
					return false
				}
				return ft.Predeclared() == PredeclaredByte
			}
		}
		return false
	}
	if len(r.Subscripts) > 0 {
		t, ok := g.localType(r.Identifier)
		if !ok {
			return false
		}
		if arr := t.Array(); arr != nil {
			return arr.ElemType.Predeclared() == PredeclaredByte
		}
		// DATA references are always byte-sized
		if g.checker != nil && g.currentScope != nil {
			if d, ok2 := g.currentScope.Lookup(r.Identifier); ok2 && d.DataValue != nil {
				return true
			}
			// DATA parameters (ParamRef, PredeclaredData) are also byte-sized
			if d, ok2 := g.currentScope.Lookup(r.Identifier); ok2 && d.ParamRef && d.Type.Predeclared() == PredeclaredData {
				return true
			}
		}
		return false
	}
	t, ok := g.localType(r.Identifier)
	if !ok {
		return false
	}
	return t.Predeclared() == PredeclaredByte
}

func (g *GenPIR) isByteOperand(op *Operand) bool {
	switch {
	case op.Literal() != nil:
		return false
	case op.Reference() != nil:
		return g.isByteRef(op.Reference())
	case op.Expr() != nil:
		return g.isByteExpression(op.Expr())
	case op.Input() != nil:
		return true
	case op.Length() != nil:
		return false
	}
	return false
}

func (g *GenPIR) isByteExpression(e *Expression) bool {
	switch {
	case e.Operand() != nil:
		return g.isByteOperand(e.Operand())
	case e.Prefix() != nil:
		p := e.Prefix()
		if p.Operator == Operator(KeywordByte) {
			return true
		}
		if p.Operator == Operator(KeywordWord) {
			return false
		}
		if p.Operator == OperatorNOT {
			return true
		}
		return g.isByteOperand(&p.Operand)
	case e.Infix() != nil:
		inf := e.Infix()
		switch inf.Operator {
		case OperatorShiftLeft, OperatorShiftRight:
			return g.isByteOperand(&inf.Operands[0])
		}
		return g.isByteOperand(&inf.Operands[0]) && g.isByteOperand(&inf.Operands[1])
	case e.Suffix() != nil:
		return false
	}
	return false
}

func (g *GenPIR) isByteInfix(i *Infix) bool {
	return g.isByteOperand(&i.Operands[0]) && g.isByteOperand(&i.Operands[1])
}

// ── Emit helpers ────────────────────────────────────────────────────

func (g *GenPIR) emit(instr pir.Instr) {
	g.prog.Instrs = append(g.prog.Instrs, instr)
}

func (g *GenPIR) emitN(op pir.Instruction) {
	g.emit(pir.Instr{Op: op})
}

func (g *GenPIR) emitNum(op pir.Instruction, n uint16) {
	g.emit(pir.Instr{Op: op, Operand: pir.Operand{Type: pir.OpNumber, Num: n}})
}

func (g *GenPIR) emitName(op pir.Instruction, name string) {
	g.emit(pir.Instr{Op: op, Operand: pir.Operand{Type: pir.OpName, Name: name}})
}

func (g *GenPIR) emitStr(op pir.Instruction, s string) {
	g.emit(pir.Instr{Op: op, Operand: pir.Operand{Type: pir.OpString, Str: s}})
}

func (g *GenPIR) emitCond(op pir.Instruction, cond pir.Condition) {
	g.emit(pir.Instr{Op: op, Operand: pir.Operand{Type: pir.OpCondition, Cond: cond}})
}

// ── Top-level entry point ───────────────────────────────────────────

// GenPIR runs the checker and translates the AST into a PIR program.
func (p Program) GenPIR() (*pir.Program, error) {
	c := NewChecker()
	if err := p.Check(c); err != nil {
		return nil, err
	}
	g := NewGenPIR(c)
	g.currentScope = c.root

	// Prologue: initialise data stack after variables and tasks.
	// The PIR model expects HL (or equivalent) past all static data.
	// Emit a NOP as placeholder for the platform-specific loader.
	g.emitN(pir.NOP)

	// Classify top-level statements
	procedures, dataStmts, dataItems, err := g.classifyStmts(p.Statements)
	if err != nil {
		return nil, err
	}

	// Main body — inline statements that are not procedures / data / declares.
	for _, s := range p.Statements {
		switch c := s.Command.(type) {
		case Procedure, Data, At:
			// emitted separately
		case Declare:
			if c.Initializer != nil {
				// Emit init code for globals; VAR is emitted by deferred dataItems.
				if err := c.Initializer.Expr.genPIRExpr(g); err != nil {
					return nil, err
				}
				size := c.StorageSize()
				if size == 1 {
					g.emitName(pir.PUT_B, string(c.Identifier))
				} else {
					g.emitName(pir.PUT_W, string(c.Identifier))
				}
			}
		default:
			if err := s.genPIR(g); err != nil {
				return nil, err
			}
		}
	}
	g.emitN(pir.HLT)

	// Emit procedures
	for _, proc := range procedures {
		if err := proc.genPIR(g); err != nil {
			return nil, err
		}
	}

	// Emit DATA statements
	for _, ds := range dataStmts {
		if err := ds.genPIR(g); err != nil {
			return nil, err
		}
	}

	// Emit task bodies
	taskDefs := c.TaskDefs()
	for i := range taskDefs {
		t := &taskDefs[i]
		g.inTask = true
		g.emitNum(pir.PRIORITY, uint16(t.Priority))
		g.emitName(pir.JOB, fmt.Sprintf("_plz_task_%d", i))
		var taskDeclares []Statement
		for j := range t.Body {
			if _, ok := t.Body[j].Command.(Declare); ok {
				taskDeclares = append(taskDeclares, t.Body[j])
				continue
			}
			if err := t.Body[j].genPIR(g); err != nil {
				return nil, err
			}
		}
		// Emit task declares after body (data at end)
		for _, ds := range taskDeclares {
			if err := ds.genPIR(g); err != nil {
				return nil, err
			}
		}
		g.inTask = false
		g.emitN(pir.BYE)
	}

	// Emit deferred data items (AT + DECLARE)
	for _, item := range dataItems {
		if item.at != nil {
			if item.at.HasBank {
				g.emitNum(pir.BANK, uint16(item.at.BankNumber))
			} else if item.at.Address.Expr != nil {
				item.at.Address.genPIRExpr(g)
				// AT with dynamic address — PIR AT is one-shot directive
			}
		}
		if item.declare != nil {
			if item.declare.ConstantValue != nil {
				g.genPIRDeclareConst(item.declare)
			} else {
				g.genPIRDeclare(item.declare)
			}
		}
	}

	// Task scheduler is emitted by the Z80 backend when it sees JOB/BYE.
	// No need to emit it in PIR — tasks are modelled directly.

	return g.prog, nil
}

// classifyStmts mirrors genClassifyStmts.
func (g *GenPIR) classifyStmts(stmts []Statement) (procedures []Procedure, dataStmts []Data, dataItems []dataItem, err error) {
	for _, s := range stmts {
		switch cmd := s.Command.(type) {
		case At:
			if cmd.HasBank {
				g.emitNum(pir.BANK, uint16(cmd.BankNumber))
			} else {
				dataItems = append(dataItems, dataItem{at: &cmd})
			}
		case Declare:
			dataItems = append(dataItems, dataItem{declare: &cmd})
		case Data:
			dataStmts = append(dataStmts, cmd)
		case Procedure:
			procedures = append(procedures, cmd)
		default:
			// handled inline
		}
	}
	return
}

// ── Statement generation ────────────────────────────────────────────

// genPIR generates PIR for a statement. It is the PIR equivalent of Gen.
type pirGenner interface {
	genPIR(*GenPIR) error
}

func (s Statement) genPIR(g *GenPIR) error {
	if s.Label != nil {
		g.emitName(pir.TAG, s.Label.Name)
	}
	genner, ok := s.Command.(pirGenner)
	if ok {
		return genner.genPIR(g)
	}
	return nil
}

// If genPIR
func (s If) genPIR(g *GenPIR) error {
	n := g.nextLabel()
	elseTag := fmt.Sprintf("_else_%d", n)
	if err := g.genPIRCondBranch(s.Condition, elseTag); err != nil {
		return err
	}
	if err := s.Then.genPIR(g); err != nil {
		return err
	}
	if s.Else != nil {
		endTag := fmt.Sprintf("_end_%d", n)
		g.emitName(pir.GO, endTag)
		g.emitName(pir.TAG, elseTag)
		if err := s.Else.genPIR(g); err != nil {
			return err
		}
		g.emitName(pir.TAG, endTag)
	} else {
		g.emitName(pir.TAG, elseTag)
	}
	return nil
}

// genPIRCondBranch evaluates an expression and emits a conditional jump
// to falseTag when the expression is false (zero).
func (g *GenPIR) genPIRCondBranch(e Expression, falseTag string) error {
	if inf := e.Infix(); inf != nil {
		switch inf.Operator {
		case OperatorEQU, OperatorNEQ, OperatorGT, OperatorLT, OperatorGTE, OperatorLTE:
			if err := inf.Operands[0].genPIRExpr(g); err != nil {
				return err
			}
			if err := inf.Operands[1].genPIRExpr(g); err != nil {
				return err
			}
			cond := pirOpCond(inf.Operator)
			isByte := g.isByteInfix(inf)
			if isByte {
				g.emitCond(pir.IS_B, cond)
			} else {
				g.emitCond(pir.IS_W, cond)
			}
			g.emitN(pir.NOT_W)
			g.emitName(pir.GO_IF, falseTag)
			return nil
		}
	}
	// General expression — evaluate, compare against 0
	if err := e.genPIRExpr(g); err != nil {
		return err
	}
	// IS_W EQ against 0: push 0, compare, GO_IF falseTag
	// Actually, simpler: DUP / NOT_W / GO_IF falseTag
	// NOT: 1 if 0 else 0 → GO_IF (jumps if true, i.e. if original was false)
	// Wait: GO_IF pops value and jumps if non-zero.
	// We want: jump to falseTag if expression is 0.
	// So: evaluate expression (stack: val), NOT_W (stack: 1 if 0 else 0),
	// then GO_IF falseTag (jumps if result is non-zero, i.e. original was 0).
	g.emitN(pir.NOT_W)
	g.emitName(pir.GO_IF, falseTag)
	return nil
}

// Let genPIR — assignment
func (s Let) genPIR(g *GenPIR) error {
	if err := s.Expression.genPIRExpr(g); err != nil {
		return err
	}
	if s.Target2 != nil {
		// After CALL with 2 returns, stack = [ret1, ret2] with ret2 as TOS.
		// We want ret1 → Reference (first target), ret2 → Target2 (second target).
		// So: pop to second target (TOS = ret2), then pop to first target (new TOS = ret1)
		if err := g.genPIRPutRef(s.Target2); err != nil {
			return err
		}
		if err := g.genPIRPutRef(&s.Reference); err != nil {
			return err
		}
	} else {
		if err := g.genPIRPutRef(&s.Reference); err != nil {
			return err
		}
	}
	return nil
}

// genPIRPutRef stores TOS into the given reference (variable, array element, or field).
func (g *GenPIR) genPIRPutRef(r *Reference) error {
	if r == nil {
		g.emitN(pir.DROP)
		return nil
	}
	if len(r.Fields) == 0 && len(r.Subscripts) == 0 {
		// Simple variable
		if g.isByteRef(r) {
			g.emitName(pir.PUT_B, string(r.Identifier))
		} else {
			g.emitName(pir.PUT_W, string(r.Identifier))
		}
		return nil
	}
	// Array or field access — push address, then WRITE
	g.genPIRRefAddr(r)
	if g.isByteRef(r) {
		g.emitN(pir.SWAP) // stack: [addr, val] → [val, addr]
		g.emitN(pir.WRITE_B)
	} else {
		g.emitN(pir.SWAP)
		g.emitN(pir.WRITE_W)
	}
	return nil
}

// genPIRRefAddr pushes the hardware address of a reference onto the data stack.
func (g *GenPIR) genPIRRefAddr(r *Reference) {
	if g.checker != nil && g.currentScope != nil {
		if d, ok := g.currentScope.Lookup(r.Identifier); ok && d.DataValue != nil {
			g.emitName(pir.PUSH_D, string(r.Identifier))
			goto doOffsets
		}
		if d, ok := g.currentScope.Lookup(r.Identifier); ok && d.ParamRef && (d.Type.Predeclared() == PredeclaredData || d.Type.Record() != nil) {
			g.emitName(pir.GET_W, string(r.Identifier))
			goto doOffsets
		}
	}
	g.emitName(pir.PUSH_A, string(r.Identifier))
doOffsets:
	// Track the type at the current address for subscript sizing.
	subType, _ := g.localType(r.Identifier)

	if len(r.Fields) > 0 {
		// If the base is an array (arr[i].x), process subscripts first.
		// If the base is a record (s.arr[i]), process the field first.
		if arr := subType.Array(); arr != nil {
			// Base is array → process subscripts, then field offset
			goto doSubscripts
		}
		// Base is record → add field offset, walk into field type
		rec := subType.Record()
		if rec != nil {
			for j, f := range rec.Fields {
				if f.Identifier == r.Fields[0] {
					off := rec.FieldOffset(j)
					if off > 0 {
						g.emitNum(pir.PUSH_W, uint16(off))
						g.emitN(pir.ADD_W)
					}
					subType = f.Type
					break
				}
			}
		}
	}
	// Subscript offsets, walking into nested Array types.
	for _, idx := range r.Subscripts {
		idx.genPIRExpr(g)
		arr := subType.Array()
		if arr != nil {
			if arr.ElemType.Size() > 1 {
				g.emitNum(pir.PUSH_W, uint16(arr.ElemType.Size()))
				g.emitN(pir.MUL_W)
			}
			subType = arr.ElemType // walk into next dimension
		} else if subType.Size() > 1 {
			g.emitNum(pir.PUSH_W, uint16(subType.Size()))
			g.emitN(pir.MUL_W)
		}
		g.emitN(pir.ADD_W)
	}
	return

doSubscripts:
	// For array-of-record (arr[i].x), process subscripts first,
	// sizing by the array element (record) size.
	subType2, _ := g.localType(r.Identifier)
	for _, idx := range r.Subscripts {
		idx.genPIRExpr(g)
		arr := subType2.Array()
		if arr != nil {
			if arr.ElemType.Size() > 1 {
				g.emitNum(pir.PUSH_W, uint16(arr.ElemType.Size()))
				g.emitN(pir.MUL_W)
			}
			subType2 = arr.ElemType
		} else if subType2.Size() > 1 {
			g.emitNum(pir.PUSH_W, uint16(subType2.Size()))
			g.emitN(pir.MUL_W)
		}
		g.emitN(pir.ADD_W)
	}
	// Then add field offset
	if len(r.Fields) > 0 {
		rec := subType2.Record()
		if rec != nil {
			for j, f := range rec.Fields {
				if f.Identifier == r.Fields[0] {
					off := rec.FieldOffset(j)
					if off > 0 {
						g.emitNum(pir.PUSH_W, uint16(off))
						g.emitN(pir.ADD_W)
					}
					break
				}
			}
		}
	}
}

func (g *GenPIR) refLeafSize(r *Reference) int {
	if g.isByteRef(r) {
		return 1
	}
	return 2
}

// Group genPIR — WHILE, FOR, DO, CASE
func (s Group) genPIR(g *GenPIR) error {
	switch {
	case s.While != nil:
		return g.genPIRWhile(s.While, s.Statements)
	case s.For != nil:
		return g.genPIRFor(s.For, s.Statements)
	case s.Case != nil:
		return g.genPIRCase(s.Case)
	default:
		// DO ... END
		for _, stmt := range s.Statements {
			if err := stmt.genPIR(g); err != nil {
				return err
			}
		}
		return nil
	}
}

func (g *GenPIR) genPIRWhile(w *While, body []Statement) error {
	n := g.nextLabel()
	startTag := fmt.Sprintf("_while_%d", n)
	endTag := fmt.Sprintf("_end_%d", n)
	g.loopStack = append(g.loopStack, struct{ start, end string }{startTag, endTag})
	defer func() { g.loopStack = g.loopStack[:len(g.loopStack)-1] }()
	g.emitName(pir.TAG, startTag)
	if err := g.genPIRCondBranch(w.Expression, endTag); err != nil {
		return err
	}
	for _, s := range body {
		if err := s.genPIR(g); err != nil {
			return err
		}
	}
	g.emitName(pir.GO, startTag)
	g.emitName(pir.TAG, endTag)
	return nil
}

func (g *GenPIR) genPIRFor(f *For, body []Statement) error {
	n := g.nextLabel()
	loopTag := fmt.Sprintf("_for_%d", n)
	contTag := fmt.Sprintf("_for_cont_%d", n)
	endTag := fmt.Sprintf("_end_%d", n)
	g.loopStack = append(g.loopStack, struct{ start, end string }{contTag, endTag})
	defer func() { g.loopStack = g.loopStack[:len(g.loopStack)-1] }()

	isByte := g.isByteRef(&f.Reference)

	// var = start
	if err := f.Start.genPIRExpr(g); err != nil {
		return err
	}
	if isByte {
		g.emitName(pir.PUT_B, string(f.Reference.Identifier))
	} else {
		g.emitName(pir.PUT_W, string(f.Reference.Identifier))
	}

	// Compute and save TO value
	if err := f.To.genPIRExpr(g); err != nil {
		return err
	}
	toTemp := fmt.Sprintf("_plz_for_to_%d", n)
	g.emitNum(pir.ALLOC, 2)
	g.emitName(pir.VAR, toTemp)
	g.emitName(pir.PUT_W, toTemp)

	// Compute and save STEP (default 1)
	if f.By != nil {
		if err := f.By.genPIRExpr(g); err != nil {
			return err
		}
	} else {
		g.emitNum(pir.PUSH_W, 1)
	}
	stepTemp := fmt.Sprintf("_plz_for_step_%d", n)
	g.emitNum(pir.ALLOC, 2)
	g.emitName(pir.VAR, stepTemp)
	g.emitName(pir.PUT_W, stepTemp)

	g.emitName(pir.TAG, loopTag)

	// Check: var > to → exit
	if isByte {
		g.emitName(pir.GET_B, string(f.Reference.Identifier))
	} else {
		g.emitName(pir.GET_W, string(f.Reference.Identifier))
	}
	g.emitName(pir.GET_W, toTemp)
	if isByte {
		g.emitCond(pir.IS_B, pir.CondGT)
	} else {
		g.emitCond(pir.IS_W, pir.CondGT)
	}
	g.emitName(pir.GO_IF, endTag)

	// Body
	for _, s := range body {
		if err := s.genPIR(g); err != nil {
			return err
		}
	}

	g.emitName(pir.TAG, contTag)

	// var = var + step
	if isByte {
		g.emitName(pir.GET_B, string(f.Reference.Identifier))
	} else {
		g.emitName(pir.GET_W, string(f.Reference.Identifier))
	}
	g.emitName(pir.GET_W, stepTemp)
	if isByte {
		g.emitN(pir.ADD_B)
	} else {
		g.emitN(pir.ADD_W)
	}
	if isByte {
		g.emitName(pir.PUT_B, string(f.Reference.Identifier))
	} else {
		g.emitName(pir.PUT_W, string(f.Reference.Identifier))
	}

	g.emitName(pir.GO, loopTag)
	g.emitName(pir.TAG, endTag)

	return nil
}

// resolveCaseVal resolves a CASE OF value to an integer. When the value
// is a named constant (cv.Name != ""), it looks up the constant in the
// current scope to get its numeric value.
func (g *GenPIR) resolveCaseVal(cv CaseVal) (int, error) {
	if cv.Name == "" {
		return cv.Value, nil
	}
	d, ok := g.currentScope.Lookup(Identifier(cv.Name))
	if !ok || d.ConstantValue == nil {
		return 0, fmt.Errorf("CASE: undefined constant %q", cv.Name)
	}
	if n := d.ConstantValue.Number(); n != nil {
		return n.Value, nil
	}
	return 0, fmt.Errorf("CASE: constant %q is not a number", cv.Name)
}

func (g *GenPIR) genPIRCase(c *Case) error {
	n := g.nextLabel()
	endTag := fmt.Sprintf("_case_end_%d", n)

	// Evaluate selector
	if err := c.Expression.genPIRExpr(g); err != nil {
		return err
	}

	for i, branch := range c.Branches {
		branchTag := fmt.Sprintf("_case_%d_%d", n, i)
		nextTag := fmt.Sprintf("_case_next_%d_%d", n, i)

		// Compare selector against any of the values
		for j, cv := range branch.Values {
			val, err := g.resolveCaseVal(cv)
			if err != nil {
				return err
			}
			g.emitN(pir.DUP)
			g.emitNum(pir.PUSH_W, uint16(val))
			g.emitCond(pir.IS_W, pir.CondEQ)
			if j < len(branch.Values)-1 {
				g.emitName(pir.GO_IF, branchTag)
			} else {
				g.emitName(pir.GO_IF, branchTag)
			}
		}
		// If last value didn't match, skip to next branch
		g.emitName(pir.GO, nextTag)

		g.emitName(pir.TAG, branchTag)
		// Execute branch statement
		if err := branch.Statement.genPIR(g); err != nil {
			return err
		}
		g.emitName(pir.GO, endTag)
		g.emitName(pir.TAG, nextTag)
	}

	// Default branch
	if c.Default != nil {
		if err := c.Default.genPIR(g); err != nil {
			return err
		}
	}

	g.emitName(pir.TAG, endTag)
	g.emitN(pir.DROP) // discard selector
	return nil
}

// Procedure genPIR
func (s Procedure) genPIR(g *GenPIR) error {
	g.procName = string(s.Name.Name)
	g.procReturnType = s.Type
	g.inFrame = s.Reentrant
	g.emitName(pir.ROUTE, string(s.Name.Name))

	g.pushScope()
	if s.Reentrant {
		// Compute frame size from parameters + locals
		size := 0
		for _, pt := range s.ParamTypes {
			if pt.Predeclared() == PredeclaredData {
				size += 4 // 2 for address + 2 for length
			} else {
				size += pt.Size()
			}
		}
		g.emitNum(pir.FRAME, uint16(size))
		// Declare locals for params (in order)
		for i, p := range s.Parameters {
			if s.ParamTypes[i].Predeclared() == PredeclaredData {
				g.emitName(pir.LOCAL_W, string(p))       // address
				g.emitName(pir.LOCAL_W, string(p)+"_len") // length
			} else if s.ParamTypes[i].Predeclared() == PredeclaredByte {
				g.emitName(pir.LOCAL_B, string(p))
			} else {
				g.emitName(pir.LOCAL_W, string(p))
			}
		}
		// Pop params from data stack into locals (TOS = first param)
		for i := 0; i < len(s.Parameters); i++ {
			p := s.Parameters[i]
			if s.ParamTypes[i].Predeclared() == PredeclaredData {
				g.emitName(pir.PUT_W, string(p))       // pop address
				g.emitName(pir.PUT_W, string(p)+"_len") // pop length
			} else if s.ParamTypes[i].Predeclared() == PredeclaredByte {
				g.emitName(pir.PUT_B, string(p))
			} else {
				g.emitName(pir.PUT_W, string(p))
			}
		}
	} else {
		// Non-reentrant: declare global vars for params and pop them in
		for i, p := range s.Parameters {
			if s.ParamTypes[i].Predeclared() == PredeclaredData {
				g.emitNum(pir.ALLOC, 2)
				g.emitName(pir.VAR, string(p))       // address
				g.emitNum(pir.ALLOC, 2)
				g.emitName(pir.VAR, string(p)+"_len") // length
			} else if s.ParamTypes[i].Predeclared() == PredeclaredByte {
				g.emitName(pir.VAR, string(p))
			} else {
				g.emitNum(pir.ALLOC, 2)
				g.emitName(pir.VAR, string(p))
			}
		}
		// Pop params from data stack (TOS = first param)
		for i := 0; i < len(s.Parameters); i++ {
			p := s.Parameters[i]
			if s.ParamTypes[i].Predeclared() == PredeclaredData {
				g.emitName(pir.PUT_W, string(p))       // pop address
				g.emitName(pir.PUT_W, string(p)+"_len") // pop length
			} else if s.ParamTypes[i].Predeclared() == PredeclaredByte {
				g.emitName(pir.PUT_B, string(p))
			} else {
				g.emitName(pir.PUT_W, string(p))
			}
		}
	}

	for _, stmt := range s.Statements {
		if err := stmt.genPIR(g); err != nil {
			return err
		}
	}

	g.popScope()
	// Return if the procedure body didn't end with a RETURN
	if s.Interrupt != nil && s.Interrupt.NMI {
		g.emitN(pir.DONE_NMI)
	} else if s.Interrupt != nil {
		g.emitN(pir.DONE_INTERRUPT)
	} else {
		g.emitN(pir.DONE)
	}

	g.procName = ""
	g.procReturnType = Type{}
	g.inFrame = false
	return nil
}

// Return genPIR
func (s Return) genPIR(g *GenPIR) error {
	for _, expr := range s.Expressions {
		if g.procReturnType.Record() != nil {
			// For RECORD return type, push the ADDRESS of the record
			if err := g.genPIRExprAsAddr(expr); err != nil {
				return err
			}
		} else {
			if err := expr.genPIRExpr(g); err != nil {
				return err
			}
		}
	}
	if g.procName != "" {
		g.emitN(pir.DONE)
	}
	return nil
}

// isDataOrRecordParam returns true when t is a DATA or RECORD type.
func isDataOrRecordParam(t Type) bool {
	return t.Record() != nil || t.Predeclared() == PredeclaredData
}

// pushDataLength pushes the element count of a DATA argument onto the data stack.
// For DATA block references it emits a compile-time constant; for DATA params
// it reads the stored length from the synthetic <name>_len slot.
func (g *GenPIR) pushDataLength(expr Expression) error {
	if g.currentScope == nil {
		return fmt.Errorf("pushDataLength: no active scope")
	}
	if ref := expr.Ref(); ref != nil && len(ref.Fields) == 0 && len(ref.Subscripts) == 0 {
		if d, ok := g.currentScope.Lookup(ref.Identifier); ok {
			if d.DataValue != nil {
				data := d.DataValue
				switch {
				case data.Tile != nil:
					g.emitNum(pir.PUSH_W, uint16(len(data.Tile.Tiles)))
				case data.Text != nil:
					g.emitNum(pir.PUSH_W, uint16(len(data.Text.Value)))
				default:
					g.emitNum(pir.PUSH_W, uint16(len(data.Values)))
				}
				return nil
			}
			if d.ParamRef && d.Type.Predeclared() == PredeclaredData {
				g.emitName(pir.GET_W, string(ref.Identifier)+"_len")
				return nil
			}
		}
	}
	// Fallback: if it has subscripts, evaluate LENGTH at the referent
	if ref := expr.Ref(); ref != nil && len(ref.Fields) == 0 {
		if len(ref.Subscripts) > 0 {
			if d, ok := g.currentScope.Lookup(ref.Identifier); ok && d.DataValue != nil {
				g.emitNum(pir.PUSH_W, uint16(len(d.DataValue.Values)))
				return nil
			}
		}
	}
	return fmt.Errorf("pushDataLength: cannot determine length of %v", expr)
}

// genPIRExprAsAddr evaluates an expression and pushes its address onto the
// data stack (instead of its value). Used when passing arguments to DATA or
// RECORD parameters.
func (g *GenPIR) genPIRExprAsAddr(expr Expression) error {
	if suff := expr.Suffix(); suff != nil {
		switch suff.Operator {
		case OperatorINDEX:
			return g.genPIRIndexAddr(suff.Operands)
		case OperatorFIELD:
			return g.genPIRFieldAddr(suff.Operands)
		}
	}
	if ref := expr.Ref(); ref != nil && len(ref.Fields) == 0 && len(ref.Subscripts) == 0 {
		if d, ok := g.currentScope.Lookup(ref.Identifier); ok {
			if d.DataValue != nil {
				g.emitName(pir.PUSH_D, string(ref.Identifier))
				return nil
			}
			if d.ParamRef {
				g.emitName(pir.GET_W, string(ref.Identifier))
				return nil
			}
		}
		g.emitName(pir.PUSH_A, string(ref.Identifier))
		return nil
	}
	return expr.genPIRExpr(g)
}

// Call genPIR
func (s Call) genPIR(g *GenPIR) error {
	name := string(s.Identifier)
	var pd *ProcData
	if g.checker != nil {
		pd = g.checker.ProcData(name)
	}
	// Push args right-to-left (last arg first)
	for i := len(s.Arguments) - 1; i >= 0; i-- {
		if pd != nil && i < len(pd.ParamTypes) && pd.ParamTypes[i].Predeclared() == PredeclaredData {
			// DATA param: push length first (underneath), then address (on top)
			if err := g.pushDataLength(s.Arguments[i]); err != nil {
				return err
			}
			if err := g.genPIRExprAsAddr(s.Arguments[i]); err != nil {
				return err
			}
		} else if pd != nil && i < len(pd.ParamTypes) && isDataOrRecordParam(pd.ParamTypes[i]) {
			// RECORD param: push address only
			if err := g.genPIRExprAsAddr(s.Arguments[i]); err != nil {
				return err
			}
		} else {
			if err := s.Arguments[i].genPIRExpr(g); err != nil {
				return err
			}
		}
	}
	g.emitName(pir.RUN, name)
	return nil
}

// GoTo genPIR
func (s GoTo) genPIR(g *GenPIR) error {
	g.emitName(pir.GO, s.Name)
	return nil
}

// Output genPIR
func (s Output) genPIR(g *GenPIR) error {
	if err := s.Value.genPIRExpr(g); err != nil {
		return err
	}
	if s.IsWord {
		g.emitNum(pir.OUT_W, uint16(s.Port))
	} else {
		g.emitNum(pir.OUT_B, uint16(s.Port))
	}
	return nil
}

// Halt genPIR
func (s Halt) genPIR(g *GenPIR) error {
	if g.inTask {
		g.emitN(pir.BYE) // mark task DEAD, reschedule
	} else {
		g.emitN(pir.HLT)
	}
	return nil
}

// Enable/Disable genPIR
func (s Enable) genPIR(g *GenPIR) error {
	g.emitN(pir.ENI)
	return nil
}
func (s Disable) genPIR(g *GenPIR) error {
	g.emitN(pir.DII)
	return nil
}

// BankStmt genPIR
func (s BankStmt) genPIR(g *GenPIR) error {
	if err := s.Number.genPIRExpr(g); err != nil {
		return err
	}
	g.emitN(pir.SWITCH)
	return nil
}

// Suspend genPIR
func (s Suspend) genPIR(g *GenPIR) error {
	if g.checker != nil {
		if idx := g.checker.TaskIndex(string(s.Name)); idx >= 0 {
			g.emitName(pir.STOP, fmt.Sprintf("_plz_task_%d", idx))
			return nil
		}
	}
	g.emitName(pir.STOP, string(s.Name))
	return nil
}

// Resume genPIR
func (s Resume) genPIR(g *GenPIR) error {
	if g.checker != nil {
		if idx := g.checker.TaskIndex(string(s.Name)); idx >= 0 {
			g.emitName(pir.START, fmt.Sprintf("_plz_task_%d", idx))
			return nil
		}
	}
	g.emitName(pir.START, string(s.Name))
	return nil
}

// Sleep genPIR
func (s Sleep) genPIR(g *GenPIR) error {
	if err := s.Duration.genPIRExpr(g); err != nil {
		return err
	}
	g.emitN(pir.SLEEP)
	return nil
}

// Yield genPIR
func (s Yield) genPIR(g *GenPIR) error {
	g.emitN(pir.YIELD)
	return nil
}

// Break genPIR — jump to end of current loop
func (s Break) genPIR(g *GenPIR) error {
	if len(g.loopStack) == 0 {
		return nil // checker already rejected this
	}
	g.emitName(pir.GO, g.loopStack[len(g.loopStack)-1].end)
	return nil
}

// Continue genPIR — jump to start of current loop
func (s Continue) genPIR(g *GenPIR) error {
	if len(g.loopStack) == 0 {
		return nil // checker already rejected this
	}
	g.emitName(pir.GO, g.loopStack[len(g.loopStack)-1].start)
	return nil
}

// Task genPIR
func (s Task) genPIR(g *GenPIR) error {
	// Tasks are handled in Program.GenPIR via TaskDefs()
	return nil
}

// InterruptStmt genPIR
func (s InterruptStmt) genPIR(g *GenPIR) error {
	if s.NMI {
		g.emitName(pir.NMI, string(s.Target))
	} else {
		g.emitName(pir.INT, string(s.Target))
	}
	return nil
}

// Save genPIR
func (s Save) genPIR(g *GenPIR) error {
	g.emitN(pir.SRAM_ON)
	// SRAM_ON enables SRAM and pushes the platform-specific base address onto
	// the data stack as the destination. If a custom AT address is given,
	// discard the default and push the user's address instead.
	if s.Location != nil {
		g.emitN(pir.DROP)
		if err := s.Location.genPIRExpr(g); err != nil {
			return err
		}
	}
	// Push src address, then length
	if err := g.genPIRExprOrRef(s.Source); err != nil {
		return err
	}
	g.emitNum(pir.PUSH_W, uint16(g.refSize(s.Source)))
	g.emitN(pir.SAVE)
	g.emitN(pir.SRAM_OFF)
	return nil
}

// Load genPIR
func (s Load) genPIR(g *GenPIR) error {
	g.emitN(pir.SRAM_ON)
	// SRAM_ON pushes the base address as src. If AT given, discard and use
	// custom address instead.
	if s.Location != nil {
		g.emitN(pir.DROP)
		if err := s.Location.genPIRExpr(g); err != nil {
			return err
		}
	}
	if err := g.genPIRExprOrRef(s.Target); err != nil {
		return err
	}
	g.emitNum(pir.PUSH_W, uint16(g.refSize(s.Target)))
	g.emitN(pir.LOAD)
	g.emitN(pir.SRAM_OFF)
	return nil
}

// Pragma genPIR
func (s Pragma) genPIR(g *GenPIR) error {
	var flags uint16
	for _, id := range s.Idents {
		if id == "BOUNDCHECK" {
			flags |= 1
		}
	}
	g.emitNum(pir.PRAGMA, flags)
	return nil
}

// At genPIR (compile-time directive)
func (s At) genPIR(g *GenPIR) error {
	if s.HasBank {
		g.emitNum(pir.BANK, uint16(s.BankNumber))
	} else if s.Address.Expr != nil {
		// Compile-time AT: need constant value
		if n, ok := g.constEval(s.Address); ok {
			g.emitNum(pir.AT, uint16(n))
		} else {
			return fmt.Errorf("AT address must be a constant expression")
		}
	}
	return nil
}

// Declare genPIR
func (s Declare) genPIR(g *GenPIR) error {
	if s.ConstantValue != nil {
		return nil // constants are compile-time
	}
	size := s.StorageSize()
	if size > 2 || size == 0 {
		g.emitNum(pir.ALLOC, uint16(size))
	} else if size == 2 {
		g.emitNum(pir.ALLOC, 2)
	}
	g.emitName(pir.VAR, string(s.Identifier))

	if s.Initializer != nil {
		if err := s.Initializer.Expr.genPIRExpr(g); err != nil {
			return err
		}
		if size == 1 {
			g.emitName(pir.PUT_B, string(s.Identifier))
		} else {
			g.emitName(pir.PUT_W, string(s.Identifier))
		}
	}
	return nil
}

// genPIRDeclare handles a DECLARE with a constant value.
func (g *GenPIR) genPIRDeclareConst(d *Declare) {
	if n := d.ConstantValue.Number(); n != nil {
		g.constValues[d.Identifier] = d.ConstantValue
	}
}

// genPIRDeclare handles a normal DECLARE.
func (g *GenPIR) genPIRDeclare(d *Declare) {
	size := d.StorageSize()
	if size > 2 || size == 0 {
		g.emitNum(pir.ALLOC, uint16(size))
	} else if size == 2 {
		g.emitNum(pir.ALLOC, 2)
	}
	g.emitName(pir.VAR, string(d.Identifier))
}

// Data genPIR
func (s Data) genPIR(g *GenPIR) error {
	if s.Name != "" {
		g.emitName(pir.TAG, s.Name)
	}
	if s.Tile != nil {
		for _, tile := range s.Tile.Tiles {
			for _, b := range tile.Bytes() {
				g.emitNum(pir.DATA_B, uint16(b))
			}
		}
		return nil
	}
	if s.Text != nil {
		g.emitStr(pir.DATA_STR, s.Text.Value)
		return nil
	}
	for _, v := range s.Values {
		n, ok := g.constEval(v)
		if ok {
			g.emitNum(pir.DATA_B, uint16(n))
		}
	}
	return nil
}

// Constant genPIR
func (s Constant) genPIR(g *GenPIR) error {
	// Constants are compile-time — no PIR emitted.
	return nil
}

// Define genPIR — type aliases are compile-time
func (s Define) genPIR(g *GenPIR) error {
	return nil
}

// ── Expression generation ───────────────────────────────────────────

// genPIRExpr evaluates an expression and leaves the result on the data stack.
func (e Expression) genPIRExpr(g *GenPIR) error {
	switch {
	case e.Operand() != nil:
		return e.Operand().genPIRExpr(g)
	case e.Prefix() != nil:
		return e.Prefix().genPIRExpr(g)
	case e.Infix() != nil:
		return e.Infix().genPIRExpr(g)
	case e.Suffix() != nil:
		return e.Suffix().genPIRExpr(g)
	}
	return nil
}

// Operand genPIRExpr
func (o Operand) genPIRExpr(g *GenPIR) error {
	switch {
	case o.Literal() != nil:
		if n := o.Literal().Number(); n != nil {
			if n.Value >= 0 && n.Value <= 255 {
				g.emitNum(pir.PUSH_B, uint16(n.Value))
			} else {
				g.emitNum(pir.PUSH_W, uint16(n.Value))
			}
		} else if t := o.Literal().Text(); t != nil {
			// String literal: emit DATA_STR + TAG, then PUSH_A
			n := g.nextLabel()
			label := fmt.Sprintf("_plz_str_%d", n)
			g.emitName(pir.TAG, label)
			g.emitStr(pir.DATA_STR, t.Value)
			g.emitName(pir.PUSH_A, label)
		}
	case o.Reference() != nil:
		ref := o.Reference()
		if g.checker != nil && g.currentScope != nil {
			if d, ok := g.currentScope.Lookup(ref.Identifier); ok && d.ConstantValue != nil {
				if n := d.ConstantValue.Number(); n != nil {
					g.emitNum(pir.PUSH_W, uint16(n.Value))
					return nil
				}
			}
			if d, ok := g.currentScope.Lookup(ref.Identifier); ok && d.DataValue != nil && len(ref.Subscripts) == 0 {
				g.emitName(pir.PUSH_D, string(ref.Identifier))
				return nil
			}
		}
		if len(ref.Fields) > 0 || len(ref.Subscripts) > 0 {
			g.genPIRRefAddr(ref)
			if g.isByteRef(ref) {
				g.emitN(pir.READ_B)
			} else {
				g.emitN(pir.READ_W)
			}
		} else {
			if g.isByteRef(ref) {
				g.emitName(pir.GET_B, string(ref.Identifier))
			} else {
				g.emitName(pir.GET_W, string(ref.Identifier))
			}
		}
	case o.Expr() != nil:
		return o.Expr().genPIRExpr(g)
	case o.Input() != nil:
		if n, ok := g.constEval(o.Input().Port); ok {
			g.emitNum(pir.IN_B, uint16(n))
		} else {
			return fmt.Errorf("INPUT port must be a constant (PIR backend limitation), got non-constant expression")
		}
	case o.Length() != nil:
		l := o.Length()
		if g.currentScope != nil {
			if d, ok := g.currentScope.Lookup(l.Identifier); ok && d.ParamRef && d.Type.Predeclared() == PredeclaredData {
				// DATA param: read runtime length from the synthetic _len slot
				g.emitName(pir.GET_W, string(l.Identifier)+"_len")
				break
			}
		}
		n, err := g.checker.evalLength(o.Length())
		if err != nil {
			return err
		}
		g.emitNum(pir.PUSH_W, uint16(n))
	}
	return nil
}

// Prefix genPIRExpr
func (p Prefix) genPIRExpr(g *GenPIR) error {
	switch p.Operator {
	case OperatorNEG:
		if err := p.Operand.genPIRExpr(g); err != nil {
			return err
		}
		if g.isByteOperand(&p.Operand) {
			g.emitN(pir.NEG_B)
		} else {
			g.emitN(pir.NEG_W)
		}
	case OperatorNOT:
		if err := p.Operand.genPIRExpr(g); err != nil {
			return err
		}
		if g.isByteOperand(&p.Operand) {
			g.emitN(pir.NOT_B)
		} else {
			g.emitN(pir.NOT_W)
		}
	case Operator(KeywordByte):
		if err := p.Operand.genPIRExpr(g); err != nil {
			return err
		}
		g.emitN(pir.CAST_B)
	case Operator(KeywordWord):
		if err := p.Operand.genPIRExpr(g); err != nil {
			return err
		}
		g.emitN(pir.CAST_W)
	}
	return nil
}

// Infix genPIRExpr
func (i Infix) genPIRExpr(g *GenPIR) error {
	// Evaluate left, then right
	if err := i.Operands[0].genPIRExpr(g); err != nil {
		return err
	}
	if err := i.Operands[1].genPIRExpr(g); err != nil {
		return err
	}
	isByte := g.isByteInfix(&i)

	switch i.Operator {
	case OperatorADD:
		if isByte {
			g.emitN(pir.ADD_B)
		} else {
			g.emitN(pir.ADD_W)
		}
	case OperatorSUB:
		if isByte {
			g.emitN(pir.SUB_B)
		} else {
			g.emitN(pir.SUB_W)
		}
	case OperatorMUL:
		if isByte {
			g.emitN(pir.MUL_B)
		} else {
			g.emitN(pir.MUL_W)
		}
	case OperatorDIV:
		if isByte {
			g.emitN(pir.DIV_B)
		} else {
			g.emitN(pir.DIV_W)
		}
	case OperatorMOD:
		if isByte {
			g.emitN(pir.MOD_B)
		} else {
			g.emitN(pir.MOD_W)
		}
	case OperatorShiftLeft:
		if isByte {
			g.emitN(pir.SHL_B)
		} else {
			g.emitN(pir.SHL_W)
		}
	case OperatorShiftRight:
		if isByte {
			g.emitN(pir.SHR_B)
		} else {
			g.emitN(pir.SHR_W)
		}
	case OperatorAND:
		if isByte {
			g.emitN(pir.AND_B)
		} else {
			g.emitN(pir.AND_W)
		}
	case OperatorLAnd:
		if isByte {
			g.emitN(pir.AND_B)
		} else {
			g.emitN(pir.AND_W)
		}
		g.emitNum(pir.PUSH_B, 0)
		g.emitCond(pir.IS_W, pir.CondNE)
	case OperatorOR:
		if isByte {
			g.emitN(pir.OR_B)
		} else {
			g.emitN(pir.OR_W)
		}
	case OperatorLOr:
		if isByte {
			g.emitN(pir.OR_B)
		} else {
			g.emitN(pir.OR_W)
		}
		g.emitNum(pir.PUSH_B, 0)
		g.emitCond(pir.IS_W, pir.CondNE)
	case OperatorXOR:
		if isByte {
			g.emitN(pir.XOR_B)
		} else {
			g.emitN(pir.XOR_W)
		}
	case OperatorEQU, OperatorNEQ, OperatorGT, OperatorLT, OperatorGTE, OperatorLTE:
		cond := pirOpCond(i.Operator)
		if isByte {
			g.emitCond(pir.IS_B, cond)
		} else {
			g.emitCond(pir.IS_W, cond)
		}
	}
	return nil
}

// Suffix genPIRExpr
func (s *Suffix) genPIRExpr(g *GenPIR) error {
	switch s.Operator {
	case OperatorINDEX:
		return g.genPIRIndexRead(s.Operands)
	case OperatorCALL:
		return g.genPIRCallExpr(s.Operands)
	case OperatorFIELD:
		return g.genPIRFieldRead(s.Operands)
	}
	return nil
}

// genPIRFieldAddr pushes the hardware address of a struct field onto the
// data stack. The base is evaluated first (must produce an address), then
// the field offset is added.
func (g *GenPIR) genPIRFieldAddr(operands []Operand) error {
	// Push base address: for a simple reference push PUSH_A/PUSH_D;
	// for an index expression (arr[i].f) push address without reading.
	baseRef := operands[0].Ref()
	if baseRef != nil && len(baseRef.Fields) == 0 && len(baseRef.Subscripts) == 0 {
		if g.checker != nil && g.currentScope != nil {
			if d, ok := g.currentScope.Lookup(baseRef.Identifier); ok && d.DataValue != nil {
				g.emitName(pir.PUSH_D, string(baseRef.Identifier))
			} else if d, ok := g.currentScope.Lookup(baseRef.Identifier); ok && d.ParamRef && (d.Type.Predeclared() == PredeclaredData || d.Type.Record() != nil) {
				g.emitName(pir.GET_W, string(baseRef.Identifier))
			} else {
				g.emitName(pir.PUSH_A, string(baseRef.Identifier))
			}
		} else {
			g.emitName(pir.PUSH_A, string(baseRef.Identifier))
		}
	} else if baseExpr := operands[0].Expr(); baseExpr != nil {
		if s := baseExpr.Suffix(); s != nil && s.Operator == OperatorINDEX {
			g.genPIRIndexAddr(s.Operands)
		} else {
			if err := operands[0].genPIRExpr(g); err != nil {
				return err
			}
		}
	} else {
		if err := operands[0].genPIRExpr(g); err != nil {
			return err
		}
	}
	// Determine field offset from the base's record type.
	ref := operands[0].Ref()
	if ref == nil {
		if expr := operands[0].Expr(); expr != nil {
			if s := expr.Suffix(); s != nil && len(s.Operands) > 0 {
				ref = s.Operands[0].Ref()
			} else {
				ref = expr.Ref()
			}
		}
	}
	if ref == nil || ref.Identifier == "" {
		return fmt.Errorf("genPIRFieldAddr: cannot determine base reference")
	}
	t, ok := g.localType(ref.Identifier)
	if !ok {
		return fmt.Errorf("genPIRFieldAddr: unknown identifier %s", ref.Identifier)
	}
	if arr := t.Array(); arr != nil {
		t = arr.ElemType
	}
	rec := t.Record()
	if rec == nil {
		return fmt.Errorf("genPIRFieldAddr: %s is not a struct", ref.Identifier)
	}
	fname := operands[1].Reference().Identifier
	off := 0
	for j, f := range rec.Fields {
		if f.Identifier == fname {
			off = rec.FieldOffset(j)
			break
		}
	}
	if off > 0 {
		g.emitNum(pir.PUSH_W, uint16(off))
		g.emitN(pir.ADD_W)
	}
	return nil
}

// genPIRFieldRead reads a struct field value and pushes it on the stack.
func (g *GenPIR) genPIRFieldRead(operands []Operand) error {
	if err := g.genPIRFieldAddr(operands); err != nil {
		return err
	}
	if g.isByteField(operands) {
		g.emitN(pir.READ_B)
	} else {
		g.emitN(pir.READ_W)
	}
	return nil
}

// genPIRIndexAddr pushes the address of arr[index] onto the stack without reading.
func (g *GenPIR) genPIRIndexAddr(operands []Operand) error {
	if len(operands) < 2 {
		return fmt.Errorf("genPIRIndexAddr: need base and index")
	}
	// Push base address
	baseExpr := operands[0].Expr()
	if baseExpr != nil {
		if s := baseExpr.Suffix(); s != nil && s.Operator == OperatorFIELD {
			if err := g.genPIRFieldAddr(s.Operands); err != nil {
				return err
			}
		} else if ref := baseExpr.Ref(); ref != nil && len(ref.Subscripts) == 0 {
			if g.checker != nil && g.currentScope != nil {
				if d, ok := g.currentScope.Lookup(ref.Identifier); ok && d.DataValue != nil {
					g.emitName(pir.PUSH_D, string(ref.Identifier))
				} else if d, ok := g.currentScope.Lookup(ref.Identifier); ok && d.ParamRef && d.Type.Predeclared() == PredeclaredData {
					g.emitName(pir.GET_W, string(ref.Identifier))
				} else {
					g.emitName(pir.PUSH_A, string(ref.Identifier))
				}
			} else {
				g.emitName(pir.PUSH_A, string(ref.Identifier))
			}
		} else if s := baseExpr.Suffix(); s != nil && s.Operator == OperatorINDEX {
			// Nested index: arr[i][j] — push address, not value
			if err := g.genPIRIndexAddr(s.Operands); err != nil {
				return err
			}
		} else {
			if err := baseExpr.genPIRExpr(g); err != nil {
				return err
			}
		}
	} else {
		if ref := operands[0].Ref(); ref != nil && len(ref.Subscripts) == 0 {
			if g.checker != nil && g.currentScope != nil {
				if d, ok := g.currentScope.Lookup(ref.Identifier); ok && d.DataValue != nil {
					g.emitName(pir.PUSH_D, string(ref.Identifier))
				} else if d, ok := g.currentScope.Lookup(ref.Identifier); ok && d.ParamRef && d.Type.Predeclared() == PredeclaredData {
					g.emitName(pir.GET_W, string(ref.Identifier))
				} else {
					g.emitName(pir.PUSH_A, string(ref.Identifier))
				}
			} else {
				g.emitName(pir.PUSH_A, string(ref.Identifier))
			}
		} else {
			if err := operands[0].genPIRExpr(g); err != nil {
				return err
			}
		}
	}
	// Evaluate index
	if err := operands[1].genPIRExpr(g); err != nil {
		return err
	}
	// Scale by element size (determine from base type)
	elemSize := g.indexBaseSize(operands)
	if elemSize > 1 {
		g.emitNum(pir.PUSH_W, uint16(elemSize))
		g.emitN(pir.MUL_W)
	}
	g.emitN(pir.ADD_W)
	return nil
}

// genPIRIndexRead reads an array element via (base)[index] and pushes the value.
func (g *GenPIR) genPIRIndexRead(operands []Operand) error {
	if err := g.genPIRIndexAddr(operands); err != nil {
		return err
	}
	elemSize := g.indexBaseSize(operands)
	if elemSize == 1 {
		g.emitN(pir.READ_B)
	} else {
		g.emitN(pir.READ_W)
	}
	return nil
}

// indexBaseSize returns the element size for the base of an index expression.
func (g *GenPIR) indexBaseSize(operands []Operand) int {
	baseExpr := operands[0].Expr()
	// Try to find a simple reference (may be inside Expression wrapping)
	ref := operands[0].Ref()
	if ref != nil && len(ref.Subscripts) == 0 {
		if g.isByteRef(ref) {
			return 1
		}
		// Check for array-of-byte type (e.g. DECLARE arr ARRAY [N] BYTE)
		if g.checker != nil && g.currentScope != nil {
			if d, ok := g.currentScope.Lookup(ref.Identifier); ok {
				if d.DataValue != nil {
					return 1
				}
				if d.ParamRef && d.Type.Predeclared() == PredeclaredData {
					return 1
				}
				if arr := d.Type.Array(); arr != nil {
					if arr.ElemType.Predeclared() == PredeclaredByte {
						return 1
					}
					return arr.ElemType.Size()
				}
			}
		}
		return 2
	}
	if baseExpr == nil {
		return 2
	}
	if s := baseExpr.Suffix(); s != nil && s.Operator == OperatorINDEX {
		// Nested index: arr[i][j] — inner index pushes address of arr[i] (type ARRAY[N]).
		// This index's element is the inner array's element.
		ref := s.Operands[0].Ref()
		if ref == nil {
			return 2
		}
		if g.checker != nil && g.currentScope != nil {
			if d, ok := g.currentScope.Lookup(ref.Identifier); ok && d.DataValue == nil {
				if arr := d.Type.Array(); arr != nil {
					// arr[i] has type arr.ElemType. If it's another array,
					// the element size is sizeof(innermost element).
					if inner := arr.ElemType.Array(); inner != nil {
						if inner.ElemType.Predeclared() == PredeclaredByte {
							return 1
						}
						return inner.ElemType.Size()
					}
					if arr.ElemType.Predeclared() == PredeclaredByte {
						return 1
					}
					return arr.ElemType.Size()
				}
			}
		}
		return 2
	}
	if s := baseExpr.Suffix(); s != nil && s.Operator == OperatorFIELD {
		// Field of a struct — determine the field's element type.
		ref := s.Operands[0].Ref()
		if ref == nil {
			return 2
		}
		t, ok := g.localType(ref.Identifier)
		if !ok {
			return 2
		}
		if arr := t.Array(); arr != nil {
			t = arr.ElemType
		}
		rec := t.Record()
		if rec == nil {
			return 2
		}
		fname := s.Operands[1].Reference().Identifier
		for _, f := range rec.Fields {
			if f.Identifier == fname {
				ft := f.Type
				if a := ft.Array(); a != nil {
					if a.ElemType.Predeclared() == PredeclaredByte {
						return 1
					}
					if a.ElemType.Record() != nil {
						return nextPow2(a.ElemType.Size())
					}
					return 2
				}
				if ft.Predeclared() == PredeclaredByte {
					return 1
				}
				return ft.Size()
			}
		}
		return 2
	}
	return 2
}

// isByteField reports whether a field access expression yields a byte value.
func (g *GenPIR) isByteField(operands []Operand) bool {
	ref := operands[0].Ref()
	if ref == nil {
		if expr := operands[0].Expr(); expr != nil {
			if s := expr.Suffix(); s != nil && len(s.Operands) > 0 {
				ref = s.Operands[0].Ref()
			} else {
				ref = expr.Ref()
			}
		}
	}
	if ref == nil || ref.Identifier == "" {
		return false
	}
	t, ok := g.localType(ref.Identifier)
	if !ok {
		return false
	}
	if arr := t.Array(); arr != nil {
		t = arr.ElemType
	}
	rec := t.Record()
	if rec == nil {
		return false
	}
	fname := operands[1].Reference().Identifier
	for _, f := range rec.Fields {
		if f.Identifier == fname {
			ft := f.Type
			if a := ft.Array(); a != nil {
				ft = a.ElemType
			}
			return ft.Predeclared() == PredeclaredByte
		}
	}
	return false
}

// genPIROperandAsAddr evaluates an operand and pushes its address onto the
// data stack (instead of its value). Used when passing arguments to DATA or
// RECORD parameters in expression-form calls.
func (g *GenPIR) genPIROperandAsAddr(op Operand) error {
	if ref := op.Reference(); ref != nil && len(ref.Fields) == 0 && len(ref.Subscripts) == 0 {
		if d, ok := g.currentScope.Lookup(ref.Identifier); ok {
			if d.DataValue != nil {
				g.emitName(pir.PUSH_D, string(ref.Identifier))
				return nil
			}
			if d.ParamRef {
				g.emitName(pir.GET_W, string(ref.Identifier))
				return nil
			}
		}
		g.emitName(pir.PUSH_A, string(ref.Identifier))
		return nil
	}
	if expr := op.Expr(); expr != nil {
		return g.genPIRExprAsAddr(*expr)
	}
	return op.genPIRExpr(g)
}

// genPIRCallExpr generates PIR for a call used as an expression value.
// operands[0] is the function reference; operands[1:] are the arguments.
// Args are pushed right-to-left so that TOS is the first argument.
func (g *GenPIR) genPIRCallExpr(operands []Operand) error {
	if len(operands) < 1 {
		return fmt.Errorf("genPIRCallExpr: missing function reference")
	}
	ref := operands[0].Ref()
	if ref == nil {
		return fmt.Errorf("genPIRCallExpr: indirect calls not supported")
	}
	name := string(ref.Identifier)
	var pd *ProcData
	if g.checker != nil {
		pd = g.checker.ProcData(name)
	}
	// Push args right-to-left (last arg first, so first arg is TOS)
	for i := len(operands) - 1; i >= 1; i-- {
		argIdx := i - 1 // operand index 1 → param 0
		if pd != nil && argIdx < len(pd.ParamTypes) && pd.ParamTypes[argIdx].Predeclared() == PredeclaredData {
			// DATA param: push length first, then address
			argExpr := Expression{Expr: &operands[i]}
			if err := g.pushDataLength(argExpr); err != nil {
				return err
			}
			if err := g.genPIROperandAsAddr(operands[i]); err != nil {
				return err
			}
		} else if pd != nil && argIdx < len(pd.ParamTypes) && isDataOrRecordParam(pd.ParamTypes[argIdx]) {
			// RECORD param: push address only
			if err := g.genPIROperandAsAddr(operands[i]); err != nil {
				return err
			}
		} else {
			if err := operands[i].genPIRExpr(g); err != nil {
				return err
			}
		}
	}
	g.emitName(pir.RUN, name)
	return nil
}

// ── Helpers ─────────────────────────────────────────────────────────

// pirOpCond maps an operator to a PIR condition.
func pirOpCond(op Operator) pir.Condition {
	switch op {
	case OperatorEQU:
		return pir.CondEQ
	case OperatorNEQ:
		return pir.CondNE
	case OperatorGT:
		return pir.CondGT
	case OperatorLT:
		return pir.CondLT
	case OperatorGTE:
		return pir.CondGE
	case OperatorLTE:
		return pir.CondLE
	default:
		return pir.CondEQ
	}
}

// constEval evaluates a constant expression.
func (g *GenPIR) constEval(e Expression) (int, bool) {
	if op := e.Operand(); op != nil {
		if lit := op.Literal(); lit != nil {
			if n := lit.Number(); n != nil {
				return n.Value, true
			}
		}
		if ref := op.Reference(); ref != nil {
			if lit, ok := g.constValues[ref.Identifier]; ok {
				if n := lit.Number(); n != nil {
					return n.Value, true
				}
			}
		}
	}
	return 0, false
}

// refSize returns the storage size of a reference expression.
func (g *GenPIR) refSize(e Expression) int {
	if op := e.Operand(); op != nil {
		if ref := op.Reference(); ref != nil {
			// For DATA references, return the actual byte count of the values.
			if g.checker != nil && g.currentScope != nil {
				if d, ok := g.currentScope.Lookup(ref.Identifier); ok {
					if d.DataValue != nil {
						if d.DataValue.Text != nil {
							return len(d.DataValue.Text.Value)
						}
						if d.DataValue.Tile != nil {
							return len(d.DataValue.Tile.Tiles) * 64
						}
						return len(d.DataValue.Values)
					}
					// DATA parameters are pointers (2 bytes), not single bytes.
					if d.ParamRef && d.Type.Predeclared() == PredeclaredData {
						return 2
					}
				}
			}
			t, ok := g.localType(ref.Identifier)
			if !ok {
				return 1
			}
			return t.Size()
		}
	}
	return 1
}

// genPIRExprOrRef evaluates an expression (usually a reference) and pushes its ADDRESS.
// For SAVE/LOAD we need the address of the data, not its value.
func (g *GenPIR) genPIRExprOrRef(e Expression) error {
	if op := e.Operand(); op != nil {
		if ref := op.Reference(); ref != nil {
			g.genPIRRefAddr(ref)
			return nil
		}
	}
	if s := e.Suffix(); s != nil && s.Operator == OperatorINDEX {
		return g.genPIRIndexAddr(s.Operands)
	}
	// For other expression types (arithmetic, cast, etc.), the value IS the address.
	return e.genPIRExpr(g)
}
