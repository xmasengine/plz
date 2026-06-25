package pir

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/xmasengine/plz/pkg/asm6502"
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
	// OutputBase is the base address for OUT_B/OUT_W writes (0 = direct port address).
	OutputBase uint16
	// IntHandlerName is the PL/Z name of the INT handler, set by Gen6502 during Gen().
	IntHandlerName string
	// NmiHandlerName is the PL/Z name of the NMI handler, set by Gen6502 during Gen().
	NmiHandlerName string
}

// Default6502Config returns a configuration suitable for testing.
func Default6502Config() Gen6502Config {
	return Gen6502Config{
		Origin:    0x1000,
		VarBase:   0x2000,
		StackBase: 0x3000,
		NES:       false,
		TaskLimit: 16,
		OutputBase: 0x5000,
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
		OutputBase: 0x5000,
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
	atVars   map[string]bool // variables placed via AT directive

	// ag tracking
	tags    map[string]bool
	labelID int

	// usage tracking — which helpers we need
	needMul   bool
	needDiv   bool
	needMod   bool
	needCmp   bool
	needShift bool
	needSched bool
	needSave  bool
	needLoad  bool
	needSeed  bool

	// task tracking
	taskCount        int
	taskEntryLabels  []string
	taskNames        map[string]int // task name → index
	taskPriorities   []int          // one per task, set by PRIORITY before JOB
	pendingPriority  int            // -1 when none

	// reentrant procedure (frame) tracking
	inFrame  bool
	frameSz  int
	localOff map[string]int // local name → offset from FP
	localNxt int            // next local offset

	// interrupt handler tracking
	intHandler string // name set by INT instruction
	nmiHandler string // name set by NMI instruction

	// one-shot directive tracking
	pendingAT   int // >=0 when AT is active; -1 when none
	pendingSize int // >0 overrides size for next VAR; 0 when none

	// bank tracking
	currentBank int            // current bank (0 = code bank)
	bankLines   map[int][]string // bank -> its assembly lines (bank 0 is main code)
}

// IntHandler returns the PL/Z name of the INT handler, or empty if none.
func (z *Gen6502) IntHandler() string { return z.intHandler }

// NmiHandler returns the PL/Z name of the NMI handler, or empty if none.
func (z *Gen6502) NmiHandler() string { return z.nmiHandler }

// NewGen6502 creates a Gen6502 with the given config.
func NewGen6502(cfg Gen6502Config) *Gen6502 {
	return &Gen6502{cfg: cfg, varSizes: make(map[string]int), pendingAT: -1, localOff: make(map[string]int), bankLines: make(map[int][]string), atVars: make(map[string]bool)}
}

func (z *Gen6502) varName(name string) string {
	return "_v_" + name
}

// localOff returns the frame offset for a local variable, or -1 if not a local.
func (z *Gen6502) localAddr(name string) int {
	if off, ok := z.localOff[name]; ok {
		return off
	}
	return -1
}

// Gen translates a PIR programme into 6502 assembly text.
// After Gen(), BankLines() returns per-bank assembly texts for banks > 0.
func (z *Gen6502) Gen(prog *Program) string {
	z.lines = nil
	z.varAddr = make(map[string]uint16)
	z.varSizes = make(map[string]int)
	z.varNext = z.cfg.VarBase
	z.tags = make(map[string]bool)
	z.pendingAT = -1
	z.pendingSize = 0
	z.currentBank = 0
	z.bankLines = make(map[int][]string)
	z.atVars = make(map[string]bool)

	z.scanProg(prog)

	z.emitHeader()
	z.emitStart()
	z.emitProg(prog)
	z.currentBank = 0 // reset after emitProg so runtime/footer/vars stay in bank 0
	z.emitRuntime()
	z.emitFooter()
	z.emitVars()

	return strings.Join(z.lines, "\n")
}

// BankLines returns per-bank assembly texts (key = bank number, 1+).
// Each text is a complete 6502 assembly snippet for that bank.
func (z *Gen6502) BankLines() map[int]string {
	result := make(map[int]string)
	for n, lines := range z.bankLines {
		if n > 0 && len(lines) > 0 {
			result[n] = strings.Join(lines, "\n")
		}
	}
	return result
}

func (z *Gen6502) scanProg(prog *Program) {
	z.pendingPriority = -1
	z.taskNames = make(map[string]int)
	for _, instr := range prog.Instrs {
		consumed := false
		switch instr.Op {
		case AT:
			z.pendingAT = int(instr.Operand.Num)
			consumed = true
		case ALLOC:
			z.pendingSize = int(instr.Operand.Num)
			consumed = true
		case VAR:
			name := z.varName(instr.Operand.Name)
			if _, ok := z.varAddr[name]; !ok {
				size := 1
				if z.pendingSize > 0 {
					size = z.pendingSize
					z.pendingSize = 0
				}
				if z.pendingAT >= 0 {
					z.varAddr[name] = uint16(z.pendingAT)
					z.varSizes[name] = size
					z.atVars[name] = true
					consumed = true
				} else {
					z.varAddr[name] = z.varNext
					z.varSizes[name] = size
					z.varNext += uint16(size)
				}
			}
		case DATA_B, DATA_W, DATA_STR:
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
		case SHL_B, SHL_W, SHR_B, SHR_W:
			z.needShift = true
		case IS_B, IS_W:
			z.needCmp = true
		case PRIORITY:
			z.pendingPriority = int(instr.Operand.Num)
			consumed = true
		case JOB:
			z.needSched = true
			z.taskEntryLabels = append(z.taskEntryLabels, instr.Operand.Name)
			z.taskNames[instr.Operand.Name] = z.taskCount
			if z.pendingPriority >= 0 {
				z.taskPriorities = append(z.taskPriorities, z.pendingPriority)
				z.pendingPriority = -1
			} else {
				z.taskPriorities = append(z.taskPriorities, 0) // default
			}
			z.taskCount++
		case BYE, YIELD, SLEEP, STOP, START:
			z.needSched = true
		case SAVE:
			z.needSave = true
		case LOAD:
			z.needLoad = true
		case SEED:
			z.needSeed = true
			// Register _plz_seed as a variable if not already
			if _, ok := z.varAddr["_plz_seed"]; !ok {
				z.varAddr["_plz_seed"] = z.varNext
				z.varSizes["_plz_seed"] = 1
				z.varNext++
			}
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

// emitBank appends a line to the current bank's buffer instead of bank 0.
func (z *Gen6502) emitBank(line string) {
	if z.currentBank > 0 {
		z.bankLines[z.currentBank] = append(z.bankLines[z.currentBank], line)
	} else {
		z.lines = append(z.lines, line)
	}
}

func (z *Gen6502) emitBankf(format string, args ...interface{}) {
	z.emitBank(fmt.Sprintf(format, args...))
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

	if z.cfg.NES {
		// MMC5: set PRG mode to 16KB fixed at $C000, selectable at $8000
		z.emit("\tlda #$01")
		z.emit("\tsta $5100")
		// MMC5: enable SRAM writes by default
		z.emit("\tlda #$02")
		z.emit("\tsta $5104")
	}

	// Init data stack pointer
	z.emitf("\tlda #<%d", z.cfg.StackBase)
	z.emitf("\tsta $00")
	z.emitf("\tlda #>%d", z.cfg.StackBase)
	z.emitf("\tsta $01")

	// Init output buffer pointer at $0C-$0D
	z.emitf("\tlda #<%d", z.cfg.OutputBase)
	z.emitf("\tsta $0c")
	z.emitf("\tlda #>%d", z.cfg.OutputBase)
	z.emitf("\tsta $0d")

	if z.needSched {
		// JSR to task init (emitted after all code so JOB labels exist)
		z.emit("\tjsr _plz_init_tasks")
		// After init, RTS to highest priority task
	}

	z.emit("")
}

func (z *Gen6502) emitFooter() {
	z.emit("_6502_all_done:")
	z.emit("\tbrk")
	z.emit("_6502_halt:")
	z.emit("\tsei")
	z.emit("\tjmp _6502_halt")
	z.emit("")
}

// ── Runtime helpers ──

func (z *Gen6502) emitRuntime() {
	z.emit("; -------------------------------------------------------------------")
	z.emit("; Runtime helpers")
	z.emit("; -------------------------------------------------------------------")
	z.emit("")

	z.emitMulHelpers()
	z.emitDivModHelpers()
	z.emitScheduler()

	if z.needCmp {
		z.emit("; Word comparison helper")
		z.emit("; On entry: A=low(NEXT), X=high(NEXT), $02=low(TOS), $03=high(TOS)")
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

	if z.needSave || z.needLoad {
		z.emit("; Block copy: ($0a-$0b) → ($02-$03), length $08-$09")
		z.emit("_plz_memcpy:")
		mcLoop := z.nextLabel()
		mcS1 := z.nextLabel()
		mcS2 := z.nextLabel()
		mcDec := z.nextLabel()
		z.emitf("_mc_%d:", mcLoop)
		z.emit("\tldy #0")
		z.emit("\tlda ($0a),y")
		z.emit("\tsta ($02),y")
		z.emit("\tinc $0a")
		z.emitf("\tbne _mc_%d", mcS1)
		z.emit("\tinc $0b")
		z.emitf("_mc_%d:", mcS1)
		z.emit("\tinc $02")
		z.emitf("\tbne _mc_%d", mcS2)
		z.emit("\tinc $03")
		z.emitf("_mc_%d:", mcS2)
		z.emit("\tlda $08")
		z.emitf("\tbne _mc_%d", mcDec)
		z.emit("\tdec $09")
		z.emitf("_mc_%d:", mcDec)
		z.emit("\tdec $08")
		z.emit("\tlda $08")
		z.emit("\tora $09")
		z.emitf("\tbne _mc_%d", mcLoop)
		z.emit("\trts")
		z.emit("")
	}

}

func (z *Gen6502) emitMulHelpers() {
	if !z.needMul {
		return
	}
	z.emit("; 8-bit multiply: A = A * $02 (unsigned)")
	z.emit("_plz_mul8:")
	z.emit("\tsta $04          ; save left operand")
	z.emit("\tlda #0           ; accumulator")
	z.emit("\tldx #8")
	_ml8_loop := z.nextLabel()
	_ml8_skip := z.nextLabel()
	z.emitf("_ml8_%d:", _ml8_loop)
	z.emit("\tlsr $02          ; shift right, LSB -> carry")
	z.emitf("\tbcc _ml8_%d", _ml8_skip)
	z.emit("\tclc")
	z.emit("\tadc $04          ; add left to accumulator")
	z.emitf("_ml8_%d:", _ml8_skip)
	z.emit("\tasl $04          ; shift left")
	z.emit("\tdex")
	z.emitf("\tbne _ml8_%d", _ml8_loop)
	z.emit("\trts")
	z.emit("")

	z.emit("; 16-bit multiply: A,X = A,X * ($02),($03) (unsigned, low 16 bits)")
	z.emit("; Uses $04-$07 as scratch")
	z.emit("_plz_mul16:")
	z.emit("\tsta $04          ; save left low")
	z.emit("\tstx $05          ; save left high")
	z.emit("\tlda #0")
	z.emit("\tsta $06          ; result low")
	z.emit("\tsta $07          ; result high")
	z.emit("\tldx #16")
	_ml16_loop := z.nextLabel()
	_ml16_skip := z.nextLabel()
	z.emitf("_ml16_%d:", _ml16_loop)
	z.emit("\tlsr $03          ; shift right high")
	z.emit("\tror $02          ; rotate right low")
	z.emitf("\tbcc _ml16_%d", _ml16_skip)
	z.emit("\tclc")
	z.emit("\tlda $06")
	z.emit("\tadc $04")
	z.emit("\tsta $06")
	z.emit("\tlda $07")
	z.emit("\tadc $05")
	z.emit("\tsta $07")
	z.emitf("_ml16_%d:", _ml16_skip)
	z.emit("\tasl $04")
	z.emit("\trol $05")
	z.emit("\tdex")
	z.emitf("\tbne _ml16_%d", _ml16_loop)
	z.emit("\tlda $06")
	z.emit("\tldx $07")
	z.emit("\trts")
	z.emit("")
}

func (z *Gen6502) emitDivModHelpers() {
	if !z.needDiv && !z.needMod {
		return
	}
	// Combined divmod: dividend in A, divisor in $02
	// Returns: A = quotient, X = remainder
	z.emit("; 8-bit divmod: A = A / $02 (quotient), X = remainder")
	z.emit("_plz_div8:")
	z.emit("_plz_mod8:")
	z.emit("\tsta $04          ; save dividend")
	z.emit("\tlda #0")
	z.emit("\tsta $05          ; quotient = 0")
	z.emit("\tldx #8")
	_dm8_loop := z.nextLabel()
	_dm8_skip := z.nextLabel()
	z.emitf("_dm8_%d:", _dm8_loop)
	z.emit("\tasl $04          ; shift dividend left, MSB -> carry")
	z.emit("\trol              ; rotate carry into remainder (A)")
	z.emit("\tcmp $02          ; compare remainder with divisor")
	z.emitf("\tbcc _dm8_%d", _dm8_skip)
	z.emit("\tsbc $02          ; subtract divisor from remainder")
	z.emit("\tsec              ; set quotient bit")
	z.emitf("_dm8_%d:", _dm8_skip)
	z.emit("\trol $05          ; shift quotient bit into $05")
	z.emit("\tdex")
	z.emitf("\tbne _dm8_%d", _dm8_loop)
	z.emit("\tldx $05          ; X = quotient")
	z.emit("\trts              ; A = remainder, X = quotient")
	z.emit("")

	// 16-bit divmod: dividend in A,X, divisor in $02,$03
	// Uses $04-$08 as scratch
	z.emit("; 16-bit divmod: A,X = (A,X) / ($02,$03)")
	z.emit("; Uses $04-$08 as scratch")
	z.emit("_plz_mod16:")
	z.emit("\tjsr _plz_divmod16")
	z.emit("\tlda $06          ; remainder low")
	z.emit("\tldx $07          ; remainder high")
	z.emit("\trts")
	z.emit("_plz_div16:")
	z.emit("\tjsr _plz_divmod16")
	z.emit("\tlda $04          ; quotient low")
	z.emit("\tldx $05          ; quotient high")
	z.emit("\trts")
	_dm16_do := z.nextLabel()
	_dm16_loop := z.nextLabel()
	_dm16_skip := z.nextLabel()
	z.emit("_plz_divmod16:")
	z.emit("\tsta $04          ; dividend low")
	z.emit("\tstx $05          ; dividend high")
	z.emit("\tlda #0")
	z.emit("\tsta $06          ; remainder low")
	z.emit("\tsta $07          ; remainder high")
	z.emit("\tlda $02")
	z.emit("\tora $03")
	z.emitf("\tbne _dm16_%d", _dm16_do)
	z.emit("\tlda #0           ; div by zero -> 0")
	z.emit("\ttax")
	z.emit("\trts")
	z.emitf("_dm16_%d:", _dm16_do)
	z.emit("\tldx #16")
	z.emitf("_dm16_%d:", _dm16_loop)
	z.emit("\tasl $04")
	z.emit("\trol $05")
	z.emit("\trol $06")
	z.emit("\trol $07")
	z.emit("\tsec")
	z.emit("\tlda $06")
	z.emit("\tsbc $02")
	z.emit("\tpha")
	z.emit("\tlda $07")
	z.emit("\tsbc $03")
	z.emitf("\tbcc _dm16_%d", _dm16_skip)
	z.emit("\tsta $07")
	z.emit("\tpla")
	z.emit("\tsta $06")
	z.emit("\tinc $04          ; set quotient bit")
	z.emitf("\tjmp _dm16_next_%d", _dm16_skip)
	z.emitf("_dm16_%d:", _dm16_skip)
	z.emit("\tpla")
	z.emitf("_dm16_next_%d:", _dm16_skip)
	z.emit("\tdex")
	z.emitf("\tbne _dm16_%d", _dm16_loop)
	z.emit("\trts")
	z.emit("")
}

func (z *Gen6502) emitScheduler() {
	if !z.needSched {
		return
	}
	z.emit("; -------------------------------------------------------------------")
	z.emit("; Task scheduler")
	z.emit("; -------------------------------------------------------------------")
	z.emit("")
	z.emit("; TCB layout (8 bytes per task, zero-page at $80):")
	z.emit("; +0: SP_low (1 byte)")
	z.emit("; +1: SP_high (1 byte)")
	z.emit("; +2: state (1 byte: 0=READY, 1=SUSPENDED, 2=SLEEPING, 3=DEAD)")
	z.emit("; +3: sleep counter (1 byte)")
	z.emit("; +4: priority (1 byte)")
	z.emit("; +5: reserved")
	z.emit("; +6: reserved")
	z.emit("; +7: reserved")
	z.emit(";")
	z.emit("; Current task index in $06")
	z.emit("")

	z.emit("_plz_scheduler:")
	// Save current SP into current task's TCB
	z.emit("\tsei")
	z.emit("\ttsx")
	z.emit("\tlda $06           ; current task index")
	z.emit("\tasl")
	z.emit("\tasl")
	z.emit("\tasl")
	z.emit("\ttay               ; Y = task_index * 8")
	z.emit("\ttxa               ; A = SP")
	z.emit("\tsta $80,y         ; save SP low")
	z.emit("\tlda #$01")
	z.emit("\tsta $81,y         ; save SP high (always page $01)")
	z.emit("")

	// Decrement all sleep counters
	z.emit("\t; Decrement sleep counters")
	z.emit("\tldx #0")
	sch_slp := z.nextLabel()
	sch_skip := z.nextLabel()
	z.emitf("_sch_slp_%d:", sch_slp)
	z.emit("\tlda $83,x          ; sleep counter")
	z.emitf("\tbeq _sch_sk_%d", sch_skip)
	z.emit("\tdec $83,x")
	z.emitf("\tbne _sch_sk_%d", sch_skip)
	z.emit("\t; Reached 0 — wake up if SLEEPING")
	z.emit("\tlda $82,x")
	z.emit("\tcmp #2")
	z.emitf("\tbne _sch_sk_%d", sch_skip)
	z.emit("\tlda #0")
	z.emit("\tsta $82,x          ; state = READY")
	z.emitf("_sch_sk_%d:", sch_skip)
	z.emit("\ttxa")
	z.emit("\tclc")
	z.emit("\tadc #8")
	z.emit("\ttax")
	z.emitf("\tcpx #%d", z.taskCount*8)
	z.emitf("\tbne _sch_slp_%d", sch_slp)
	z.emit("")

	// Scan for best READY task (lowest priority value, round-robin)
	// $07 = candidate byte offset (task_idx * 8), $08 = best priority found
	z.emit("\t; Scan for best READY task")
	sch_start := z.nextLabel()
	sch_loop := z.nextLabel()
	sch_next := z.nextLabel()
	sch_wrap := z.nextLabel()
	sch_done := z.nextLabel()
	// candidate starts at current+1 (with wrap), stored as byte offset
	z.emit("\tlda $06")
	z.emit("\tclc")
	z.emit("\tadc #1")
	z.emitf("\tcmp #%d", z.taskCount)
	z.emitf("\tbcc _sch_st_%d", sch_start)
	z.emit("\tlda #0")
	z.emitf("_sch_st_%d:", sch_start)
	z.emit("\tasl")              // *2
	z.emit("\tasl")              // *4
	z.emit("\tasl")              // *8
	z.emit("\tsta $07           ; candidate byte offset")
	z.emit("\tlda #15           ; best priority = worst")
	z.emit("\tsta $08")
	z.emit("\tldx #0            ; iteration count")
	z.emitf("_sch_lp_%d:", sch_loop)
	// Check candidate at byte offset $07
	z.emit("\tldy $07")
	z.emit("\tlda $82,y          ; state (at TCB_base + 2)")
	z.emitf("\tbne _sch_nx_%d", sch_next)
	z.emit("\t; READY — check priority")
	z.emit("\tlda $84,y          ; priority")
	z.emit("\tcmp $08")
	z.emitf("\tbcs _sch_nx_%d", sch_next)
	z.emit("\tsta $08           ; new best priority")
	// Convert byte offset back to task index for $06
	z.emit("\tlda $07")
	z.emit("\tlsr")              // /2
	z.emit("\tlsr")              // /4
	z.emit("\tlsr")              // /8
	z.emit("\tsta $06           ; current_task = candidate")
	z.emitf("_sch_nx_%d:", sch_next)
	// Advance to next candidate
	z.emit("\tinx")
	z.emitf("\tcpx #%d", z.taskCount)
	z.emitf("\tbeq _sch_dn_%d", sch_done)
	z.emit("\tlda $07")
	z.emit("\tclc")
	z.emit("\tadc #8")           // advance by 8 bytes
	z.emitf("\tcmp #%d", z.taskCount*8)
	z.emitf("\tbcc _sch_wr_%d", sch_wrap)
	z.emit("\tlda #0")
	z.emitf("_sch_wr_%d:", sch_wrap)
	z.emit("\tsta $07")
	z.emitf("\tjmp _sch_lp_%d", sch_loop)
	z.emitf("_sch_dn_%d:", sch_done)
	z.emit("")

	// Check if chosen task is DEAD — if so, halt
	z.emit("\t; Check if chosen task is DEAD")
	z.emit("\tlda $06")
	z.emit("\tasl")
	z.emit("\tasl")
	z.emit("\tasl")
	z.emit("\ttay")
	z.emit("\tlda $82,y          ; state")
	z.emit("\tcmp #3")
	z.emit("\tbeq _6502_sch_halt")
	z.emit("")
	// Restore chosen task's SP
	z.emit("\t; Restore chosen task")
	z.emit("\tlda $80,y          ; SP low")
	z.emit("\ttax")
	z.emit("\ttxs")
	z.emit("\tcli")
	z.emit("\trts                ; jump to chosen task")
	z.emit("")
	z.emit("_6502_sch_halt:")
	z.emit("\t; No runnable task — halt")
	z.emit("\tsei")
	z.emit("\tbrk")
	z.emit("")

	// Task init routine (called from _6502_main)
	z.emit("_plz_init_tasks:")
	taskStackSize := 256 / z.cfg.TaskLimit
	z.emitf("\t; Init %d tasks, %d bytes stack each", z.taskCount, taskStackSize)
	// Zero TCBs
	z.emit("\tldx #0")
	z.emit("\tlda #0")
	z_clr := z.nextLabel()
	z.emitf("_init_clr_%d:", z_clr)
	z.emit("\tsta $80,x")
	z.emit("\tinx")
	z.emitf("\tcpx #%d", z.taskCount*8)
	z.emitf("\tbne _init_clr_%d", z_clr)
	z.emit("")
	// Set priorities from stored values (TCBs zeroed above, so all priority=0 by default)
	for i, p := range z.taskPriorities {
		if p != 0 {
			z.emitf("\tlda #%d", p)
			z.emitf("\tsta $%02x", 0x80+i*8+4)
		}
	}
	z.emit("")
	// For each task, set SP to top of its stack area, push entry address,
	// and save SP in TCB
	for i, name := range z.taskEntryLabels {
		spOffset := byte((i+1)*taskStackSize - 1)
		z.emitf("\t; Task %d: %s", i, name)
		z.emitf("\tldx #%d", spOffset)
		z.emit("\ttxs")
		z.emitf("\tlda #>(_plz_task_entry_%s - 1)", name)
		z.emit("\tpha")
		z.emitf("\tlda #<(_plz_task_entry_%s - 1)", name)
		z.emit("\tpha")
		z.emit("\ttsx")
		z.emitf("\tstx $%02x", 0x80+i*8)
		z.emit("\tlda #$01")
		z.emitf("\tsta $%02x", 0x80+i*8+1)
		z.emit("")
	}
	// Pick the task with the lowest priority number (highest priority)
	bestIdx := 0
	bestPri := 15
	for i, p := range z.taskPriorities {
		if p < bestPri {
			bestPri = p
			bestIdx = i
		}
	}
	z.emitf("\t; Start task %d (highest priority)", bestIdx)
	z.emitf("\tldx $%02x", 0x80+bestIdx*8)
	z.emit("\ttxs")
	z.emitf("\tlda #%d", bestIdx)
	z.emit("\tsta $06           ; current task = best priority")
	z.emit("\trts               ; RTS into best task")
	z.emit("")

}

// ── Variables ──

func (z *Gen6502) emitVars() {
	type kv struct {
		name string
		addr uint16
	}
	var sorted []kv
	for name, addr := range z.varAddr {
		sorted = append(sorted, kv{name, addr})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].addr < sorted[j].addr })
	if len(sorted) == 0 {
		return
	}
	z.emit("; -------------------------------------------------------------------")
	z.emit("; Variable storage")
	z.emit("; -------------------------------------------------------------------")
	for _, kv := range sorted {
		if z.atVars[kv.name] {
			z.emitf("\t.org $%04x", kv.addr)
		}
		z.emitf("%s:", kv.name)
		z.emitf("\t.pad 0, %d", z.varSize(kv.name))
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
		if instr.Op == ALLOC {
			z.pendingSize = int(instr.Operand.Num)
			continue
		}
		if z.pendingAT >= 0 {
			switch instr.Op {
			case VAR:
				// handled by emitVars
			case DATA_B, DATA_W, DATA_STR, ROUTE, JOB:
				z.emitf("\t.org $%04x", z.pendingAT)
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

	// ── Stack Frame (Reentrant Procedures) ──
	case FRAME:
		z.inFrame = true
		z.frameSz = int(o.Num)
		// Save old frame pointer to hardware stack
		z.emit("\tlda $0e")
		z.emit("\tpha")
		z.emit("\tlda $0f")
		z.emit("\tpha")
		// Allocate N bytes (SP -= N)
		if o.Num > 0 {
			z.emit("\ttsx")
			z.emit("\ttxa")
			z.emitf("\tsec\n\tsbc #%d", o.Num)
			z.emit("\ttax")
			z.emit("\ttxs")
		}
		// Set frame pointer to current SP (first local byte)
		z.emit("\ttsx")
		z.emit("\tstx $0f")
		z.emit("\tlda #$01")
		z.emit("\tsta $0e")

	case LOCAL_B:
		z.localOff[o.Name] = z.localNxt
		z.localNxt += 1

	case LOCAL_W:
		z.localOff[o.Name] = z.localNxt
		z.localNxt += 2

	case PUSH_B:
		z.emitf("\tlda #%d", o.Num)
		z.emit("\tldx #0")
		z.emitSpillW()

	case PUSH_W:
		z.emitf("\tlda #%d", o.Num&0xFF)
		z.emitf("\tldx #%d", (o.Num>>8)&0xFF)
		z.emitSpillW()

	case VAR:
	// Handled in scan phase

	case GET_B:
		if off := z.localAddr(o.Name); off >= 0 {
			z.emitf("\tldy #%d", off)
			z.emit("\tlda ($0e),y")
		} else {
			z.emitf("\tlda %s", z.varName(o.Name))
		}
		z.emit("\tldx #0")
		z.emitSpillW()

	case GET_W:
		if off := z.localAddr(o.Name); off >= 0 {
			z.emitf("\tldy #%d", off)
			z.emit("\tlda ($0e),y")
			z.emit("\tsta $02        ; save low byte")
			z.emitf("\tldy #%d", off+1)
			z.emit("\tlda ($0e),y")
			z.emit("\ttax")
			z.emit("\tlda $02")
		} else {
			z.emitf("\tlda %s", z.varName(o.Name))
			z.emitf("\tldx %s+1", z.varName(o.Name))
		}
		z.emitSpillW()

	case PUT_B:
		z.emitFillW() // A=low, X=high (only low byte stored)
		if off := z.localAddr(o.Name); off >= 0 {
			z.emitf("\tldy #%d", off)
			z.emit("\tsta ($0e),y")
		} else {
			z.emitf("\tsta %s", z.varName(o.Name))
		}

	case PUT_W:
		z.emitFillW() // A=low, X=high
		if off := z.localAddr(o.Name); off >= 0 {
			z.emitf("\tldy #%d", off)
			z.emit("\tsta ($0e),y")
			z.emit("\ttxa")
			z.emitf("\tldy #%d", off+1)
			z.emit("\tsta ($0e),y")
		} else {
			z.emitf("\tsta %s", z.varName(o.Name))
			z.emitf("\tstx %s+1", z.varName(o.Name))
		}

	case PUSH_A:
		z.emitf("\tlda #<%s", z.varName(o.Name))
		z.emitf("\tldx #>%s", z.varName(o.Name))
		z.emitSpillW()

	case PUSH_D:
		z.emitf("\tlda #<%s", o.Name)
		z.emitf("\tldx #>%s", o.Name)
		z.emitSpillW()

	case READ_B:
		z.emitFillW() // A=low addr, X=high addr
		z.emit("\tstx $05  ; high byte of address")
		z.emit("\tsta $04  ; low byte of address")
		z.emit("\tldy #0")
		z.emit("\tlda ($04),y")
		z.emit("\tldx #0")
		z.emitSpillW()

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
		z.emitFillW() // A=low(value), X=high(value)
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
		z.emitFillW() // A=low(TOS), X=high(TOS)
		z.emit("\tsta $02")
		z.emitFillW() // A=low(NEXT), X=high(NEXT)
		z.emit("\tclc")
		z.emit("\tadc $02")
		z.emit("\tldx #0")
		z.emitSpillW()

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
		z.emitFillW() // A=low(TOS), X=high(TOS)
		z.emit("\tsta $02")
		z.emitFillW() // A=low(NEXT), X=high(NEXT)
		z.emit("\tsec")
		z.emit("\tsbc $02")
		z.emit("\tldx #0")
		z.emitSpillW()

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

	// ── Multiply / Divide / Modulo ──
	case MUL_B:
		z.emitFillW()             // A=right low, X=right high
		z.emit("\tsta $02")       // save right (multiplier)
		z.emitFillW()             // A=left low, X=left high
		z.emit("\tjsr _plz_mul8") // A = result
		z.emit("\tldx #0")
		z.emitSpillW()

	case MUL_W:
		z.emitFillW()             // A=right low, X=right high
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()             // A=left low, X=left high
		z.emit("\tjsr _plz_mul16")
		z.emitSpillW()            // A=result low, X=result high

	case DIV_B:
		z.emitFillW()             // A=right low, X=right high
		z.emit("\tsta $02")       // save divisor
		z.emitFillW()             // A=left low, X=left high
		z.emit("\tjsr _plz_div8") // A = remainder, X = quotient
		z.emit("\ttxa")           // A = quotient
		z.emit("\tldx #0")
		z.emitSpillW()

	case DIV_W:
		z.emitFillW()             // A=right low, X=right high
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()             // A=left low, X=left high
		z.emit("\tjsr _plz_div16")
		z.emitSpillW()

	case MOD_B:
		z.emitFillW()             // A=right low, X=right high
		z.emit("\tsta $02")       // save divisor
		z.emitFillW()             // A=left low, X=left high
		z.emit("\tjsr _plz_mod8") // A = remainder
		z.emit("\tldx #0")
		z.emitSpillW()

	case MOD_W:
		z.emitFillW()             // A=right low, X=right high
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()             // A=left low, X=left high
		z.emit("\tjsr _plz_mod16")
		z.emitSpillW()

	case AND_B:
		z.emitFillW() // A=low(TOS), X=high(TOS)
		z.emit("\tsta $02")
		z.emitFillW() // A=low(NEXT), X=high(NEXT)
		z.emit("\tand $02")
		z.emit("\tldx #0")
		z.emitSpillW()

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
		z.emitFillW() // A=low(TOS), X=high(TOS)
		z.emit("\tsta $02")
		z.emitFillW() // A=low(NEXT), X=high(NEXT)
		z.emit("\tora $02")
		z.emit("\tldx #0")
		z.emitSpillW()

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
		z.emitFillW() // A=low(TOS), X=high(TOS)
		z.emit("\tsta $02")
		z.emitFillW() // A=low(NEXT), X=high(NEXT)
		z.emit("\teor $02")
		z.emit("\tldx #0")
		z.emitSpillW()

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
		z.emitFillW() // A=low, X=high
		z.emit("\teor #$ff")
		z.emit("\tand #1")
		z.emit("\tldx #0")
		z.emitSpillW()

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
		z.emitFillW() // A=low, X=high
		z.emit("\teor #$ff")
		z.emit("\tclc")
		z.emit("\tadc #1")
		z.emit("\tldx #0")
		z.emitSpillW()

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

	// ── Shifts ──
	case SHL_B:
		_shl_b_loop := z.nextLabel()
		_shl_b_done := z.nextLabel()
		z.emitFillW()             // A=shift count low, X=shift count high
		z.emit("\tsta $02")       // save shift count (low byte)
		z.emitFillW()             // A=value low, X=value high
		z.emit("\tsta $04")       // save value in temp
		z.emit("\tlda $02")       // A = shift count
		z.emitf("\tbeq _shl_b_%d", _shl_b_done)
		z.emit("\tldx $02")       // X = shift count
		z.emit("\tlda $04")       // A = value
		z.emitf("_shl_b_%d:", _shl_b_loop)
		z.emit("\tasl")
		z.emit("\tdex")
		z.emitf("\tbne _shl_b_%d", _shl_b_loop)
		z.emitf("_shl_b_%d:", _shl_b_done)
		z.emit("\tldx #0")
		z.emitSpillW()

	case SHL_W:
		_shl_w_loop := z.nextLabel()
		_shl_w_done := z.nextLabel()
		z.emitFillW()             // A=shift count low, X=shift count high
		z.emit("\tsta $02")       // save shift count (low byte)
		z.emitFillW()             // A=value low, X=value high
		z.emit("\tsta $04")       // value low in $04
		z.emit("\tstx $05")       // value high in $05
		z.emit("\tlda $02")       // A = shift count
		z.emitf("\tbeq _shl_w_%d", _shl_w_done)
		z.emit("\tldx $02")       // X = shift count
		z.emitf("_shl_w_%d:", _shl_w_loop)
		z.emit("\tasl $04")
		z.emit("\trol $05")
		z.emit("\tdex")
		z.emitf("\tbne _shl_w_%d", _shl_w_loop)
		z.emitf("_shl_w_%d:", _shl_w_done)
		z.emit("\tlda $04")
		z.emit("\tldx $05")
		z.emitSpillW()

	case SHR_B:
		_shr_b_loop := z.nextLabel()
		_shr_b_done := z.nextLabel()
		z.emitFillW()             // A=shift count low, X=shift count high
		z.emit("\tsta $02")       // save shift count (low byte)
		z.emitFillW()             // A=value low, X=value high
		z.emit("\tsta $04")       // save value in temp
		z.emit("\tlda $02")       // A = shift count
		z.emitf("\tbeq _shr_b_%d", _shr_b_done)
		z.emit("\tldx $02")       // X = shift count
		z.emit("\tlda $04")       // A = value
		z.emitf("_shr_b_%d:", _shr_b_loop)
		z.emit("\tlsr")
		z.emit("\tdex")
		z.emitf("\tbne _shr_b_%d", _shr_b_loop)
		z.emitf("_shr_b_%d:", _shr_b_done)
		z.emit("\tldx #0")
		z.emitSpillW()

	case SHR_W:
		_shr_w_loop := z.nextLabel()
		_shr_w_done := z.nextLabel()
		z.emitFillW()             // A=shift count low, X=shift count high
		z.emit("\tsta $02")       // save shift count (low byte)
		z.emitFillW()             // A=value low, X=value high
		z.emit("\tsta $04")       // value low
		z.emit("\tstx $05")       // value high
		z.emit("\tlda $02")       // A = shift count
		z.emitf("\tbeq _shr_w_%d", _shr_w_done)
		z.emit("\tldx $02")       // X = shift count
		z.emitf("_shr_w_%d:", _shr_w_loop)
		z.emit("\tlsr $05")
		z.emit("\tror $04")
		z.emit("\tdex")
		z.emitf("\tbne _shr_w_%d", _shr_w_loop)
		z.emitf("_shr_w_%d:", _shr_w_done)
		z.emit("\tlda $04")
		z.emit("\tldx $05")
		z.emitSpillW()

	// ── Cast ──
	case CAST_W:
		z.emitFillW()
		z.emit("\tldx #0")
		z.emitSpillW()

	case CAST_B:
		z.emitFillW()
		z.emit("\tldx #0")
		z.emitSpillW()

	// ── Stack ──
	case DUP:
		z.emitFillW()     // A=low, X=high
		z.emit("\tpha")   // save low byte
		z.emitSpillW()    // push copy 1 (A clobbered to high byte)
		z.emit("\tpla")   // restore low byte
		z.emitSpillW()    // push copy 2 (A=low, correct)

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
		z.emitFillW()           // A=low(TOS), X=high(TOS)
		z.emit("\tsta $02")     // save TOS low byte
		z.emitFillW()           // A=low(NEXT), X=high(NEXT)
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
			z.emit("\tlda #1")
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
		z.emit("\tldx #0")
		z.emitSpillW()

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
		// High bytes equal, low bytes differ. Since we only reach this
		// code when there was no borrow (C=1), low(NEXT) > low(TOS).
		// Evaluate condition inline to avoid extra PLA in _is_w_gt.
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
		if z.currentBank > 0 {
			z.emitBankf("\t.export %s", o.Name)
			z.emitBankf("%s:", o.Name)
		} else {
			z.emitf("%s:", o.Name)
		}

	case GO:
		z.emitf("\tjmp %s", o.Name)

	case GO_IF:
		z.emitFillW()            // A=low, X=high
		z.emit("\tstx $02")      // $02 = high byte
		z.emit("\tora $02")      // A |= high byte
		_goif_skip := z.nextLabel()
		z.emitf("\tbeq _goif_skip_%d", _goif_skip)
		z.emitf("\tjmp %s", o.Name)
		z.emitf("_goif_skip_%d:", _goif_skip)

	// ── Procedures ──
	case ROUTE:
		z.inFrame = false
		z.frameSz = 0
		z.localOff = make(map[string]int)
		z.localNxt = 0
		z.emitf("%s:", o.Name)

	case RUN:
		z.emitf("\tjsr %s", o.Name)

	case DONE:
		if z.inFrame {
			z.emitFrameRestore()
		}
		z.emit("\trts")

	case DONE_INTERRUPT:
		if z.inFrame {
			z.emitFrameRestore()
		}
		z.emit("\trti")

	case DONE_NMI:
		if z.inFrame {
			z.emitFrameRestore()
		}
		z.emit("\trti")

	// ── Tasks ──
	case JOB:
		// JOB declares a task entry point; at boot the scheduler pushes
		// the address onto the task's stack so it can be RETed into.
		// In the 6502 backend, task initialisation is handled at startup:
		// we emit TAG + a stub that the startup code references.
		z.emitf("\t; JOB %s", o.Name)
		z.emitf("_plz_task_entry_%s:", o.Name)

	case PRIORITY:
		// Handled in scanProg; no code emission needed.

	case BYE:
		// Mark current task DEAD (3), then call scheduler
		z.emit("\tsei")
		// current_task_idx * 8 + 2 = TCB + state offset
		z.emit("\tlda $06")          // current task index
		z.emit("\tasl")              // *2
		z.emit("\tasl")              // *4
		z.emit("\tasl")              // *8
		z.emit("\tadc #2")           // +2 (state offset)
		z.emit("\ttax")
		z.emit("\tlda #3")           // DEAD
		z.emit("\tsta $80,x")
		z.emit("\tjmp _plz_scheduler")

	case YIELD:
		// Yield: call scheduler (task stays READY)
		z.emit("\tjsr _plz_scheduler")

	case SLEEP:
		// Pop sleep duration from data stack, store in current task's
		// TCB sleep field (offset +3), then call scheduler
		z.emitFill()                  // A = sleep duration
		z.emit("\tpha")
		z.emit("\tlda $06")           // current task index
		z.emit("\tasl")
		z.emit("\tasl")
		z.emit("\tasl")
		z.emit("\tadc #3")            // +3 (sleep offset)
		z.emit("\ttax")
		z.emit("\tpla")
		z.emit("\tsta $80,x")
		z.emit("\tjmp _plz_scheduler")

	case STOP:
		// Suspend named task: set state to 1 (SUSPENDED)
		z.emitf("\t; STOP %s: set state to SUSPENDED", o.Name)
		if idx, ok := z.taskNames[o.Name]; ok {
			z.emitf("\tlda #1")
		z.emitf("\tsta $%02x", 0x80+idx*8+2)
	}

	case START:
		// Resume named task: set state to 0 (READY)
		z.emitf("\t; START %s: set state to READY", o.Name)
		if idx, ok := z.taskNames[o.Name]; ok {
			z.emitf("\tlda #0")
			z.emitf("\tsta $%02x", 0x80+idx*8+2)
		}

	// ── Port I/O ──
	case IN_B:
		z.emitf("\tlda $%04x", z.cfg.OutputBase+uint16(o.Num))
		z.emit("\tldx #0")
		z.emitSpillW()

	case IN_W:
		z.emitf("\tlda $%04x", z.cfg.OutputBase+uint16(o.Num))
		z.emit("\tpha")
		z.emitf("\tlda $%04x", z.cfg.OutputBase+uint16(o.Num+1))
		z.emit("\ttax")
		z.emit("\tpla")
		z.emitSpillW()

	case OUT_B:
		_out_b_skip := z.nextLabel()
		z.emitFillW()            // A=low, X=high
		z.emit("\tldy #0")
		z.emit("\tsta ($0c),y")  // write to *out_ptr
		z.emit("\tinc $0c")       // out_ptr++
		z.emitf("\tbne _out_b_%d", _out_b_skip)
		z.emit("\tinc $0d")
		z.emitf("_out_b_%d:", _out_b_skip)

	case OUT_W:
		_out_w_skip1 := z.nextLabel()
		_out_w_skip2 := z.nextLabel()
		z.emitFillW()            // A=low, X=high
		z.emit("\tldy #0")
		z.emit("\tsta ($0c),y")  // write low byte
		z.emit("\tinc $0c")       // out_ptr++
		z.emitf("\tbne _out_w_%d", _out_w_skip1)
		z.emit("\tinc $0d")
		z.emitf("_out_w_%d:", _out_w_skip1)
		z.emit("\ttxa")
		z.emit("\tldy #0")
		z.emit("\tsta ($0c),y")  // write high byte
		z.emit("\tinc $0c")
		z.emitf("\tbne _out_w_%d", _out_w_skip2)
		z.emit("\tinc $0d")
		z.emitf("_out_w_%d:", _out_w_skip2)

	// ── Interrupts ──
	case INT:
		z.intHandler = o.Name
		z.emitf("\t.export %s", o.Name)
	case NMI:
		z.nmiHandler = o.Name
		z.emitf("\t.export %s", o.Name)
	case HLT:
		z.emit("\tjmp _6502_all_done")
	case DII:
		z.emit("\tsei")
	case ENI:
		z.emit("\tcli")

	// ── Random ──
	case SEED:
		// LCG: seed = seed * 5 + 1
		// Compute 4*seed in A, save to scratch, then add seed back
		z.emit("\tlda _plz_seed")
		z.emit("\tasl")           // *2
		z.emit("\tasl")           // *4
		z.emit("\tsta $04")       // save 4*seed
		z.emit("\tlda _plz_seed")
		z.emit("\tclc")
		z.emit("\tadc $04")       // + seed = *5
		z.emit("\tadc #1")        // +1
		z.emit("\tsta _plz_seed") // store new seed
		z.emit("\tldx #0")
		z.emitSpillW()

	// ── Bank Switching ──
	case BANK:
		z.currentBank = int(o.Num)
		if z.cfg.NES {
			if z.currentBank > 0 {
				z.emitf("\t.export _plz_bank_%d", z.currentBank)
				z.emitf("_plz_bank_%d:", z.currentBank)
			}
		}
	case SWITCH:
		if z.cfg.NES {
			// Pop bank number (word) from data stack, write low byte to MMC5 $5113 (16KB PRG bank select)
			z.emitFillW()
			z.emit("\tsta $5113       ; MMC5: select PRG bank at $8000")
		} else {
			z.emitFillW() // pop bank number (keep stack balanced)
			z.emit("\t; SWITCH (runtime bank switching) not supported on non-NES 6502")
			z.emit("\t.byte <_switch_not_supported_on_non_nes_6502")
		}

	// ── Data Emission ──
	case DATA_B:
		if z.currentBank > 0 {
			z.emitBankf("\t.byte %d", o.Num)
		} else {
			z.emitf("\t.byte %d", o.Num)
		}
	case DATA_W:
		if z.currentBank > 0 {
			z.emitBankf("\t.word %d", o.Num)
		} else {
			z.emitf("\t.word %d", o.Num)
		}
	case DATA_STR:
		{
			lines := make([]string, 0, len(o.Str)+1)
			lines = append(lines, fmt.Sprintf("\t.byte %d", len(o.Str)))
			for _, ch := range o.Str {
				lines = append(lines, fmt.Sprintf("\t.byte %d", byte(ch)))
			}
			if z.currentBank > 0 {
				z.bankLines[z.currentBank] = append(z.bankLines[z.currentBank], lines...)
			} else {
				z.lines = append(z.lines, lines...)
			}
		}

	// ── Pragma ──
	case PRAGMA:
		z.emitf("\t; PRAGMA %d", o.Num)

	// ── Inline Assembly ──
	case INLINE:
		z.emit("\t" + o.Str)

	// ── Battery RAM ──
	case SRAM_ON:
		if z.cfg.NES {
			// MMC5: 8KB SRAM at $6000-$7FFF, enable via $5104, push base address
			z.emit("\tlda #$02")
			z.emit("\tsta $5104")
			z.emit("\tlda #$00")
			z.emit("\tldx #$60")
			z.emitSpillW()
		} else {
			// Push VarBase as SRAM base for memory copy
			z.emitf("\tlda #<%d", z.cfg.VarBase)
			z.emitf("\tldx #>%d", z.cfg.VarBase)
			z.emitSpillW()
		}
	case SRAM_OFF:
		if z.cfg.NES {
			// MMC5: disable SRAM writes
			z.emit("\tlda #$00")
			z.emit("\tsta $5104")
		} else {
			z.emit("\t; SRAM_OFF: no-op on non-NES 6502")
		}
	case SAVE:
		// Stack: [dest, src, len] — pop len, src, dest; copy src→dest
		z.emitFillW()
		z.emit("\tsta $08")
		z.emit("\tstx $09")
		z.emitFillW()
		z.emit("\tsta $0a")
		z.emit("\tstx $0b")
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emit("\tjsr _plz_memcpy")
	case LOAD:
		// Stack: [src, dest, len] — pop len, dest, src; copy src→dest
		z.emitFillW()
		z.emit("\tsta $08")
		z.emit("\tstx $09")
		z.emitFillW()
		z.emit("\tsta $02")
		z.emit("\tstx $03")
		z.emitFillW()
		z.emit("\tsta $0a")
		z.emit("\tstx $0b")
		z.emit("\tjsr _plz_memcpy")

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

// emitFrameRestore restores the frame pointer and stack for a reentrant procedure.
func (z *Gen6502) emitFrameRestore() {
	// SP = FP + frameSz (skip past locals to saved FP)
	z.emit("\tlda $0f")
	z.emitf("\tclc\n\tadc #%d", z.frameSz)
	z.emit("\ttax")
	z.emit("\ttxs")
	// Pop old frame pointer
	z.emit("\tpla")
	z.emit("\tsta $0f")
	z.emit("\tpla")
	z.emit("\tsta $0e")
	z.inFrame = false
	z.frameSz = 0
	z.localOff = make(map[string]int)
	z.localNxt = 0
}

// Assemble6502 assembles 6502 assembly text into binary code.
// For NES configs, the iNES header and vector table are prepended/appended.
// bankLines is per-bank assembly texts (key = bank number, 1+ for data banks).
func Assemble6502(cfg Gen6502Config, code string, bankLines map[int]string) ([]byte, error) {
	// Assemble code bank at cfg.Origin (typically $C000 for NES).
	r := bytes.NewReader([]byte(code))
	assembly, sourceMap, err := asm.Assemble(r, "plz", cfg.Origin, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("assembly failed: %w", err)
	}
	if len(assembly.Errors) > 0 {
		return nil, fmt.Errorf("assembly errors: %s", strings.Join(assembly.Errors, "; "))
	}
	bin := assembly.Code
	if cfg.NES {
		const bankSize = 0x4000 // 16KB per PRG bank
		const prgStart = 0x8000
		const dataBankOrigin = 0x8000

		// Assemble data banks and store their binaries.
		type dataBankBin struct {
			bankNum int
			code    []byte
		}
		var dataBanks []dataBankBin
		if len(bankLines) > 0 {
			for bankNum := 1; bankNum <= highestBank(bankLines); bankNum++ {
				bk, ok := bankLines[bankNum]
				if !ok || bk == "" {
					continue
				}
				bankAsm := "\t.org $8000\n" + bk
				bankR := bytes.NewReader([]byte(bankAsm))
				bankAssembly, bankSM, err := asm.Assemble(bankR, "plz", dataBankOrigin, nil, 0)
				if err != nil {
					return nil, fmt.Errorf("data bank %d assembly failed: %w", bankNum, err)
				}
				if len(bankAssembly.Errors) > 0 {
					return nil, fmt.Errorf("data bank %d assembly errors: %s", bankNum, strings.Join(bankAssembly.Errors, "; "))
				}
				dataBanks = append(dataBanks, dataBankBin{bankNum: bankNum, code: bankAssembly.Code})
				// Merge source map from this bank.
				sourceMap.Merge(bankSM)
			}
		}

		// Determine total PRG size in banks.
		// The last bank is always the fixed bank at $C000 (contains code and vectors).
		// Data banks (if any) go into switchable slots before the fixed bank.
		// Minimum is 2 banks (one switchable bank 0, one fixed bank 1) even without data.
		maxDataBank := 0
		for _, db := range dataBanks {
			if db.bankNum > maxDataBank {
				maxDataBank = db.bankNum
			}
		}
		prgBanks := maxDataBank + 2 // data banks + fixed bank, minimum 2
		prgSize := prgBanks * bankSize

		// The code bank is the fixed bank at $C000-$FFFF.
		// Its offset within the PRG data is the start of the last bank.
		fixedBankStart := (prgBanks - 1) * bankSize
		codeInBankOffset := int(cfg.Origin - 0xC000)
		codePRGOffset := fixedBankStart + codeInBankOffset
		if codeInBankOffset < 0 || codePRGOffset+len(bin) > prgSize-6 {
			return nil, fmt.Errorf("code bank at $%04X too large: %d bytes, PRG size %d", cfg.Origin, len(bin), prgSize)
		}

		// Allocate PRG data. Each bank is bankSize bytes.
		prgData := make([]byte, prgSize)

		// Copy code bank into its PRG slot (fixed bank at $C000).
		copy(prgData[codePRGOffset:], bin)

		// Copy data banks into their PRG slots.
		for _, db := range dataBanks {
			bankSlot := db.bankNum * bankSize
			if bankSlot+bankSize <= prgSize {
				copy(prgData[bankSlot:], db.code)
			}
		}

		// Look up handler addresses from source map exports.
		var nmiAddr, irqAddr uint16
		for _, ex := range sourceMap.Exports {
			if cfg.NmiHandlerName != "" && ex.Label == cfg.NmiHandlerName {
				nmiAddr = ex.Address
			}
			if cfg.IntHandlerName != "" && ex.Label == cfg.IntHandlerName {
				irqAddr = ex.Address
			}
		}
		// NMI vector at $FFFA (within the last PRG bank)
		vecOffset := fixedBankStart + bankSize - 6
		prgData[vecOffset] = byte(nmiAddr)
		prgData[vecOffset+1] = byte(nmiAddr >> 8)
		// Reset vector at $FFFC
		resetVec := cfg.Origin
		prgData[vecOffset+2] = byte(resetVec)
		prgData[vecOffset+3] = byte(resetVec >> 8)
		// IRQ vector at $FFFE
		prgData[vecOffset+4] = byte(irqAddr)
		prgData[vecOffset+5] = byte(irqAddr >> 8)

		// iNES header
		flags6 := byte(0x01 | ((5 & 0x0F) << 4)) // vertical mirroring | MMC5 low nibble
		flags7 := byte((5 >> 4) & 0x0F)           // MMC5 high nibble
		header := []byte{
			0x4e, 0x45, 0x53, 0x1a, // "NES" + MS-DOS EOF
			byte(prgBanks), // PRG banks
			0x00,           // CHR-RAM (0 banks)
			flags6,
			flags7,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
		}
		bin = append(header, prgData...)
	}
	return bin, nil
}

// highestBank returns the highest bank number in the bankLines map.
func highestBank(bankLines map[int]string) int {
	max := 0
	for n := range bankLines {
		if n > max {
			max = n
		}
	}
	return max
}
