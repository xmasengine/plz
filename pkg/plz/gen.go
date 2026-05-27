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

func (p Program) Gen(g *Gen) error {
	g.Emit(ProgramHeader)
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
	g.Emitf("// statement not yet implemented: %v\n", s)
	return nil
}

// ---------------------------------------------------------------------------
// Expression code generator
// ---------------------------------------------------------------------------

// genExpr evaluates the expression and leaves the result in hl.
func (g *Gen) genExpr(e *Expression) error {
	switch {
	case e.Operand != nil:
		return g.genOperand(e.Operand)
	case e.Prefix != nil:
		return g.genPrefix(e.Prefix)
	case e.Infix != nil:
		return g.genInfix(e.Infix)
	case e.Suffix != nil:
		return g.genSuffix(e.Suffix)
	}
	return nil
}

func (g *Gen) genOperand(o *Operand) error {
	switch {
	case o.Literal != nil:
		if o.Literal.Number != nil {
			g.Emitf("\tld hl, %d\n", *o.Literal.Number)
		}
	case o.Reference != nil:
		g.Emitf("\tld hl, (%s)\n", o.Reference.Identifier)
	case o.Expression != nil:
		return g.genExpr(o.Expression)
	}
	return nil
}

func (g *Gen) genPrefix(p *Prefix) error {
	switch p.Operator {
	case OperatorNEG:
		if err := g.genOperand(&p.Operand); err != nil {
			return err
		}
		g.Emitln("\tex de, hl")
		g.Emitln("\tld hl, 0")
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")

	case OperatorNOT:
		if err := g.genOperand(&p.Operand); err != nil {
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

func (g *Gen) genInfix(i *Infix) error {
	// Left operand
	if err := g.genOperand(&i.Operands[0]); err != nil {
		return err
	}
	g.Emitln("\tpush hl")

	// Right operand
	if err := g.genOperand(&i.Operands[1]); err != nil {
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

	case OperatorMUL, OperatorDIV, OperatorMOD:
		g.Emitf("\t// %c not yet implemented\n", i.Operator)

	case OperatorEQU:
		g.genCmp("nz")

	case OperatorNEQ:
		g.genCmp("z")

	case OperatorGT:
		// hl > de  (unsigned)
		n := g.nextLabel()
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")
		g.Emitf("\tjr c, _lbl_%d\n", n)
		g.Emitf("\tjr z, _lbl_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("_lbl_%d:\n", n)

	case OperatorLT:
		n := g.nextLabel()
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")
		g.Emitf("\tjr nc, _lbl_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("_lbl_%d:\n", n)

	case OperatorGTE:
		n := g.nextLabel()
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")
		g.Emitf("\tjr c, _lbl_%d\n", n)
		g.Emitln("\tld hl, 1")
		g.Emitf("_lbl_%d:\n", n)

	case OperatorLTE:
		n := g.nextLabel()
		g.Emitln("\tor a")
		g.Emitln("\tsbc hl, de")
		g.Emitf("\tld hl, 0\n")
		g.Emitf("\tjr nz, _lbl_%d\n", n)
		g.Emitln("\tinc l") // hl == de → true
		g.Emitf("_lbl_%d:\n", n)
		g.Emitf("\tjr nc, _lbl_%d\n", n+1)
		g.Emitln("\tinc l") // hl < de (carry) → true
		g.Emitf("_lbl_%d:\n", n+1)
	}

	return nil
}

// genCmp generates code for EQU / NEQ: compares hl with de, sets hl = 0 or 1.
// jmpCond is the condition to SKIP setting hl=1 (jr <jmpCond> skips the inc l).
func (g *Gen) genCmp(jmpCond string) {
	n := g.nextLabel()
	g.Emitln("\tor a")
	g.Emitln("\tsbc hl, de")
	g.Emitf("\tld hl, 0\n")
	g.Emitf("\tjr %s, _lbl_%d\n", jmpCond, n)
	g.Emitln("\tinc l")
	g.Emitf("_lbl_%d:\n", n)
}

func (g *Gen) genSuffix(s *Suffix) error {
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
		if err := g.genExpr(operands[1].Expression); err != nil {
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
	if err := g.genExpr(&s.Expression); err != nil {
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
		if err := g.genExpr(&s.Subscripts[i]); err != nil {
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

func (s Group) Gen(g *Gen) error     { return nil }
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
