package plz

import (
	"fmt"
	"os"
	"strconv"
)

// HeapBase is the base address of the heap/RAM region at 0xC000.
const HeapBase = 0xC000 // RAM memory.

// Gen is the code generator that emits Z80 assembly text from a checked AST.
// It holds an output file, a scope stack for name resolution, and generation
// state such as the current procedure name and label counter.
type Gen struct {
	file           *os.File
	Heap           int // Pointer to last allocated heap RAM memory.
	label          int // counter for unique local labels
	Checker        *Checker
	InTask         bool                      // set when generating inside a task body
	procName       string                    // current procedure name (empty = global scope)
	symStack       []map[Identifier]symEntry // scope stack
	ProcReturnType Type                      // return type of current procedure (for BYTE zero-extend in Return.Gen)
	ProcInterrupt  *Interrupt                // interrupt type of current procedure (for reti/retn in Return.Gen)
	strings        []strEntry                // string literal labels and content for ROM emission
}

// strEntry records a string literal that needs to be emitted as ROM data.
type strEntry struct {
	label string
	data  string
}

// symEntry records the assembly-level label, type, and whether the symbol is a
// record or data parameter (passed by reference) for a single identifier on the
// scope stack.
type symEntry struct {
	label    string
	typ      Type
	paramRef bool
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

// localType resolves an identifier's type by walking the scope stack from
// innermost to outermost, falling back to the checker's root scope.
func (g *Gen) localType(id Identifier) (Type, bool) {
	for i := len(g.symStack) - 1; i >= 0; i-- {
		if e, ok := g.symStack[i][id]; ok {
			return e.typ, true
		}
	}
	if g.Checker != nil {
		if d, ok := g.Checker.Lookup(id); ok {
			return d.Type, true
		}
	}
	return Type{}, false
}

// isParamRef returns true when id is a record or data parameter passed by reference.
func (g *Gen) isParamRef(id Identifier) bool {
	for i := len(g.symStack) - 1; i >= 0; i-- {
		if e, ok := g.symStack[i][id]; ok {
			return e.paramRef
		}
	}
	if g.Checker != nil {
		if d, ok := g.Checker.Lookup(id); ok {
			return d.ParamRef
		}
	}
	return false
}

// pushScope pushes a new empty scope onto the symbol stack.
// Every call should be paired with a deferred popScope.
func (g *Gen) pushScope() {
	g.symStack = append(g.symStack, make(map[Identifier]symEntry))
}

// popScope removes the innermost scope from the symbol stack.
func (g *Gen) popScope() {
	g.symStack = g.symStack[:len(g.symStack)-1]
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

// ProgramFooter is the assembly epilogue that switches to the RAM region at 0xC000.
const ProgramFooter = `
org 0xC000 // RAM memory.
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
_plz_divmod:
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

	// Emit main code.

	g.Emit(ProgramHeader)
	g.Emitln("\tjp _plz_start")
	g.Emit(RuntimeHeader)
	g.Emitln("_plz_start:")

	// Task initialization
	if len(c.TaskDefs) > 0 {
		g.Emitf("\tld hl, _plz_tcbs\n")
		g.Emitf("\tld de, _plz_tcbs+1\n")
		g.Emitf("\tld bc, %d\n", 128)
		g.Emitln("\tld (hl), 0")
		g.Emitln("\tldir")
		for i, task := range c.TaskDefs {
			g.Emitf("\tld hl, _plz_task_%d\n", i)
			g.Emitf("\tld sp, _plz_task%d_stack+128\n", i)
			g.Emitln("\tpush hl")
			g.Emitf("\tld (_plz_tcbs+%d), sp\n", i*8)
			g.Emitf("\tld a, %d\n", task.Priority)
			g.Emitf("\tld (_plz_tcbs+%d), a\n", i*8+4)
		}
		// Mark unused task slots as dead
		for i := len(c.TaskDefs); i < 16; i++ {
			g.Emitf("\tld a, 3\n")
			g.Emitf("\tld (_plz_tcbs+%d), a\n", i*8+2)
		}
		g.Emitln("\txor a")
		g.Emitln("\tld (_plz_current_task), a")
		// Restore task 0's SP (which points to push'd ret addr) and "return" to it.
		g.Emitln("\tld sp, (_plz_tcbs+0)")
		g.Emitln("\tret")
	}

	type dataItem struct {
		at      *At
		declare *Declare
	}
	var dataItems []dataItem
	var procedures []Procedure
	var dataStmts []Data
	for _, statement := range p.Statements {
		switch cmd := statement.Command.(type) {
		case At:
			dataItems = append(dataItems, dataItem{at: &cmd})
		case Declare:
			dataItems = append(dataItems, dataItem{declare: &cmd})
		case Data:
			dataStmts = append(dataStmts, cmd)
		case Procedure:
			procedures = append(procedures, cmd)
		case Task:
			continue // emitted separately
		default:
			genner, ok := (cmd.(interface{ Gen(*Gen) error }))
			if ok {
				if err := genner.Gen(g); err != nil {
					return err
				}
			} else {
				g.Emitf("// statement not implemented: %v\n", statement)
			}
		}
	}

	// Emit loop to stop fallthough.
	g.Emitln("_plz_all_done:")
	g.Emitln("\tdi")
	g.Emitln("\thalt")
	g.Emitln("\tjp _plz_all_done")

	// Emit procedure bodies after main code (must not be reachable by fall-through).
	for _, proc := range procedures {
		if err := proc.Gen(g); err != nil {
			return err
		}
	}

	// Emit data statements after procedures (must not be reachable by fall-through).
	for _, ds := range dataStmts {
		if err := ds.Gen(g); err != nil {
			return err
		}
	}

	// Emit task bodies after procedures and data.
	for i := range c.TaskDefs {
		t := c.TaskDefs[i]
		g.Emitf("_plz_task_%d:\n", i)
		g.InTask = true
		for _, stmt := range t.Body {
			if err := stmt.Gen(g); err != nil {
				return err
			}
		}
		g.InTask = false
		g.Emitln("\tjp _plz_task_done")
	}

	// Emit string literal data in ROM (after all code, before RAM data).
	for _, s := range g.strings {
		g.Emitf("%s: db %d", s.label, len(s.data))
		for _, c := range s.data {
			g.Emitf(", %d", byte(c))
		}
		g.Emit("\n")
	}

	// Emit scheduler and task runtime.
	if len(c.TaskDefs) > 0 {
		g.Emit(SchedulerCode)
		g.Emitf("org 0x%x\n", g.Heap)
		g.Emitf("_plz_current_task: db 0\n")
		g.Emitf("_plz_sch_sp: dw 0\n")
		g.Emitf("_plz_tcbs: ds 128\n")
		for i := range c.TaskDefs {
			g.Emitf("_plz_task%d_stack: ds 128\n", i)
			g.Heap += 128
		}
		g.Heap += 131 // current_task(1) + sch_sp(2) + tcbs(128)
	}

	// Emit procedure parameter and local variable storage in RAM.
	for _, stmt := range p.Statements {
		proc, ok := stmt.Command.(Procedure)
		if !ok || proc.Reentrant {
			continue
		}
		for i, param := range proc.Parameters {
			psize := proc.paramByteSize(i)
			g.emitStorage("_plz_%s_%s", psize, proc.Name.Name, param)
		}
		var emitLocals func(stmts []Statement, depth int)
		emitLocals = func(stmts []Statement, depth int) {
			for _, s := range stmts {
				switch cmd := s.Command.(type) {
				case Declare:
					var label string
					if depth == 0 {
						label = fmt.Sprintf("_plz_%s_%s", proc.Name.Name, cmd.Identifier)
					} else {
						label = fmt.Sprintf("_plz_%s_%d_%s", proc.Name.Name, depth, cmd.Identifier)
					}
					g.emitStorageRaw(label, cmd.StorageSize())
				case Group:
					emitLocals(cmd.Statements, depth+1)
				}
			}
		}
		emitLocals(proc.Statements, 0)
	}

	// Emit data items (AT directives and global declarations) at the end in
	// source order. Each AT directive sets the ORG address; each DECLARE
	// allocates storage at the current heap position.
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

	return nil
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

// Gen generates assembly for an IF statement. It evaluates the condition, jumps
// to the else branch if false, emits the then-body, optionally emits the else-
// body, then jumps past the else to end.
func (s If) Gen(g *Gen) error {
	if err := s.Condition.Gen(g); err != nil {
		return err
	}
	n := g.nextLabel()
	g.Emitln("\tld a, h")
	g.Emitln("\tor l")
	g.Emitf("\tjr z, _else_%d\n", n)
	if err := s.Then.Gen(g); err != nil {
		return err
	}
	if s.Else != nil {
		g.Emitf("\tjr _end_%d\n", n)
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
// ROM and load HL with its address. Variable references load the value from
// memory (byte references zero-extend through A). Parenthesized sub-expressions
// recurse.
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

// Gen generates assembly for a prefix expression. OperatorNEG computes two's
// complement negation of the operand into HL. OperatorNOT computes logical
// negation (0 → 1, non-zero → 0).
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
		n := g.nextLabel()
		g.Emitln("\tld a, h")
		g.Emitln("\tor l")
		g.Emitf("\tld hl, 0\n")
		g.Emitf("\tjr nz, _lbl_%d\n", n)
		g.Emitln("\tinc l")
		g.Emitf("_lbl_%d:\n", n)
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

// Gen generates assembly for an infix expression. It evaluates the left operand
// into HL, pushes it, evaluates the right operand into HL, moves it to DE, pops
// the left operand back into HL, then emits the operation-specific code. Binary
// arithmetic, shifts, bitwise operations, comparisons, and multiplication/division/
// modulus all delegate to inline code or runtime calls.
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
		g.Emitln("\tadd hl, de")

	case OperatorSUB:
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")

	case OperatorShiftLeft:
		if n := constShift(i.Operands[1]); n >= 0 {
			for j := 0; j < n; j++ {
				g.Emitln("\tadd hl, hl")
			}
		} else {
			loop := g.nextLabel()
			end := g.nextLabel()
			g.Emitln("\tld a, e")
			g.Emitf("_lbl_%d:\n", loop)
			g.Emitln("\tor a")
			g.Emitf("\tjr z, _lbl_%d\n", end)
			g.Emitln("\tadd hl, hl")
			g.Emitln("\tdec a")
			g.Emitf("\tjr _lbl_%d\n", loop)
			g.Emitf("_lbl_%d:\n", end)
		}

	case OperatorShiftRight:
		if n := constShift(i.Operands[1]); n >= 0 {
			for j := 0; j < n; j++ {
				g.Emitln("\tsrl h")
				g.Emitln("\trr l")
			}
		} else {
			loop := g.nextLabel()
			end := g.nextLabel()
			g.Emitln("\tld a, e")
			g.Emitf("_lbl_%d:\n", loop)
			g.Emitln("\tor a")
			g.Emitf("\tjr z, _lbl_%d\n", end)
			g.Emitln("\tsrl h")
			g.Emitln("\trr l")
			g.Emitln("\tdec a")
			g.Emitf("\tjr _lbl_%d\n", loop)
			g.Emitf("_lbl_%d:\n", end)
		}

	case OperatorAND:
		g.Emitln("\tld a, h")
		g.Emitln("\tand d")
		g.Emitln("\tld h, a")
		g.Emitln("\tld a, l")
		g.Emitln("\tand e")
		g.Emitln("\tld l, a")

	case OperatorOR:
		g.Emitln("\tld a, h")
		g.Emitln("\tor d")
		g.Emitln("\tld h, a")
		g.Emitln("\tld a, l")
		g.Emitln("\tor e")
		g.Emitln("\tld l, a")

	case OperatorXOR:
		g.Emitln("\tld a, h")
		g.Emitln("\txor d")
		g.Emitln("\tld h, a")
		g.Emitln("\tld a, l")
		g.Emitln("\txor e")
		g.Emitln("\tld l, a")

	case OperatorMUL:
		g.Emitln("\tcall _plz_mul")
	case OperatorDIV:
		g.Emitln("\tcall _plz_div")
	case OperatorMOD:
		g.Emitln("\tcall _plz_mod")

	case OperatorEQU:
		g.Emitln("\tcall _plz_eq")
	case OperatorNEQ:
		g.Emitln("\tcall _plz_ne")
	case OperatorGT:
		g.Emitln("\tcall _plz_gt")
	case OperatorLT:
		g.Emitln("\tcall _plz_lt")
	case OperatorGTE:
		g.Emitln("\tcall _plz_gte")
	case OperatorLTE:
		g.Emitln("\tcall _plz_lte")
	}

	return nil
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
	if !ok || t.Record() == nil {
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
func (g *Gen) genIndexAddr(operands []Operand) (elemSize int, err error) {
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

	elem := 2 // default fallback
	if baseSuffix != nil && baseSuffix.Operator == OperatorFIELD {
		// Field expression base (e.g., rec.arr[i]): compute field address first.
		// genFieldAddr returns the field type, which may be an array.
		ft, err := g.genFieldAddr(baseSuffix.Operands)
		if err != nil {
			return 0, err
		}
		if ft.Array() != nil {
			if ft.Array().ElemType.Predeclared() == PredeclaredByte {
				elem = 1
			} else if ft.Array().ElemType.Record() != nil {
				elem = ft.Array().ElemType.Record().TotalSize()
				elem = nextPow2(elem)
			} else {
				elem = 2
			}
		}
	} else if ref != nil {
		elem = g.elemSize(ref.Identifier)
		if g.isParamRef(ref.Identifier) {
			g.Emitf("\tld hl, (%s)\n", g.localSym(ref.Identifier))
			goto gotAddr
		}
		g.Emitf("\tld hl, %s\n", g.localSym(ref.Identifier))
	} else {
		return 0, fmt.Errorf("genIndexAddr: first operand must be a reference or field expression")
	}
gotAddr:

	if len(operands) >= 2 {
		g.Emitln("\tpush hl")
		if err := operands[1].Expr().Gen(g); err != nil {
			return 0, err
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

	// Look up the procedure definition for param type info.
	proc, ok := g.Checker.Procedures[name]

	genCallArg := func(i int) error {
		if ok && i < len(proc.ParamTypes) {
			pt := proc.ParamTypes[i]
			if pt.Record() != nil || pt.Predeclared() == PredeclaredData {
				refArg := args[i].Ref()
				if refArg != nil && refArg.Identifier != "" && len(refArg.Fields) == 0 && len(refArg.Subscripts) == 0 {
					if g.isParamRef(refArg.Identifier) {
						g.Emitf("\tld hl, (%s)\n", g.localSym(refArg.Identifier))
					} else {
						g.Emitf("\tld hl, %s\n", g.localSym(refArg.Identifier))
					}
					return nil
				}
				// Non-trivial argument: compute address instead of value.
				if expr := args[i].Expr(); expr != nil {
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
				}
				return fmt.Errorf("cannot take address of argument %d", i)
			}
		}
		return args[i].Expr().Gen(g)
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
		isReentrant := !ok || proc.Reentrant
		var totalExtra int
		if !isReentrant {
			for i := 2; i < len(args); i++ {
				if err := genCallArg(i); err != nil {
					return err
				}
				paramName := proc.Parameters[i]
				if ok && i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared() == PredeclaredByte {
					g.Emitf("\tld a, l\n\tld (_plz_%s_%s), a\n", name, paramName)
				} else {
					g.Emitf("\tld (_plz_%s_%s), hl\n", name, paramName)
				}
			}
		} else {
			for i := len(args) - 1; i >= 2; i-- {
				if err := genCallArg(i); err != nil {
					return err
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
		if err := genCallArg(1); err != nil {
			return err
		}
		g.Emitln("\tpush hl")
		if err := genCallArg(0); err != nil {
			return err
		}
		g.Emitln("\tpop de")

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
// word targets store both bytes.
func (s Let) Gen(g *Gen) error {
	// Evaluate RHS into hl.
	if err := s.Expression.Gen(g); err != nil {
		return err
	}

	// Determine base decl type for record field access.
	t, _ := g.localType(s.Identifier)

	// Three cases:
	// 1. Simple store (no subscripts, no fields)
	// 2. Array of records field store: arr[i].x (both subscripts and fields)
	// 3. Subscripted field store: rec.arr[i] (fields then subscripts, field is array)
	// 4. Simple field store: rec.x (fields only)
	// 5. Simple array store: arr[i] (subscripts only)

	if len(s.Subscripts) == 0 && len(s.Fields) == 0 {
		// Simple variable store.
		if s.isByteRef(g) {
			g.Emitln("\tld a, l")
			g.Emitf("\tld (%s), a\n", g.localSym(s.Identifier))
		} else {
			g.Emitf("\tld (%s), hl\n", g.localSym(s.Identifier))
		}
		return nil
	}

	// Save RHS value.
	g.Emitln("\tpush hl")

	// Compute target address.
	if len(s.Subscripts) > 0 && len(s.Fields) > 0 {
		// Two cases look the same in the flat Reference:
		//   arr[i].x  -> subscripts index the base array, field accesses the result record
		//   rec.arr[i] -> field accesses the base record, subscripts index the result array
		// Distinguish: if the first field has an Array type, it's rec.arr[i].
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
			// arr[i].x: array of records, field write.
			if t.Record() == nil {
				return fmt.Errorf("let: %s is not an array of records", s.Identifier)
			}
			recSize := g.elemSize(s.Identifier)
			fname := s.Fields[0]
			fieldIdx := -1
			for i, f := range t.Record().Fields {
				if f.Identifier == fname {
					fieldIdx = i
					break
				}
			}
			if fieldIdx < 0 {
				return fmt.Errorf("let: struct %s has no field %s", s.Identifier, fname)
			}
			off := t.Record().FieldOffset(fieldIdx)
			ft := t.Record().Fields[fieldIdx].Type

			if g.isParamRef(s.Identifier) {
				g.Emitf("\tld hl, (%s)\n", g.localSym(s.Identifier))
			} else {
				g.Emitf("\tld hl, %s\n", g.localSym(s.Identifier))
			}
			// Add scaled index.
			if len(s.Subscripts) >= 1 {
				g.Emitln("\tpush hl")
				if err := s.Subscripts[0].Gen(g); err != nil {
					return err
				}
				for size := recSize; size > 1; size >>= 1 {
					g.Emitln("\tadd hl, hl")
				}
				g.Emitln("\tex de, hl")
				g.Emitln("\tpop hl")
				g.Emitln("\tadd hl, de")
			}
			// Add field offset.
			if off > 0 {
				g.Emitf("\tld de, %d\n", off)
				g.Emitln("\tadd hl, de")
			}
			g.Emitln("\tpop de") // value to store
			if ft.Predeclared() == PredeclaredByte {
				g.Emitln("\tld (hl), e")
			} else {
				g.Emitln("\tld (hl), e")
				g.Emitln("\tinc hl")
				g.Emitln("\tld (hl), d")
			}
			return nil
		}
	}

	if len(s.Fields) > 0 {
		// Struct field store: s.field = rhs (possibly with subscripts if field is array).
		if t.Record() == nil {
			return fmt.Errorf("let: %s is not a struct", s.Identifier)
		}
		fname := s.Fields[0]
		fieldIdx := -1
		for i, f := range t.Record().Fields {
			if f.Identifier == fname {
				fieldIdx = i
				break
			}
		}
		if fieldIdx < 0 {
			return fmt.Errorf("let: struct %s has no field %s", s.Identifier, fname)
		}
		off := t.Record().FieldOffset(fieldIdx)
		ft := t.Record().Fields[fieldIdx].Type

		if g.isParamRef(s.Identifier) {
			g.Emitf("\tld hl, (%s)\n", g.localSym(s.Identifier))
		} else {
			g.Emitf("\tld hl, %s\n", g.localSym(s.Identifier))
		}
		if off > 0 {
			g.Emitf("\tld de, %d\n", off)
			g.Emitln("\tadd hl, de")
		}
		// If field is an array and there are subscripts, add scaled index.
		if ft.Array() != nil && len(s.Subscripts) > 0 {
			arrElemSize := 1
			if ft.Array().ElemType.Predeclared() == PredeclaredWord {
				arrElemSize = 2
			} else if ft.Array().ElemType.Record() != nil {
				arrElemSize = ft.Array().ElemType.Record().TotalSize()
				arrElemSize = nextPow2(arrElemSize)
			}
			g.Emitln("\tpush hl")
			if err := s.Subscripts[0].Gen(g); err != nil {
				return err
			}
			for size := arrElemSize; size > 1; size >>= 1 {
				g.Emitln("\tadd hl, hl")
			}
			g.Emitln("\tex de, hl")
			g.Emitln("\tpop hl")
			g.Emitln("\tadd hl, de")
		}
		g.Emitln("\tpop de")
		if ft.Predeclared() == PredeclaredByte {
			g.Emitln("\tld (hl), e")
		} else if ft.Array() != nil && ft.Array().ElemType.Predeclared() == PredeclaredByte {
			g.Emitln("\tld (hl), e")
		} else {
			g.Emitln("\tld (hl), e")
			g.Emitln("\tinc hl")
			g.Emitln("\tld (hl), d")
		}
		return nil
	}

	// Array element set: lhs[i] = rhs

	elem := g.elemSize(s.Identifier)
	if g.isParamRef(s.Identifier) {
		g.Emitf("\tld hl, (%s)\n", g.localSym(s.Identifier))
	} else {
		g.Emitf("\tld hl, %s\n", g.localSym(s.Identifier))
	}
	for i := range s.Subscripts {
		g.Emitln("\tpush hl")
		if err := s.Subscripts[i].Gen(g); err != nil {
			return err
		}
		for size := elem; size > 1; size >>= 1 {
			g.Emitln("\tadd hl, hl")
		}
		g.Emitln("\tex de, hl")
		g.Emitln("\tpop hl")
		g.Emitln("\tadd hl, de")
	}

	g.Emitln("\tpop de")
	if elem == 1 {
		g.Emitln("\tld (hl), e")
	} else {
		g.Emitln("\tld (hl), e")
		g.Emitln("\tinc hl")
		g.Emitln("\tld (hl), d")
	}
	return nil
}

// Gen generates assembly for a group statement. It handles three forms:
// WHILE loops (condition checked at top), FOR loops (with start, end, optional
// step), and bare DO...END compound blocks (introducing a new scope).
func (s Group) Gen(g *Gen) error {
	switch {
	case s.While != nil:
		n := g.nextLabel()
		g.Emitf("_while_%d:\n", n)
		if err := s.While.Expression.Gen(g); err != nil {
			return err
		}
		g.Emitln("\tld a, h")
		g.Emitln("\tor l")
		g.Emitf("\tjr z, _end_%d\n", n)
		g.pushScope()
		defer g.popScope()
		for _, stmt := range s.Statements {
			if err := stmt.Gen(g); err != nil {
				return err
			}
		}
		g.Emitf("\tjr _while_%d\n", n)
		g.Emitf("_end_%d:\n", n)

	case s.For != nil:
		// Evaluate step (default 1), push on stack
		if s.For.By != nil {
			if err := s.For.By.Gen(g); err != nil {
				return err
			}
		} else {
			g.Emitln("\tld hl, 1")
		}
		g.Emitln("\tpush hl")

		// Evaluate end, push on stack
		if err := s.For.To.Gen(g); err != nil {
			return err
		}
		g.Emitln("\tpush hl")

		// Initialize var = start
		if err := s.For.Start.Gen(g); err != nil {
			return err
		}
		g.Emitf("\tld (%s), hl\n", g.localSym(s.For.Reference.Identifier))

		n := g.nextLabel()
		g.Emitf("_for_%d:\n", n)
		// Compare var with end (hl = end - var)
		g.Emitln("\tpop de")                                               // de = end, stack: [step]
		g.Emitln("\tpush de")                                              // push back, stack: [step, end]
		g.Emitf("\tld hl, (%s)\n", g.localSym(s.For.Reference.Identifier)) // hl = var
		g.Emitln("\tex de, hl")                                            // hl = end, de = var
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")        // hl = end - var
		g.Emitf("\tjr c, _end_%d\n", n) // end < var → exit

		// Body
		g.pushScope()
		defer g.popScope()
		for _, stmt := range s.Statements {
			if err := stmt.Gen(g); err != nil {
				return err
			}
		}

		// var += step
		g.Emitln("\tpop de")  // de = end, stack: [step]
		g.Emitln("\tpop hl")  // hl = step, stack: []
		g.Emitln("\tpush hl") // push step back, stack: [step]
		g.Emitln("\tpush de") // push end back, stack: [step, end]
		g.Emitf("\tld de, (%s)\n", g.localSym(s.For.Reference.Identifier))
		g.Emitln("\tadd hl, de") // hl = step + var
		g.Emitf("\tld (%s), hl\n", g.localSym(s.For.Reference.Identifier))
		g.Emitf("\tjr _for_%d\n", n)
		g.Emitf("_end_%d:\n", n)
		g.Emitln("\tpop hl") // discard end
		g.Emitln("\tpop hl") // discard step

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
				g.Emitf("\tjr z, %s\n", branchLabel)
			}
		}

		// No match: clean stack.
		g.Emitln("\tpop hl")

		// Compute the default jump target (default label or end).
		nomatchLabel := endLabel
		if s.Case.Default != nil {
			nomatchLabel = fmt.Sprintf("_case_dflt_%d", n)
		}
		g.Emitf("\tjr %s\n", nomatchLabel)

		// Emit branch bodies (all jump to end).
		for i, branch := range s.Case.Branches {
			label := fmt.Sprintf("_case_%d_%d", n, i)
			g.Emitf("%s:\n", label)
			g.Emitln("\tpop hl")
			if err := branch.Statement.Gen(g); err != nil {
				return err
			}
			g.Emitf("\tjr %s\n", endLabel)
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

	// Push scope and register parameters so localSym/localType can find them.
	g.pushScope()
	defer g.popScope()
	for i, param := range s.Parameters {
		ptype := Type{Typ: &PredeclaredType{Kind: PredeclaredWord}}
		if i < len(s.ParamTypes) {
			ptype = s.ParamTypes[i]
		}
		g.symStack[len(g.symStack)-1][param] = symEntry{
			label:    fmt.Sprintf("_plz_%s_%s", s.Name.Name, param),
			typ:      ptype,
			paramRef: ptype.Record() != nil || ptype.Predeclared() == PredeclaredData,
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

	// Look up the procedure definition for param type info.
	proc, procOk := g.Checker.Procedures[name]

	// genCallArg emits code to load the i-th argument into HL.
	// For RECORD params it loads the ADDRESS (not the value).
	genCallArg := func(i int) error {
		if i < len(proc.ParamTypes) {
			pt := proc.ParamTypes[i]
			if pt.Record() != nil || pt.Predeclared() == PredeclaredData {
				// Load address of record or DATA argument.
				ref := args[i].Ref()
				if ref != nil && ref.Identifier != "" && len(ref.Fields) == 0 && len(ref.Subscripts) == 0 {
					if g.isParamRef(ref.Identifier) {
						g.Emitf("\tld hl, (%s)\n", g.localSym(ref.Identifier))
					} else {
						g.Emitf("\tld hl, %s\n", g.localSym(ref.Identifier))
					}
					return nil
				}
				// Non-trivial argument: compute address instead of value.
				if suff, ok := args[i].Expr.(*Suffix); ok {
					switch suff.Operator {
					case OperatorINDEX:
						_, err := g.genIndexAddr(suff.Operands)
						return err
					case OperatorFIELD:
						_, err := g.genFieldAddr(suff.Operands)
						return err
					}
				}
				return fmt.Errorf("cannot take address of argument %d", i)
			}
		}
		return args[i].Gen(g)
	}

	switch len(args) {
	case 0:
		// No arguments.
	case 1:
		if err := genCallArg(0); err != nil {
			return err
		}
	case 2:
		// Arg2 into DE, arg1 into HL.
		if err := genCallArg(1); err != nil {
			return err
		}
		g.Emitln("\tpush hl")
		if err := genCallArg(0); err != nil {
			return err
		}
		g.Emitln("\tpop de")
	default:
		// 3+ arguments.
		isReentrant := !procOk || proc.Reentrant
		var totalExtra int // total bytes of extra args for REENTRANT cleanup
		if !isReentrant {
			// Non-REENTRANT: store extra args in the procedure's individual labels.
			for i := 2; i < len(args); i++ {
				if err := genCallArg(i); err != nil {
					return err
				}
				paramName := proc.Parameters[i]
				if i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared() == PredeclaredByte {
					g.Emitf("\tld a, l\n\tld (_plz_%s_%s), a\n", name, paramName)
				} else {
					g.Emitf("\tld (_plz_%s_%s), hl\n", name, paramName)
				}
			}
		} else {
			// REENTRANT or no proc info: push remaining args onto stack right-to-left.
			for i := len(args) - 1; i >= 2; i-- {
				if err := genCallArg(i); err != nil {
					return err
				}
				psize := 2
				if procOk && i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared() == PredeclaredByte {
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
		// Set up HL=arg1, DE=arg2.
		if err := genCallArg(1); err != nil {
			return err
		}
		g.Emitln("\tpush hl")
		if err := genCallArg(0); err != nil {
			return err
		}
		g.Emitln("\tpop de")

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

// Gen generates assembly for a GOTO statement, emitting a JP to the named label.
func (s GoTo) Gen(g *Gen) error {
	g.Emit("\tjp ")
	g.Emitln(s.Name)
	return nil
}

// Gen generates assembly for an AT directive, emitting an ORG directive at
// the specified address and updating the heap pointer so subsequent data
// declarations are placed there.
func (s At) Gen(g *Gen) error {
	addr, err := g.evalConstLiteral(s.Address)
	if err != nil {
		return err
	}
	g.Emitf("org 0x%x\n", addr)
	g.Heap = addr
	return nil
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

// evalConstLiteral evaluates a Literal to an integer at codegen time.
// The literal must be a number (NumberLit) or a reference to a named
// constant (ReferenceLit) that has been registered during semantic checking.
func (g *Gen) evalConstLiteral(l Literal) (int, error) {
	if n := l.Number(); n != nil {
		return n.Value, nil
	}
	if r := l.Reference(); r != nil {
		id := r.Value.Identifier
		if lit, ok := g.Checker.Constants[id]; ok {
			if n := lit.Number(); n != nil {
				return n.Value, nil
			}
			return 0, fmt.Errorf("AT: constant %q is not a number", id)
		}
		return 0, fmt.Errorf("AT: undefined constant %q", id)
	}
	return 0, fmt.Errorf("AT: expected a number or constant, got text literal")
}

// Gen generates assembly for a DECLARE statement. Inside a procedure body it
// registers the identifier in the current scope so later references resolve to
// the correct RAM label. At global scope it emits an ORG directive followed by
// zero-initialized storage.
func (s Declare) Gen(g *Gen) error {
	if g.procName != "" {
		// Local variable: register in the current scope so localSym/localType can find it.
		var label string
		if len(g.symStack) <= 1 {
			label = fmt.Sprintf("_plz_%s_%s", g.procName, s.Identifier)
		} else {
			label = fmt.Sprintf("_plz_%s_%d_%s", g.procName, len(g.symStack)-1, s.Identifier)
		}
		g.symStack[len(g.symStack)-1][s.Identifier] = symEntry{
			label: label,
			typ:   s.Type,
		}
		return nil
	}
	elemSize := g.elemSize(s.Identifier)
	total := s.Size * elemSize
	if total == 0 {
		total = elemSize // unbounded → 1 element minimum
	}

	// If AT is set, the variable is at a fixed absolute address.
	// Do not advance g.Heap so subsequent declarations are unaffected.
	if s.At != nil {
		addr, err := g.evalConstLiteral(*s.At)
		if err != nil {
			return err
		}
		g.Emitf("org 0x%x\n%s: db 0", addr, s.Identifier)
		for i := 1; i < total; i++ {
			g.Emit(", 0")
		}
		g.Emit("\n")
		return nil
	}

	g.Emitf("org 0x%x\n%s: db 0", g.Heap, s.Identifier)
	for i := 1; i < total; i++ {
		g.Emit(", 0")
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

// Gen generates assembly for a DATA declaration, emitting DB directives for
// numeric literals and DS directives for text literals.
func (s Data) Gen(g *Gen) error {
	g.Emitf("%s:\n", s.Name)
	for _, lit := range s.Literals {
		if n := lit.Number(); n != nil {
			g.Emitf("\tdb %d\n", n.Value)
		} else if t := lit.Text(); t != nil {
			g.Emitf("\tds %s\n", strconv.Quote(t.Value))
		}
	}
	return nil
}

// Gen generates assembly for a CONSTANT declaration, emitting a const directive
// with the literal value in hexadecimal or quoted form.
func (s Constant) Gen(g *Gen) error {
	if n := s.Literal.Number(); n != nil {
		g.Emitf("\tconst %s = %x\n", s.Name, n.Value)
	} else if t := s.Literal.Text(); t != nil {
		g.Emitf("\tconst %s = %s\n", s.Name, strconv.Quote(t.Value))
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
// tasks' counters (waking those that reach zero), performs round-robin task
// selection among ready tasks, and restores the chosen task's stack pointer.
const SchedulerCode = `
// -------------------------------------------------------------------
// Task scheduler: save current SP, pick next ready task, restore SP
// -------------------------------------------------------------------
_plz_scheduler:
	// Compute current TCB address = _plz_tcbs + current_task * 8
	ld a, (_plz_current_task)
	ld l, a
	ld h, 0
	add hl, hl
	add hl, hl
	add hl, hl
	ld de, _plz_tcbs
	add hl, de
	// Save SP at TCB[0-1]
	ld (_plz_sch_sp), sp
	ld de, (_plz_sch_sp)
	ld (hl), e
	inc hl
	ld (hl), d
	// HL now at TCB+2 (state byte) - not needed

	// Decrement all sleeping tasks' sleep counters.
	// When a counter reaches 0, set state to ready.
	ld hl, _plz_tcbs+3
	ld b, 16
	ld de, 8
_slp_dec_loop:
	ld a, (hl)
	or a
	jr z, _slp_dec_cont
	dec (hl)
	jr nz, _slp_dec_cont
	// Sleep counter hit 0 => set state to ready
	dec hl
	ld (hl), 0
	inc hl
_slp_dec_cont:
	add hl, de
	djnz _slp_dec_loop

	// Find next ready task (round-robin, start from current+1)
	ld a, (_plz_current_task)
	ld c, a
	inc a
	cp 16
	jr c, _sch_ok
	xor a
_sch_ok:
	ld (_plz_current_task), a
	ld b, 16
_sch_loop:
	ld a, (_plz_current_task)
	ld l, a
	ld h, 0
	add hl, hl
	add hl, hl
	add hl, hl
	ld de, _plz_tcbs+2
	add hl, de
	ld a, (hl)
	or a
	jr z, _sch_found
	ld a, (_plz_current_task)
	inc a
	cp 16
	jr c, _sch_next
	xor a
_sch_next:
	ld (_plz_current_task), a
	djnz _sch_loop
	// No ready task found - halt
	halt
	jp _plz_scheduler
_sch_found:
	// Restore SP from TCB
	ld a, (_plz_current_task)
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

`
