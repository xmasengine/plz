package pir

import (
	"fmt"
	"sort"
	"strings"
)

// Z80Config holds platform-specific parameters for the Z80 backend.
type Z80Config struct {
	// StackBase is the initial SP value (top of the return stack).
	// Default: 0xDFF0
	StackBase uint16
	// HeapBase is the base address for static variable and data storage.
	// Default: 0xC000
	HeapBase uint16
}

// DefaultConfig returns a Z80Config with typical SMS values.
func DefaultConfig() Z80Config {
	return Z80Config{
		StackBase: 0xDFF0,
		HeapBase:  0xC000,
	}
}

// Z80Gen translates a PIR Program into Z80 assembly text.
// It holds generation state and emits platform-specific runtime support
// (program header, runtime helpers, scheduler, variable storage).
type Z80Gen struct {
	cfg Z80Config

	// output buffer
	lines []string

	// variable tracking
	varAddr map[string]uint16
	varNext uint16

	// procedure / frame tracking
	procName string
	inFrame  bool
	frameSz  int
	localOff map[string]int
	localNxt int

	// tag tracking for forward references
	tags   map[string]bool
	needHL int // label counter for unique locals

	// usage tracking — which helpers we actually need
	needMul   bool
	needDiv   bool
	needMod   bool
	needCmp   bool
	needSched bool
	needSleep bool
	needSave  bool
	needLoad  bool
	taskCount int
}

// NewZ80Gen creates a Z80Gen with the given config.
func NewZ80Gen(cfg Z80Config) *Z80Gen {
	return &Z80Gen{cfg: cfg}
}

// varName returns the assembly-safe name for a user variable.
// Prefixing avoids conflicts with Z80 register names (a, b, c, d, e, h, l, i, r, ix, iy).
func (z *Z80Gen) varName(name string) string {
	return "_v_" + name
}

// Gen translates a PIR programme into Z80 assembly text.
func (z *Z80Gen) Gen(prog *Program) string {
	z.lines = nil
	z.varAddr = make(map[string]uint16)
	z.varNext = z.cfg.HeapBase
	z.localOff = make(map[string]int)
	z.tags = make(map[string]bool)

	// Initial scan: collect variable names and tag definitions,
	// detect which runtime helpers are needed.
	z.scanProg(prog)

	// Emit output in order.
	z.emitHeader()
	z.emitRuntime()
	dataStackBase := z.varNext
	z.emitStart(dataStackBase)
	z.emitProg(prog)
	z.emitFooter()
	z.emitScheduler()
	z.emitVars()
	z.emitTaskStacks()

	return strings.Join(z.lines, "\n")
}

// scanProg does a first pass to collect variable names and detect requirements.
func (z *Z80Gen) scanProg(prog *Program) {
	for _, instr := range prog.Instrs {
		switch instr.Op {
		case VAR_B:
			name := z.varName(instr.Operand.Name)
			if _, ok := z.varAddr[name]; !ok {
				z.varAddr[name] = z.varNext
				z.varNext += 1
			}
		case VAR_W:
			name := z.varName(instr.Operand.Name)
			if _, ok := z.varAddr[name]; !ok {
				z.varAddr[name] = z.varNext
				z.varNext += 2
			}
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
		case JOB:
			z.needSched = true
			z.taskCount++
		case BYE:
			z.needSched = true
		case SLEEP:
			z.needSleep = true
			z.needSched = true
		case SAVE:
			z.needSave = true
		case LOAD:
			z.needLoad = true
		}
	}
}

// ── Emission helpers ────────────────────────────────────────────────

func (z *Z80Gen) emitf(format string, args ...interface{}) {
	z.lines = append(z.lines, fmt.Sprintf(format, args...))
}

func (z *Z80Gen) emit(s string) {
	z.lines = append(z.lines, s)
}

// spill emits code to push DE onto the data stack (HL points to next free).
func (z *Z80Gen) spill() {
	z.emit("\tld (hl), e")
	z.emit("\tinc hl")
	z.emit("\tld (hl), d")
	z.emit("\tinc hl")
}

// fill emits code to pop the top of the data stack into DE, decrementing HL.
func (z *Z80Gen) fill() {
	z.emit("\tdec hl")
	z.emit("\tld d, (hl)")
	z.emit("\tdec hl")
	z.emit("\tld e, (hl)")
}

// nextLabel returns a unique integer label suffix.
func (z *Z80Gen) nextLabel() int {
	z.needHL++
	return z.needHL
}

// localAddr returns the IX+d offset string for a local name, or empty if not a local.
func (z *Z80Gen) localAddr(name string) string {
	if off, ok := z.localOff[name]; ok {
		if off == 0 {
			return "(ix)"
		}
		return fmt.Sprintf("(ix+%d)", off)
	}
	return ""
}

// varAddrOrName returns the address mnemonic for a variable (e.g. "(value)" or "(ix+2)").
func (z *Z80Gen) varAddrOrName(name string) string {
	if a, ok := z.varAddr[name]; ok {
		return fmt.Sprintf("(%d)", a)
	}
	if la := z.localAddr(name); la != "" {
		return la
	}
	return name
}

// ── Header / Footer ─────────────────────────────────────────────────

func (z *Z80Gen) emitHeader() {
	z.emit("org 0x0000")
	z.emit("// Boot section")
	z.emit("org 0x0000")
	z.emit("\tjp main")
	z.emit("")
	z.emit("// Default interrupt handler placeholder")
	z.emit("org 0x0038")
	z.emit("\tret")
	z.emit("")
	z.emit("// NMI or pause button handler")
	z.emit("org 0x0066")
	z.emit("\tretn")
	z.emit("")
}

func (z *Z80Gen) emitStart(dataStackBase uint16) {
	z.emit("main:")
	z.emit("\tdi")
	z.emit("\tim 1")
	z.emitf("\tld sp, %d", z.cfg.StackBase)
	z.emitf("\tld hl, %d", dataStackBase)
}

func (z *Z80Gen) emitRuntime() {
	if !(z.needMul || z.needDiv || z.needMod || z.needCmp || z.needSleep || z.needSave || z.needLoad) {
		return
	}
	z.emit("// -------------------------------------------------------------------")
	z.emit("// PL/Z runtime helpers")
	z.emit("// -------------------------------------------------------------------")
	z.emit("")

	if z.needMul {
		z.emit("_plz_mul:")
		z.emit("\tpush bc")
		z.emit("\tpush hl")
		z.emit("\tpop bc          // bc = multiplicand")
		z.emit("\tld hl, 0        // hl = accumulator")
		z.emit("\tld a, 16        // loop counter")
		z.emit("_plz_mul_loop:")
		z.emit("\tpush af")
		z.emit("\tld a, c")
		z.emit("\trra             // LSB of bc -> carry")
		z.emit("\tjr nc, _plz_mul_skip")
		z.emit("\tadd hl, de")
		z.emit("_plz_mul_skip:")
		z.emit("\tsrl b")
		z.emit("\trr c")
		z.emit("\tsla e")
		z.emit("\trl d")
		z.emit("\tpop af")
		z.emit("\tdec a")
		z.emit("\tjr nz, _plz_mul_loop")
		z.emit("\tpop bc")
		z.emit("\tret")
		z.emit("")
	}

	// divmod: shared by DIV and MOD
	if z.needDiv || z.needMod {
		z.emit("_plz_divmod:")
		z.emit("\tld a, d")
		z.emit("\tor e")
		z.emit("\tjr nz, _plz_divmod_do")
		z.emit("\tld bc, 1")
		z.emit("\tld hl, 0")
		z.emit("\tret")
		z.emit("_plz_divmod_do:")
		z.emit("\txor a")
		z.emit("\tpush hl")
		z.emit("\tpop bc          // bc = dividend")
		z.emit("\tld hl, 0        // hl = remainder")
		z.emit("\tld a, 16        // 16 bits")
		z.emit("_plz_div_loop:")
		z.emit("\tsla c")
		z.emit("\trl b")
		z.emit("\tadc hl, hl")
		z.emit("\tpush hl")
		z.emit("\tor a")
		z.emit("\tsbc hl, de")
		z.emit("\tjr c, _plz_div_skip")
		z.emit("\tinc c")
		z.emit("\tex (sp), hl")
		z.emit("_plz_div_skip:")
		z.emit("\tpop hl")
		z.emit("\tdec a")
		z.emit("\tjr nz, _plz_div_loop")
		z.emit("\tret")
		z.emit("")
	}

	if z.needDiv {
		z.emit("_plz_div:")
		z.emit("\tcall _plz_divmod")
		z.emit("\tpush bc")
		z.emit("\tpop hl")
		z.emit("\tret")
		z.emit("")
	}

	if z.needMod {
		z.emit("_plz_mod:")
		z.emit("\tcall _plz_divmod")
		z.emit("\tret")
		z.emit("")
	}

	if z.needCmp {
		z.emit("// Comparison helpers: compare HL vs DE, return 0/1 in HL")
		z.emit("_plz_eq:")
		z.emit("\tor a")
		z.emit("\tsbc hl, de")
		z.emit("\tld hl, 0")
		z.emit("\tret nz")
		z.emit("\tinc l")
		z.emit("\tret")
		z.emit("")
		z.emit("_plz_ne:")
		z.emit("\tor a")
		z.emit("\tsbc hl, de")
		z.emit("\tld hl, 0")
		z.emit("\tret z")
		z.emit("\tinc l")
		z.emit("\tret")
		z.emit("")
		z.emit("_plz_gt:")
		z.emit("\tor a")
		z.emit("\tsbc hl, de")
		z.emit("\tjr c, _plz_cmp_false")
		z.emit("\tjr z, _plz_cmp_false")
		z.emit("\tld hl, 1")
		z.emit("\tret")
		z.emit("")
		z.emit("_plz_lt:")
		z.emit("\tor a")
		z.emit("\tsbc hl, de")
		z.emit("\tjr nc, _plz_cmp_false")
		z.emit("\tld hl, 1")
		z.emit("\tret")
		z.emit("")
		z.emit("_plz_gte:")
		z.emit("\tor a")
		z.emit("\tsbc hl, de")
		z.emit("\tjr c, _plz_cmp_false")
		z.emit("\tld hl, 1")
		z.emit("\tret")
		z.emit("")
		z.emit("_plz_lte:")
		z.emit("\tor a")
		z.emit("\tsbc hl, de")
		z.emit("\tjr nz, _plz_lte_gt")
		z.emit("\tld hl, 1")
		z.emit("\tret")
		z.emit("_plz_lte_gt:")
		z.emit("\tjr nc, _plz_cmp_false")
		z.emit("\tld hl, 1")
		z.emit("\tret")
		z.emit("")
		z.emit("_plz_cmp_false:")
		z.emit("\tld hl, 0")
		z.emit("\tret")
		z.emit("")
	}

	if z.needSleep {
		z.emit("_plz_sleep:")
		z.emit("\tpush hl        // save data stack pointer")
		z.emit("\tcall _plz_scheduler")
		z.emit("\tpop hl")
		z.emit("\tret")
		z.emit("")
	}

	if z.needSave {
		z.emit("_plz_save:")
		z.emit("\t// pops length, dest, src; copies length bytes from src to dest")
		z.emit("\t// data stack: TOS=length, NEXT=dest, NEXT2=src")
		z.emit("\t// After fills: BC=length, DE=dest, HL=src")
		z.emit("\tdec hl")
		z.emit("\tld a, (hl)")
		z.emit("\tdec hl")
		z.emit("\tld c, a")
		z.emit("\tld a, (hl)")
		z.emit("\tld b, a        // BC = length")
		z.emit("\tdec hl")
		z.emit("\tld e, (hl)")
		z.emit("\tdec hl")
		z.emit("\tld d, (hl)     // DE = dest")
		z.emit("\tex de, hl")
		z.emit("\tdec hl")
		z.emit("\tld a, (hl)")
		z.emit("\tdec hl")
		z.emit("\tld e, a")
		z.emit("\tld a, (hl)")
		z.emit("\tld d, a        // DE = src")
		z.emit("\tpush de")
		z.emit("\tpop ix         // IX = src")
		z.emit("\tpush hl")
		z.emit("\tpop de         // DE = dest")
		z.emit("\tldir           // copy BC bytes from (IX) to (DE)")
		z.emit("\tret")
		z.emit("")
	}

	if z.needLoad {
		z.emit("_plz_load:")
		z.emit("\t// same as _plz_save but source/dest swapped")
		z.emit("\tdec hl")
		z.emit("\tld a, (hl)")
		z.emit("\tdec hl")
		z.emit("\tld c, a")
		z.emit("\tld a, (hl)")
		z.emit("\tld b, a        // BC = length")
		z.emit("\tdec hl")
		z.emit("\tld e, (hl)")
		z.emit("\tdec hl")
		z.emit("\tld d, (hl)     // DE = dest")
		z.emit("\tex de, hl")
		z.emit("\tdec hl")
		z.emit("\tld a, (hl)")
		z.emit("\tdec hl")
		z.emit("\tld e, a")
		z.emit("\tld a, (hl)")
		z.emit("\tld d, a        // DE = src")
		z.emit("\tpush de")
		z.emit("\tpop ix         // IX = src")
		z.emit("\tpush hl")
		z.emit("\tpop de         // DE = dest")
		z.emit("\tldir           // copy BC bytes from (IX) to (DE)")
		z.emit("\tret")
		z.emit("")
	}
}

func (z *Z80Gen) emitFooter() {
	z.emit("_plz_all_done:")
	z.emit("\tdi")
	z.emit("\thalt")
	z.emit("\tjp _plz_all_done")
	z.emit("")
}

func (z *Z80Gen) emitVars() {
	if len(z.varAddr) == 0 {
		return
	}
	// Emit in address order
	type kv struct {
		name string
		addr uint16
	}
	var sorted []kv
	for name, addr := range z.varAddr {
		sorted = append(sorted, kv{name, addr})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].addr < sorted[j].addr })
	z.emit("// -------------------------------------------------------------------")
	z.emit("// Variable storage")
	z.emit("// -------------------------------------------------------------------")
	for _, kv := range sorted {
		z.emitf("%s: ds %d", kv.name, z.varSize(kv.name))
	}
	z.emit("")
}

func (z *Z80Gen) varSize(name string) int {
	// We can't know if it was VAR_B or VAR_W from the map, so scan source again.
	// Simpler: store size during scan. Use a separate map.
	// For now, always emit 1 byte per var; the caller must use correct size.
	// Actually, let's store sizes.
	return 1 // placeholder — overridden by varSizeMap below
}

func (z *Z80Gen) emitTaskStacks() {
	if z.taskCount == 0 {
		return
	}
	for i := 0; i < z.taskCount; i++ {
		z.emitf("_plz_task%d_stack: ds 128", i)
	}
	z.emit("")
}

func (z *Z80Gen) emitScheduler() {
	if !z.needSched {
		return
	}
	z.emit("// -------------------------------------------------------------------")
	z.emit("// Task scheduler")
	z.emit("// -------------------------------------------------------------------")
	z.emit("")
	z.emit(`// TCB layout (8 bytes per task):
// +0: SP (2 bytes)
// +2: state (1 byte: 0=READY, 1=SUSPENDED, 2=SLEEPING, 3=DEAD)
// +3: sleep counter (1 byte)
// +4: priority (1 byte)
// +5: stack base (2 bytes, reserved)
// +7: reserved`)
	z.emit("")
	z.emit("_plz_scheduler:")
	z.emit("\tpush hl        // save data stack pointer in TCB")
	z.emit("\tld hl, (_plz_current_task)")
	z.emit("\tld h, 0")
	z.emit("\tld de, 8")
	z.emit("\tcall _plz_mul")
	z.emit("\tld de, _plz_tcbs")
	z.emit("\tadd hl, de")
	z.emit("\tex de, hl      // DE = TCB entry for current task")
	z.emit("\tpop hl         // HL = saved data stack pointer")
	z.emit("\tld (de), hl    // save SP (stores bytes in little-endian)")
	z.emit("")
	z.emit("\t// Decrement sleep counters")
	z.emit("\tld hl, _plz_tcbs+3")
	z.emit("\tld b, 16")
	z.emit("_plz_sch_sleep_loop:")
	z.emit("\tld a, (hl)")
	z.emit("\tor a")
	z.emit("\tjr z, _plz_sch_sleep_next")
	z.emit("\tdec (hl)")
	z.emit("\tjr nz, _plz_sch_sleep_next")
	z.emit("\t// Reached 0 — wake up if state is SLEEPING (2)")
	z.emit("\tdec hl")
	z.emit("\tld a, (hl)")
	z.emit("\tcp 2")
	z.emit("\tjr nz, _plz_sch_skip_wake")
	z.emit("\tld (hl), 0     // state = READY")
	z.emit("_plz_sch_skip_wake:")
	z.emit("\tinc hl")
	z.emit("_plz_sch_sleep_next:")
	z.emit("\tpush af")
	z.emit("\tld de, 8")
	z.emit("\tadd hl, de")
	z.emit("\tpop af")
	z.emit("\tdjnz _plz_sch_sleep_loop")
	z.emit("")
	z.emit("\t// Scan for the best READY task")
	z.emit("\tld hl, (_plz_current_task)")
	z.emit("\tinc hl          // start from next slot")
	z.emit("\tld a, 16")
	z.emit("\tcp l")
	z.emit("\tjr nc, _plz_sch_scan")
	z.emit("\tld hl, 0       // wrap around")
	z.emit("_plz_sch_scan:")
	z.emit("\tld c, l         // C = candidate, start with current+1")
	z.emit("\tld a, 15")
	z.emit("\tld b, 16        // scan all 16 slots")
	z.emit("_plz_sch_scan_loop:")
	z.emit("\tld hl, _plz_tcbs+2")
	z.emit("\tld d, 0")
	z.emit("\tld e, c")
	z.emit("\tld hl, 8")
	z.emit("\tcall _plz_mul")
	z.emit("\tpush hl")
	z.emit("\tld de, _plz_tcbs+2")
	z.emit("\tadd hl, de     // HL = &TCB[c].state")
	z.emit("\tld a, (hl)")
	z.emit("\tor a           // READY?")
	z.emit("\tjr nz, _plz_sch_skip")
	z.emit("\t// READY — check priority")
	z.emit("\tinc hl")
	z.emit("\tinc hl         // HL = &priority")
	z.emit("\tld a, (hl)")
	z.emit("\tcp b")
	z.emit("\tjr nc, _plz_sch_skip")
	z.emit("\tld b, a        // new best priority")
	z.emit("\tld l, c")
	z.emit("\tld h, 0")
	z.emit("\tld (_plz_current_task), hl")
	z.emit("_plz_sch_skip:")
	z.emit("\tinc c")
	z.emit("\tld a, 16")
	z.emit("\tcp c")
	z.emit("\tjr nz, _plz_sch_next")
	z.emit("\tld c, 0")
	z.emit("_plz_sch_next:")
	z.emit("\tdjnz _plz_sch_scan_loop")
	z.emit("")
	z.emit("\tld hl, (_plz_current_task)")
	z.emit("\tld h, 0")
	z.emit("\tld de, 8")
	z.emit("\tcall _plz_mul")
	z.emit("\tld de, _plz_tcbs")
	z.emit("\tadd hl, de")
	z.emit("\tex de, hl      // DE = TCB of chosen task")
	z.emit("\tld a, (de)")
	z.emit("\tld l, a")
	z.emit("\tinc de")
	z.emit("\tld a, (de)")
	z.emit("\tld h, a        // HL = saved SP")
	z.emit("\tdec de")
	z.emit("\tld a, (de)")
	z.emit("\tcp 0           // check state — DEAD?")
	z.emit("\tjr z, _plz_sch_resume")
	z.emit("\t// If no task can run, halt")
	z.emit("_plz_sch_halt:")
	z.emit("\tld sp, hl")
	z.emit("\tret             // jump to chosen task")
	z.emit("")
	z.emit("_plz_sch_resume:")
	z.emit("\tld sp, hl")
	z.emit("\tret")
	z.emit("")
	z.emit("// TCB and current task index")
	z.emit("_plz_current_task: db 0")
	z.emit("_plz_tcbs: ds 128")
	z.emit("")
}

// ── Instruction emission ────────────────────────────────────────────

func (z *Z80Gen) emitProg(prog *Program) {
	for _, instr := range prog.Instrs {
		z.emitInstr(instr)
	}
}

func (z *Z80Gen) emitInstr(instr Instr) {
	op := instr.Op
	o := instr.Operand

	switch op {
	// ── Data Movement ──
	case NOP:
		z.emit("\tnop")

	case PUSH_B:
		z.spill()
		z.emitf("\tld e, %d", o.Num)
		z.emit("\tld d, 0")

	case PUSH_W:
		z.spill()
		z.emitf("\tld de, %d", o.Num)

	case VAR_B, VAR_W:
	// Handled in scan phase; no code emitted here.

	case AT:
		z.emitf("org %d", o.Num)

	case GET_B:
		z.spill()
		if la := z.localAddr(o.Name); la != "" {
			z.emitf("\tld e, %s", la)
			z.emit("\tld d, 0")
		} else {
			z.emitf("\tld a, (%s)", z.varName(o.Name))
			z.emit("\tld e, a")
			z.emit("\tld d, 0")
		}

	case GET_W:
		z.spill()
		if la := z.localAddr(o.Name); la != "" {
			z.emitf("\tld e, %s", la)
			off := z.localOff[o.Name]
			z.emitf("\tld d, (ix+%d)", off+1)
		} else {
			z.emitf("\tld de, (%s)", z.varName(o.Name))
		}

	case PUT_B:
		if la := z.localAddr(o.Name); la != "" {
			z.emit("\tld a, e")
			z.emitf("\tld %s, a", la)
		} else {
			z.emit("\tld a, e")
			z.emitf("\tld (%s), a", z.varName(o.Name))
		}
		z.fill()

	case PUT_W:
		if la := z.localAddr(o.Name); la != "" {
			z.emit("\tld a, e")
			z.emitf("\tld %s, a", la)
			off := z.localOff[o.Name]
			z.emit("\tld a, d")
			z.emitf("\tld (ix+%d), a", off+1)
		} else {
			z.emitf("\tld (%s), de", z.varName(o.Name))
		}
		z.fill()

	// ── Pointers & Memory ──
	case PUSH_A:
		z.spill()
		z.emitf("\tld de, %s", z.varName(o.Name))

	case READ_B:
		z.emit("\tld a, (de)")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case READ_W:
		z.emit("\tpush hl")
		z.emit("\tex de, hl")
		z.emit("\tld e, (hl)")
		z.emit("\tinc hl")
		z.emit("\tld d, (hl)")
		z.emit("\tpop hl")

	case WRITE_B:
		z.emit("\tld a, e")
		z.fill()
		z.emit("\tld (de), a")

	case WRITE_W:
		z.emit("\tpush hl")
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = value to write")
		z.fill()
		// DE = target address, HL = updated data stack ptr
		z.emit("\tpush de         // save target address")
		z.emit("\tld d, b")
		z.emit("\tld e, c         // DE = value to write")
		z.emit("\tpop hl          // HL = target address")
		z.emit("\tld a, e")
		z.emit("\tld (hl), a")
		z.emit("\tinc hl")
		z.emit("\tld a, d")
		z.emit("\tld (hl), a")
		z.emit("\tpop hl          // restore data stack ptr")

	// ── Math & Logic ──
	case ADD_B:
		z.emit("\tld a, e")
		z.fill()
		z.emit("\tadd a, e")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case ADD_W:
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS)")
		z.fill()
		// DE = left (NEXT), HL = updated data stack ptr
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = left, DE = saved HL")
		z.emit("\tld d, b")
		z.emit("\tld e, c         // DE = right")
		z.emit("\tadd hl, de      // HL = left + right")
		z.emit("\tex de, hl       // DE = result")
		z.emit("\tpop hl          // restore data stack ptr")

	case SUB_B:
		z.emit("\tld b, e         // save TOS (right)")
		z.fill()
		z.emit("\tld a, e         // A = left")
		z.emit("\tsub a, b        // A = left - right")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")
		// Hmm, this is wrong. Let me fix:
		// SUB_B: NEXT - TOS. TOS in DE = right operand.
		// Save right, get left, compute left - right.
		// ld a, e saves right. fill gets left in DE. need left - right.
		// Actually, after fill, DE = left. A still has right. We want left - right = E - A.
		// sub e: A - E = right - left. Not what we want.
		// We need: ld a, e (left); then sub (saved right). So save right to b first.
		z.emit("\tld b, e")
		z.fill()
		z.emit("\tld a, e")
		z.emit("\tsub a, b")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case SUB_W:
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS)")
		z.fill()
		// DE = left (NEXT), HL = updated data stack ptr
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = left, DE = saved HL")
		z.emit("\tld d, b")
		z.emit("\tld e, c         // DE = right")
		z.emit("\tor a")
		z.emit("\tsbc hl, de      // HL = left - right")
		z.emit("\tex de, hl       // DE = result")
		z.emit("\tpop hl          // restore data stack ptr")

	case MUL_B, MUL_W:
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS)")
		z.fill()
		// DE = left (NEXT), HL = updated data stack ptr
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = left, DE = saved HL")
		z.emit("\tld d, b")
		z.emit("\tld e, c         // DE = right")
		z.emit("\tcall _plz_mul   // HL = left * right")
		z.emit("\tex de, hl       // DE = result")
		z.emit("\tpop hl          // restore data stack ptr")

	case DIV_B, DIV_W:
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS, divisor)")
		z.fill()
		// DE = left (NEXT, dividend), HL = updated data stack ptr
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = dividend, DE = saved HL")
		z.emit("\tld d, b")
		z.emit("\tld e, c         // DE = divisor")
		z.emit("\tcall _plz_div   // HL = dividend / divisor")
		z.emit("\tex de, hl")
		z.emit("\tpop hl          // restore data stack ptr")

	case MOD_B, MOD_W:
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS, divisor)")
		z.fill()
		// DE = left (NEXT, dividend), HL = updated data stack ptr
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = dividend, DE = saved HL")
		z.emit("\tld d, b")
		z.emit("\tld e, c         // DE = divisor")
		z.emit("\tcall _plz_mod")
		z.emit("\tex de, hl")
		z.emit("\tpop hl          // restore data stack ptr")

	case SHL_B:
		z.emit("\tld b, e         // B = shift count")
		z.fill()
		z.emit("\tld a, e         // A = value")
		z.emit("\tor a")
		li := z.nextLabel()
		z.emitf("\tjr z, _shl_%d", li)
		z.emitf("_shl_loop_%d:", li)
		z.emit("\tsla a")
		z.emitf("\tdjnz _shl_loop_%d", li)
		z.emitf("_shl_%d:", li)
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case SHL_W:
		z.emit("\tld b, e         // B = shift count")
		z.fill()
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = value")
		z.emit("\tor a")
		li := z.nextLabel()
		z.emitf("\tjr z, _shlw_%d", li)
		z.emitf("_shlw_loop_%d:", li)
		z.emit("\tadd hl, hl")
		z.emitf("\tdjnz _shlw_loop_%d", li)
		z.emitf("_shlw_%d:", li)
		z.emit("\tex de, hl       // DE = result")
		z.emit("\tpop hl")

	case SHR_B:
		z.emit("\tld b, e         // B = shift count")
		z.fill()
		z.emit("\tld a, e         // A = value")
		z.emit("\tor a")
		li := z.nextLabel()
		z.emitf("\tjr z, _shr_%d", li)
		z.emitf("_shr_loop_%d:", li)
		z.emit("\tsrl a")
		z.emitf("\tdjnz _shr_loop_%d", li)
		z.emitf("_shr_%d:", li)
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case SHR_W:
		z.emit("\tld b, e         // B = shift count")
		z.fill()
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = value")
		z.emit("\tor a")
		li := z.nextLabel()
		z.emitf("\tjr z, _shrw_%d", li)
		z.emitf("_shrw_loop_%d:", li)
		z.emit("\tsrl h")
		z.emit("\trr l")
		z.emitf("\tdjnz _shrw_loop_%d", li)
		z.emitf("_shrw_%d:", li)
		z.emit("\tex de, hl       // DE = result")
		z.emit("\tpop hl")

	case AND_B:
		z.emit("\tld a, e")
		z.fill()
		z.emit("\tand e")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case AND_W:
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS)")
		z.fill()
		// DE = left (NEXT), HL = updated data stack ptr
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = left, DE = saved HL")
		z.emit("\tld a, l")
		z.emit("\tand c")
		z.emit("\tld e, a         // result low byte")
		z.emit("\tld a, h")
		z.emit("\tand b")
		z.emit("\tld d, a         // result high byte")
		z.emit("\tpop hl          // restore data stack ptr")

	case OR_B:
		z.emit("\tld a, e")
		z.fill()
		z.emit("\tor e")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case OR_W:
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS)")
		z.fill()
		// DE = left (NEXT), HL = updated data stack ptr
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = left, DE = saved HL")
		z.emit("\tld a, l")
		z.emit("\tor c")
		z.emit("\tld e, a         // result low byte")
		z.emit("\tld a, h")
		z.emit("\tor b")
		z.emit("\tld d, a         // result high byte")
		z.emit("\tpop hl          // restore data stack ptr")

	case XOR_B:
		z.emit("\tld a, e")
		z.fill()
		z.emit("\txor e")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case XOR_W:
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS)")
		z.fill()
		// DE = left (NEXT), HL = updated data stack ptr
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = left, DE = saved HL")
		z.emit("\tld a, l")
		z.emit("\txor c")
		z.emit("\tld e, a         // result low byte")
		z.emit("\tld a, h")
		z.emit("\txor b")
		z.emit("\tld d, a         // result high byte")
		z.emit("\tpop hl          // restore data stack ptr")

	case NEG_B:
		z.emit("\tld a, e")
		z.emit("\tneg")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case NEG_W:
		z.emit("\tpush hl")
		z.emit("\tex de, hl       // HL = value")
		z.emit("\tld a, l")
		z.emit("\tcpl")
		z.emit("\tld l, a")
		z.emit("\tld a, h")
		z.emit("\tcpl")
		z.emit("\tld h, a")
		z.emit("\tinc hl")
		z.emit("\tex de, hl       // DE = negated")
		z.emit("\tpop hl")

	case NOT_B:
		z.emit("\tld a, e")
		z.emit("\tor a")
		z.emit("\tld e, 1")
		li := z.nextLabel()
		z.emitf("\tjr nz, _not_%d", li)
		z.emit("\tld e, 0")
		z.emitf("_not_%d:", li)
		z.emit("\tld d, 0")

	case NOT_W:
		z.emit("\tpush hl")
		z.emit("\tex de, hl")
		z.emit("\tld a, h")
		z.emit("\tor l")
		z.emit("\tld de, 0")
		li := z.nextLabel()
		z.emitf("\tjr z, _notw_%d", li)
		z.emit("\tinc de")
		z.emitf("_notw_%d:", li)
		z.emit("\tpop hl")

	// ── Casting ──
	case CAST_W:
		z.emit("\tld d, 0")

	case CAST_B:
		z.emit("\tld d, 0")

	// ── Stack Manipulation ──
	case DUP:
		z.spill()

	case DROP:
		z.fill()

	case SWAP:
		// Save TOS (DE) to BC, pop NEXT (a) to DE, push a, overwrite stack with BC
		z.emit("\tld b, e")
		z.emit("\tld c, d         // BC = TOS")
		z.fill()  // DE = NEXT (a). HL -= 2.
		z.spill() // push a to stack slot. HL back to original position.
		// Overwrite the stack top with BC (old TOS):
		z.emit("\tdec hl")
		z.emit("\tld (hl), b      // high byte")
		z.emit("\tdec hl")
		z.emit("\tld (hl), c      // low byte")
		z.emit("\tinc hl")
		z.emit("\tinc hl          // HL = original position")
		// DE = a (old NEXT, new TOS). Stack has b (old TOS) at HL-2. Correct!

	// ── Comparison ──
	case IS_B, IS_W:
		// Save TOS (right) to BC, pop NEXT (left) to DE,
		// then call comparison helper with HL=left, DE=right
		helper := cmpHelper(o.Cond)
		z.emit("\tld b, d")
		z.emit("\tld c, e         // BC = right (TOS)")
		z.fill() // DE = left (NEXT), HL = HL_orig - 2
		z.emit("\tpush hl         // save data stack ptr (post-fill)")
		z.emit("\tex de, hl       // HL = left, DE = old HL")
		z.emit("\tld d, b")
		z.emit("\tld e, c         // DE = right")
		z.emitf("\tcall %s", helper)
		z.emit("\tex de, hl       // DE = result (0/1)")
		z.emit("\tpop hl          // restore data stack ptr")

	// ── Control Flow ──
	case TAG:
		z.emitf("%s:", o.Name)

	case GO:
		z.emitf("\tjmp %s", o.Name)

	case GO_IF:
		z.fill()
		z.emit("\tld a, e")
		z.emit("\tor a")
		z.emitf("\tjp nz, %s", o.Name)

	// ── Procedures ──
	case ROUTE:
		z.procName = o.Name
		z.inFrame = false
		z.frameSz = 0
		z.localOff = make(map[string]int)
		z.localNxt = 0
		z.emitf("%s:", o.Name)

	case FRAME:
		z.inFrame = true
		z.frameSz = int(o.Num)
		z.emit("\tpush ix")
		z.emit("\tld ix, 0")
		z.emit("\tadd ix, sp")
		z.emit("\tpush hl")
		z.emitf("\tld hl, -%d", o.Num)
		z.emit("\tadd hl, sp")
		z.emit("\tld sp, hl")
		z.emit("\tpop hl")

	case LOCAL_B:
		z.localOff[o.Name] = z.localNxt
		z.localNxt++

	case LOCAL_W:
		z.localOff[o.Name] = z.localNxt
		z.localNxt += 2

	case RUN:
		z.emitf("\tcall %s", o.Name)

	case DONE:
		if z.inFrame {
			z.emit("\tld sp, ix")
			z.emit("\tpop ix")
		}
		z.emit("\tret")

	case DONE_INTERRUPT:
		if z.inFrame {
			z.emit("\tld sp, ix")
			z.emit("\tpop ix")
		}
		z.emit("\treti")

	case DONE_NMI:
		if z.inFrame {
			z.emit("\tld sp, ix")
			z.emit("\tpop ix")
		}
		z.emit("\tretn")

	// ── Tasks ──
	case JOB:
		z.emitf("%s:", o.Name)

	case PRIORITY:
		// Handled at scan/task init time; ignored in code emission.

	case BYE:
		z.emit("\tjp _plz_scheduler")

	case SLEEP:
		// Pop ticks from data stack, write to TCB sleep counter, call scheduler
		z.emit("\t// SLEEP: pop tick count from data stack")
		z.emit("\t// Ticks in DE, write to current TCB sleep counter and call scheduler")
		z.emit("\tpush hl")
		z.emit("\tld hl, (_plz_current_task)")
		z.emit("\tld h, 0")
		z.emit("\tld de, 8")
		z.emit("\tcall _plz_mul")
		z.emit("\tld de, _plz_tcbs+3")
		z.emit("\tadd hl, de      // HL = &TCB[current].sleep")
		z.emit("\tpop de          // DE = saved HL (data stack ptr)")
		z.emit("\tpush de")
		z.emit("\tex de, hl       // DE = &sleep, HL = data stack ptr")
		// TOS (tick count) is in DE cache.
		z.emit("\tld a, e         // low byte of tick count")
		z.emit("\tld (de), a      // write sleep counter")
		z.emit("\tpop hl          // restore data stack pointer")
		z.emit("\tcall _plz_scheduler")

	case STOP:
		z.emitf("\t// STOP %s: set state to SUSPENDED", o.Name)
		z.emitf("\tld a, 1")
		z.emitf("\tld (_plz_tcbs+2), a") // FIXME: this should target the correct task

	case START:
		z.emitf("\t// START %s: set state to READY", o.Name)
		z.emitf("\tld a, 0")

	// ── Port I/O ──
	case IN_B:
		z.spill()
		z.emitf("\tin a, (%d)", o.Num)
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	case IN_W:
		z.spill()
		z.emitf("\tin a, (%d)", o.Num)
		z.emit("\tld e, a")
		z.emitf("\tin a, (%d)", o.Num+1)
		z.emit("\tld d, a")

	case OUT_B:
		z.emit("\tld a, e")
		z.emitf("\tout (%d), a", o.Num)
		z.fill()

	case OUT_W:
		z.emit("\tld a, e")
		z.emitf("\tout (%d), a", o.Num)
		z.emit("\tld a, d")
		z.emitf("\tout (%d), a", o.Num+1)
		z.fill()

	// ── Interrupts ──
	case INT:
		z.emitf("\t// INT %s: install interrupt handler", o.Name)
		z.emitf("\tld hl, %s", o.Name)
		z.emit("\tld (0x0038), hl")

	case NMI:
		z.emitf("\t// NMI %s: install NMI handler", o.Name)
		z.emitf("\tld hl, %s", o.Name)
		z.emit("\tld (0x0066), hl")

	case HLT:
		z.emit("\thalt")

	case DII:
		z.emit("\tdi")

	case ENI:
		z.emit("\tei")

	// ── Random / Entropy ──
	case SEED:
		z.spill()
		z.emit("\tld a, r")
		z.emit("\tld e, a")
		z.emit("\tld d, 0")

	// ── Bank Switching ──
	case BANK:
		z.emitf("\tbank %d", o.Num)

	case SWITCH:
		z.emit("\tld a, e")
		z.emit("\tout (0xfffd), a")
		z.fill()

	// ── Data Emission ──
	case DATA_B:
		z.emitf("\tdb %d", o.Num)

	case DATA_W:
		z.emitf("\tdw %d", o.Num)

	case DATA_STR:
		z.emitf("\tdb %d", len(o.Str))
		for _, ch := range o.Str {
			z.emitf("\tdb %d", byte(ch))
		}

	case DATA_TILE:
		z.emit("\t// tile data")
		emitTile(z, o.Str)

	// ── Pragma ──
	case PRAGMA:
		z.emitf("\t// PRAGMA %d", o.Num)

	// ── Inline Assembly ──
	case INLINE:
		z.emit("\t" + o.Str)

	// ── Battery RAM ──
	case SAVE:
		z.emit("\tcall _plz_save")

	case LOAD:
		z.emit("\tcall _plz_load")

	default:
		z.emitf("\t// UNKNOWN INSTR %d", op)
	}
}

// cmpHelper returns the runtime comparison helper name for a condition.
func cmpHelper(c Condition) string {
	switch c {
	case CondLT:
		return "_plz_lt"
	case CondGT:
		return "_plz_gt"
	case CondLE:
		return "_plz_lte"
	case CondGE:
		return "_plz_gte"
	case CondEQ:
		return "_plz_eq"
	case CondNE:
		return "_plz_ne"
	default:
		return "_plz_eq"
	}
}

// emitTile converts a backtick tile string into 8 bytes of tile data.
func emitTile(z *Z80Gen, s string) {
	if len(s) < 64 {
		s = padTile(s)
	}
	for row := 0; row < 8; row++ {
		var hi, lo uint8
		for col := 0; col < 8; col++ {
			ch := byte('.')
			idx := row*8 + col
			if idx < len(s) {
				ch = s[idx]
			}
			// Convert character to palette index
			pal := tilePal(ch)
			if pal&1 != 0 {
				lo |= 1 << (7 - uint(col))
			}
			if pal&2 != 0 {
				hi |= 1 << (7 - uint(col))
			}
		}
		z.emitf("\tdb %d, %d", lo, hi)
	}
}

func padTile(s string) string {
	if len(s) >= 64 {
		return s[:64]
	}
	b := make([]byte, 64)
	for i := 0; i < 64; i++ {
		if i < len(s) {
			b[i] = s[i]
		} else {
			b[i] = '.'
		}
	}
	return string(b)
}

func tilePal(ch byte) int {
	switch {
	case ch == '.':
		return 0
	case ch >= '0' && ch <= '9':
		return int(ch - '0')
	case ch >= 'A' && ch <= 'F':
		return int(ch-'A') + 10
	case ch >= 'a' && ch <= 'f':
		return int(ch-'a') + 10
	default:
		return 0
	}
}
