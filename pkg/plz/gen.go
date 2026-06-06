package plz

import (
	"fmt"
	"os"
	"strconv"
)

// HeapBase is the base address of the heap/RAM region at 0xC000 for the SMS.
const HeapBase = 0xC000 // RAM memory.

// Gen is the code generator that emits Z80 assembly text from a checked AST.
// It holds an output file, a scope stack for name resolution, and generation
// state such as the current procedure name and label counter.
//
// 8-bit code generation is handled inline: Infix.Gen, Prefix.Gen, and the
// byte-variable load/store paths in Operand.Gen and Let.Gen all check the
// types of their operands and emit 8-bit Z80 instructions (A register, cp,
// add a, etc.) when all values involved are BYTE-typed. Remaining
// opportunities: 8-bit MUL/DIV runtime helpers, 8-bit suffix (index/field)
// operations, and 8-bit CALL argument passing.
type Gen struct {
	file               *os.File
	Heap               int                       // Heap pointer to last allocated heap RAM memory.
	label              int                       // counter for unique local labels
	Checker            *Checker                  // Checker for semantic information
	InTask             bool                      // InTaks is set when generating inside a task body
	procName           string                    // current procedure name (empty = global scope)
	symStack           []map[Identifier]symEntry // scope stack for assembly label resolution
	currentScope       *Scope                    // current position in checker's persistent scope tree
	scopeStack         []*Scope                  // stack of checker scopes (parallel to symStack)
	scopeChildIdx      map[*Scope]int            // number of children consumed per parent scope
	ProcReturnType     Type                      // return type of current procedure
	ProcInterrupt      *Interrupt                // interrupt type of current procedure
	BoundCheck         bool                      // when true, emit runtime bounds checks before array accesses
	boundsErrorEmitted bool                      // tracks whether _plz_bounds_error label has been emitted
	strings            []strEntry                // string literal labels and content for ROM emission
	forTemps           []string                  // FOR loop temp variable labels (step/end), emitted in data section
}

// strEntry records a string literal that needs to be emitted as ROM data.
type strEntry struct {
	label string
	data  string
}

// symEntry records the assembly-level label for a local identifier.
// Type and paramRef are read from the checker's persistent scope tree
// via the Gen.currentScope pointer — not duplicated here.
type symEntry struct {
	label string
}

// NewGenFile creates a Gen that writes assembly output to a file with the given name.
func NewGenFile(name string) (*Gen, error) {
	fout, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	return NewGen(fout), nil
}

// NewGenTmp creates a Gen that writes assembly output to a temporary file.
func NewGenTmp() (*Gen, error) {
	fout, err := os.CreateTemp("", "plz_*.asm")
	if err != nil {
		return nil, err
	}
	return NewGen(fout), nil
}

// NewGen creates a Gen that writes assembly output to the given file.
func NewGen(fout *os.File) *Gen {
	return &Gen{file: fout, Heap: HeapBase}
}

// FileName returns the name of the output assembly file.
func (g *Gen) FileName() string {
	return g.file.Name()
}

// localSym resolves an identifier to its assembly-level symbol by walking the
// scope stack from innermost to outermost, falling back to the plain identifier.
func (g *Gen) localSym(id Identifier) Identifier {
	if g.procName == "" {
		return id
	}
	for i := len(g.symStack) - 1; i >= 0; i-- {
		if e, ok := g.symStack[i][id]; ok {
			return Identifier(e.label)
		}
	}
	return id
}

// localType resolves an identifier's type by walking the checker's
// persistent scope tree from the current position outward. This replaces
// the previous approach of storing type/paramRef copies on symEntry.
func (g *Gen) localType(id Identifier) (Type, bool) {
	for s := g.currentScope; s != nil; s = s.Parent {
		if d, ok := s.Symbols[id]; ok {
			return d.Type, true
		}
	}
	return Type{}, false
}

// isParamRef returns true when id is a record or data parameter passed
// by reference. It walks the checker's scope tree from current position
// outward, reading paramRef from the persistent Declare stored there.
func (g *Gen) isParamRef(id Identifier) bool {
	for s := g.currentScope; s != nil; s = s.Parent {
		if d, ok := s.Symbols[id]; ok {
			return d.ParamRef
		}
	}
	return false
}

// pushScope pushes a new empty scope onto the symbol stack and advances
// the checker scope pointer to the next child of the current scope (if any).
// Every call should be paired with a deferred popScope.
func (g *Gen) pushScope() {
	g.symStack = append(g.symStack, make(map[Identifier]symEntry))
	if g.currentScope != nil {
		idx := g.scopeChildIdx[g.currentScope]
		if idx < len(g.currentScope.Children) {
			g.scopeChildIdx[g.currentScope] = idx + 1
			g.scopeStack = append(g.scopeStack, g.currentScope)
			g.currentScope = g.currentScope.Children[idx]
		}
	}
}

// popScope removes the innermost scope from the symbol stack and restores
// the checker scope pointer to the enclosing scope.
func (g *Gen) popScope() {
	g.symStack = g.symStack[:len(g.symStack)-1]
	if len(g.scopeStack) > 0 {
		g.currentScope = g.scopeStack[len(g.scopeStack)-1]
		g.scopeStack = g.scopeStack[:len(g.scopeStack)-1]
	}
}

// Close closes the output assembly file.
func (g *Gen) Close() error {
	return g.file.Close()
}

// Emitf writes a formatted string to the output assembly file.
func (g *Gen) Emitf(form string, args ...any) (int, error) {
	return fmt.Fprintf(g.file, form, args...)
}

// Emitln writes arguments separated by spaces followed by a newline to the
// output assembly file.
func (g *Gen) Emitln(args ...any) (int, error) {
	return fmt.Fprintln(g.file, args...)
}

// Emit writes arguments without formatting to the output assembly file.
func (g *Gen) Emit(args ...any) (int, error) {
	return fmt.Fprint(g.file, args...)
}

// nextLabel returns a new unique integer suitable for constructing local label names.
func (g *Gen) nextLabel() int {
	g.label++
	return g.label
}

// emitReturn emits the correct return instruction based on the current
// procedure's interrupt type. Normal procedures emit ret, INTERRUPT
// procedures emit reti, and NMI procedures emit retn.
func (g *Gen) emitReturn() {
	if g.ProcInterrupt != nil && g.ProcInterrupt.NMI {
		g.Emitln("\tretn")
	} else if g.ProcInterrupt != nil {
		g.Emitln("\treti")
	} else {
		g.Emitln("\tret")
	}
}

// ProgramHeader is the assembly prologue emitted at the start of every program.
// It sets up the Z80 in interrupt mode 1, disables interrupts, and initializes
// the stack pointer to 0xDFF0.
const ProgramHeader = `org 0x0000
// Boot section
org 0x0000
    jp main         // Jump to main program

// Default interrupt handler placeholder (no handler installed)
org 0x0038
	ret

// NMI or pause button handler
org 0x0066
    retn

// Main program
main:
    di            // Disable interrupts
    im 1          // Interrupt mode 1
    ld sp, 0xdff0 // Set up stack pointer at end of RAM.
`

// ProgramFooter is the assembly epilogue that stops the program from
// running into the data section.
const ProgramFooter = `
_plz_all_done:
	di
	halt
	jp _plz_all_done

`

// RuntimeHeader is the assembly block containing PL/Z runtime helper routines
// for multiplication, division, modulus, and six comparison operators
// (EQ, NE, GT, LT, GTE, LTE). All helpers expect arguments in HL and DE and
// return the result in HL.
const RuntimeHeader = `
// -------------------------------------------------------------------
// PL/Z runtime helpers
// -------------------------------------------------------------------

// _plz_mul: HL = HL * DE (unsigned 16-bit)
_plz_mul:
	push bc
	push hl
	pop bc          // bc = multiplicand
	ld hl, 0        // hl = accumulator
	ld a, 16        // loop counter
_plz_mul_loop:
	push af
	ld a, c
	rra             // LSB of bc -> carry
	jr nc, _plz_mul_skip
	add hl, de
_plz_mul_skip:
	srl b
	rr c
	sla e
	rl d
	pop af
	dec a
	jr nz, _plz_mul_loop
	pop bc
	ret

// _plz_div: HL = HL / DE (unsigned 16-bit)
_plz_div:
	call _plz_divmod
	push bc
	pop hl
	ret

// _plz_mod: HL = HL % DE (unsigned 16-bit)
_plz_mod:
	call _plz_divmod
	ret

// Internal: divide HL by DE
// Output: BC = quotient, HL = remainder
// If DE == 0, returns quotient=1, remainder=0 (safe fallback on systems
// without runtime exception handling).
_plz_divmod:
	ld a, d
	or e
	jr nz, _plz_divmod_do
	ld bc, 1
	ld hl, 0
	ret
_plz_divmod_do:
	xor a
	push hl
	pop bc          // bc = dividend
	ld hl, 0        // hl = remainder
	ld a, 16        // 16 bits
_plz_div_loop:
	sla c
	rl b
	adc hl, hl
	push hl
	or a
	sbc hl, de
	jr c, _plz_div_skip
	inc c
	ex (sp), hl
_plz_div_skip:
	pop hl
	dec a
	jr nz, _plz_div_loop
	ret

// _plz_eq: HL = (HL == DE) ? 1 : 0
_plz_eq:
	or a
	sbc hl, de
	ld hl, 0
	ret nz
	inc l
	ret

// _plz_ne: HL = (HL != DE) ? 1 : 0
_plz_ne:
	or a
	sbc hl, de
	ld hl, 0
	ret z
	inc l
	ret

// _plz_gt: HL = (HL > DE) ? 1 : 0 (unsigned)
_plz_gt:
	or a
	sbc hl, de
	jr c, _plz_cmp_false
	jr z, _plz_cmp_false
	ld hl, 1
	ret

// _plz_lt: HL = (HL < DE) ? 1 : 0 (unsigned)
_plz_lt:
	or a
	sbc hl, de
	jr nc, _plz_cmp_false
	ld hl, 1
	ret

// _plz_gte: HL = (HL >= DE) ? 1 : 0 (unsigned)
_plz_gte:
	or a
	sbc hl, de
	jr c, _plz_cmp_false
	ld hl, 1
	ret

// _plz_lte: HL = (HL <= DE) ? 1 : 0 (unsigned)
_plz_lte:
	or a
	sbc hl, de
	jr nz, _plz_lte_gt
	ld hl, 1
	ret
_plz_lte_gt:
	jr nc, _plz_cmp_false
	ld hl, 1
	ret

_plz_cmp_false:
	ld hl, 0
	ret
`

// Gen generates the complete assembly output for a program. It runs the semantic
// checker, emits the program header, runtime helpers, main code body, procedure
// bodies, task bodies, the task scheduler, and global variable storage. Tasks
// and procedures are emitted after the main body so they are not reachable by
// fall-through.
func (p Program) Gen(g *Gen) error {
	c := NewChecker()
	if err := p.Check(c); err != nil {
		return err
	}
	g.Checker = c
	g.currentScope = c.root
	g.scopeChildIdx = make(map[*Scope]int)

	g.Emit(ProgramHeader)
	g.Emitln("\tjp _plz_start")
	g.Emit(RuntimeHeader)
	g.Emitln("_plz_start:")

	if len(c.TaskDefs) > 0 {
		g.genTaskInit(c.TaskDefs)
	}

	var dataItems []dataItem
	var procedures []Procedure
	var dataStmts []Data
	for _, statement := range p.Statements {
		switch cmd := statement.Command.(type) {
		case At:
			if cmd.HasBank {
				if err := cmd.Gen(g); err != nil {
					return err
				}
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
			if err := statement.Gen(g); err != nil {
				return err
			}
		}
	}

	g.Emit(ProgramFooter)

	for _, proc := range procedures {
		if err := proc.Gen(g); err != nil {
			return err
		}
	}
	for _, ds := range dataStmts {
		if err := ds.Gen(g); err != nil {
			return err
		}
	}

	taskDeclares := g.genTaskBodies(c.TaskDefs)

	g.genStringData()
	if len(c.TaskDefs) > 0 {
		g.genSchedulerRuntime(c.TaskDefs)
	}
	g.genProcStorage(p.Statements)
	g.genForTemps()

	for _, item := range dataItems {
		if item.at != nil {
			if err := item.at.Gen(g); err != nil {
				return err
			}
		} else {
			g.Emitln("")
			item.declare.Gen(g)
		}
	}
	for _, s := range taskDeclares {
		if err := s.Gen(g); err != nil {
			return err
		}
	}

	if g.BoundCheck {
		g.genOnceBoundsError()
	}
	return nil
}

type dataItem struct {
	at      *At
	declare *Declare
}

func (g *Gen) genTaskInit(tasks []Task) {
	g.Emitf("\tld hl, _plz_tcbs\n")
	g.Emitf("\tld de, _plz_tcbs+1\n")
	g.Emitf("\tld bc, %d\n", 128)
	g.Emitln("\tld (hl), 0")
	g.Emitln("\tldir")
	for i, task := range tasks {
		g.Emitf("\tld hl, _plz_task_%d\n", i)
		g.Emitf("\tld sp, _plz_task%d_stack+128\n", i)
		g.Emitln("\tpush hl")
		g.Emitf("\tld (_plz_tcbs+%d), sp\n", i*8)
		g.Emitf("\tld a, %d\n", task.Priority)
		g.Emitf("\tld (_plz_tcbs+%d), a\n", i*8+4)
		// Store stack base address (stack bottom) in TCB reserved bytes.
		g.Emitf("\tld hl, _plz_task%d_stack\n", i)
		g.Emitf("\tld (_plz_tcbs+%d), hl\n", i*8+5)
		// Write canary bytes at bottom of stack.
		g.Emitf("\tld a, 0xDE\n\tld (_plz_task%d_stack), a\n", i)
		g.Emitf("\tld a, 0xAD\n\tld (_plz_task%d_stack+1), a\n", i)
	}
	for i := len(tasks); i < 16; i++ {
		g.Emitf("\tld a, 3\n")
		g.Emitf("\tld (_plz_tcbs+%d), a\n", i*8+2)
	}
	g.Emitln("\txor a")
	g.Emitln("\tld (_plz_current_task), a")
	g.Emitln("\tld sp, (_plz_tcbs+0)")
	g.Emitln("\tret")
}

func (g *Gen) genTaskBodies(tasks []Task) []Statement {
	var taskDeclares []Statement
	for i := range tasks {
		t := tasks[i]
		g.Emitf("_plz_task_%d:\n", i)
		g.InTask = true
		for j := range t.Body {
			if _, ok := t.Body[j].Command.(Declare); ok {
				taskDeclares = append(taskDeclares, t.Body[j])
				continue
			}
			if err := t.Body[j].Gen(g); err != nil {
				panic(err) // or return error — kept for simplicity
			}
		}
		g.InTask = false
		g.Emitln("\tjp _plz_task_done")
	}
	return taskDeclares
}

func (g *Gen) genStringData() {
	for _, s := range g.strings {
		g.Emitf("%s: db %d", s.label, len(s.data))
		for _, c := range s.data {
			g.Emitf(", %d", byte(c))
		}
		g.Emit("\n")
	}
}

func (g *Gen) genSchedulerRuntime(tasks []Task) {
	g.Emit(SchedulerCode)
	g.Emitf("org 0x%x\n", g.Heap)
	g.Emitf("_plz_current_task: db 0\n")
	g.Emitf("_plz_sch_sp: dw 0\n")
	g.Emitf("_plz_tcbs: ds 128\n")
	for i := range tasks {
		g.Emitf("_plz_task%d_stack: ds 128\n", i)
		g.Heap += 128
	}
	g.Heap += 131
}

func (g *Gen) genProcStorage(stmts []Statement) {
	for _, stmt := range stmts {
		proc, ok := stmt.Command.(Procedure)
		if !ok || proc.Reentrant {
			continue
		}
		for i, param := range proc.Parameters {
			psize := proc.paramByteSize(i)
			g.emitStorage("_plz_%s_%s", psize, proc.Name.Name, param)
		}
		g.emitProcLocals(proc.Statements, proc.Name.Name, 0)
	}
}

func (g *Gen) emitProcLocals(stmts []Statement, procName string, depth int) {
	for _, s := range stmts {
		switch cmd := s.Command.(type) {
		case Declare:
			var label string
			if depth == 0 {
				label = fmt.Sprintf("_plz_%s_%s", procName, cmd.Identifier)
			} else {
				label = fmt.Sprintf("_plz_%s_%d_%s", procName, depth, cmd.Identifier)
			}
			g.emitStorageRaw(label, cmd.StorageSize())
		case Group:
			g.emitProcLocals(cmd.Statements, procName, depth+1)
		}
	}
}

// Gen generates assembly code for a statement, first emitting its label if
// present, then dispatching to the Gen method of the contained command.
func (s Statement) Gen(g *Gen) error {
	if s.Label != nil {
		s.Label.Gen(g)
	}
	genner, ok := (s.Command.(interface{ Gen(*Gen) error }))
	if ok {
		if err := genner.Gen(g); err != nil {
			return err
		}
	} else {
		g.Emitf("// statement not implemented: %v\n", s)
	}

	return nil
}

// Gen emits assembly for a named label, emitting it as an assembly label
// (e.g. "loop:") for use as a GOTO target.
func (l Label) Gen(g *Gen) error {
	if l.Name != "" {
		g.Emitf("%s:", l.Name)
	}
	return nil
}

// genCondBranch evaluates an expression and emits a conditional jump to
// falseLabel when the expression is false (zero). For comparison infix
// expressions (==, !=, <, >, <=, >=) it emits optimized inline code that
// uses the Z80 flags directly instead of calling a runtime helper and then
// testing HL for zero.
func (g *Gen) genCondBranch(e Expression, falseLabel string) error {
	inf := e.Infix()
	if inf != nil {
		switch inf.Operator {
		case OperatorEQU, OperatorNEQ, OperatorGT, OperatorLT, OperatorGTE, OperatorLTE:
			if err := inf.Operands[0].Gen(g); err != nil {
				return err
			}
			g.Emitln("\tpush hl")
			if err := inf.Operands[1].Gen(g); err != nil {
				return err
			}
			g.Emitln("\tex de, hl")
			g.Emitln("\tpop hl")

			g.Emitln("\tor a")
			g.Emitln("\tsbc hl, de")

			switch inf.Operator {
			case OperatorEQU:
				g.Emitf("\tjmp nz, %s\n", falseLabel)
			case OperatorNEQ:
				g.Emitf("\tjmp z, %s\n", falseLabel)
			case OperatorGT:
				g.Emitf("\tjmp c, %s\n", falseLabel)
				g.Emitf("\tjmp z, %s\n", falseLabel)
			case OperatorLT:
				g.Emitf("\tjmp nc, %s\n", falseLabel)
			case OperatorGTE:
				g.Emitf("\tjmp c, %s\n", falseLabel)
			case OperatorLTE:
				g.Emitf("\tjmp z, _lte_%d\n", g.nextLabel())
				g.Emitf("\tjmp nc, %s\n", falseLabel)
				g.Emitf("_lte_%d:\n", g.nextLabel()-1)
			}
			return nil
		}
	}
	if err := e.Gen(g); err != nil {
		return err
	}
	g.Emitln("\tld a, h")
	g.Emitln("\tor l")
	g.Emitf("\tjmp z, %s\n", falseLabel)
	return nil
}

// Gen generates assembly for an IF statement. It evaluates the condition, jumps
// to the else branch if false, emits the then-body, optionally emits the else-
// body, then jumps past the else to end.
func (s If) Gen(g *Gen) error {
	n := g.nextLabel()
	elseLabel := fmt.Sprintf("_else_%d", n)
	if err := g.genCondBranch(s.Condition, elseLabel); err != nil {
		return err
	}
	if err := s.Then.Gen(g); err != nil {
		return err
	}
	if s.Else != nil {
		g.Emitf("\tjmp _end_%d\n", n)
	}
	g.Emitf("_else_%d:\n", n)
	if s.Else != nil {
		if err := s.Else.Gen(g); err != nil {
			return err
		}
	}
	if s.Else != nil {
		g.Emitf("_end_%d:\n", n)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Expression code generator
// ---------------------------------------------------------------------------

// Gen evaluates the expression and leaves the result in HL. It dispatches to
// the appropriate operand, prefix, infix, or suffix generator.
func (e Expression) Gen(g *Gen) error {
	switch {
	case e.Operand() != nil:
		return e.Operand().Gen(g)
	case e.Prefix() != nil:
		return e.Prefix().Gen(g)
	case e.Infix() != nil:
		return e.Infix().Gen(g)
	case e.Suffix() != nil:
		return e.Suffix().Gen(g)
	}
	return nil
}

// Gen generates assembly code for an operand. Numeric literals load the value
// into HL. String literals emit a TEXT record (length-prefixed byte array) in
// ROM and load HL with its address (an HL pointing to a TEXT-compatible
// struct { byte length; byte text[] }). The TEXT type alias (record with
// length and text fields) allows field access via .length and .text[i].
// Assignment of a string literal to a TEXT variable is not yet supported;
// string literals are ROM-based and can only be used as expression operands
// (passed to procedures, compared, etc.).
func (o Operand) Gen(g *Gen) error {
	switch {
	case o.Literal() != nil:
		if n := o.Literal().Number(); n != nil {
			g.Emitf("\tld hl, %d\n", n.Value)
		} else if t := o.Literal().Text(); t != nil {
			n := g.nextLabel()
			label := fmt.Sprintf("_plz_str_%d", n)
			g.strings = append(g.strings, strEntry{label: label, data: t.Value})
			g.Emitf("\tld hl, %s\n", label)
		}
	case o.Reference() != nil:
		if g.Checker != nil {
			if lit, ok := g.Checker.Constants[o.Reference().Identifier]; ok {
				if n := lit.Number(); n != nil {
					g.Emitf("\tld hl, %d\n", n.Value)
					break
				}
			}
		}
		if o.Reference().isByteRef(g) {
			g.Emitf("\tld a, (%s)\n", g.localSym(o.Reference().Identifier))
			g.Emitln("\tld l, a")
			g.Emitln("\tld h, 0")
		} else {
			g.Emitf("\tld hl, (%s)\n", g.localSym(o.Reference().Identifier))
		}
	case o.Expr() != nil:
		return o.Expr().Gen(g)
	case o.Input() != nil:
		if err := o.Input().Port.Gen(g); err != nil {
			return err
		}
		g.Emitln("\tld c, l")
		g.Emitln("\tin a, (c)")
		g.Emitln("\tld l, a")
		g.Emitln("\tld h, 0")
	case o.Length() != nil:
		n, err := g.Checker.evalLength(o.Length())
		if err != nil {
			return err
		}
		g.Emitf("\tld hl, %d\n", n)
	}
	return nil
}

// isByteRef reports whether the reference's resolved type is BYTE (including
// struct fields resolved to BYTE), so the generator knows to load a single byte
// rather than a 16-bit word.
func (r *Reference) isByteRef(g *Gen) bool {
	if g.Checker == nil || r == nil {
		return false
	}
	if len(r.Fields) > 0 {
		// Field access — check the field's type.
		t, ok := g.localType(r.Identifier)
		if !ok || t.Record() == nil {
			return false
		}
		fname := r.Fields[0]
		for _, f := range t.Record().Fields {
			if f.Identifier == fname {
				return f.Type.Predeclared() == PredeclaredByte
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

// isByteOperand reports whether evaluating the operand produces a BYTE value
// (fits in 0–255). This is used to select 8-bit code generation paths.
// Literals are treated conservatively (non-BYTE) since their type depends on
// the expression context — only an explicit BYTE() cast or a BYTE-typed
// reference guarantees a byte-wide result.
func (op Operand) isByteOperand(g *Gen) bool {
	switch {
	case op.Literal() != nil:
		return false // literal type depends on context; don't guess
	case op.Reference() != nil:
		return op.Reference().isByteRef(g)
	case op.Expr() != nil:
		return op.Expr().isByteExpression(g)
	case op.Input() != nil:
		return true // INPUT returns a byte
	case op.Length() != nil:
		return false // LENGTH is WORD
	}
	return false
}

// isByteExpression reports whether the expression produces a BYTE value.
func (e Expression) isByteExpression(g *Gen) bool {
	switch {
	case e.Operand() != nil:
		return e.Operand().isByteOperand(g)
	case e.Prefix() != nil:
		p := e.Prefix()
		if p.Operator == Operator(KeywordByte) {
			return true // BYTE(expr) always produces a byte
		}
		if p.Operator == Operator(KeywordWord) {
			return false // WORD(expr) always produces a word
		}
		if p.Operator == OperatorNOT {
			return true // !expr always produces 0 or 1
		}
		return p.Operand.isByteOperand(g)
	case e.Infix() != nil:
		inf := e.Infix()
		switch inf.Operator {
		case OperatorShiftLeft, OperatorShiftRight:
			return false
		}
		return inf.Operands[0].isByteOperand(g) && inf.Operands[1].isByteOperand(g)
	case e.Suffix() != nil:
		return false
	}
	return false
}

// isByteInfix reports whether both operands of an infix expression are
// BYTE-typed, enabling 8-bit arithmetic/logic code generation.
func (i Infix) isByteInfix(g *Gen) bool {
	return i.Operands[0].isByteOperand(g) && i.Operands[1].isByteOperand(g)
}

// Gen generates assembly for a prefix expression. OperatorNEG computes two's
// complement negation of the operand into HL. OperatorNOT computes logical
// negation (0 → 1, non-zero → 0). When the operand is BYTE-typed, 8-bit
// instructions are used for smaller/faster code.
func (p Prefix) Gen(g *Gen) error {
	switch p.Operator {
	case OperatorNEG:
		if err := p.Operand.Gen(g); err != nil {
			return err
		}
		g.Emitln("\tex de, hl")
		g.Emitln("\tld hl, 0")
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")

	case OperatorNOT:
		if err := p.Operand.Gen(g); err != nil {
			return err
		}
		if p.Operand.isByteOperand(g) {
			n := g.nextLabel()
			g.Emitln("\tld a, l")
			g.Emitf("\tld hl, 0\n")
			g.Emitln("\tor a")
			g.Emitf("\tjr nz, _lbl_%d\n", n)
			g.Emitln("\tinc l")
			g.Emitf("_lbl_%d:\n", n)
		} else {
			n := g.nextLabel()
			g.Emitln("\tld a, h")
			g.Emitln("\tor l")
			g.Emitf("\tld hl, 0\n")
			g.Emitf("\tjr nz, _lbl_%d\n", n)
			g.Emitln("\tinc l")
			g.Emitf("_lbl_%d:\n", n)
		}

	case Operator(KeywordByte):
		if err := p.Operand.Gen(g); err != nil {
			return err
		}
		g.Emitln("\tld h, 0")

	case Operator(KeywordWord):
		if err := p.Operand.Gen(g); err != nil {
			return err
		}
	}
	return nil
}

// constShift returns the constant shift amount if the operand is a numeric
// literal (possibly wrapped in Expression/Operand chains as produced by the
// parser), or -1 if it is not a compile-time constant.
func constShift(op Operand) int {
	if lit := op.Literal(); lit != nil {
		if n := lit.Number(); n != nil {
			return n.Value
		}
	}
	if expr := op.Expr(); expr != nil {
		if op2 := expr.Operand(); op2 != nil {
			return constShift(*op2)
		}
	}
	return -1
}

// isSimpleOperand checks if evaluating the operand does not clobber DE.
// Simple operands are: literals, variable references, parenthesized simple
// expressions, and INPUT with a simple port expression.
// TODO: this should be a method of Operand, not a function
func isSimpleOperand(op Operand) bool {
	switch {
	case op.Literal() != nil:
		return true
	case op.Reference() != nil:
		r := op.Reference()
		return len(r.Subscripts) == 0 && len(r.Fields) == 0
	case op.Expr() != nil:
		return isSimpleExpr(*op.Expr())
	case op.Input() != nil:
		return isSimpleExpr(op.Input().Port)
	case op.Length() != nil:
		return true // constant, no register clobber
	}
	return false
}

// isSimpleExpr checks if evaluating the expression does not clobber DE.
// Simple expressions are: simple operands and NOT-of-simple-operand.
// Infix, suffix, and NEG expressions may clobber DE and are not simple.
// TODO: this should be a method of Expression, not a function
func isSimpleExpr(e Expression) bool {
	switch {
	case e.Operand() != nil:
		return isSimpleOperand(*e.Operand())
	case e.Prefix() != nil:
		p := e.Prefix()
		switch p.Operator {
		case OperatorNOT:
			return isSimpleOperand(p.Operand)
		case OperatorNEG:
			return false
		}
	}
	return false
}

// Gen generates assembly for an infix expression. It evaluates the left operand
// into HL, saves it, evaluates the right operand into HL, moves it to DE, then
// restores the left operand into HL and emits the operation-specific code.
//
// When the right operand is simple (literal, variable, etc.) the left operand
// is saved in DE via ex de,hl, avoiding a push/pop pair. When the right operand
// is complex (infix, suffix, NEG) the left operand is saved on the stack.
func (i Infix) Gen(g *Gen) error {
	// Left operand
	if err := i.Operands[0].Gen(g); err != nil {
		return err
	}
	g.Emitln("\tpush hl")
	// Right operand
	if err := i.Operands[1].Gen(g); err != nil {
		return err
	}
	g.Emitln("\tex de, hl")
	g.Emitln("\tpop hl")
	switch i.Operator {
	case OperatorADD:
		g.genInfixAdd()
	case OperatorSUB:
		g.genInfixSub()
	case OperatorShiftLeft:
		g.genInfixShiftLeft(i.Operands[1])
	case OperatorShiftRight:
		g.genInfixShiftRight(i.Operands[1])
	case OperatorAND:
		if i.isByteInfix(g) {
			g.genInfixBitwise8("\tand e")
		} else {
			g.genInfixBitwise("\tand e", "\tand d")
		}
	case OperatorOR:
		if i.isByteInfix(g) {
			g.genInfixBitwise8("\tor e")
		} else {
			g.genInfixBitwise("\tor e", "\tor d")
		}
	case OperatorXOR:
		if i.isByteInfix(g) {
			g.genInfixBitwise8("\txor e")
		} else {
			g.genInfixBitwise("\txor e", "\txor d")
		}
	case OperatorMUL:
		g.Emitln("\tcall _plz_mul")
	case OperatorDIV:
		g.Emitln("\tcall _plz_div")
	case OperatorMOD:
		g.Emitln("\tcall _plz_mod")
	case OperatorEQU:
		if i.isByteInfix(g) {
			g.genInfixCmp8(cmpEQ)
		} else {
			g.genInfixCmp(cmpEQ)
		}
	case OperatorNEQ:
		if i.isByteInfix(g) {
			g.genInfixCmp8(cmpNEQ)
		} else {
			g.genInfixCmp(cmpNEQ)
		}
	case OperatorGT:
		if i.isByteInfix(g) {
			g.genInfixCmp8(cmpGT)
		} else {
			g.genInfixCmp(cmpGT)
		}
	case OperatorLT:
		if i.isByteInfix(g) {
			g.genInfixCmp8(cmpLT)
		} else {
			g.genInfixCmp(cmpLT)
		}
	case OperatorGTE:
		if i.isByteInfix(g) {
			g.genInfixCmp8(cmpGTE)
		} else {
			g.genInfixCmp(cmpGTE)
		}
	case OperatorLTE:
		if i.isByteInfix(g) {
			g.genInfixCmp8(cmpLTE)
		} else {
			g.genInfixCmp(cmpLTE)
		}
	}
	return nil
}

type cmpKind int

const (
	cmpEQ cmpKind = iota
	cmpNEQ
	cmpGT
	cmpLT
	cmpGTE
	cmpLTE
)

func (g *Gen) genInfixAdd() {
	g.Emitln("\tadd hl, de")
}

func (g *Gen) genInfixSub() {
	g.Emitln("\tor a")
	g.Emitln("\tsbc hl, de")
}

func (g *Gen) genInfixShiftLeft(rhs Operand) {
	if n := constShift(rhs); n >= 0 {
		for j := 0; j < n; j++ {
			g.Emitln("\tadd hl, hl")
		}
	} else {
		g.genVarShift("add hl, hl", rhs)
	}
}

func (g *Gen) genInfixShiftRight(rhs Operand) {
	if n := constShift(rhs); n >= 0 {
		for j := 0; j < n; j++ {
			g.Emitln("\tsrl h")
			g.Emitln("\trr l")
		}
	} else {
		g.genVarShift("srl h\n\trr l", rhs)
	}
}

func (g *Gen) genVarShift(shiftOp string, rhs Operand) {
	loop := g.nextLabel()
	end := g.nextLabel()
	g.Emitln("\tld a, e")
	g.Emitf("_lbl_%d:\n", loop)
	g.Emitln("\tor a")
	g.Emitf("\tjr z, _lbl_%d\n", end)
	g.Emit(shiftOp)
	g.Emit("\n")
	g.Emitln("\tdec a")
	g.Emitf("\tjr _lbl_%d\n", loop)
	g.Emitf("_lbl_%d:\n", end)
}

func (g *Gen) genInfixBitwise(byteOp, wordHighOp string) {
	g.Emitln("\tld a, h")
	g.Emit(wordHighOp)
	g.Emit("\n")
	g.Emitln("\tld h, a")
	g.Emitln("\tld a, l")
	g.Emit(byteOp)
	g.Emit("\n")
	g.Emitln("\tld l, a")
}

func (g *Gen) genInfixBitwise8(byteOp string) {
	g.Emitln("\tld a, l")
	g.Emit(byteOp)
	g.Emit("\n")
	g.Emitln("\tld l, a")
	g.Emitln("\tld h, 0")
}

func (g *Gen) genInfixCmp(kind cmpKind) {
	n := g.nextLabel()
	g.genWordCmp(kind, n)
}

func (g *Gen) genInfixCmp8(kind cmpKind) {
	n := g.nextLabel()
	g.genByteCmp(kind, n)
}

func (g *Gen) genByteCmp(kind cmpKind, n int) {
	switch kind {
	case cmpEQ:
		g.Emitln("\tld a, l")
		g.Emitln("\tcp e")
		g.Emitln("\tld hl, 0")
		g.Emitf("\tjr nz, _cmp_%d\n", n)
		g.Emitln("\tinc l")
		g.Emitf("_cmp_%d:\n", n)
	case cmpNEQ:
		g.Emitln("\tld a, l")
		g.Emitln("\tcp e")
		g.Emitln("\tld hl, 0")
		g.Emitf("\tjr z, _cmp_%d\n", n)
		g.Emitln("\tinc l")
		g.Emitf("_cmp_%d:\n", n)
	case cmpGT:
		g.Emitln("\tld a, e")
		g.Emitln("\tcp l")
		g.Emitf("\tjr nc, _cmp_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("\tjr _cmpd_%d\n", n)
		g.Emitf("_cmp_%d:\n", n)
		g.Emitln("\tld hl, 0")
		g.Emitf("_cmpd_%d:\n", n)
	case cmpLT:
		g.Emitln("\tld a, l")
		g.Emitln("\tcp e")
		g.Emitf("\tjr nc, _cmp_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("\tjr _cmpd_%d\n", n)
		g.Emitf("_cmp_%d:\n", n)
		g.Emitln("\tld hl, 0")
		g.Emitf("_cmpd_%d:\n", n)
	case cmpGTE:
		g.Emitln("\tld a, l")
		g.Emitln("\tcp e")
		g.Emitf("\tjr c, _cmp_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("\tjr _cmpd_%d\n", n)
		g.Emitf("_cmp_%d:\n", n)
		g.Emitln("\tld hl, 0")
		g.Emitf("_cmpd_%d:\n", n)
	case cmpLTE:
		g.Emitln("\tld a, l")
		g.Emitln("\tcp e")
		g.Emitln("\tld hl, 1")
		g.Emitf("\tjr z, _cmp_%d\n", n)
		g.Emitf("\tjr c, _cmp_%d\n", n)
		g.Emitln("\tdec l")
		g.Emitf("_cmp_%d:\n", n)
	}
}

func (g *Gen) genWordCmp(kind cmpKind, n int) {
	g.Emitln("\tor a")
	g.Emitln("\tsbc hl, de")
	switch kind {
	case cmpEQ:
		g.Emitln("\tld hl, 0")
		g.Emitf("\tjr nz, _cmp_%d\n", n)
		g.Emitln("\tinc l")
		g.Emitf("_cmp_%d:\n", n)
	case cmpNEQ:
		g.Emitln("\tld hl, 0")
		g.Emitf("\tjr z, _cmp_%d\n", n)
		g.Emitln("\tinc l")
		g.Emitf("_cmp_%d:\n", n)
	case cmpGT:
		g.Emitf("\tjr c, _cmp_%d\n", n)
		g.Emitf("\tjr z, _cmp_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("\tjr _cmpd_%d\n", n)
		g.Emitf("_cmp_%d:\n", n)
		g.Emitln("\tld hl, 0")
		g.Emitf("_cmpd_%d:\n", n)
	case cmpLT:
		g.Emitf("\tjr nc, _cmp_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("\tjr _cmpd_%d\n", n)
		g.Emitf("_cmp_%d:\n", n)
		g.Emitln("\tld hl, 0")
		g.Emitf("_cmpd_%d:\n", n)
	case cmpGTE:
		g.Emitf("\tjr c, _cmp_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("\tjr _cmpd_%d\n", n)
		g.Emitf("_cmp_%d:\n", n)
		g.Emitln("\tld hl, 0")
		g.Emitf("_cmpd_%d:\n", n)
	case cmpLTE:
		g.Emitln("\tld hl, 1")
		g.Emitf("\tjr z, _cmp_%d\n", n)
		g.Emitf("\tjr c, _cmp_%d\n", n)
		g.Emitln("\tdec l")
		g.Emitf("_cmp_%d:\n", n)
	}
}

// Gen generates assembly for a suffix expression (index, function call, or field access).
func (s *Suffix) Gen(g *Gen) error {
	switch s.Operator {
	case OperatorINDEX:
		return g.genIndexRead(s.Operands)
	case OperatorCALL:
		return g.genCallExpr(s.Operands)
	case OperatorFIELD:
		return g.genFieldRead(s.Operands)
	}
	return nil
}

// genFieldAddr computes the address of a struct field into HL and returns the
// field's type. Operands are [struct_ref_or_expr, field_ref]. It handles both
// simple field access (s.f) and indexed access on an array of records (arr[i].f).
func (g *Gen) genFieldAddr(operands []Operand) (fieldType Type, err error) {
	ref := operands[0].Ref()
	var baseSuffix *Suffix
	if ref == nil {
		if expr := operands[0].Expr(); expr != nil {
			baseSuffix = expr.Suffix()
			if baseSuffix != nil && len(baseSuffix.Operands) > 0 {
				ref = baseSuffix.Operands[0].Ref()
			} else {
				ref = expr.Ref()
			}
		}
	}
	if ref == nil {
		return fieldType, fmt.Errorf("genFieldAddr: first operand must have a reference")
	}
	t, ok := g.localType(ref.Identifier)
	if !ok {
		return fieldType, fmt.Errorf("genFieldAddr: unknown identifier %s", ref.Identifier)
	}
	if arr := t.Array(); arr != nil {
		t = arr.ElemType
	}
	if t.Record() == nil {
		return fieldType, fmt.Errorf("genFieldAddr: %s is not a struct", ref.Identifier)
	}
	fname := operands[1].Reference().Identifier
	fieldIdx := -1
	for i, f := range t.Record().Fields {
		if f.Identifier == fname {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 0 {
		return fieldType, fmt.Errorf("genFieldAddr: struct %s has no field %s", ref.Identifier, fname)
	}
	off := t.Record().FieldOffset(fieldIdx)
	ft := t.Record().Fields[fieldIdx].Type

	if baseSuffix != nil {
		// The base is an INDEX suffix (e.g., arr[i] for array of records).
		// Compute element address, then add field offset.
		if baseSuffix.Operator != OperatorINDEX {
			return fieldType, fmt.Errorf("genFieldAddr: unsupported base expression")
		}
		if _, err := g.genIndexAddr(baseSuffix.Operands); err != nil {
			return fieldType, err
		}
	} else {
		// Simple reference: load base address.
		if g.isParamRef(ref.Identifier) {
			g.Emitf("\tld hl, (%s)\n", g.localSym(ref.Identifier))
		} else {
			g.Emitf("\tld hl, %s\n", g.localSym(ref.Identifier))
		}
	}
	if off > 0 {
		g.Emitf("\tld de, %d\n", off)
		g.Emitln("\tadd hl, de")
	}
	return ft, nil
}

// genFieldRead generates code to read a struct field into HL.
// Operands are [struct_ref, field_ref].
func (g *Gen) genFieldRead(operands []Operand) error {
	ft, err := g.genFieldAddr(operands)
	if err != nil {
		return err
	}
	if ft.Predeclared() == PredeclaredByte {
		g.Emitln("\tld a, (hl)")
		g.Emitln("\tld l, a")
		g.Emitln("\tld h, 0")
	} else {
		g.Emitln("\tld a, (hl)")
		g.Emitln("\tinc hl")
		g.Emitln("\tld h, (hl)")
		g.Emitln("\tld l, a")
	}
	return nil
}

// elemSize returns the storage size in bytes for elements declared under the
// given identifier. Record types round up to the next power of two; BYTE and
// DATA are 1; everything else defaults to 2.
func (g *Gen) elemSize(name Identifier) int {
	t, ok := g.localType(name)
	if !ok {
		return 2
	}
	if arr := t.Array(); arr != nil {
		t = arr.ElemType
	}
	if t.Record() != nil {
		total := t.Record().TotalSize()
		return nextPow2(total)
	}
	if t.Predeclared() == PredeclaredByte || t.Predeclared() == PredeclaredData {
		return 1
	}
	return 2
}

// genIndexAddr computes the address of arr[index] into HL without loading the
// value. It handles simple array references, field-of-array references (rec.arr[i]),
// and array-of-record references (arr[i].f). It returns the element size in bytes.
//
// When g.BoundCheck is true and the array size is known, it emits a runtime
// bounds check that halts the CPU if the index is out of range.
func (g *Gen) genIndexAddr(operands []Operand) (elemSize int, err error) {
	ref, baseSuffix := g.resolveIndexBase(operands)
	if ref == nil {
		return 0, fmt.Errorf("genIndexAddr: first operand must be a reference or field expression")
	}

	elem, arrSize, err := g.indexElemAndBounds(ref, baseSuffix)
	if err != nil {
		return 0, err
	}
	g.emitIndexBaseAddr(ref, baseSuffix)

	if len(operands) >= 2 {
		g.Emitln("\tpush hl")
		if err := operands[1].Expr().Gen(g); err != nil {
			return 0, err
		}
		if g.BoundCheck && arrSize > 0 {
			g.Emitf("\tld de, %d\n", arrSize)
			g.Emitln("\tor a")
			g.Emitln("\tsbc hl, de")
			g.Emitln("\tjr nc, _plz_bounds_error")
			g.Emitln("\tadd hl, de")
		}
		for size := elem; size > 1; size >>= 1 {
			g.Emitln("\tadd hl, hl")
		}
		g.Emitln("\tex de, hl")
		g.Emitln("\tpop hl")
		g.Emitln("\tadd hl, de")
	}
	return elem, nil
}

// resolveIndexBase extracts the underlying Reference and optional field Suffix
// from the first operand of an index expression.
func (g *Gen) resolveIndexBase(operands []Operand) (*Reference, *Suffix) {
	ref := operands[0].Ref()
	if ref != nil {
		return ref, nil
	}
	if expr := operands[0].Expr(); expr != nil {
		if s := expr.Suffix(); s != nil && len(s.Operands) > 0 {
			return s.Operands[0].Ref(), s
		}
		return expr.Ref(), nil
	}
	return nil, nil
}

// indexElemAndBounds determines the element size (in bytes) and the array size
// (for bounds checking) for the reference or field-suffix at the base of an
// index expression.
func (g *Gen) indexElemAndBounds(ref *Reference, baseSuffix *Suffix) (elem int, arrSize int, err error) {
	elem = 2
	if baseSuffix != nil && baseSuffix.Operator == OperatorFIELD {
		ft, err := g.genFieldAddr(baseSuffix.Operands)
		if err != nil {
			return 2, 0, err
		}
		if ft.Array() != nil {
			if g.BoundCheck {
				arrSize = ft.Array().Size
			}
			if ft.Array().ElemType.Predeclared() == PredeclaredByte {
				elem = 1
			} else if ft.Array().ElemType.Record() != nil {
				elem = ft.Array().ElemType.Record().TotalSize()
				elem = nextPow2(elem)
			}
		}
		return elem, arrSize, nil
	}

	elem = g.elemSize(ref.Identifier)
	if g.BoundCheck {
		if t, ok := g.localType(ref.Identifier); ok {
			if arr := t.Array(); arr != nil {
				arrSize = arr.Size
			}
		} else if data, ok := g.Checker.Datas[ref.Identifier]; ok {
			if data.Tile != nil {
				arrSize = len(data.Tile.Tiles)
			} else if data.Text != nil {
				arrSize = len(data.Text.Value) + 1
			} else {
				arrSize = len(data.Values)
			}
		}
	}
	if _, ok := g.Checker.Datas[ref.Identifier]; ok {
		elem = 1
	}
	return elem, arrSize, nil
}

// emitIndexBaseAddr emits a load of the base address for an index expression.
// For field suffixes (rec.arr[i]), genFieldAddr already emitted the address,
// so this is a no-op.
func (g *Gen) emitIndexBaseAddr(ref *Reference, baseSuffix *Suffix) {
	if baseSuffix != nil && baseSuffix.Operator == OperatorFIELD {
		return
	}
	if g.isParamRef(ref.Identifier) {
		g.Emitf("\tld hl, (%s)\n", g.localSym(ref.Identifier))
	} else {
		g.Emitf("\tld hl, %s\n", g.localSym(ref.Identifier))
	}
}

// genIndexRead generates code to read arr[index] into HL. Operands are
// [array_expr, index_expr].
func (g *Gen) genIndexRead(operands []Operand) error {
	elem, err := g.genIndexAddr(operands)
	if err != nil {
		return err
	}
	if elem == 1 {
		g.Emitln("\tld a, (hl)")
		g.Emitln("\tld l, a")
		g.Emitln("\tld h, 0")
	} else {
		g.Emitln("\tld a, (hl)")
		g.Emitln("\tinc hl")
		g.Emitln("\tld h, (hl)")
		g.Emitln("\tld l, a")
	}
	return nil
}

// genCallExpr generates code to call a function identified by the first operand.
// Arguments are passed in HL and DE (first two), with additional arguments either
// stored in procedure-specific RAM labels (non-REENTRANT) or pushed on the stack
// (REENTRANT). Record and DATA arguments pass the address rather than the value.
func (g *Gen) genCallExpr(operands []Operand) error {
	ref := operands[0].Ref()
	if ref == nil {
		return fmt.Errorf("genCallExpr: indirect calls not yet supported")
	}
	name := string(ref.Identifier)
	args := operands[1:]
	proc, ok := g.Checker.Procedures[name]

	genCallArg := func(i int) error {
		e := args[i].Expr()
		if e == nil {
			return fmt.Errorf("cannot evaluate argument %d", i)
		}
		return g.genCallArg(*e, proc, ok, i)
	}

	switch len(args) {
	case 0:
	case 1:
		if err := genCallArg(0); err != nil {
			return err
		}
	case 2:
		if err := genCallArg(1); err != nil {
			return err
		}
		g.Emitln("\tpush hl")
		if err := genCallArg(0); err != nil {
			return err
		}
		g.Emitln("\tpop de")
	default:
		totalExtra := g.emitExtraCallArgs(name, proc, ok, genCallArg, len(args))
		if err := genCallArg(1); err != nil {
			return err
		}
		g.Emitln("\tpush hl")
		if err := genCallArg(0); err != nil {
			return err
		}
		g.Emitln("\tpop de")
		isReentrant := !ok || proc.Reentrant
		g.Emitf("\tcall _plz_%s\n", name)
		if isReentrant && totalExtra > 0 {
			g.Emitf("\tld hl, %d\n", totalExtra)
			g.Emitln("\tadd hl, sp")
			g.Emitln("\tld sp, hl")
		}
		return nil
	}
	g.Emitf("\tcall _plz_%s\n", name)
	return nil
}

// ---------------------------------------------------------------------------
// Let (assignment)
// ---------------------------------------------------------------------------

// Gen generates assembly for a LET assignment. It evaluates the RHS, pushes it,
// computes the target address (handling subscripts, field access, and their
// combinations), then stores the value. Byte targets store only the low byte;
// word targets store both bytes. When Target2 is set (multi-return CALL), DE
// holds the second return value and is stored into Target2 after the first.
func (s Let) Gen(g *Gen) error {
	if err := s.Expression.Gen(g); err != nil {
		return err
	}
	t, _ := g.localType(s.Identifier)
	if arr := t.Array(); arr != nil {
		t = arr.ElemType
	}

	// Simple store (no subscripts, no fields).
	if len(s.Subscripts) == 0 && len(s.Fields) == 0 {
		return g.genLetSimple(s)
	}

	// Complex stores: save RHS value(s) and compute target address.
	if s.Target2 != nil {
		g.Emitln("\tpush de")
	}
	g.Emitln("\tpush hl")

	if len(s.Subscripts) > 0 && len(s.Fields) > 0 {
		return g.genLetArrField(s, t)
	}
	if len(s.Fields) > 0 {
		return g.genLetField(s, t)
	}
	return g.genLetArray(s)
}

func (g *Gen) genLetSimple(s Let) error {
	if s.isByteRef(g) {
		g.Emitln("\tld a, l")
		g.Emitf("\tld (%s), a\n", g.localSym(s.Identifier))
	} else {
		g.Emitf("\tld (%s), hl\n", g.localSym(s.Identifier))
	}
	if s.Target2 != nil && s.Target2.Identifier != "" {
		g.emitTarget2Store(s.Target2)
	}
	return nil
}

func (g *Gen) genLetArrField(s Let, t Type) error {
	isArrOfRecords := true
	if t.Record() != nil {
		fname := s.Fields[0]
		for _, f := range t.Record().Fields {
			if f.Identifier == fname && f.Type.Array() != nil {
				isArrOfRecords = false
				break
			}
		}
	}
	if isArrOfRecords {
		return g.genLetArrOfRecField(s, t)
	}
	return g.genLetFieldArr(s, t)
}

// genLetArrOfRecField handles arr[i].x — field write into array of records.
func (g *Gen) genLetArrOfRecField(s Let, t Type) error {
	if t.Record() == nil {
		return fmt.Errorf("let: %s is not an array of records", s.Identifier)
	}
	recSize := g.elemSize(s.Identifier)
	fname := s.Fields[0]
	fieldIdx := findField(t.Record(), fname)
	if fieldIdx < 0 {
		return fmt.Errorf("let: struct %s has no field %s", s.Identifier, fname)
	}
	off := t.Record().FieldOffset(fieldIdx)
	ft := t.Record().Fields[fieldIdx].Type

	g.emitLetBaseAddr(s.Identifier)
	if len(s.Subscripts) >= 1 {
		if err := g.emitScaledIndex(s.Subscripts[0], recSize); err != nil {
			return err
		}
	}
	if off > 0 {
		g.Emitf("\tld de, %d\n", off)
		g.Emitln("\tadd hl, de")
	}
	g.Emitln("\tpop de")
	if ft.Predeclared() == PredeclaredByte {
		g.emitStoreDE8()
	} else {
		g.emitStoreDE()
	}
	return nil
}

// genLetFieldArr handles rec.arr[i] — array field access with index.
func (g *Gen) genLetFieldArr(s Let, t Type) error {
	return g.genLetField(s, t) // same path; genLetField handles subscripted array fields
}

func (g *Gen) genLetField(s Let, t Type) error {
	if t.Record() == nil {
		return fmt.Errorf("let: %s is not a struct", s.Identifier)
	}
	fname := s.Fields[0]
	fieldIdx := findField(t.Record(), fname)
	if fieldIdx < 0 {
		return fmt.Errorf("let: struct %s has no field %s", s.Identifier, fname)
	}
	off := t.Record().FieldOffset(fieldIdx)
	ft := t.Record().Fields[fieldIdx].Type

	g.emitLetBaseAddr(s.Identifier)
	if off > 0 {
		g.Emitf("\tld de, %d\n", off)
		g.Emitln("\tadd hl, de")
	}
	if ft.Array() != nil && len(s.Subscripts) > 0 {
		arrElemSize := 1
		if ft.Array().ElemType.Predeclared() == PredeclaredWord {
			arrElemSize = 2
		} else if ft.Array().ElemType.Record() != nil {
			arrElemSize = ft.Array().ElemType.Record().TotalSize()
			arrElemSize = nextPow2(arrElemSize)
		}
		if err := g.emitScaledIndex(s.Subscripts[0], arrElemSize); err != nil {
			return err
		}
	}
	g.Emitln("\tpop de")
	if ft.Predeclared() == PredeclaredByte ||
		(ft.Array() != nil && ft.Array().ElemType.Predeclared() == PredeclaredByte) {
		g.emitStoreDE8()
	} else {
		g.emitStoreDE()
	}
	return nil
}

func (g *Gen) genLetArray(s Let) error {
	elem := g.elemSize(s.Identifier)
	g.emitLetBaseAddr(s.Identifier)
	for i := range s.Subscripts {
		if err := g.emitScaledIndex(s.Subscripts[i], elem); err != nil {
			return err
		}
	}
	g.Emitln("\tpop de")
	if elem == 1 {
		g.emitStoreDE8()
	} else {
		g.emitStoreDE()
	}
	if s.Target2 != nil && s.Target2.Identifier != "" {
		g.Emitln("\tpop de")
		g.emitTarget2Store(s.Target2)
	}
	return nil
}

func (g *Gen) emitLetBaseAddr(id Identifier) {
	if g.isParamRef(id) {
		g.Emitf("\tld hl, (%s)\n", g.localSym(id))
	} else {
		g.Emitf("\tld hl, %s\n", g.localSym(id))
	}
}

// emitScaledIndex computes address = hl + expr * scale and leaves result in HL.
func (g *Gen) emitScaledIndex(expr Expression, scale int) error {
	g.Emitln("\tpush hl")
	if err := expr.Gen(g); err != nil {
		return err
	}
	for size := scale; size > 1; size >>= 1 {
		g.Emitln("\tadd hl, hl")
	}
	g.Emitln("\tex de, hl")
	g.Emitln("\tpop hl")
	g.Emitln("\tadd hl, de")
	return nil
}

func (g *Gen) emitStoreDE() {
	g.Emitln("\tld (hl), e")
	g.Emitln("\tinc hl")
	g.Emitln("\tld (hl), d")
}

func (g *Gen) emitStoreDE8() {
	g.Emitln("\tld (hl), e")
}

func findField(rec *Record, name Identifier) int {
	for i, f := range rec.Fields {
		if f.Identifier == name {
			return i
		}
	}
	return -1
}

// emitTarget2Store emits code to store the second return value (in DE) into
// the target reference. Only simple variable targets are supported.
func (g *Gen) emitTarget2Store(t2 *Reference) {
	if t2.isByteRef(g) {
		g.Emitln("\tld a, e")
		g.Emitf("\tld (%s), a\n", g.localSym(t2.Identifier))
	} else {
		g.Emitf("\tld (%s), de\n", g.localSym(t2.Identifier))
	}
}

// Gen generates assembly for a group statement. It handles three forms:
// WHILE loops (condition checked at top), FOR loops (with start, end, optional
// step), and bare DO...END compound blocks (introducing a new scope).
func (s Group) Gen(g *Gen) error {
	switch {
	case s.While != nil:
		n := g.nextLabel()
		g.Emitf("_while_%d:\n", n)
		if err := g.genCondBranch(s.While.Expression, fmt.Sprintf("_end_%d", n)); err != nil {
			return err
		}
		g.pushScope()
		defer g.popScope()
		for _, stmt := range s.Statements {
			if err := stmt.Gen(g); err != nil {
				return err
			}
		}
		g.Emitf("\tjmp _while_%d\n", n)
		g.Emitf("_end_%d:\n", n)

	case s.For != nil:
		n := g.nextLabel()
		stepLabel := fmt.Sprintf("_for_step_%d", n)
		endLabel := fmt.Sprintf("_for_end_%d", n)
		g.addForTemp(stepLabel)
		g.addForTemp(endLabel)

		// Evaluate step (default 1), store in temp variable
		if s.For.By != nil {
			if err := s.For.By.Gen(g); err != nil {
				return err
			}
		} else {
			g.Emitln("\tld hl, 1")
		}
		g.Emitf("\tld (%s), hl\n", stepLabel)

		// Evaluate end, store in temp variable
		if err := s.For.To.Gen(g); err != nil {
			return err
		}
		g.Emitf("\tld (%s), hl\n", endLabel)

		// Initialize var = start
		if err := s.For.Start.Gen(g); err != nil {
			return err
		}
		g.Emitf("\tld (%s), hl\n", g.localSym(s.For.Reference.Identifier))

		g.Emitf("_for_%d:\n", n)
		// Compare var with end (hl = end - var)
		g.Emitf("\tld hl, (%s)\n", endLabel)
		g.Emitf("\tld de, (%s)\n", g.localSym(s.For.Reference.Identifier))
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")         // hl = end - var
		g.Emitf("\tjmp c, _end_%d\n", n) // end < var → exit

		// Body
		g.pushScope()
		defer g.popScope()
		for _, stmt := range s.Statements {
			if err := stmt.Gen(g); err != nil {
				return err
			}
		}

		// var += step
		g.Emitf("\tld hl, (%s)\n", stepLabel)
		g.Emitf("\tld de, (%s)\n", g.localSym(s.For.Reference.Identifier))
		g.Emitln("\tadd hl, de")
		g.Emitf("\tld (%s), hl\n", g.localSym(s.For.Reference.Identifier))
		g.Emitf("\tjmp _for_%d\n", n)
		g.Emitf("_end_%d:\n", n)

	case s.Case != nil:
		// CASE: evaluate selector, compare against each branch value, jump
		// to the matching branch body, then to end. If no branch matches,
		// execute the DEFAULT body (if present) or fall through to end.

		if err := s.Case.Expression.Gen(g); err != nil {
			return err
		}

		n := g.nextLabel()
		endLabel := fmt.Sprintf("_case_end_%d", n)

		// Save selector on stack for comparison against each value, then
		// emit a comparison chain. Each branch may have multiple values.
		g.Emitln("\tpush hl")
		for i, branch := range s.Case.Branches {
			branchLabel := fmt.Sprintf("_case_%d_%d", n, i)
			for _, cv := range branch.Values {
				v, err := g.resolveCaseVal(cv)
				if err != nil {
					return err
				}
				g.Emitln("\tpop hl")
				g.Emitln("\tpush hl")
				g.Emitf("\tld de, %d\n", v)
				g.Emitln("\tor a")
				g.Emitln("\tsbc hl, de")
				g.Emitf("\tjmp z, %s\n", branchLabel)
			}
		}

		// No match: clean stack.
		g.Emitln("\tpop hl")

		// Compute the default jump target (default label or end).
		nomatchLabel := endLabel
		if s.Case.Default != nil {
			nomatchLabel = fmt.Sprintf("_case_dflt_%d", n)
		}
		g.Emitf("\tjmp %s\n", nomatchLabel)

		// Emit branch bodies (all jump to end).
		for i, branch := range s.Case.Branches {
			label := fmt.Sprintf("_case_%d_%d", n, i)
			g.Emitf("%s:\n", label)
			g.Emitln("\tpop hl")
			if err := branch.Statement.Gen(g); err != nil {
				return err
			}
			g.Emitf("\tjmp %s\n", endLabel)
		}

		// Emit default body if present, falling through to end.
		if s.Case.Default != nil {
			g.Emitf("%s:\n", nomatchLabel)
			if err := s.Case.Default.Gen(g); err != nil {
				return err
			}
		}

		g.Emitf("%s:\n", endLabel)

	default:
		// Bare DO...END: push scope and emit statements
		g.pushScope()
		defer g.popScope()
		for _, stmt := range s.Statements {
			if err := stmt.Gen(g); err != nil {
				return err
			}
		}
	}
	return nil
}

// Gen generates assembly for a PROCEDURE definition. It emits the procedure
// entry label, pushes a new scope populated with parameter symbols, saves
// parameters to their RAM labels (for non-REENTRANT procedures), generates the
// body, and appends a RET if the last statement is not a RETURN.
func (s Procedure) Gen(g *Gen) error {
	g.Emitf("_plz_%s:\n", s.Name.Name)

	// Save return type and interrupt type so Return.Gen can emit correct ret.
	g.ProcReturnType = s.Type
	g.ProcInterrupt = s.Interrupt

	// Push scope and register parameters so localSym can find their labels.
	g.pushScope()
	defer g.popScope()
	for _, param := range s.Parameters {
		g.symStack[len(g.symStack)-1][param] = symEntry{
			label: fmt.Sprintf("_plz_%s_%s", s.Name.Name, param),
		}
	}

	// For non-REENTRANT: save parameters to their individual RAM labels.
	if !s.Reentrant && len(s.Parameters) > 0 {
		p0size := s.paramByteSize(0)
		if p0size == 1 {
			g.Emitf("\tld a, l\n\tld (_plz_%s_%s), a\n", s.Name.Name, s.Parameters[0])
		} else {
			g.Emitf("\tld (_plz_%s_%s), hl\n", s.Name.Name, s.Parameters[0])
		}
		if len(s.Parameters) > 1 {
			p1size := s.paramByteSize(1)
			if p1size == 1 {
				g.Emitf("\tld a, e\n\tld (_plz_%s_%s), a\n", s.Name.Name, s.Parameters[1])
			} else {
				g.Emitf("\tld (_plz_%s_%s), de\n", s.Name.Name, s.Parameters[1])
			}
		}
		// Params 3+ are already stored by the call site.
	}

	// Generate body; Declare.Gen registers locals in the current scope.
	g.procName = s.Name.Name
	for _, stmt := range s.Statements {
		if err := stmt.Gen(g); err != nil {
			return err
		}
	}
	g.procName = ""

	// Emit ret (or reti/retn) if the procedure doesn't end with RETURN.
	if len(s.Statements) == 0 {
		g.emitReturn()
	} else if _, ok := s.Statements[len(s.Statements)-1].Command.(Return); !ok {
		g.emitReturn()
	}
	return nil
}

// Gen generates assembly for a RETURN statement. With no expressions it emits a
// plain RET. With one expression it evaluates it into HL (zero-extending BYTE
// return values); with two expressions it puts the second in DE and the first in
// HL. For RECORD return types, it returns the address of the record variable
// rather than its value.
func (s Return) Gen(g *Gen) error {
	switch len(s.Expressions) {
	case 0:
		// No return value → just ret.
	case 1:
		// For RECORD return type, return the ADDRESS of the record.
		if g.ProcReturnType.Record() != nil {
			ref := s.Expressions[0].Ref()
			if ref != nil && ref.Identifier != "" && len(ref.Fields) == 0 && len(ref.Subscripts) == 0 {
				g.Emitf("\tld hl, %s\n", g.localSym(ref.Identifier))
				break
			}
			// Non-trivial expression: compute address instead of value.
			if expr := s.Expressions[0].Suffix(); expr != nil {
				switch expr.Operator {
				case OperatorINDEX:
					_, err := g.genIndexAddr(expr.Operands)
					return err
				case OperatorFIELD:
					_, err := g.genFieldAddr(expr.Operands)
					return err
				}
			}
			return fmt.Errorf("cannot take address of return expression")
		}
		if err := s.Expressions[0].Gen(g); err != nil {
			return err
		}
		// Zero-extend BYTE return values.
		if g.ProcReturnType.Predeclared() == PredeclaredByte {
			g.Emitln("\tld h, 0")
		}
	case 2:
		// Second return value in DE.
		if err := s.Expressions[1].Gen(g); err != nil {
			return err
		}
		g.Emitln("\tpush hl")
		if err := s.Expressions[0].Gen(g); err != nil {
			return err
		}
		g.Emitln("\tpop de")
	default:
		return fmt.Errorf("return: too many return values (%d)", len(s.Expressions))
	}
	g.emitReturn()
	return nil
}

// Gen generates assembly for a CALL statement. Arguments are loaded into HL,
// DE, or stored in procedure-specific RAM labels (for non-REENTRANT) or pushed
// on the stack (for REENTRANT). Record and DATA arguments pass addresses.
func (s Call) Gen(g *Gen) error {
	name := string(s.Identifier)
	args := s.Arguments
	proc, procOk := g.Checker.Procedures[name]

	genCallArg := func(i int) error {
		return g.genCallArg(args[i], proc, procOk, i)
	}

	switch len(args) {
	case 0:
	case 1:
		if err := genCallArg(0); err != nil {
			return err
		}
	case 2:
		if err := genCallArg(1); err != nil {
			return err
		}
		g.Emitln("\tpush hl")
		if err := genCallArg(0); err != nil {
			return err
		}
		g.Emitln("\tpop de")
	default:
		totalExtra := g.emitExtraCallArgs(name, proc, procOk, genCallArg, len(args))
		if err := genCallArg(1); err != nil {
			return err
		}
		g.Emitln("\tpush hl")
		if err := genCallArg(0); err != nil {
			return err
		}
		g.Emitln("\tpop de")
		isReentrant := !procOk || proc.Reentrant
		g.Emitf("\tcall _plz_%s\n", name)
		if isReentrant && totalExtra > 0 {
			g.Emitf("\tld hl, %d\n", totalExtra)
			g.Emitln("\tadd hl, sp")
			g.Emitln("\tld sp, hl")
		}
		return nil
	}
	g.Emitf("\tcall _plz_%s\n", name)
	return nil
}

// genCallArg emits code to load the i-th argument into HL.
// For RECORD or DATA params it loads the address rather than the value.
// expr is the AST expression for the argument; proc/ok provide type info.
func (g *Gen) genCallArg(expr Expression, proc Procedure, ok bool, i int) error {
	if !ok || i >= len(proc.ParamTypes) {
		return expr.Gen(g)
	}
	pt := proc.ParamTypes[i]
	if pt.Record() == nil && pt.Predeclared() != PredeclaredData {
		return expr.Gen(g)
	}

	ref := expr.Ref()
	if ref != nil && ref.Identifier != "" && len(ref.Fields) == 0 && len(ref.Subscripts) == 0 {
		if g.isParamRef(ref.Identifier) {
			g.Emitf("\tld hl, (%s)\n", g.localSym(ref.Identifier))
		} else {
			g.Emitf("\tld hl, %s\n", g.localSym(ref.Identifier))
		}
		return nil
	}

	if suff := expr.Suffix(); suff != nil {
		switch suff.Operator {
		case OperatorINDEX:
			_, err := g.genIndexAddr(suff.Operands)
			return err
		case OperatorFIELD:
			_, err := g.genFieldAddr(suff.Operands)
			return err
		}
	}

	if op := expr.Operand(); op != nil {
		if lit := op.Literal(); lit != nil {
			if t := lit.Text(); t != nil {
				n := g.nextLabel()
				label := fmt.Sprintf("_plz_str_%d", n)
				g.strings = append(g.strings, strEntry{label: label, data: t.Value})
				g.Emitf("\tld hl, %s\n", label)
				return nil
			}
		}
	}
	return fmt.Errorf("cannot take address of argument %d", i)
}

// emitExtraCallArgs emits code for arguments 3+ of a procedure call. For
// non-REENTRANT procedures it stores each extra arg into its dedicated RAM
// label; for REENTRANT it pushes them right-to-left. Returns total extra bytes
// pushed (for REENTRANT cleanup).
func (g *Gen) emitExtraCallArgs(name string, proc Procedure, ok bool, genArg func(int) error, argc int) int {
	isReentrant := !ok || proc.Reentrant
	var totalExtra int
	if !isReentrant {
		for i := 2; i < argc; i++ {
			if err := genArg(i); err != nil {
				panic(err)
			}
			paramName := proc.Parameters[i]
			if ok && i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared() == PredeclaredByte {
				g.Emitf("\tld a, l\n\tld (_plz_%s_%s), a\n", name, paramName)
			} else {
				g.Emitf("\tld (_plz_%s_%s), hl\n", name, paramName)
			}
		}
	} else {
		for i := argc - 1; i >= 2; i-- {
			if err := genArg(i); err != nil {
				panic(err)
			}
			psize := 2
			if ok && i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared() == PredeclaredByte {
				psize = 1
			}
			totalExtra += psize
			if psize == 1 {
				g.Emitln("\tld a, l")
				g.Emitln("\tpush af")
			} else {
				g.Emitln("\tpush hl")
			}
		}
	}
	return totalExtra
}

// Gen generates assembly for a GOTO statement, emitting a JP to the named label.
func (s GoTo) Gen(g *Gen) error {
	g.Emit("\tjp ")
	g.Emitln(s.Name)
	return nil
}

// Gen generates assembly for an AT directive. If HasBank is true,
// it emits a "bank" directive to switch the active ROM bank. Otherwise,
// it emits an ORG directive at the specified address and updates the heap
// pointer so subsequent data declarations are placed there.
func (s At) Gen(g *Gen) error {
	if s.HasBank {
		g.Emitf("bank %d\n", s.BankNumber)
		return nil
	}
	addr, err := g.Checker.EvalConstExpr(s.Address)
	if err != nil {
		return err
	}
	g.Emitf("org 0x%x\n", addr)
	g.Heap = addr
	return nil
}

// Gen processes a PRAGMA directive at code-generation time.
// PRAGMA BOUNDCHECK and PRAGMA NOBOUNDCHECK toggle runtime array bounds checking.
func (s Pragma) Gen(g *Gen) error {
	for _, id := range s.Idents {
		switch string(id) {
		case "BOUNDCHECK":
			g.BoundCheck = true
		case "NOBOUNDCHECK":
			g.BoundCheck = false
		}
	}
	return nil
}

// genOnceBoundsError emits the bounds error handler once if not already emitted.
func (g *Gen) genOnceBoundsError() {
	if g.boundsErrorEmitted {
		return
	}
	g.boundsErrorEmitted = true
	g.Emitln("_plz_bounds_error:")
	g.Emitln("\thalt")
	g.Emitln("\tjr _plz_bounds_error")
}

// resolveCaseVal resolves a CaseVal to an integer at codegen time.
func (g *Gen) resolveCaseVal(cv CaseVal) (int, error) {
	if cv.Name == "" {
		return cv.Value, nil
	}
	lit, ok := g.Checker.Constants[Identifier(cv.Name)]
	if !ok {
		return 0, fmt.Errorf("CASE: undefined constant %q", cv.Name)
	}
	if n := lit.Number(); n != nil {
		return n.Value, nil
	}
	return 0, fmt.Errorf("CASE: constant %q is not a number", cv.Name)
}

// evalConstExpr evaluates an Expression to an integer at codegen time.
// It delegates to the Checker's EvalConstExpr which resolves constants and
// evaluates all compile-time operators.
func (g *Gen) evalConstExpr(e Expression) (int, error) {
	return g.Checker.EvalConstExpr(e)
}

// Gen generates assembly for a DECLARE statement. Inside a procedure body it
// registers the identifier in the current scope so later references resolve to
// the correct RAM label. At global scope it emits an ORG directive followed by
// zero-initialized storage (or the initializer value if one is provided).
func (s Declare) Gen(g *Gen) error {
	if g.procName != "" {
		// Local variable: register in the current scope so localSym can find it.
		var label string
		if len(g.symStack) <= 1 {
			label = fmt.Sprintf("_plz_%s_%s", g.procName, s.Identifier)
		} else {
			label = fmt.Sprintf("_plz_%s_%d_%s", g.procName, len(g.symStack)-1, s.Identifier)
		}
		g.symStack[len(g.symStack)-1][s.Identifier] = symEntry{
			label: label,
		}
		// Emit initialization code for the local variable.
		if s.Initializer != nil {
			initVal, err := g.evalConstExpr(s.Initializer.Expr)
			if err != nil {
				return err
			}
			if g.elemSize(s.Identifier) == 1 {
				g.Emitf("\tld a, %d\n", initVal&0xFF)
				g.Emitf("\tld (%s), a\n", label)
			} else {
				g.Emitf("\tld hl, %d\n", initVal&0xFFFF)
				g.Emitf("\tld (%s), hl\n", label)
			}
		}
		return nil
	}
	elemSize := g.elemSize(s.Identifier)
	total := elemSize
	if arr := s.Type.Array(); arr != nil && arr.Size > 0 {
		total = arr.Size * elemSize
	}

	// If AT is set, the variable is at a fixed absolute address (for
	// memory-mapped I/O or direct memory access). Only emit the label so
	// references resolve to the correct address; do not emit data bytes,
	// which could overwrite code already placed in that region.
	// Do not advance g.Heap so subsequent declarations are unaffected.
	if s.At != nil {
		addr, err := g.evalConstExpr(*s.At)
		if err != nil {
			return err
		}
		g.Emitf("org 0x%x\n%s:\n", addr, s.Identifier)
		return nil
	}

	g.Emitf("org 0x%x\n%s: ", g.Heap, s.Identifier)
	if s.Initializer != nil {
		initVal, err := g.evalConstExpr(s.Initializer.Expr)
		if err != nil {
			return err
		}
		if elemSize == 1 {
			g.Emitf("db %d", initVal&0xFF)
			for i := 1; i < total; i++ {
				g.Emitf(", %d", initVal&0xFF)
			}
		} else {
			n := total / elemSize
			g.Emitf("dw %d", initVal&0xFFFF)
			for i := 1; i < n; i++ {
				g.Emitf(", %d", initVal&0xFFFF)
			}
		}
	} else {
		g.Emitf("db 0")
		for i := 1; i < total; i++ {
			g.Emit(", 0")
		}
	}
	g.Emit("\n")
	g.Heap += total
	return nil
}

// Gen generates assembly for an INTERRUPT or NMI install statement.
// It emits a JP to the named procedure at the appropriate vector address:
// 0x0038 for maskable interrupts, 0x0066 for NMI.
// The current ORG position is saved before and restored after, so subsequent
// code continues at the correct address.
func (s InterruptStmt) Gen(g *Gen) error {
	addr := 0x0038
	if s.NMI {
		addr = 0x0066
	}
	label := fmt.Sprintf("_plz_org_%d", g.nextLabel())
	g.Emitf("%s:\n", label)
	g.Emitf("org 0x%04x\n", addr)
	g.Emitf("\tjp _plz_%s\n", s.Target)
	g.Emitf("org %s\n", label)
	return nil
}

// Gen generates assembly for a HALT statement. Inside a task body it jumps to
// the task-done handler; otherwise it emits a HALT instruction.
func (s Halt) Gen(g *Gen) error {
	if g.InTask {
		g.Emitln("\tjp _plz_task_done")
	} else {
		g.Emitln("\thalt")
	}
	return nil
}

// Gen generates assembly for an ENABLE statement, emitting an EI instruction.
func (s Enable) Gen(g *Gen) error {
	g.Emitln("\tei")
	return nil
}

// Gen generates assembly for a DISABLE statement, emitting a DI instruction.
func (s Disable) Gen(g *Gen) error {
	g.Emitln("\tdi")
	return nil
}

// Gen generates assembly for a BANK statement, emitting code to switch
// the active ROM bank on the Sega mapper by writing to port 0xFFFD.
func (s BankStmt) Gen(g *Gen) error {
	val, err := g.Checker.EvalConstExpr(s.Number)
	if err != nil {
		return fmt.Errorf("bank: %v", err)
	}
	if val < 0 || val > 255 {
		return fmt.Errorf("bank: value %d out of range (0-255)", val)
	}
	g.Emitf("\tld a, %d\n", val)
	g.Emitln("\tld (0xFFFD), a")
	return nil
}

// Gen generates assembly for an OUTPUT statement. It evaluates the value into
// HL, then writes the low byte (and optionally the high byte for WORD output)
// to the given port.
func (s Output) Gen(g *Gen) error {
	if err := s.Value.Gen(g); err != nil {
		return err
	}
	g.Emitf("\tld a, l\n")
	g.Emitf("\tout (%d), a\n", s.Port)
	if s.IsWord {
		g.Emitf("\tld a, h\n")
		g.Emitf("\tout (%d), a\n", s.Port)
	}
	return nil
}

const SRAMBase = 0x8000
const SRAMControl = 0xFFFC
const SRAMEnable = 0x8
const SRAMDisable = 0x0

// generates code to enable SRAM
func genEnableSRAM(g *Gen) {
	g.Emitf("\tld a, %d\n", SRAMEnable)
	g.Emitf("\tld (%d), a\n", SRAMControl)
}

// generates code to disable SRAM
func genDisableSRAM(g *Gen) {
	g.Emitf("\tld a, %d\n", SRAMDisable)
	g.Emitf("\tld (%d), a\n", SRAMControl)
}

// Gen generates assembly for a SAVE statement. It emits an LDIR loop to copy
// data from the source reference to the destination address (from AT or
// SRAMBase if not given).
func (s Save) Gen(g *Gen) error {
	ref := s.Source.Ref()
	if ref == nil {
		return fmt.Errorf("SAVE source must be a reference")
	}
	// must enable SRAM
	genEnableSRAM(g)
	defer genDisableSRAM(g)

	// Load source address into HL.
	if g.isParamRef(ref.Identifier) {
		g.Emitf("\tld hl, (%s)\n", g.localSym(ref.Identifier))
	} else {
		g.Emitf("\tld hl, %s\n", g.localSym(ref.Identifier))
	}

	// Load destination address into DE, default SRAMBase.
	var destAddr int = SRAMBase
	if s.Location != nil {
		var err error
		destAddr, err = g.Checker.EvalConstExpr(*s.Location)
		if err != nil {
			return err
		}
	}
	g.Emitf("\tld de, 0x%x\n", destAddr)

	// Compute byte count into BC.
	size, err := g.saveSize(ref.Identifier)
	if err != nil {
		return err
	}
	if size == 0 {
		return fmt.Errorf("SAVE: %q has zero size", ref.Identifier)
	}
	g.Emitf("\tld bc, %d\n", size)
	g.Emitln("\tldir")

	return nil
}

// Gen generates assembly for a LOAD statement. It emits an LDIR loop to copy
// data from the SRAM address (from AT or SRAMBase) into the
// target variable.
func (s Load) Gen(g *Gen) error {
	ref := s.Target.Ref()
	if ref == nil {
		return fmt.Errorf("LOAD target must be a reference")
	}
	genEnableSRAM(g)
	defer genDisableSRAM(g)

	// Load source address (SRAM) into HL.
	var srcAddr int = SRAMBase
	if s.Location != nil {
		var err error
		srcAddr, err = g.Checker.EvalConstExpr(*s.Location)
		if err != nil {
			return err
		}
	}
	g.Emitf("\tld hl, 0x%x\n", srcAddr)

	// Load destination address (variable) into DE.
	if g.isParamRef(ref.Identifier) {
		g.Emitf("\tld de, (%s)\n", g.localSym(ref.Identifier))
	} else {
		g.Emitf("\tld de, %s\n", g.localSym(ref.Identifier))
	}

	// Compute byte count into BC.
	size, err := g.saveSize(ref.Identifier)
	if err != nil {
		return err
	}
	if size == 0 {
		return fmt.Errorf("LOAD: %q has zero size", ref.Identifier)
	}
	g.Emitf("\tld bc, %d\n", size)
	g.Emitln("\tldir")

	return nil
}

// saveSize returns the byte size of the data referenced by the given identifier.
// It handles Declare entries (via Checker.Lookup) and Data blocks (via Checker.Datas).
func (g *Gen) saveSize(id Identifier) (int, error) {
	// Check for a DATA block first.
	if data, ok := g.Checker.Datas[id]; ok {
		if data.Tile != nil {
			size := 0
			for _, tile := range data.Tile.Tiles {
				size += len(tile.Bytes())
			}
			return size, nil
		}
		if data.Text != nil {
			return len(data.Text.Value) + 1, nil // length byte + string
		}
		size := 0
		for _, val := range data.Values {
			if op := val.Operand(); op != nil {
				if lit := op.Literal(); lit != nil {
					if t := lit.Text(); t != nil {
						size += len(t.Value)
						continue
					}
				}
			}
			size++
		}
		return size, nil
	}
	// Fall back to a declared variable/array/record.
	if d, ok := g.Checker.Lookup(id); ok {
		return d.StorageSize(), nil
	}
	return 0, fmt.Errorf("%q not found", id)
}

// emitStorageRaw emits a RAM storage allocation at g.Heap for the given label
// and byte size, advancing the heap pointer.
func (g *Gen) emitStorageRaw(label string, size int) {
	g.Emitf("org 0x%x\n%s: db 0", g.Heap, label)
	for i := 1; i < size; i++ {
		g.Emit(", 0")
	}
	g.Emit("\n")
	g.Heap += size
}

// emitStorage is like emitStorageRaw but takes a format string and variadic
// arguments for constructing the label.
func (g *Gen) emitStorage(format string, size int, args ...any) {
	g.emitStorageRaw(fmt.Sprintf(format, args...), size)
}

// addForTemp registers a FOR loop temp variable label to be emitted in the
// data section. Each temp is 2 bytes (one word).
func (g *Gen) addForTemp(label string) {
	g.forTemps = append(g.forTemps, label)
}

// genForTemps emits all registered FOR loop temp variables in the data section.
func (g *Gen) genForTemps() {
	for _, label := range g.forTemps {
		g.emitStorageRaw(label, 2)
	}
}

// Gen generates assembly for a DATA declaration, emitting DB directives for
// numeric expressions and DS directives for text literals.
func (s Data) Gen(g *Gen) error {
	g.Emitf("%s:\n", s.Name)
	if s.Tile != nil {
		return s.Tile.Gen(g)
	}
	if s.Text != nil {
		g.Emitf("\tdb %d\n", len(s.Text.Value))
		g.Emitf("\tds %s\n", strconv.Quote(s.Text.Value))
		return nil
	}

	for _, val := range s.Values {
		if v, err := g.Checker.EvalConstExpr(val); err == nil {
			g.Emitf("\tdb %d\n", v)
		} else if op := val.Operand(); op != nil {
			if lit := op.Literal(); lit != nil {
				if t := lit.Text(); t != nil {
					g.Emitf("\tds %s\n", strconv.Quote(t.Value))
					continue
				}
			}
			return fmt.Errorf("data: cannot evaluate value %v", val)
		} else {
			return fmt.Errorf("data: cannot evaluate expression")
		}
	}
	return nil
}

// Gen generates data for the Tile,
func (b Tile) Gen(g *Gen) error {
	if len(b.Tiles) < 1 {
		return fmt.Errorf("No tiles to generate")
	}
	for ti, tile := range b.Tiles {
		g.Emitf("\t// Tile %d\n", ti)
		for y := 0; y < tile.Size(); y++ {
			g.Emitf("\t// ")
			for x := 0; x < tile.Size(); x++ {
				id, _ := tile.PaletteIdAt(y, x)
				g.Emitf("%x", int(id))
			}
			g.Emitf("\n")
		}

		buf := tile.Bytes()
		for _, b := range buf {
			g.Emitf("\tdb %d\n", b)
		}
	}
	g.Emitln()
	return nil
}

// Gen generates assembly for a CONSTANT declaration, emitting a const directive
// with the evaluated expression value.
func (s Constant) Gen(g *Gen) error {
	if s.Expr.Expr == nil {
		return nil
	}
	if v, err := g.Checker.EvalConstExpr(s.Expr); err == nil {
		g.Emitf("\tconst %s = %d\n", s.Name, v)
	} else if op := s.Expr.Operand(); op != nil {
		if lit := op.Literal(); lit != nil {
			if t := lit.Text(); t != nil {
				g.Emitf("\tconst %s = %s\n", s.Name, strconv.Quote(t.Value))
			}
		}
	}
	return nil
}

// Gen generates assembly for a SUSPEND statement, setting the target task's
// state byte to 1 (SUSPENDED).
func (s Suspend) Gen(g *Gen) error {
	idx, ok := g.Checker.Tasks[string(s.Name)]
	if !ok {
		return fmt.Errorf("undeclared task %q", s.Name)
	}
	g.Emitf("\tld a, 1\n")
	g.Emitf("\tld (_plz_tcbs+%d), a\n", idx*8+2)
	return nil
}

// Gen generates assembly for a RESUME statement, clearing the target task's
// state byte to 0 (READY).
func (r Resume) Gen(g *Gen) error {
	idx, ok := g.Checker.Tasks[string(r.Name)]
	if !ok {
		return fmt.Errorf("undeclared task %q", r.Name)
	}
	g.Emitf("\txor a\n")
	g.Emitf("\tld (_plz_tcbs+%d), a\n", idx*8+2)
	return nil
}

// Gen generates assembly for a SLEEP statement. It evaluates the duration,
// sets the current task's state to SLEEPING (2) with the duration as counter,
// then calls the scheduler. If the duration is zero, the scheduler call is
// skipped.
func (s Sleep) Gen(g *Gen) error {
	if err := s.Duration.Gen(g); err != nil {
		return err
	}
	n := g.nextLabel()
	g.Emitln("\tld a, l")
	g.Emitln("\tor a")
	g.Emitf("\tjr z, _slp_%d\n", n)
	g.Emitln("\tpush af")
	g.Emitln("\tld a, (_plz_current_task)")
	g.Emitln("\tld l, a")
	g.Emitln("\tld h, 0")
	g.Emitln("\tadd hl, hl")
	g.Emitln("\tadd hl, hl")
	g.Emitln("\tadd hl, hl")
	g.Emitln("\tld de, _plz_tcbs+2")
	g.Emitln("\tadd hl, de")
	g.Emitln("\tld (hl), 2") // state = SLEEPING
	g.Emitln("\tinc hl")     // HL = &sleep counter
	g.Emitln("\tpop af")
	g.Emitln("\tld (hl), a")
	g.Emitln("\tcall _plz_scheduler")
	g.Emitf("_slp_%d:\n", n)
	return nil
}

// Gen generates assembly for a YIELD statement, calling the task scheduler to
// potentially switch to another ready task.
func (y Yield) Gen(g *Gen) error {
	g.Emitln("\tcall _plz_scheduler")
	return nil
}

// Gen generates assembly for a TASK definition. Task bodies are emitted in
// Program.Gen rather than inline, so this method is a no-op.
func (t Task) Gen(g *Gen) error {
	// Task bodies are emitted in Program.Gen, not inline.
	return nil
}

// SchedulerCode is the assembly block implementing the cooperative task
// scheduler. It saves the current task's stack pointer, decrements all sleeping
// tasks' counters (waking only SLEEPING tasks that reach zero), performs
// priority-based task selection among ready tasks (lower priority value =
// higher priority, round-robin among equal-priority tasks), and restores the
// chosen task's stack pointer.
const SchedulerCode = `
// -------------------------------------------------------------------
// Task scheduler: save current SP, pick next ready task, restore SP
// -------------------------------------------------------------------
_plz_scheduler:
	// Save current task's SP in its TCB
	ld a, (_plz_current_task)
	ld l, a
	ld h, 0
	add hl, hl
	add hl, hl
	add hl, hl
	ld de, _plz_tcbs
	add hl, de
	ld (_plz_sch_sp), sp
	ld de, (_plz_sch_sp)
	ld (hl), e
	inc hl
	ld (hl), d

	// Check stack canary for the current task (2 bytes at stack_base).
	// If overwritten, the task's stack overflowed — mark it DEAD.
	// HL = TCB_entry + 1; compute TCB_entry + 5 for the stack base.
	inc hl
	inc hl
	inc hl
	inc hl		// HL = TCB_entry + 5 (stack base address)
	ld c, (hl)
	inc hl
	ld b, (hl)	// BC = stack base address
	ld a, (bc)	// canary byte 0
	cp 0xDE
	jp nz, _sch_stack_dead
	inc bc
	ld a, (bc)	// canary byte 1
	cp 0xAD
	jp nz, _sch_stack_dead

	// Decrement all sleeping tasks' sleep counters.
	// When a counter reaches 0, if the task is SLEEPING (state=2),
	// set its state to READY. SUSPENDED tasks are not woken.
	ld hl, _plz_tcbs+3
	ld b, 16
	ld de, 8
_slp_dec_loop:
	ld a, (hl)
	or a
	jr z, _slp_dec_cont
	dec (hl)
	jr nz, _slp_dec_cont
	// Sleep counter hit 0 — only wake if task is SLEEPING
	dec hl
	ld a, (hl)
	cp 2
	jr nz, _slp_nowake
	ld (hl), 0
_slp_nowake:
	inc hl
_slp_dec_cont:
	add hl, de
	djnz _slp_dec_loop

	// Priority-based task selection.
	// Scan all 16 slots starting from current+1 (wrap).
	// Among READY tasks, pick the one with the lowest priority value
	// (0 = highest, 15 = lowest). If multiple have the same priority,
	// the first encountered in scan order wins (round-robin).
	// Uses _plz_sch_sp as a temporary scan pointer.
	ld a, (_plz_current_task)
	inc a
	cp 16
	jr c, _sch_init
	xor a
_sch_init:
	ld (_plz_sch_sp), a	// scan pointer
	ld b, 16		// loop counter
	ld d, 0xFF		// best_pri (lowest value = highest priority)
	ld e, 0xFF		// best_idx (0xFF = none)
_sch_loop:
	ld a, (_plz_sch_sp)
	ld l, a
	ld h, 0
	add hl, hl
	add hl, hl
	add hl, hl
	push de
	ld de, _plz_tcbs+2
	add hl, de
	pop de
	ld a, (hl)		// state
	or a
	jr nz, _sch_next	// not READY, skip
	// READY — compare priority
	inc hl
	inc hl			// HL = &priority
	ld a, (hl)
	cp d			// compare with best so far
	jr nc, _sch_next	// priority >= best_pri, skip
	ld d, a			// best_pri = priority
	ld a, (_plz_sch_sp)
	ld e, a			// best_idx = current slot
_sch_next:
	ld a, (_plz_sch_sp)
	inc a
	cp 16
	jr c, _sch_wrap
	xor a
_sch_wrap:
	ld (_plz_sch_sp), a
	djnz _sch_loop

	// Check if any READY task was found
	ld a, e
	cp 0xFF
	jr nz, _sch_found
	// No ready task — enable interrupts and halt, wait for interrupt
	ei
	halt
	jp _plz_scheduler
_sch_found:
	ld (_plz_current_task), a
	// Restore SP from TCB
	ld l, a
	ld h, 0
	add hl, hl
	add hl, hl
	add hl, hl
	ld de, _plz_tcbs
	add hl, de
	ld e, (hl)
	inc hl
	ld d, (hl)
	ex de, hl
	ld sp, hl
	ret

_plz_task_done:
	// Mark current task as dead
	ld a, (_plz_current_task)
	ld l, a
	ld h, 0
	add hl, hl
	add hl, hl
	add hl, hl
	ld de, _plz_tcbs+2
	add hl, de
	ld (hl), 3
	jp _plz_scheduler

_sch_stack_dead:
	// Stack canary corrupted — mark current task as DEAD
	ld a, (_plz_current_task)
	ld l, a
	ld h, 0
	add hl, hl
	add hl, hl
	add hl, hl
	ld de, _plz_tcbs+2
	add hl, de
	ld (hl), 3
	jp _plz_scheduler

`
