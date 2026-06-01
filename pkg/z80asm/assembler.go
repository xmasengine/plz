package z80asm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/scanner"
)

var baseCommandTable = map[string]instrAssembler{
	"org":      commandOrg{},
	"db":       cmdData(const8),
	"dw":       cmdData(const16),
	"ds":       cmdData(argstring),
	"const":    commandConst{},
	"include":  commandInclude{},
	"jmp":      commandJmp{},
	"bank":     commandBank{},
	"banksize": commandBankSize{},
	"bankat":   commandBankAt{},
}

type commandAssembler struct {
	cmd  string
	args map[arg][]byte
}

// An Assembler can assemble Z80 instructions
// into RAM.
type Assembler struct {
	commandTable map[string]instrAssembler
	opener       func(string) (io.ReadCloser, error)
	pass         int
	pc           int // The PC from the point of view of the code
	target       int // Where in the total memory the code is written
	l            map[string]uint16
	consts       map[string]int64
	constsDef    map[string]bool

	currentMajorLabel string
	labelAssign       map[string]string
	m                 []uint8

	bankSize int // ROM bank size in bytes (default 0x4000 = 16KB)
	bankAt   int // CPU address of the bank window (default 0x4000)

	// These are stacks, used when we "include" another file.
	scanners  []*scanner.Scanner
	closers   []io.Closer
	openFiles []string // to avoid recursive includes

	scanErr   error
	lastToken token
}

func openFile(filename string) (io.ReadCloser, error) {
	f, err := os.Open(filename)
	return f, err
}

type Z80Core int

const (
	Z80CoreStandard Z80Core = 0
	Z80CoreNext1    Z80Core = 1
	Z80CoreNext2    Z80Core = 2
)

type assemblerOption struct {
	core Z80Core
}

type AssemblerOpt func(*assemblerOption) error

// UseNextCore include Z80N opcodes for the given core.
func UseNextCore(core Z80Core) AssemblerOpt {
	return func(a *assemblerOption) error {
		a.core = core
		return nil
	}
}

// NewAssembler constructs a new assembler.
// By default, the assembler will assemble code starting at address
// 0x8000.
func NewAssembler(opts ...AssemblerOpt) (*Assembler, error) {
	var aopt assemblerOption
	for _, opt := range opts {
		if err := opt(&aopt); err != nil {
			return nil, err
		}
	}

	cmdTable := make(map[string]instrAssembler)
	for k, v := range baseCommandTable {
		cmdTable[k] = v
	}

	cmd0s := []map[string][]byte{commands0arg}
	cmds := []map[string]args{commandsArgs, ixCommands, iyCommands}

	if aopt.core > 0 {
		cmd0s = append(cmd0s, commands0argNext1)
		cmds = append(cmds, commandsArgsNext1)
	}
	if aopt.core > 1 {
		cmds = append(cmds, commandsArgsNext2)
	}

	for _, c0a := range cmd0s {
		for c0, bs := range c0a {
			if _, ok := cmdTable[c0]; ok {
				panic("duplicate command: " + c0)
			}
			cmdTable[c0] = commandAssembler{c0, map[arg][]byte{void: bs}}
		}
	}

	for c0, os := range joinCommands(cmds...) {
		if _, ok := cmdTable[c0]; ok {
			panic("duplicate command: " + c0)
		}
		cmdTable[c0] = commandAssembler{c0, os}
	}

	a := &Assembler{
		commandTable: cmdTable,
		opener:       openFile,
		pc:           0x8000,
		target:       0x8000,
		l:            make(map[string]uint16),
		consts:       make(map[string]int64),
		constsDef:    make(map[string]bool),
		labelAssign:  make(map[string]string),
		m:            make([]uint8, 64*1024),
		bankSize:     0x4000,
		bankAt:       0x4000,
	}
	return a, nil
}

func (asm *Assembler) RAM() []uint8 {
	return asm.m
}

// AssembleFile reads the named file, and assembles it as z80
// instructions.
func (asm *Assembler) AssembleFile(filename string) error {
	pc := asm.pc
	target := asm.target
	defer func() {
		asm.pc = pc
		asm.target = target
	}()
	for pass := 0; pass < 2; pass++ {
		asm.pc = pc
		asm.target = target
		asm.pass = pass
		asm.currentMajorLabel = ""
		// Reset the map that says whether we've seen a const.
		// We use this to prevent use of const before definition.
		asm.constsDef = make(map[string]bool)
		if err := asm.assembleFile(filename); pass == 1 && err != nil {
			return err
		}
	}
	return nil
}

func endStatement(t token) bool {
	return t.t == ';' || t.t == scanner.EOF || t.t == '\n'
}

func (asm *Assembler) popScanner() (bool, error) {
	if err := asm.closers[len(asm.closers)-1].Close(); err != nil {
		return true, asm.scanErrorf("error closing file: %v", err)
	}
	asm.closers = asm.closers[:len(asm.closers)-1]
	asm.scanners = asm.scanners[:len(asm.scanners)-1]
	asm.openFiles = asm.openFiles[:len(asm.openFiles)-1]
	return len(asm.scanners) == 0, nil
}

func (asm *Assembler) pushScanner(filename string) error {
	for _, f := range asm.openFiles {
		if f == filename {
			return asm.scanErrorf("recursive include of file %q", filename)
		}
	}
	f, err := asm.opener(filename)
	if err != nil {
		return fmt.Errorf("failed to assemble %q: %v", filename, err)
	}

	asm.openFiles = append(asm.openFiles, filename)
	var scan scanner.Scanner
	scan.Init(f)
	scan.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanChars | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments | scanner.SkipComments
	scan.Whitespace = (1 << ' ') | (1 << '\t')
	scan.Position.Filename = filename
	scan.Error = func(s *scanner.Scanner, msg string) {
		asm.scanErr = asm.scanErrorf("%s", msg)
	}
	asm.scanners = append(asm.scanners, &scan)
	asm.closers = append(asm.closers, f)
	return nil
}

func (asm *Assembler) assembleFile(filename string) error {
	err := asm.pushScanner(filename)
	if err != nil {
		return err
	}

	var errs []string
	for asm.canContinue() && len(errs) < 20 {
		if err := asm.assemble(); err != nil {
			errs = append(errs, err.Error())
			for asm.canContinue() && !endStatement(asm.lastToken) {
				asm.nextToken()
			}
		} else {
			break
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (asm *Assembler) scan() *scanner.Scanner {
	return asm.scanners[len(asm.scanners)-1]
}

func (asm *Assembler) location() string {
	return fmt.Sprintf("%s:%d.%d", asm.scan().Position.Filename, asm.scan().Position.Line, asm.scan().Position.Column)
}

func (asm *Assembler) scanErrorf(fs string, args ...interface{}) error {
	return errors.New(asm.location() + ": " + fmt.Sprintf(fs, args...))
}

type token struct {
	t rune
	s string
}

const (
	tokLTLT rune = -iota - 20
	tokGTGT
	tokAndNot
	tokEqEq
	tokNotEq
	tokLTEq
	tokGTEq
	tokAndAnd
	tokOrOr
)

var tokStrings = map[rune]string{
	tokLTLT:   "<<",
	tokGTGT:   ">>",
	tokAndNot: "&^",
	tokEqEq:   "==",
	tokNotEq:  "!=",
	tokLTEq:   "<=",
	tokGTEq:   ">=",
	tokAndAnd: "&&",
	tokOrOr:   "||",
}

var tokOperatorPrefixes = makeOperatorCompletions()

func makeOperatorCompletions() map[rune]map[rune]rune {
	var r = map[rune]map[rune]rune{}
	for k, v := range tokStrings {
		if _, ok := r[rune(v[0])]; !ok {
			r[rune(v[0])] = map[rune]rune{}
		}
		r[rune(v[0])][rune(v[1])] = k
	}
	return r
}

func (asm *Assembler) nextToken() (token, error) {
	t := asm.scan().Scan()
	if asm.scanErr != nil {
		return token{}, asm.scanErr
	}
	if m2 := tokOperatorPrefixes[t]; m2 != nil {
		if tok := m2[asm.scan().Peek()]; tok != 0 {
			asm.scan().Scan()
			return token{tok, ""}, asm.scanErr
		}
	}
	asm.lastToken = token{t, asm.scan().TokenText()}
	return asm.lastToken, asm.scanErr
}

func (t token) String() string {
	switch t.t {
	case scanner.Int:
		return t.s
	case scanner.Ident:
		return fmt.Sprintf(`identifier "%s"`, t.s)
	}
	if s, ok := tokStrings[t.t]; ok {
		return s
	}
	return scanner.TokenString(t.t)
}

func (asm *Assembler) canContinue() bool {
	return asm.scanErr == nil
}

func (asm *Assembler) assemble() error {
	if asm.scanErr != nil {
		return asm.scanErr
	}
	for {
		tok, err := asm.nextToken()
		if err != nil {
			return err
		}
		switch tok.t {
		case scanner.EOF:
			done, err := asm.popScanner()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		case scanner.Ident:
			// Might be a command
			if f, ok := asm.commandTable[strings.ToLower(tok.s)]; ok {
				if err := f.W(asm); err != nil {
					return err
				}
				continue
			}
			// We try to parse the identifier as a major label.
			// That means the next token should be a ':'
			labName := tok.s
			tok, err = asm.nextToken()
			if err != nil {
				return err
			}
			if tok.t != ':' {
				return asm.scanErrorf("unknown command %s", labName)
			}
			if err := asm.setLabel(labName, 0); err != nil {
				return err
			}
			continue
		case ';':
			continue
		case '\n':
			continue
		case '.':
			if err := asm.assembleMinorLabel(); err != nil {
				return err
			}
		default:
			return asm.scanErrorf("unexpected %s", tok)
		}
	}
}

func (asm *Assembler) writeByte(u uint8) error {
	if int(asm.target) >= len(asm.m) {
		newLen := (asm.target + 16*1024 - 1) / (16 * 1024) * 16 * 1024
		asm.m = append(asm.m, make([]uint8, newLen-len(asm.m))...)
	}
	if asm.pc >= 64*1024 || asm.pc < 0 {
		return fmt.Errorf("pc out of range: %x", asm.pc)
	}
	asm.m[asm.target] = u
	asm.pc++
	asm.target++
	return nil
}

func (asm *Assembler) writeBytes(bs []byte) error {
	for _, b := range bs {
		if err := asm.writeByte(b); err != nil {
			return err
		}
	}
	return nil
}

// GetLabel returns the value of the given label.
// It is only valid after the assembler has run.
func (asm *Assembler) GetLabel(majLabel, l string) (uint16, bool) {
	if strings.HasPrefix(l, ".") {
		v, ok := asm.l[majLabel+l]
		return v, ok
	}
	v, ok := asm.l[majLabel+"."+l]
	if ok {
		return v, ok
	}
	v, ok = asm.l[l]
	return v, ok
}

// GetConst returns the value of the given const.
// It is only valid after the assembler has run.
func (asm *Assembler) GetConst(c string) (int64, bool, error) {
	if !asm.constsDef[c] {
		if _, ok := asm.consts[c]; ok {
			return 0, false, asm.scanErrorf("use of const %q before definition", c)
		}
		return 0, false, nil
	}
	v, ok := asm.consts[c]
	return v, ok, nil
}

type cmdData arg

func (n cmdData) W(asm *Assembler) error {
	args, err := asm.parseArgs(true)
	if err != nil {
		return err
	}
	for _, arg0 := range args {
		bs, ok, err := arg0.evalAs(asm, arg(n), false)
		if err != nil {
			return err
		}
		if !ok {
			return asm.scanErrorf("bad data value: %s", arg0)
		}
		if err := asm.writeBytes(bs); err != nil {
			return err
		}
	}
	return nil
}

type instrAssembler interface {
	W(a *Assembler) error
}

func (asm *Assembler) argsCompatible(vals []expr, a arg) ([]byte, bool, error) {
	if len(vals) != argLen(a) {
		return nil, false, nil
	}
	if a == 0 {
		return nil, true, nil
	}
	if a >= 1024 {
		a0, ok, err := vals[0].evalAs(asm, a/1024, true)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return nil, false, nil
		}
		a1, ok, err := vals[1].evalAs(asm, a%1024, true)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return nil, false, nil
		}
		return append(a0, a1...), true, nil
	}
	return vals[0].evalAs(asm, a, true)
}

func (ca commandAssembler) W(asm *Assembler) error {
	vals, err := asm.parseArgs(false)
	if err != nil {
		return err
	}
	found := false
	for argVariant, bs := range ca.args {
		argData, ok, err := asm.argsCompatible(vals, argVariant)
		if err != nil {
			return err
		}
		if ok {
			if found {
				return asm.scanErrorf("more than one variant of %s possible: args %#v, found alt variant %s", ca.cmd, vals, argVariant)
			}
			found = true
			// Longer instructions (bit operations on ix or iy)
			// interleave the fixed part of the instruction with
			// the variable part.
			// For example: sla (ix+4), * -> dd cb 04 26
			n := len(bs)
			if n > 2 {
				n = 2
			}
			if err := asm.writeBytes(bs[:n]); err != nil {
				return err
			}
			if err := asm.writeBytes(argData); err != nil {
				return err
			}
			if err := asm.writeBytes(bs[n:]); err != nil {
				return err
			}
		}
	}
	if !found {
		vs := []string{}
		for _, v := range vals {
			vs = append(vs, fmt.Sprintf("%s", v))
		}
		return asm.scanErrorf("no suitable form of %s found that matches %s %s", ca.cmd, ca.cmd, strings.Join(vs, ", "))
	}

	return nil
}

func joinCommands(cmdss ...map[string]args) map[string]args {
	r := map[string]args{}
	for _, cmds := range cmdss {
		for k, argss := range cmds {
			if r[k] == nil {
				r[k] = args{}
			}
			for a, bs := range argss {
				if _, ok := r[k][a]; ok {
					panic(fmt.Sprintf("duplicate args %s found for %s", a, k))
				}
				r[k][a] = bs
			}
		}
	}
	return r
}

type commandJmp struct{}

// jr-to-jp opcode mapping for the same condition code.
var jmpOpcodes = [...]struct {
	cond arg
	jr   byte
	jp   byte
}{
	{ccNZ, 0x20, 0xC2},
	{ccZ, 0x28, 0xCA},
	{ccNC, 0x30, 0xD2},
	{ccC, 0x38, 0xDA},
}

// W implements the jmp pseudo-instruction. It emits jr+NOP (3 bytes) when the
// relative offset is in [-128,127] and jp (3 bytes) otherwise. Using jr for
// nearby jumps produces slightly faster code. The 3-byte encoding is consistent
// across both assembler passes so label addresses don't shift.
func (commandJmp) W(asm *Assembler) error {
	vals, err := asm.parseArgs(false)
	if err != nil {
		return err
	}
	if len(vals) < 1 || len(vals) > 2 {
		return asm.scanErrorf("jmp requires a label, or a condition and a label")
	}

	jrOp := byte(0x18) // unconditional
	jpOp := byte(0xC3)
	addrIdx := 0

	if len(vals) == 2 {
		// Try to match the first argument as a condition code.
		matched := false
		for _, oc := range jmpOpcodes {
			_, ok, _ := vals[0].evalAs(asm, oc.cond, false)
			if ok {
				jrOp = oc.jr
				jpOp = oc.jp
				matched = true
				break
			}
		}
		if !matched {
			return asm.scanErrorf("jmp: unknown condition in %v", vals[0])
		}
		addrIdx = 1
	}

	// Evaluate the address (label or expression) as a 16-bit target.
	argData, ok, err := vals[addrIdx].evalAs(asm, addr16, true)
	if err != nil {
		return err
	}
	if !ok {
		return asm.scanErrorf("jmp: cannot evaluate address expression")
	}
	target := uint16(argData[0]) | uint16(argData[1])<<8

	// We always emit exactly 3 bytes (same on both passes) so label addresses
	// computed on pass 0 remain valid on pass 1.
	if asm.pass == 0 {
		asm.writeByte(jpOp)
		asm.writeBytes(argData)
		return nil
	}

	// Pass 1: check whether a relative jump fits.
	offset := int(target) - (asm.pc + 2)
	if offset >= -128 && offset <= 127 {
		asm.writeByte(jrOp)
		asm.writeByte(byte(offset))
		asm.writeByte(0x00) // nop padding → 3 bytes total
	} else {
		asm.writeByte(jpOp)
		asm.writeBytes(argData)
	}
	return nil
}

type commandInclude struct{}

func getString(e expr) (string, error) {
	switch v := e.(type) {
	case exprString:
		return v.s, nil
	}
	return "", fmt.Errorf("expected string, got %v", e)
}

func (commandInclude) W(asm *Assembler) error {
	args, err := asm.parseArgs(false)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return asm.scanErrorf("expected \"filename.asm\" to follow include, got: %v", args)
	}
	name, err := getString(args[0])
	if err != nil {
		return asm.scanErrorf("expected \"filename.asm\" to follow include, got: %v", args[0])
	}
	return asm.pushScanner(name)
}

type commandConst struct{}

func getIdent(e expr) (string, error) {
	switch v := e.(type) {
	case exprIdent:
		if v.cc != 0 || v.r != 0 {
			return "", fmt.Errorf("expected identifier, got register or condition code")
		}
		return v.id, nil
	}
	return "", fmt.Errorf("expected identifier, got %v", e)
}

func (commandConst) W(asm *Assembler) error {
	args, err := asm.parseSepArgs('=', false)
	if err != nil {
		return err
	}
	if len(args) != 2 {
		return asm.scanErrorf("expected syntax: const <ident> = <value>, got: const %v", args)
	}
	name, err := getIdent(args[0])
	if err != nil {
		return err
	}
	n, ok, err := getIntValue(asm, args[1])
	if err != nil {
		return err
	}
	if !ok {
		return asm.scanErrorf("failed to evaluate const %q value %q", name, args[1])
	}
	if asm.constsDef[name] {
		return asm.scanErrorf("redefining %q", name)
	}
	asm.constsDef[name] = true
	asm.consts[name] = n
	return nil
}

type commandOrg struct{}

const MaxTarget = 1024 * 1024 * 128

func (commandOrg) W(asm *Assembler) error {
	args, err := asm.parseArgs(true)
	if err != nil {
		return err
	}
	if len(args) < 1 || len(args) > 2 {
		return asm.scanErrorf("org takes one or two arguments: %d found", len(args))
	}
	n, ok, err := getIntValue(asm, args[0])
	if err != nil {
		return err
	}
	arg1 := args[0]
	if len(args) >= 2 {
		arg1 = args[1]
	}
	if !ok {
		return asm.scanErrorf("org first (pc) argument should be an address, found %s", args[0])
	}
	if n < 0 || n >= 65536 {
		return asm.scanErrorf("org first (pc) argument %x out of range", n)
	}

	t, ok, err := getIntValue(asm, arg1)
	if err != nil {
		return err
	}
	if !ok {
		return asm.scanErrorf("org second (target) argument should be an address, found %s", arg1)
	}
	if t < 0 || t >= MaxTarget {
		return asm.scanErrorf("org second (target) argument %x out of range, must be between 0 and %d", t, MaxTarget)
	}

	asm.pc = int(n)
	asm.target = int(t)
	return nil
}

type commandBank struct{}

func (commandBank) W(asm *Assembler) error {
	args, err := asm.parseArgs(true)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return asm.scanErrorf("bank takes one argument: %d found", len(args))
	}
	n, ok, err := getIntValue(asm, args[0])
	if err != nil {
		return err
	}
	if !ok {
		return asm.scanErrorf("bank argument should be a number, found %s", args[0])
	}
	if n < 0 {
		return asm.scanErrorf("bank argument %d out of range (negative)", n)
	}
	asm.pc = asm.bankAt
	asm.target = asm.bankAt + int(n)*asm.bankSize
	return nil
}

type commandBankSize struct{}

func (commandBankSize) W(asm *Assembler) error {
	args, err := asm.parseArgs(true)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return asm.scanErrorf("banksize takes one argument: %d found", len(args))
	}
	n, ok, err := getIntValue(asm, args[0])
	if err != nil {
		return err
	}
	if !ok {
		return asm.scanErrorf("banksize argument should be a number, found %s", args[0])
	}
	if n <= 0 {
		return asm.scanErrorf("banksize must be positive, got %d", n)
	}
	asm.bankSize = int(n)
	return nil
}

type commandBankAt struct{}

func (commandBankAt) W(asm *Assembler) error {
	args, err := asm.parseArgs(true)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return asm.scanErrorf("bankat takes one argument: %d found", len(args))
	}
	n, ok, err := getIntValue(asm, args[0])
	if err != nil {
		return err
	}
	if !ok {
		return asm.scanErrorf("bankat argument should be a number, found %s", args[0])
	}
	if n < 0 || n >= 65536 {
		return asm.scanErrorf("bankat argument %x out of CPU address range", n)
	}
	asm.bankAt = int(n)
	return nil
}

func (asm *Assembler) setLabel(label string, level int) error {
	if level == 0 {
		asm.currentMajorLabel = label
	} else {
		label = asm.currentMajorLabel + "." + label
	}
	if asm.pass == 1 {
		fass := asm.labelAssign[label]
		if asm.location() != fass {
			return asm.scanErrorf("label %q redefined. First defined at %s", label, fass)
		}
		return nil
	}
	asm.l[label] = uint16(asm.pc)
	if asm.pass == 0 && asm.labelAssign[label] == "" {
		asm.labelAssign[label] = asm.location()
	}
	return nil
}

func (asm *Assembler) assembleMinorLabel() error {
	tok, err := asm.nextToken()
	if err != nil {
		return err
	}
	switch tok.t {
	case scanner.Ident:
		return asm.setLabel(tok.s, 1)
	default:
		return asm.scanErrorf("unexpected %s", tok)
	}
}

func getByte(prefix, bs []byte) (byte, bool) {
	n := len(bs)
	if !bytes.HasPrefix(bs, prefix) || n != len(prefix)+1 {
		return 0, false
	}
	return bs[n-1], true
}

func AssembleFiles(output string, sources []string) error {
	fout, err := os.Create(output)
	if err != nil {
		return err
	}
	defer fout.Close()

	assemblies := []*Assembler{}
	for _, source := range sources {
		asm, err := NewAssembler()
		if err != nil {
			return err
		}
		err = asm.AssembleFile(source)
		if err != nil {
			return err
		}
		assemblies = append(assemblies, asm)
	}

	return AssembleWriter(fout, assemblies)
}

func AssembleWriter(fout io.Writer, assemblies []*Assembler) (err error) {
	for _, asm := range assemblies {
		// Just concatenate the assemblies for now.
		bin := asm.RAM()
		_, err = fout.Write(bin)
		if err != nil {
			return err
		}
	}
	return nil
}

func AssembleSMS(output string, sources []string) error {
	fout, err := os.Create(output)
	if err != nil {
		return err
	}
	defer fout.Close()

	assemblies := []*Assembler{}
	for _, source := range sources {
		asm, err := NewAssembler()
		if err != nil {
			return err
		}
		err = asm.AssembleFile(source)
		if err != nil {
			return err
		}
		assemblies = append(assemblies, asm)
	}

	return AssembleSMSWriter(fout, assemblies)
}

func injectSMSChecksumAndHeader(rom []byte, headerOffset int) {
	// Ensure the ROM is large enough for the header
	if len(rom) < headerOffset+16 {
		// Pad the ROM if necessary
		padding := make([]byte, headerOffset+16-len(rom))
		rom = append(rom, padding...)
	}

	// Calculate the checksum range based on the ROM size
	var checksumEnd int
	switch {
	case len(rom) <= 0x4000: // 16KB
		checksumEnd = 0x3FF0
	case len(rom) <= 0x8000: // 32KB
		checksumEnd = 0x7FEF
	default: // 128KB, 256KB, etc.
		checksumEnd = 0x7FEF
	}

	// Calculate the checksum
	var checksum uint16 = 0
	for i := 0; i < checksumEnd; i++ {
		checksum += uint16(rom[i])
	}

	// Write the "TMR SEGA" string
	copy(rom[headerOffset:], "TMR SEGA")

	// Write the 16-bit checksum in little-endian format
	checksumOffset := headerOffset + 0x0A
	rom[checksumOffset] = byte(checksum & 0xFF)          // Low byte
	rom[checksumOffset+1] = byte((checksum >> 8) & 0xFF) // High byte

	// Set the region code (4 = SMS Export) and ROM size code
	rom[headerOffset+0x0F] = 0x40 // High nibble = 4, low nibble (size) can be set based on actual size
}

func AssembleSMSWriter(fout io.Writer, assemblies []*Assembler) (err error) {
	var rom []byte
	for _, asm := range assemblies {
		// Concatenate the assemblies.
		rom = append(rom, asm.RAM()...)
	}

	location := 0x7FF0
	if len(rom) < 0x8000 {
		location = 0x3FF0
	}
	if len(rom) < 0x400 {
		location = 0x1FF0
	}
	injectSMSChecksumAndHeader(rom, location)

	_, err = fout.Write(rom)
	if err != nil {
		return err
	}
	return nil
}
