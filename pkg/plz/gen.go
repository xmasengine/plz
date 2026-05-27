package plz

import "fmt"
import "os"
import "strconv"

const HeapBase = 0xC000 // RAM memory.

type Gen struct {
	*os.File
	Heap           int // Pointer to last allocated heap RAM memory.
	label          int // counter for unique local labels
	Checker        *Checker
	procFrame      map[string]int // proc name → frame heap base addr
	InProcedure    bool           // set when generating inside a procedure body
	ProcReturnType Type           // return type of current procedure (for BYTE zero-extend in Return.Gen)
}

func NewGenFile(name string) (*Gen, error) {
	fout, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	return NewGen(fout), nil
}

func NewGenTmp() (*Gen, error) {
	fout, err := os.CreateTemp("", "plz_*.asm")
	if err != nil {
		return nil, err
	}
	return NewGen(fout), nil
}

func NewGen(fout *os.File) *Gen {
	res := &Gen{File: fout, Heap: HeapBase}
	return res
}

func (g *Gen) Close() error {
	return g.File.Close()
}

func (g *Gen) Emitf(form string, args ...any) (int, error) {
	return fmt.Fprintf(g.File, form, args...)
}

func (g *Gen) Emitln(args ...any) (int, error) {
	return fmt.Fprintln(g.File, args...)
}

func (g *Gen) Emit(args ...any) (int, error) {
	return fmt.Fprint(g.File, args...)
}

func (g *Gen) nextLabel() int {
	g.label++
	return g.label
}

const ProgramHeader = `org 0x0000
// Boot section
org 0x0000
    jp main         // Jump to main program

// Interrupt handler
org 0x0038
	reti // do nothing for now

// NMI or pause button handler
org 0x0066
    retn // Do nothing

// Main program
main:
    di            // Disable interrupts
    im 1          // Interrupt mode 1
    ld sp, 0xdff0 // Set up stack pointer at end of RAM.
`

const ProgramFooter = `
org 0xC000 // RAM memory.
`

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

func (p Program) Gen(g *Gen) error {
	c := NewChecker()
	if err := p.Check(c); err != nil {
		return err
	}
	g.Checker = c
	g.procFrame = make(map[string]int)

	// First pass: allocate frames for non-REENTRANT procedures.
	for _, stmt := range p.Statements {
		if stmt.Procedure != nil && !stmt.Procedure.Reentrant {
		proc := stmt.Procedure
		localsSize := 0
		for _, local := range proc.Locals {
			localsSize += localDeclareSize(local)
		}
		total := totalParamSize(proc) + localsSize
			if total > 0 {
				g.procFrame[proc.Name.Name] = g.Heap
				g.Heap += total
			}
		}
	}

	// Emit procedure frame storage and parameter/locals consts first (must precede use).
	for _, stmt := range p.Statements {
		if stmt.Procedure == nil || stmt.Procedure.Reentrant {
			continue
		}
		proc := stmt.Procedure
		addr, ok := g.procFrame[proc.Name.Name]
		if !ok {
			continue
		}
		// Compute total frame size.
		localsSize := 0
		for _, local := range proc.Locals {
			localsSize += localDeclareSize(local)
		}
		total := totalParamSize(proc) + localsSize
		if total == 0 {
			continue
		}
		g.Emitf("org 0x%x\n_plz_%s_frame: db 0", addr, proc.Name.Name)
		for i := 1; i < total; i++ {
			g.Emit(", 0")
		}
		g.Emit("\n")
		for i, param := range proc.Parameters {
			g.Emitf("\tconst %s = _plz_%s_frame+%d\n", param, proc.Name.Name, paramOffset(proc, i))
		}
		// Emit const mappings for local variables.
		off := totalParamSize(proc)
		for _, local := range proc.Locals {
			g.Emitf("\tconst %s = _plz_%s_frame+%d\n", local.Identifier, proc.Name.Name, off)
			off += localDeclareSize(local)
		}
	}

	g.Emit(ProgramHeader)
	g.Emitln("\tjp _plz_start")
	g.Emit(RuntimeHeader)
	g.Emitln("_plz_start:")

	var declares []*Declare
	var procedures []*Procedure
	for _, statement := range p.Statements {
		if statement.Declare != nil {
			declares = append(declares, statement.Declare)
			continue
		}
		if statement.Procedure != nil {
			procedures = append(procedures, statement.Procedure)
			continue
		}
		if err := statement.Gen(g); err != nil {
			return err
		}
	}

	// Emit procedure bodies after main code (must not be reachable by fall-through).
	for _, proc := range procedures {
		if err := proc.Gen(g); err != nil {
			return err
		}
	}

	// Emit declarations at the end.
	for _, d := range declares {
		g.Emitln("")
		d.Gen(g)
	}
	return nil
}

func (s Statement) Gen(g *Gen) error {
	if s.Label != nil {
		s.Label.Gen(g)
	}
	switch {
	case s.If != nil:
		return s.If.Gen(g)
	case s.Let != nil:
		return s.Let.Gen(g)
	case s.Constant != nil:
		return s.Constant.Gen(g)
	case s.Declare != nil:
		return s.Declare.Gen(g)
	case s.Group != nil:
		return s.Group.Gen(g)
	case s.Procedure != nil:
		return s.Procedure.Gen(g)
	case s.Return != nil:
		return s.Return.Gen(g)
	case s.Call != nil:
		return s.Call.Gen(g)
	case s.GoTo != nil:
		return s.GoTo.Gen(g)
	case s.Halt != nil:
		return s.Halt.Gen(g)
	case s.Enable != nil:
		return s.Enable.Gen(g)
	case s.Data != nil:
		return s.Data.Gen(g)
	case s.Define != nil:
		return nil // compile-time only
	case s.Disable != nil:
		return s.Disable.Gen(g)
	case s.Output != nil:
		return s.Output.Gen(g)
	default:
		g.Emitf("// statement not implemented: %v\n", s)
	}
	return nil
}

const maxUint16 = 1 << 16

func (l Label) Gen(g *Gen) error {
	if l.Location > 0 {
		org := l.Location % maxUint16
		target := l.Location / maxUint16
		if target > 0 {
			g.Emitf("org %x, %x\n", org, target)
		} else {
			g.Emitf("org %x\n", org)
		}
	}
	if l.Name != "" {
		g.Emitf("%s:", l.Name)
	}
	return nil
}

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

// Gen evaluates the expression and leaves the result in hl.
func (e Expression) Gen(g *Gen) error {
	switch {
	case e.Operand != nil:
		return e.Operand.Gen(g)
	case e.Prefix != nil:
		return e.Prefix.Gen(g)
	case e.Infix != nil:
		return e.Infix.Gen(g)
	case e.Suffix != nil:
		return e.Suffix.Gen(g)
	}
	return nil
}

func (o Operand) Gen(g *Gen) error {
	switch {
	case o.Literal != nil:
		if o.Literal.Number != nil {
			g.Emitf("\tld hl, %d\n", *o.Literal.Number)
		}
	case o.Reference != nil:
		if o.Reference.isByteRef(g) {
			g.Emitf("\tld a, (%s)\n", o.Reference.Identifier)
			g.Emitln("\tld l, a")
			g.Emitln("\tld h, 0")
		} else {
			g.Emitf("\tld hl, (%s)\n", o.Reference.Identifier)
		}
	case o.Expression != nil:
		return o.Expression.Gen(g)
	}
	return nil
}

func (r *Reference) isByteRef(g *Gen) bool {
	if g.Checker == nil || r == nil {
		return false
	}
	if len(r.Fields) > 0 {
		// Field access — check the field's type.
		d, ok := g.Checker.Symbols[r.Identifier]
		if !ok || d.Type.Record == nil {
			return false
		}
		fname := r.Fields[0]
		for _, f := range d.Type.Record.Fields {
			if f.Identifier == fname {
				return f.Type.Predeclared == PredeclaredByte
			}
		}
		return false
	}
	d, ok := g.Checker.Symbols[r.Identifier]
	if !ok {
		return false
	}
	return d.Type.Predeclared == PredeclaredByte
}

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

// genFieldRead generates code to read s.field into HL.
// Operands: [struct_ref, field_ref].
func (g *Gen) genFieldRead(operands []Operand) error {
	ref := operands[0].Reference
	if ref == nil && operands[0].Expression != nil && operands[0].Expression.Operand != nil {
		ref = operands[0].Expression.Operand.Reference
	}
	if ref == nil {
		return fmt.Errorf("genFieldRead: first operand must be a reference")
	}
	d, ok := g.Checker.Symbols[ref.Identifier]
	if !ok || d.Type.Record == nil {
		return fmt.Errorf("genFieldRead: %s is not a struct", ref.Identifier)
	}
	fname := operands[1].Reference.Identifier
	fieldIdx := -1
	for i, f := range d.Type.Record.Fields {
		if f.Identifier == fname {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 0 {
		return fmt.Errorf("genFieldRead: struct %s has no field %s", ref.Identifier, fname)
	}
	off := fieldOffset(d.Type.Record.Fields, fieldIdx)
	ft := d.Type.Record.Fields[fieldIdx].Type

	// ParamRef symbols hold a pointer; dereference it to get the data address.
	if d.ParamRef {
		g.Emitf("\tld hl, (%s)\n", ref.Identifier)
	} else {
		g.Emitf("\tld hl, %s\n", ref.Identifier)
	}
	if off > 0 {
		g.Emitf("\tld de, %d\n", off)
		g.Emitln("\tadd hl, de")
	}
	if ft.Predeclared == PredeclaredByte {
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

// elemSize returns the storage size of elements declared under name.
func (g *Gen) elemSize(name Identifier) int {
	if g.Checker == nil {
		return 2
	}
	d, ok := g.Checker.Symbols[name]
	if !ok {
		return 2
	}
	if d.Type.Record != nil {
		total := recordTotalSize(d.Type.Record.Fields)
		return nextPow2(total)
	}
	if d.Type.Predeclared == PredeclaredByte {
		return 1
	}
	return 2
}

// genIndexRead generates code to read arr[index].
// Operands: [array_expr, index_expr].
func (g *Gen) genIndexRead(operands []Operand) error {
	// Get the array base address into hl.
	// The first operand must be a reference (we take its address).
	ref := operands[0].Reference
	if ref == nil && operands[0].Expression != nil && operands[0].Expression.Operand != nil {
		ref = operands[0].Expression.Operand.Reference
	}
	if ref != nil {
		// ParamRef symbols hold a pointer; dereference it.
		if g.Checker != nil {
			if d, ok := g.Checker.Symbols[ref.Identifier]; ok && d.ParamRef {
				g.Emitf("\tld hl, (%s)\n", ref.Identifier)
				goto gotAddr
			}
		}
		g.Emitf("\tld hl, %s\n", ref.Identifier)
	} else {
		return fmt.Errorf("genIndexRead: first operand must be a reference")
	}
gotAddr:
	elem := g.elemSize(ref.Identifier)

	// If there's an index, add it (scaled by element size).
	if len(operands) >= 2 {
		g.Emitln("\tpush hl")
		if err := operands[1].Expression.Gen(g); err != nil {
			return err
		}
		if elem == 2 {
			g.Emitln("\tadd hl, hl") // index * 2 (word)
		}
		g.Emitln("\tex de, hl")
		g.Emitln("\tpop hl")
		g.Emitln("\tadd hl, de")
	}

	if elem == 1 {
		// Load byte, zero-extend.
		g.Emitln("\tld a, (hl)")
		g.Emitln("\tld l, a")
		g.Emitln("\tld h, 0")
	} else {
		// Load word.
		g.Emitln("\tld a, (hl)")
		g.Emitln("\tinc hl")
		g.Emitln("\tld h, (hl)")
		g.Emitln("\tld l, a")
	}
	return nil
}

// genCallExpr generates code to call a function expression.
// Operands: [func_expr, arg1, arg2, ...].
func (g *Gen) genCallExpr(operands []Operand) error {
	ref := operands[0].Reference
	if ref == nil && operands[0].Expression != nil && operands[0].Expression.Operand != nil {
		ref = operands[0].Expression.Operand.Reference
	}
	if ref == nil {
		return fmt.Errorf("genCallExpr: indirect calls not yet supported")
	}
	name := string(ref.Identifier)
	args := operands[1:]

	// Look up the procedure definition for param type info.
	proc, _ := g.Checker.Procedures[name]

	genCallArg := func(i int) error {
		if proc != nil && i < len(proc.ParamTypes) && proc.ParamTypes[i].Record != nil {
			refArg := args[i].Reference
			if refArg == nil && args[i].Expression != nil && args[i].Expression.Operand != nil {
				refArg = args[i].Expression.Operand.Reference
			}
			if refArg != nil && refArg.Identifier != "" && len(refArg.Fields) == 0 && len(refArg.Subscripts) == 0 {
				g.Emitf("\tld hl, %s\n", refArg.Identifier)
				return nil
			}
		}
		return args[i].Expression.Gen(g)
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
		_, hasFrame := g.procFrame[name]
		var totalExtra int
		if hasFrame {
			for i := 2; i < len(args); i++ {
				if err := genCallArg(i); err != nil {
					return err
				}
				off := paramOffset(proc, i)
				if proc != nil && i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared == PredeclaredByte {
					g.Emitf("\tld a, l\n\tld (_plz_%s_frame+%d), a\n", name, off)
				} else {
					g.Emitf("\tld (_plz_%s_frame+%d), hl\n", name, off)
				}
			}
		} else {
			for i := len(args) - 1; i >= 2; i-- {
				if err := genCallArg(i); err != nil {
					return err
				}
				psize := 2
				if proc != nil && i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared == PredeclaredByte {
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
		if !hasFrame && totalExtra > 0 {
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

func (s Let) Gen(g *Gen) error {
	// Evaluate RHS into hl.
	if err := s.Expression.Gen(g); err != nil {
		return err
	}

	if len(s.Fields) > 0 {
		// Struct field store: s.field = rhs
		// Compute base + field offset into hl, then pop value and store.
		d, ok := g.Checker.Symbols[s.Identifier]
		if !ok || d.Type.Record == nil {
			return fmt.Errorf("let: %s is not a struct", s.Identifier)
		}
		fname := s.Fields[0]
		fieldIdx := -1
		for i, f := range d.Type.Record.Fields {
			if f.Identifier == fname {
				fieldIdx = i
				break
			}
		}
		if fieldIdx < 0 {
			return fmt.Errorf("let: struct %s has no field %s", s.Identifier, fname)
		}
		off := fieldOffset(d.Type.Record.Fields, fieldIdx)
		ft := d.Type.Record.Fields[fieldIdx].Type

		g.Emitln("\tpush hl")
		// ParamRef symbols hold a pointer; dereference it to get the data address.
		if d.ParamRef {
			g.Emitf("\tld hl, (%s)\n", s.Identifier)
		} else {
			g.Emitf("\tld hl, %s\n", s.Identifier)
		}
		if off > 0 {
			g.Emitf("\tld de, %d\n", off)
			g.Emitln("\tadd hl, de")
		}
		g.Emitln("\tpop de")
		if ft.Predeclared == PredeclaredByte {
			g.Emitln("\tld (hl), e")
		} else {
			g.Emitln("\tld (hl), e")
			g.Emitln("\tinc hl")
			g.Emitln("\tld (hl), d")
		}
		return nil
	}

	if len(s.Subscripts) == 0 {
		// Simple variable store.
		if s.isByteRef(g) {
			g.Emitln("\tld a, l")
			g.Emitf("\tld (%s), a\n", s.Identifier)
		} else {
			g.Emitf("\tld (%s), hl\n", s.Identifier)
		}
		return nil
	}

	// Array element set: lhs[sub1][sub2]... = rhs
	// hl still holds the RHS value; save it.
	g.Emitln("\tpush hl")

	// Compute target address into hl.
	elem := g.elemSize(s.Identifier)
	if g.Checker != nil {
		if d, ok := g.Checker.Symbols[s.Identifier]; ok && d.ParamRef {
			g.Emitf("\tld hl, (%s)\n", s.Identifier)
			goto gotElem
		}
	}
	g.Emitf("\tld hl, %s\n", s.Identifier)
gotElem:
	for i := range s.Subscripts {
		g.Emitln("\tpush hl")
		if err := s.Subscripts[i].Gen(g); err != nil {
			return err
		}
		if elem == 2 {
			g.Emitln("\tadd hl, hl") // * 2 (word)
		}
		g.Emitln("\tex de, hl")
		g.Emitln("\tpop hl")
		g.Emitln("\tadd hl, de")
	}

	// hl = target address.  Pop the value from the stack.
	g.Emitln("\tpop de") // de = value to store
	if elem == 1 {
		g.Emitln("\tld (hl), e")
	} else {
		g.Emitln("\tld (hl), e")
		g.Emitln("\tinc hl")
		g.Emitln("\tld (hl), d")
	}
	return nil
}

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
		g.Emitf("\tld (%s), hl\n", s.For.Reference.Identifier)

		n := g.nextLabel()
		g.Emitf("_for_%d:\n", n)
		// Compare var with end (hl = end - var)
		g.Emitln("\tpop de")  // de = end, stack: [step]
		g.Emitln("\tpush de") // push back, stack: [step, end]
		g.Emitf("\tld hl, (%s)\n", s.For.Reference.Identifier) // hl = var
		g.Emitln("\tex de, hl") // hl = end, de = var
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de") // hl = end - var
		g.Emitf("\tjr c, _end_%d\n", n) // end < var → exit

		// Body
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
		g.Emitf("\tld de, (%s)\n", s.For.Reference.Identifier)
		g.Emitln("\tadd hl, de") // hl = step + var
		g.Emitf("\tld (%s), hl\n", s.For.Reference.Identifier)
		g.Emitf("\tjr _for_%d\n", n)
		g.Emitf("_end_%d:\n", n)
		g.Emitln("\tpop hl") // discard end
		g.Emitln("\tpop hl") // discard step

	default:
		// Bare DO...END: just emit statements
		for _, stmt := range s.Statements {
			if err := stmt.Gen(g); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s Procedure) Gen(g *Gen) error {
	g.Emitf("_plz_%s:\n", s.Name.Name)

	// Save return type so Return.Gen can zero-extend BYTE values.
	g.ProcReturnType = s.Type

	// For non-REENTRANT: save parameters to frame slots.
	if !s.Reentrant {
		if _, ok := g.procFrame[s.Name.Name]; ok && len(s.Parameters) > 0 {
			// Save param1 (HL) to frame slot 0.
			off0 := paramOffset(&s, 0)
			p0size := paramByteSize(&s, 0)
			if p0size == 1 {
				g.Emitf("\tld a, l\n\tld (_plz_%s_frame+%d), a\n", s.Name.Name, off0)
			} else {
				g.Emitf("\tld (_plz_%s_frame+%d), hl\n", s.Name.Name, off0)
			}
			if len(s.Parameters) > 1 {
				// Save param2 (DE) to frame slot.
				off1 := paramOffset(&s, 1)
				p1size := paramByteSize(&s, 1)
				if p1size == 1 {
					g.Emitf("\tld a, e\n\tld (_plz_%s_frame+%d), a\n", s.Name.Name, off1)
				} else {
					g.Emitf("\tld (_plz_%s_frame+%d), de\n", s.Name.Name, off1)
				}
			}
			// Params 3+ are already stored in frame by the call site.
		}
	}

	// Generate body with InProcedure set so local Declare.Gen emits no storage.
	g.InProcedure = true
	for _, stmt := range s.Statements {
		if err := stmt.Gen(g); err != nil {
			return err
		}
	}
	g.InProcedure = false

	// Implicit ret if the last statement is not a return.
	if len(s.Statements) == 0 || s.Statements[len(s.Statements)-1].Return == nil {
		g.Emitln("\tret")
	}
	return nil
}

func (s Return) Gen(g *Gen) error {
	switch len(s.Expressions) {
	case 0:
		// No return value → just ret.
	case 1:
		// For RECORD return type, return the ADDRESS of the record.
		if g.ProcReturnType.Record != nil {
			ref := s.Expressions[0].Operand.Reference
			if ref != nil && ref.Identifier != "" && len(ref.Fields) == 0 && len(ref.Subscripts) == 0 {
				g.Emitf("\tld hl, %s\n", ref.Identifier)
				break
			}
		}
		if err := s.Expressions[0].Gen(g); err != nil {
			return err
		}
		// Zero-extend BYTE return values.
		if g.ProcReturnType.Predeclared == PredeclaredByte {
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
	g.Emitln("\tret")
	return nil
}

func (s Call) Gen(g *Gen) error {
	name := s.Name
	args := s.Arguments

	// Look up the procedure definition for param type info.
	proc, _ := g.Checker.Procedures[name]

	// genCallArg emits code to load the i-th argument into HL.
	// For RECORD params it loads the ADDRESS (not the value).
	genCallArg := func(i int) error {
		if proc != nil && i < len(proc.ParamTypes) && proc.ParamTypes[i].Record != nil {
			// Load address of record argument.
			ref := args[i].Operand.Reference
			if ref != nil && ref.Identifier != "" && len(ref.Fields) == 0 && len(ref.Subscripts) == 0 {
				g.Emitf("\tld hl, %s\n", ref.Identifier)
				return nil
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
		_, hasFrame := g.procFrame[name]
		var totalExtra int // total bytes of extra args for REENTRANT cleanup
		if hasFrame {
			// Non-REENTRANT: store extra args in the procedure's frame.
			for i := 2; i < len(args); i++ {
				if err := genCallArg(i); err != nil {
					return err
				}
				off := paramOffset(proc, i)
				if proc != nil && i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared == PredeclaredByte {
					g.Emitf("\tld a, l\n\tld (_plz_%s_frame+%d), a\n", name, off)
				} else {
					g.Emitf("\tld (_plz_%s_frame+%d), hl\n", name, off)
				}
			}
		} else {
			// REENTRANT or no frame: push remaining args onto stack right-to-left.
			for i := len(args) - 1; i >= 2; i-- {
				if err := genCallArg(i); err != nil {
					return err
				}
				psize := 2
				if proc != nil && i < len(proc.ParamTypes) && proc.ParamTypes[i].Predeclared == PredeclaredByte {
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
		if !hasFrame && totalExtra > 0 {
			g.Emitf("\tld hl, %d\n", totalExtra)
			g.Emitln("\tadd hl, sp")
			g.Emitln("\tld sp, hl")
		}
		return nil
	}

	g.Emitf("\tcall _plz_%s\n", name)
	return nil
}

func (s GoTo) Gen(g *Gen) error {
	if s.Name != "" {
		g.Emitln("\tjp ", s.Name)
	} else {
		g.Emitln("\tjp ", s.Location)
	}
	return nil
}

func (s Declare) Gen(g *Gen) error {
	if g.InProcedure {
		// Local variable; storage is part of the procedure frame.
		return nil
	}
	elemSize := 1
	if s.Type.Predeclared == PredeclaredWord {
		elemSize = 2
	}
	total := s.Size * elemSize
	if total == 0 {
		total = elemSize // unbounded → 1 element minimum
	}
	// For structs, override total.
	if s.Type.Record != nil {
		total = recordTotalSize(s.Type.Record.Fields)
		total = nextPow2(total)
	}
	g.Emitf("org 0x%x\n%s: db 0", g.Heap, s.Identifier)
	for i := 1; i < total; i++ {
		g.Emit(", 0")
	}
	g.Emit("\n")
	g.Heap += total
	return nil
}

func (s Halt) Gen(g *Gen) error {
	g.Emitln("\thalt")
	return nil
}

func (s Enable) Gen(g *Gen) error {
	g.Emitln("\tei")
	return nil
}

func (s Disable) Gen(g *Gen) error {
	g.Emitln("\tdi")
	return nil
}

func (s Output) Gen(g *Gen) error {
	if err := s.Value.Gen(g); err != nil {
		return err
	}
	g.Emitf("\tld a, l\n")
	g.Emitf("\tout (%d), a\n", s.Port)
	return nil
}

// paramByteSize returns the storage size (in bytes) for the i-th parameter of proc.
// Records are passed by reference so they occupy 2 bytes (a pointer) in the frame.
func paramByteSize(proc *Procedure, i int) int {
	if i < len(proc.ParamTypes) {
		if proc.ParamTypes[i].Predeclared == PredeclaredByte {
			return 1
		}
		// Records and arrays are passed by reference → 2-byte pointer.
		if proc.ParamTypes[i].Record != nil {
			return 2
		}
	}
	return 2
}

// paramOffset returns the byte offset of the i-th parameter within the procedure frame.
func paramOffset(proc *Procedure, i int) int {
	off := 0
	for j := 0; j < i; j++ {
		off += paramByteSize(proc, j)
	}
	return off
}

// totalParamSize returns the total byte size of all parameters for a procedure.
func totalParamSize(proc *Procedure) int {
	total := 0
	for i := range proc.Parameters {
		total += paramByteSize(proc, i)
	}
	return total
}

// localDeclareSize returns the byte size needed for a local variable declaration.
func localDeclareSize(d Declare) int {
	elemSize := 1
	if d.Type.Predeclared == PredeclaredWord {
		elemSize = 2
	}
	total := d.Size * elemSize
	if total == 0 {
		total = elemSize // unbounded → 1 element minimum
	}
	if d.Type.Record != nil {
		total = recordTotalSize(d.Type.Record.Fields)
		total = nextPow2(total)
	}
	return total
}

// fieldOffset returns the byte offset of the i-th field within a struct.
func fieldOffset(fields []Field, i int) int {
	off := 0
	for j := 0; j < i; j++ {
		if fields[j].Type.Predeclared == PredeclaredByte {
			off += 1
		} else {
			off += 2
		}
	}
	return off
}

// fieldSize returns the storage size of a single field/type.
func fieldTypeSize(t Type) int {
	if t.Predeclared == PredeclaredByte {
		return 1
	}
	return 2
}

// recordTotalSize returns the raw sum of field sizes.
func recordTotalSize(fields []Field) int {
	s := 0
	for _, f := range fields {
		s += fieldTypeSize(f.Type)
	}
	return s
}

// nextPow2 rounds n up to the next power of two.
func nextPow2(n int) int {
	if n <= 0 {
		return 0
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

func (s Data) Gen(g *Gen) error {
	if s.Literal.Number != nil {
		g.Emitf("\tdb %x\n", *s.Literal.Number)
	} else if s.Literal.Text != nil {
		g.Emitf("\tds %s\n", strconv.Quote(*s.Literal.Text))
	}
	return nil
}

func (s Constant) Gen(g *Gen) error {
	if s.Literal.Number != nil {
		g.Emitf("\tconst %s = %x\n", s.Name, *s.Literal.Number)
	} else if s.Literal.Text != nil {
		g.Emitf("\tconst %s = %s\n", s.Name, strconv.Quote(*s.Literal.Text))
	}
	return nil
}
