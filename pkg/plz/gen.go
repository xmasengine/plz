package plz

import "fmt"
import "os"
import "strconv"

const HeapBase = 0xC000 // RAM memory.

type Gen struct {
	*os.File
	Heap int // Pointer to last allocated heap RAM memory.
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
			g.Emitf("org %x, %x\n", org)
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

func (s Let) Gen(g *Gen) error {
	if s.Literal.Number != nil {
		g.Emitf("ld hl, %d\n", *s.Literal.Number)
		g.Emitf("ld (%s),hl\n", s.Identifier)
	}
	return nil
}

func (s Group) Gen(g *Gen) error     { return nil }
func (s Procedure) Gen(g *Gen) error { return nil }

func (s Return) Gen(g *Gen) error {
	g.Emitln("ret")
	return nil
}

func (s Call) Gen(g *Gen) error {
	if s.Name != "" {
		g.Emitln("call ", s.Name)
	} else {
		g.Emitln("call ", s.Location)
	}
	return nil
}

func (s GoTo) Gen(g *Gen) error {
	if s.Name != "" {
		g.Emitln("jp ", s.Name)
	} else {
		g.Emitln("jp ", s.Location)
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
	g.Emitln("halt")
	return nil
}

func (s Enable) Gen(g *Gen) error {
	g.Emitln("ei")
	return nil
}

func (s Disable) Gen(g *Gen) error {
	g.Emitln("di")
	return nil
}

func (s Output) Gen(g *Gen) error {
	g.Emitf("ld a, %d\n", s.Value)
	g.Emitf("out (%d), a\n", s.Port)
	return nil
}

func (s Data) Gen(g *Gen) error {
	if s.Literal.Number != nil {
		g.Emitf("db %x\n", *s.Literal.Number)
	} else if s.Literal.Text != nil {
		g.Emitf("ds %s\n", strconv.Quote(*s.Literal.Text))
	}
	return nil
}

func (s Constant) Gen(g *Gen) error {
	if s.Literal.Number != nil {
		g.Emitf("const %s = %x\n", s.Name, *s.Literal.Number)
	} else if s.Literal.Text != nil {
		g.Emitf("const %s = %s\n", s.Name, strconv.Quote(*s.Literal.Text))
	}
	return nil
}
