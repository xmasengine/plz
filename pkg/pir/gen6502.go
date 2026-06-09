package pir

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/beevik/go6502/asm"
)

// Gen6502Config holds platform-specific parameters for the 6502 backend.
type Gen6502Config struct {
	// Origin is the starting address of the program.
	Origin uint16
	// VarBase is the base address for static variable storage.
	VarBase uint16
	// StackBase is the base address for the data stack (grows upward).
	StackBase uint16
	// NES enables NES-specific output (iNES header, vector table).
	NES bool
	// TaskLimit limits the number of cooperative tasks (NES zero-page constraint).
	TaskLimit int
}

// Default6502Config returns a configuration suitable for testing.
func Default6502Config() Gen6502Config {
	return Gen6502Config{
		Origin:    0x1000,
		VarBase:   0x2000,
		StackBase: 0x3000,
		NES:       false,
		TaskLimit: 16,
	}
}

// NES6502Config returns a configuration for NES ROM output.
// Uses $C000 as PRG origin, RAM-based variable/data stack, and zero-page
// for frame pointers and task control blocks.
func NES6502Config() Gen6502Config {
	return Gen6502Config{
		Origin:    0xC000,
		VarBase:   0x0300,
		StackBase: 0x0200,
		NES:       true,
		TaskLimit: 8,
	}
}

// Gen6502 translates a PIR Program into 6502 assembly text.
type Gen6502 struct {
	cfg Gen6502Config

	lines []string

	// ariable tracking
	varAddr  map[string]uint16
	varSizes map[string]int
	varNext  uint16

	// ag tracking
	tags    map[string]bool
	labelID int

	// sage tracking — which helpers we need
	needMul bool
	needDiv bool
	needMod bool
	needCmp bool

	// ne-shot AT tracking
	pendingAT int
}

// NewGen6502 creates a Gen6502 with the given config.
func NewGen6502(cfg Gen6502Config) *Gen6502 {
	return &Gen6502{cfg: cfg, varSizes: make(map[string]int), pendingAT: -1}
}

func (z *Gen6502) varName(name string) string {
	return "_v_" + name
}

// Gen translates a PIR programme into 6502 assembly text.
func (z *Gen6502) Gen(prog *Program) string {
	z.lines = nil
	z.varAddr = make(map[string]uint16)
	z.varSizes = make(map[string]int)
	z.varNext = z.cfg.VarBase
	z.tags = make(map[string]bool)
	z.pendingAT = -1

	z.scanProg(prog)

	z.emitHeader()
	z.emitStart()
	z.emitProg(prog)
	z.emitFooter()
	z.emitVars()

	return strings.Join(z.lines, "\n")
}

func (z *Gen6502) scanProg(prog *Program) {
	for _, instr := range prog.Instrs {
		consumed := false
		switch instr.Op {
		case AT:
			z.pendingAT = int(instr.Operand.Num)
		case VAR_B:
			name := z.varName(instr.Operand.Name)
			if _, ok := z.varAddr[name]; !ok {
				if z.pendingAT >= 0 {
					z.varAddr[name] = uint16(z.pendingAT)
					z.varSizes[name] = 1
					consumed = true
				} else {
					z.varAddr[name] = z.varNext
					z.varSizes[name] = 1
					z.varNext += 1
				}
			}
		case VAR_W:
			name := z.varName(instr.Operand.Name)
			if _, ok := z.varAddr[name]; !ok {
				if z.pendingAT >= 0 {
					z.varAddr[name] = uint16(z.pendingAT)
					z.varSizes[name] = 2
					consumed = true
				} else {
					z.varAddr[name] = z.varNext
					z.varSizes[name] = 2
					z.varNext += 2
				}
			}
		case DATA_B, DATA_W, DATA_STR, DATA_TILE:
			consumed = true
		case ROUTE:
			consumed = true
		case TAG:
			z.tags[instr.Operand.Name] = true
		case MUL_B, MUL_W:
			z.needMul = true
		case DIV_B, DIV_W:
			z.needDiv = true
		case MOD_B, MOD_W:
			z.needMod = true
		case IS_B, IS_W:
			z.needCmp = true
		}
		if !consumed && z.pendingAT >= 0 {
			z.pendingAT = -1
		}
	}
}

// ── Emission helpers ──

func (z *Gen6502) emitf(format string, args ...interface{}) {
	z.lines = append(z.lines, fmt.Sprintf(format, args...))
}

func (z *Gen6502) emit(s string) {
	z.lines = append(z.lines, s)
}

// nextLabel returns a unique numeric label suffix.
func (z *Gen6502) nextLabel() int {
	z.labelID++
	return z.labelID
}

// ── Header / Footer ──

func (z *Gen6502) emitHeader() {
	z.emitf("\t.org $%04x", z.cfg.Origin)
	z.emit("")
	z.emitf("\tjmp _6502_main")
	z.emit("")
}

func (z *Gen6502) emitStart() {
	z.emit("_6502_main:")
	z.emit("\tsei")
	z.emit("\tcld")
	z.emit("\tldx #$ff")
	z.emit("\ttxs")
	z.emitf("\tlda #<%d", z.cfg.StackBase)
	z.emitf("\tsta $00")
	z.emitf("\tlda #>%d", z.cfg.StackBase)
	z.emitf("\tsta $01")
	z.emit("")
}

func (z *Gen6502) emitFooter() {
	z.emit("_6502_all_done:")
	z.emit("\tsei")
	z.emit("\tbeq _6502_all_done")
	z.emit("")
}

// ── Runtime helpers ──

func (z *Gen6502) emitRuntime() {
	if z.needCmp {
		z.emit("; Word comparison helper")
		z.emit("; On entry: A=low(NEXT), X=high(NEXT), _t0=low(TOS), _t0+1=high(TOS)")
		z.emit("; On exit: A=1 if NEXT < TOS (unsigned), else 0")
		z.emit("_plz_ult:")
		z.emit("\tsec")
		z.emit("\tsbc $02")
		z.emit("\ttxa")
		z.emit("\tsbc $03")
		z.emit("\tbcs _plz_ult_ge")
		z.emit("\tlda #1")
		z.emit("\trts")
		z.emit("_plz_ult_ge:")
		z.emit("\tlda #0")
		z.emit("\trts")
		z.emit("")
	}
}

// ── Variables ──

func (z *Gen6502) emitVars() {
	if len(z.varAddr) == 0 {
		return
	}
	type kv struct {
		name string
		addr uint16
	}
	var sorted []kv
	for name, addr := range z.varAddr {
		sorted = append(sorted, kv{name, addr})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].addr < sorted[j].addr })
	z.emit("; -------------------------------------------------------------------")
	z.emit("; Variable storage")
	z.emit("; -------------------------------------------------------------------")
	for _, kv := range sorted {
		z.emitf("%s:", kv.name)
		z.emitf("\t.ds %d", z.varSize(kv.name))
	}
	z.emit("")
}

func (z *Gen6502) varSize(name string) int {
	if s, ok := z.varSizes[name]; ok {
		return s
	}
	return 1
}

// ── Instruction emission ──

func (z *Gen6502) emitProg(prog *Program) {
	for _, instr := range prog.Instrs {
		if instr.Op == AT {
			z.pendingAT = int(instr.Operand.Num)
			continue
		}
		if z.pendingAT >= 0 {
			switch instr.Op {
			case VAR_B, VAR_W:
				// handled by emitVars
			case DATA_B, DATA_W, DATA_STR, DATA_TILE, ROUTE:
				z.emitf("\t; AT $%04x", z.pendingAT)
			}
			z.pendingAT = -1
		}
		z.emitInstr(instr)
	}
}

func (z *Gen6502) emitInstr(instr Instr) {
	op := instr.Op
	o := instr.Operand

	switch op {
	// ── Data Movement ──
	case NOP:
		z.emit("\tnop")

	case PUSH_B:
		z.emitf("\tlda #%d", o.Num)
		z.emitSpill()

	case PUSH_W:
		z.emitf("\tlda #%d", o.Num&0xFF)
		z.emitf("\tldx #%d", (o.Num>>8)&0xFF)
		z.emitSpillW()

	case VAR_B, VAR_W:
		// andled in scan phase

	case GET_B:
		z.emitf("\tlda %s", z.varName(o.Name))
		z.emitSpill()

	case GET_W:
		z.emitf("\tlda %s", z.varName(o.Name))
		z.emitf("\tldx %s+1", z.varName(o.Name))
		z.emitSpillW()

	case PUT_B:
		z.emitFill()
		z.emitf("\tsta %s", z.varName(o.Name))

	case PUT_W:
		z.emitFillW() // A=low, X=high
		z.emitf("\tsta %s", z.varName(o.Name))
		z.emitf("\tstx %s+1", z.varName(o.Name))

	case PUSH_A:
		z.emitf("\tlda #<%s", o.Name)
		z.emitf("\tldx #>%s", o.Name)
		z.emitSpillW()

	case READ_B:
		z.emitFillW() // A=low addr, X=high addr
		z.emit("\tstx $05  ; high byte of address")
		z.emit("\tsta $04  ; low byte of address")
		z.emit("\tldy #0")
		z.emit("\tlda ($04),y")
		z.emitSpill()

	case READ_W:
		z.emitFillW() // A=low addr, X=high addr
		z.emit("\tstx $05")
		z.emit("\tsta $04")
		z.emit("\tldy #0")
		z.emit("\tlda ($04),y")
		z.emit("\tpha")
		z.emit("\tldy #1")
		z.emit("\tlda ($04),y")
		z.emit("\ttax")
		z.emit("\tpla")
		z.emitSpillW()

	case WRITE_B:
		z.emitFill() // A = value
		z.emit("\tpha")
		z.emitFillW() // A=low addr, X=high addr
		z.emit("\tstx $05")
		z.emit("\tsta $04")
		z.emit("\tpla")
		z.emit("\tldy #0")
		z.emit("\tsta ($04),y")

	case WRITE_W:
		z.emitFillW() // A=low, X=high
		z.emit("\tpha")
		z.emit("\ttxa")
		z.emit("\tpha")
		z.emitFillW() // A=low addr, X=high addr
		z.emit("\tstx $05")
		z.emit("\tsta $04")
		z.emit("\tpla")
		z.emit("\tldy #1")
		z.emit("\tsta ($04),y")
		z.emit("\tpla")
		z.emit("\tldy #0")
		z.emit("\tsta ($04),y")

	// ── Math & Logic ──
	case ADD_B:
		z.emitFill()
		z.emit("\tsta $02")
		z.emitFill()
		z.emit("\tclc")
		z.emit("\tadc $02")
		z.emitSpill()

	case ADD_W:
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()
		z.emit("\tpha              ; save low(NEXT)")
		z.emit("\tclc")
		z.emit("\tadc $02         ; low(NEXT) + low(TOS)")
		z.emit("\tsta $02         ; temp result low")
		z.emit("\tpla")
		z.emit("\ttxa              ; A = high(NEXT)")
		z.emit("\tadc $03         ; high(NEXT) + high(TOS) + carry")
		z.emit("\ttax              ; X = result high")
		z.emit("\tlda $02         ; A = result low")
		z.emitSpillW()

	case SUB_B:
		z.emitFill()
		z.emit("\tsta $02")
		z.emitFill()
		z.emit("\tsec")
		z.emit("\tsbc $02")
		z.emitSpill()

	case SUB_W:
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()
		z.emit("\tsec")
		z.emit("\tsbc $02         ; low(NEXT) - low(TOS)")
		z.emit("\tpha              ; save low result")
		z.emit("\ttxa              ; A = high(NEXT)")
		z.emit("\tsbc $03         ; high(NEXT) - high(TOS) - borrow")
		z.emit("\ttax              ; X = result high")
		z.emit("\tpla              ; A = result low")
		z.emitSpillW()

	case AND_B:
		z.emitFill()
		z.emit("\tsta $02")
		z.emitFill()
		z.emit("\tand $02")
		z.emitSpill()

	case AND_W:
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()
		z.emit("\tand $02")
		z.emit("\tpha")
		z.emit("\ttxa")
		z.emit("\tand $03")
		z.emit("\ttax")
		z.emit("\tpla")
		z.emitSpillW()

	case OR_B:
		z.emitFill()
		z.emit("\tsta $02")
		z.emitFill()
		z.emit("\tora $02")
		z.emitSpill()

	case OR_W:
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()
		z.emit("\tora $02")
		z.emit("\tpha")
		z.emit("\ttxa")
		z.emit("\tora $03")
		z.emit("\ttax")
		z.emit("\tpla")
		z.emitSpillW()

	case XOR_B:
		z.emitFill()
		z.emit("\tsta $02")
		z.emitFill()
		z.emit("\teor $02")
		z.emitSpill()

	case XOR_W:
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()
		z.emit("\teor $02")
		z.emit("\tpha")
		z.emit("\ttxa")
		z.emit("\teor $03")
		z.emit("\ttax")
		z.emit("\tpla")
		z.emitSpillW()

	case NOT_B:
		z.emitFill()
		z.emit("\teor #$ff")
		z.emit("\tclc")
		z.emit("\tadc #1")
		z.emit("\tand #1")
		z.emitSpill()

	case NOT_W:
		nz := z.nextLabel()
		nd := z.nextLabel()
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\ttxa")
		z.emit("\tora $02")
		z.emitf("\tbne _notw_z_%d", nz)
		z.emit("\tlda #1")
		z.emit("\tldx #0")
		z.emitf("\tbeq _notw_d_%d", nd)
		z.emitf("_notw_z_%d:", nz)
		z.emit("\tlda #0")
		z.emit("\ttax")
		z.emitf("_notw_d_%d:", nd)
		z.emitSpillW()

	case NEG_B:
		z.emitFill()
		z.emit("\teor #$ff")
		z.emit("\tclc")
		z.emit("\tadc #1")
		z.emitSpill()

	case NEG_W:
		z.emitFillW()
		z.emit("\teor #$ff")
		z.emit("\tpha")
		z.emit("\ttxa")
		z.emit("\teor #$ff")
		z.emit("\ttax")
		z.emit("\tpla")
		z.emit("\tclc")
		z.emit("\tadc #1")
		z.emit("\tbcc _neg_w_done")
		z.emit("\tinx")
		z.emit("_neg_w_done:")
		z.emitSpillW()

	// ── Cast ──
	case CAST_W:
		z.emitFill()
		z.emit("\tldx #0")
		z.emitSpillW()

	case CAST_B:
		z.emitFillW()
		//  already has the low byte
		z.emitSpill()

	// ── Stack ──
	case DUP:
		z.emitFillW()
		z.emitSpillW()
		z.emitSpillW()

	case DROP:
		z.emitFillW()

	case SWAP:
		z.emitFillW() // A=low(TOS), X=high(TOS)
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW() // A=low(NEXT), X=high(NEXT)
		z.emit("\tpha")
		z.emit("\ttxa")
		z.emit("\tpha")
		z.emit("\tlda $02")
		z.emit("\tldx $03")
		z.emitSpillW()
		z.emit("\tpla")
		z.emit("\ttax")
		z.emit("\tpla")
		z.emitSpillW()

	// ── Comparison ──
	case IS_B:
		_is_b_lt := z.nextLabel()
		_is_b_ge := z.nextLabel()
		_is_b_eq := z.nextLabel()
		_is_b_done := z.nextLabel()
		z.emitFill()
		z.emit("\tsta $02")
		z.emitFill()
		z.emit("\tcmp $02")
		z.emitf("\tbeq _is_b_%d", _is_b_eq)
		z.emitf("\tbcs _is_b_%d", _is_b_ge)
		switch o.Cond {
		case CondLT:
			z.emitf("_is_b_%d:", _is_b_lt)
			z.emit("\tlda #1")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_ge)
			z.emit("\tlda #0")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_eq)
			z.emit("\tlda #0")
		case CondGT:
			z.emitf("_is_b_%d:", _is_b_lt)
			z.emit("\tlda #0")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_ge)
			z.emit("\tlda #0")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_eq)
			z.emit("\tlda #0")
		case CondLE:
			z.emitf("_is_b_%d:", _is_b_lt)
			z.emit("\tlda #1")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_ge)
			z.emit("\tlda #0")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_eq)
			z.emit("\tlda #1")
		case CondGE:
			z.emitf("_is_b_%d:", _is_b_lt)
			z.emit("\tlda #0")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_ge)
			z.emit("\tlda #1")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_eq)
			z.emit("\tlda #1")
		case CondEQ:
			z.emitf("_is_b_%d:", _is_b_lt)
			z.emit("\tlda #0")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_ge)
			z.emit("\tlda #0")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_eq)
			z.emit("\tlda #1")
		case CondNE:
			z.emitf("_is_b_%d:", _is_b_lt)
			z.emit("\tlda #1")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_ge)
			z.emit("\tlda #1")
			z.emitf("\tjmp _is_b_%d", _is_b_done)
			z.emitf("_is_b_%d:", _is_b_eq)
			z.emit("\tlda #0")
		}
		z.emitf("_is_b_%d:", _is_b_done)
		z.emitSpill()

	case IS_W:
		_is_w_lt := z.nextLabel()
		_is_w_gt := z.nextLabel()
		_is_w_eq := z.nextLabel()
		_is_w_done := z.nextLabel()
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()
		// =low(NEXT), X=high(NEXT), $02=low(TOS), $03=high(TOS)
		z.emit("\tsec")
		z.emit("\tsbc $02")
		z.emit("\tpha")
		z.emit("\ttxa")
		z.emit("\tsbc $03")
		z.emitf("\tbcc _is_w_%d", _is_w_lt)
		z.emitf("\tbne _is_w_%d", _is_w_gt)
		z.emit("\tpla")
		z.emitf("\tbeq _is_w_%d", _is_w_eq)
		z.emitf("\tbcs _is_w_%d", _is_w_gt)
		z.emitf("\tjmp _is_w_%d", _is_w_lt)
		z.emit("")
		z.emitf("_is_w_%d:", _is_w_lt)
		z.emit("\tpla")
		switch o.Cond {
		case CondLT:
			z.emit("\tlda #1")
		case CondGT:
			z.emit("\tlda #0")
		case CondLE:
			z.emit("\tlda #1")
		case CondGE:
			z.emit("\tlda #0")
		case CondEQ:
			z.emit("\tlda #0")
		case CondNE:
			z.emit("\tlda #1")
		}
		z.emitf("\tjmp _is_w_%d", _is_w_done)
		z.emit("")
		z.emitf("_is_w_%d:", _is_w_gt)
		z.emit("\tpla")
		switch o.Cond {
		case CondLT:
			z.emit("\tlda #0")
		case CondGT:
			z.emit("\tlda #1")
		case CondLE:
			z.emit("\tlda #0")
		case CondGE:
			z.emit("\tlda #1")
		case CondEQ:
			z.emit("\tlda #0")
		case CondNE:
			z.emit("\tlda #1")
		}
		z.emitf("\tjmp _is_w_%d", _is_w_done)
		z.emit("")
		z.emitf("_is_w_%d:", _is_w_eq)
		switch o.Cond {
		case CondLT:
			z.emit("\tlda #0")
		case CondGT:
			z.emit("\tlda #0")
		case CondLE:
			z.emit("\tlda #1")
		case CondGE:
			z.emit("\tlda #1")
		case CondEQ:
			z.emit("\tlda #1")
		case CondNE:
			z.emit("\tlda #0")
		}
		z.emitf("\tjmp _is_w_%d", _is_w_done)
		z.emit("")
		z.emitf("_is_w_%d:", _is_w_done)
		z.emit("\tldx #0")
		z.emitSpillW()

	// ── Control Flow ──
	case TAG:
		z.emitf("%s:", o.Name)

	case GO:
		z.emitf("\tjmp %s", o.Name)

	case GO_IF:
		z.emitFillW()
		z.emit("\tora $02")
		z.emit("\tora $03")
		z.emitf("\tbne %s", o.Name)

	// ── Procedures ──
	case ROUTE:
		z.emitf("%s:", o.Name)

	case RUN:
		z.emitf("\tjsr %s", o.Name)

	case FRAME:
		// 6502 has no frame pointer — skip for now
		z.emit("\t; FRAME not yet implemented")

	case LOCAL_B, LOCAL_W:
		// 6502 has no frame pointer — skip for now
		z.emit("\t; LOCAL not yet implemented")

	case DONE:
		z.emit("\trts")

	// ── Tasks ──
	case JOB, BYE, SLEEP, STOP, START, PRIORITY:
		z.emitf("\t; %s not yet implemented", instructionNames[op])

	// ── Port I/O ──
	case IN_B:
		z.emitf("\tlda $%04x", uint16(o.Num))
		z.emitSpill()

	case IN_W:
		z.emitf("\tlda $%04x", uint16(o.Num))
		z.emit("\tpha")
		z.emitf("\tlda $%04x", uint16(o.Num+1))
		z.emit("\ttax")
		z.emit("\tpla")
		z.emitSpillW()

	case OUT_B:
		z.emitFill()
		z.emitf("\tsta $%04x", uint16(o.Num))

	case OUT_W:
		z.emitFillW()
		z.emitf("\tsta $%04x", uint16(o.Num))
		z.emitf("\tstx $%04x", uint16(o.Num+1))

	// ── Interrupts ──
	case INT:
		z.emitf("\t; INT %s not yet implemented", o.Name)
	case NMI:
		z.emitf("\t; NMI %s not yet implemented", o.Name)
	case HLT:
		z.emit("\tbeq _6502_all_done")
	case DII:
		z.emit("\tsei")
	case ENI:
		z.emit("\tcli")

	// ── Random ──
	case SEED:
		z.emit("\t; SEED not yet implemented")
		z.emit("\tlda #42")
		z.emitSpill()

	// ── Bank Switching ──
	case BANK:
		z.emitf("\t; BANK %d not implemented", o.Num)
	case SWITCH:
		z.emit("\t; SWITCH not implemented")

	// ── Data Emission ──
	case DATA_B:
		z.emitf("\t.byte %d", o.Num)
	case DATA_W:
		z.emitf("\t.word %d", o.Num)
	case DATA_STR:
		z.emitf("\t.byte %d", len(o.Str))
		for _, ch := range o.Str {
			z.emitf("\t.byte %d", byte(ch))
		}
	case DATA_TILE:
		z.emit("\t; tile data")
		emitTile6502(z, o.Str)

	// ── Pragma ──
	case PRAGMA:
		z.emitf("\t; PRAGMA %d", o.Num)

	// ── Inline Assembly ──
	case INLINE:
		z.emit("\t" + o.Str)

	// ── Battery RAM ──
	case SRAM_ON:
		z.emit("\t; SRAM_ON not implemented on 6502")
	case SRAM_OFF:
		z.emit("\t; SRAM_OFF not implemented on 6502")
	case SAVE:
		z.emit("\t; SAVE not yet implemented")
	case LOAD:
		z.emit("\t; LOAD not yet implemented")

	default:
		z.emitf("\t; UNKNOWN INSTR %d", op)
	}
}

// ── Data stack helpers ──

// spill pushes A onto the data stack.
func (z *Gen6502) emitSpill() {
	n := z.nextLabel()
	z.emit("\tldy #0")
	z.emit("\tsta ($00),y")
	z.emit("\tinc $00")
	z.emitf("\tbne _spill_%d", n)
	z.emit("\tinc $01")
	z.emitf("_spill_%d:", n)
}

// spillW pushes A (low byte) and X (high byte) onto the data stack.
func (z *Gen6502) emitSpillW() {
	n1 := z.nextLabel()
	n2 := z.nextLabel()
	z.emit("\tldy #0")
	z.emit("\tsta ($00),y")
	z.emit("\tinc $00")
	z.emitf("\tbne _spw_%d", n1)
	z.emit("\tinc $01")
	z.emitf("_spw_%d:", n1)
	z.emit("\ttxa")
	z.emit("\tldy #0")
	z.emit("\tsta ($00),y")
	z.emit("\tinc $00")
	z.emitf("\tbne _spw_%d", n2)
	z.emit("\tinc $01")
	z.emitf("_spw_%d:", n2)
}

// fill pops a byte from the data stack into A.
func (z *Gen6502) emitFill() {
	n := z.nextLabel()
	z.emit("\tdec $00")
	z.emit("\tlda #$ff")
	z.emit("\tcmp $00")
	z.emitf("\tbne _fill_%d", n)
	z.emit("\tdec $01")
	z.emitf("_fill_%d:", n)
	z.emit("\tldy #0")
	z.emit("\tlda ($00),y")
}

// fillW pops a word from the data stack into A (low byte) and X (high byte).
func (z *Gen6502) emitFillW() {
	n1 := z.nextLabel()
	n2 := z.nextLabel()
	z.emit("\tdec $00")
	z.emit("\tlda #$ff")
	z.emit("\tcmp $00")
	z.emitf("\tbne _fw_%d", n1)
	z.emit("\tdec $01")
	z.emitf("_fw_%d:", n1)
	z.emit("\tldy #0")
	z.emit("\tlda ($00),y")
	z.emit("\ttax")
	z.emit("\tdec $00")
	z.emit("\tlda #$ff")
	z.emit("\tcmp $00")
	z.emitf("\tbne _fw_%d", n2)
	z.emit("\tdec $01")
	z.emitf("_fw_%d:", n2)
	z.emit("\tldy #0")
	z.emit("\tlda ($00),y")
}

// Assemble6502 assembles 6502 assembly text into binary code.
// For NES configs, the iNES header and vector table are prepended/appended.
func Assemble6502(cfg Gen6502Config, code string) ([]byte, error) {
	r := bytes.NewReader([]byte(code))
	assembly, _, err := asm.Assemble(r, "plz", cfg.Origin, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("assembly failed: %w", err)
	}
	if len(assembly.Errors) > 0 {
		return nil, fmt.Errorf("assembly errors: %s", strings.Join(assembly.Errors, "; "))
	}
	bin := assembly.Code
	if cfg.NES {
		// iNES header: 16 bytes
		header := []byte{
			0x4e, 0x45, 0x53, 0x1a, // "NES" + MS-DOS EOF
			0x02,       // 2× 16KB PRG banks
			0x00,       // 0× 8KB CHR banks (CHR-RAM)
			0x00,       // flags: horizontal mirroring, NROM
			0x00,       // no mapper
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
		}
		// Pad PRG to fill CPU address space ($C000-$FFFF = 16KB)
		prgEnd := int(cfg.Origin) + len(bin)
		padLen := 0x10000 - prgEnd
		if padLen < 0 {
			return nil, fmt.Errorf("code exceeds PRG-ROM: %d bytes at $%04X", len(bin), cfg.Origin)
		}
		padded := make([]byte, padLen)
		bin = append(bin, padded...)
		// Vectors at $FFFA-$FFFF
		resetVec := uint16(len(bin) - padLen) // relative to start of PRG
		bin = append(bin, 0x00, 0x00) // NMI (dummy)
		bin = append(bin, byte(resetVec), byte(resetVec>>8)) // RESET
		bin = append(bin, 0x00, 0x00) // IRQ (dummy)
		bin = append(header, bin...)
	}
	return bin, nil
}

// emitTile6502 emits 8x8 SMS tile data as .byte directives.
func emitTile6502(z *Gen6502, s string) {
	for _, ch := range s {
		var val byte
		switch {
		case ch == '.':
			val = 0
		case ch >= '0' && ch <= '9':
			val = byte(ch - '0')
		case ch >= 'A' && ch <= 'F':
			val = byte(ch - 'A' + 10)
		case ch >= 'a' && ch <= 'f':
			val = byte(ch - 'a' + 10)
		default:
			val = 0
		}
		z.emitf("\t.byte %d", val)
	}
}
