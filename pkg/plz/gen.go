package plz

import "fmt"
import "os"
import "strconv"

const HeapBase = 0xC000 // RAM memory.

type Gen struct {
	*os.File
	Heap  int // Pointer to last allocated heap RAM memory.
	label int // counter for unique local labels
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
; -------------------------------------------------------------------
; PL/Z runtime helpers
; -------------------------------------------------------------------

; _plz_mul: HL = HL * DE (unsigned 16-bit)
_plz_mul:
	push bc
	push hl
	pop bc          ; bc = multiplicand
	ld hl, 0        ; hl = accumulator
	ld a, 16        ; loop counter
_plz_mul_loop:
	push af
	ld a, c
	rra             ; LSB of bc -> carry
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

; _plz_div: HL = HL / DE (unsigned 16-bit)
_plz_div:
	call _plz_divmod
	push bc
	pop hl
	ret

; _plz_mod: HL = HL % DE (unsigned 16-bit)
_plz_mod:
	call _plz_divmod
	ret

; Internal: divide HL by DE
; Output: BC = quotient, HL = remainder
_plz_divmod:
	xor a
	push hl
	pop bc          ; bc = dividend
	ld hl, 0        ; hl = remainder
	ld a, 16        ; 16 bits
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

; _plz_eq: HL = (HL == DE) ? 1 : 0
_plz_eq:
	or a
	sbc hl, de
	ld hl, 0
	ret nz
	inc l
	ret

; _plz_ne: HL = (HL != DE) ? 1 : 0
_plz_ne:
	or a
	sbc hl, de
	ld hl, 0
	ret z
	inc l
	ret

; _plz_gt: HL = (HL > DE) ? 1 : 0 (unsigned)
_plz_gt:
	or a
	sbc hl, de
	jr c, _plz_cmp_false
	jr z, _plz_cmp_false
	ld hl, 1
	ret

; _plz_lt: HL = (HL < DE) ? 1 : 0 (unsigned)
_plz_lt:
	or a
	sbc hl, de
	jr nc, _plz_cmp_false
	ld hl, 1
	ret

; _plz_gte: HL = (HL >= DE) ? 1 : 0 (unsigned)
_plz_gte:
	or a
	sbc hl, de
	jr c, _plz_cmp_false
	ld hl, 1
	ret

; _plz_lte: HL = (HL <= DE) ? 1 : 0 (unsigned)
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
	g.Emit(ProgramHeader)
	g.Emit(RuntimeHeader)
	for _, statement := range p.Statements {
		err := statement.Gen(g)
		if err != nil {
			return err
		}
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
		g.Emitf("\tld hl, (%s)\n", o.Reference.Identifier)
	case o.Expression != nil:
		return o.Expression.Gen(g)
	}
	return nil
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
	}
	return nil
}

// genIndexRead generates code to read arr[index].
// Operands: [array_expr, index_expr].
func (g *Gen) genIndexRead(operands []Operand) error {
	// Get the array base address into hl.
	// The first operand must be a reference (we take its address).
	if operands[0].Reference != nil {
		g.Emitf("\tld hl, %s\n", operands[0].Reference.Identifier)
	} else {
		return fmt.Errorf("genIndexRead: first operand must be a reference")
	}

	// If there's an index, add it (scaled by 2 for word-size).
	if len(operands) >= 2 {
		g.Emitln("\tpush hl")
		if err := operands[1].Expression.Gen(g); err != nil {
			return err
		}
		g.Emitln("\tadd hl, hl") // index * 2 (word)
		g.Emitln("\tex de, hl")
		g.Emitln("\tpop hl")
		g.Emitln("\tadd hl, de")
	}

	// Load the word value at the computed address.
	g.Emitln("\tld a, (hl)")
	g.Emitln("\tinc hl")
	g.Emitln("\tld h, (hl)")
	g.Emitln("\tld l, a")
	return nil
}

// genCallExpr generates code to call a function expression.
// Operands: [func_expr, arg1, arg2, ...].
func (g *Gen) genCallExpr(operands []Operand) error {
	// For now, only support calling a named label.
	if operands[0].Reference != nil {
		name := operands[0].Reference.Identifier
		// Push arguments right-to-left (no args for now – stub).
		g.Emitf("\tcall %s\n", name)
		// Result comes back in hl by convention.
		return nil
	}
	return fmt.Errorf("genCallExpr: indirect calls not yet supported")
}

// ---------------------------------------------------------------------------
// Let (assignment)
// ---------------------------------------------------------------------------

func (s Let) Gen(g *Gen) error {
	// Evaluate RHS into hl.
	if err := s.Expression.Gen(g); err != nil {
		return err
	}

	if len(s.Subscripts) == 0 {
		// Simple variable store.
		g.Emitf("\tld (%s), hl\n", s.Identifier)
		return nil
	}

	// Array element set: lhs[sub1][sub2]... = rhs
	// hl still holds the RHS value; save it.
	g.Emitln("\tpush hl")

	// Compute target address into hl.
	g.Emitf("\tld hl, %s\n", s.Identifier)
	for i := range s.Subscripts {
		g.Emitln("\tpush hl")
		if err := s.Subscripts[i].Gen(g); err != nil {
			return err
		}
		g.Emitln("\tadd hl, hl") // * 2 (word)
		g.Emitln("\tex de, hl")
		g.Emitln("\tpop hl")
		g.Emitln("\tadd hl, de")
	}

	// hl = target address.  Pop the value from the stack.
	g.Emitln("\tpop de") // de = value to store
	g.Emitln("\tld (hl), e")
	g.Emitln("\tinc hl")
	g.Emitln("\tld (hl), d")
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
		// Push end value on stack
		if err := s.For.To.Gen(g); err != nil {
			return err
		}
		g.Emitln("\tpush hl")

		// Evaluate step (default 1), store in IX for preservation
		if s.For.By != nil {
			if err := s.For.By.Gen(g); err != nil {
				return err
			}
		} else {
			g.Emitln("\tld hl, 1")
		}
		g.Emitln("\tpush hl")
		g.Emitln("\tpop ix")

		// Initialize var = start
		if err := s.For.Start.Gen(g); err != nil {
			return err
		}
		g.Emitf("\tld (%s), hl\n", s.For.Reference.Identifier)

		n := g.nextLabel()
		g.Emitf("_for_%d:\n", n)
		// Compare var with end (hl = end from stack)
		g.Emitln("\tpop hl")
		g.Emitln("\tpush hl")
		g.Emitf("\tld de, (%s)\n", s.For.Reference.Identifier)
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")
		g.Emitf("\tjr c, _end_%d\n", n) // end < var → exit

		// Body
		for _, stmt := range s.Statements {
			if err := stmt.Gen(g); err != nil {
				return err
			}
		}

		// var += ix (step)
		g.Emitf("\tld hl, (%s)\n", s.For.Reference.Identifier)
		g.Emitln("\tadd hl, ix")
		g.Emitf("\tld (%s), hl\n", s.For.Reference.Identifier)
		g.Emitf("\tjr _for_%d\n", n)
		g.Emitf("_end_%d:\n", n)
		g.Emitln("\tpop hl") // discard end

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
func (s Procedure) Gen(g *Gen) error { return nil }

func (s Return) Gen(g *Gen) error {
	g.Emitln("\tret")
	return nil
}

func (s Call) Gen(g *Gen) error {
	if s.Name != "" {
		g.Emitln("\tcall ", s.Name)
	} else {
		g.Emitln("\tcall ", s.Location)
	}
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
	if s.Type.Predeclared == PredeclaredByte {
		g.Emitf("org 0x%x\n%s: db 0\n", g.Heap, s.Identifier)
		g.Heap += 1
	} else if s.Type.Predeclared == PredeclaredWord {
		g.Emitf("org 0x%x\n%s: db 0,0\n", g.Heap, s.Identifier)
		g.Heap += 2
	}
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
	g.Emitf("\tld a, %d\n", s.Value)
	g.Emitf("\tout (%d), a\n", s.Port)
	return nil
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
