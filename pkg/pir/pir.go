// Package pir defines the PLZ Intermediate Representation (PIR).
// PIR models an abstract stack machine (PAM) with a data stack, a return
// stack, named storage locations, and a one-shot AT directive.
// Each instruction has at most one operand.
package pir

import (
	"fmt"
	"strconv"
	"strings"
)

// Instruction identifies a PIR opcode.
type Instruction int

const (
	// ── Data Movement ──────────────────────────────────────────────

	NOP    Instruction = iota // no operation
	PUSH_B                    // [number] Push 8-bit literal onto the data stack.
	PUSH_W                    // [number] Push 16-bit literal onto the data stack.
	ALLOC                     // [number] One-shot: set allocation size (bytes) for next VAR.
	VAR                       // [name] Define a global variable (size set by preceding ALLOC, default 1).
	AT                        // [number] One-shot: assign next VAR/DATA/ROUTE/JOB to this hardware address.
	GET_B                     // [name] Fetch 8-bit variable value; push to data stack.
	GET_W                     // [name] Fetch 16-bit variable value; push to data stack.
	PUT_B                     // [name] Pop 8-bit value from stack; write to variable.
	PUT_W                     // [name] Pop 16-bit value from stack; write to variable.

	// ── Pointers & Memory ──────────────────────────────────────────

	PUSH_A  // [name] Push the 16-bit RAM address of a variable onto the data stack.
	PUSH_D  // [name] Push the 16-bit ROM address of a data label onto the data stack.
	READ_B  // Pop 16-bit address; read byte from RAM; push result.
	READ_W  // Pop 16-bit address; read word from RAM; push result.
	WRITE_B // Pop byte value, then pop 16-bit address; write value to RAM.
	WRITE_W // Pop word value, then pop 16-bit address; write value to RAM.

	// ── Math & Logic (typed) ───────────────────────────────────────

	ADD_B // Byte add: NEXT + TOS → byte result.
	ADD_W // Word add: NEXT + TOS → word result.
	SUB_B // Byte subtract: NEXT − TOS → byte result.
	SUB_W // Word subtract: NEXT − TOS → word result.
	MUL_B // Byte multiply: NEXT * TOS → 16-bit result.
	MUL_W // Word multiply: NEXT * TOS → 16-bit result (truncated).
	DIV_B // Byte unsigned divide: NEXT / TOS → byte result.
	DIV_W // Word unsigned divide: NEXT / TOS → word result.
	MOD_B // Byte unsigned modulo: NEXT % TOS → byte result.
	MOD_W // Word unsigned modulo: NEXT % TOS → word result.
	SHL_B // Byte shift left: pop count (TOS), pop value (NEXT), push NEXT << count.
	SHL_W // Word shift left: pop count (TOS), pop value (NEXT), push NEXT << count.
	SHR_B // Byte shift right: pop count (TOS), pop value (NEXT), push NEXT >> count.
	SHR_W // Word shift right: pop count (TOS), pop value (NEXT), push NEXT >> count.
	AND_B // Byte bitwise AND: NEXT & TOS.
	AND_W // Word bitwise AND: NEXT & TOS.
	OR_B  // Byte bitwise OR:  NEXT | TOS.
	OR_W  // Word bitwise OR:  NEXT | TOS.
	XOR_B // Byte bitwise XOR: NEXT ^ TOS.
	XOR_W // Word bitwise XOR: NEXT ^ TOS.
	NEG_B // Byte unary negation: 0 − TOS, byte result.
	NEG_W // Word unary negation: 0 − TOS, word result.
	NOT_B // Byte logical not: 1 if TOS==0 else 0, byte result.
	NOT_W // Word logical not: 1 if TOS==0 else 0, word result.

	// ── Casting ────────────────────────────────────────────────────

	CAST_W // Zero-extend byte to word (from TOS, push result).
	CAST_B // Truncate word to byte (keep low 8 bits, push result).

	// ── Stack Manipulation ─────────────────────────────────────────

	DUP  // Duplicate TOS: [a] → [a, a]
	DROP // Discard TOS:   [a] → []
	SWAP // Exchange TOS and NEXT: [a, b] → [b, a]

	// ── Comparison ─────────────────────────────────────────────────

	IS_B // [cond] Pop two byte values; compare NEXT against TOS using condition; push 0/1.
	IS_W // [cond] Pop two word values; compare NEXT against TOS using condition; push 0/1.

	// ── Control Flow ───────────────────────────────────────────────

	TAG   // [name] Declare a jump target label. Global scope, forward-referencable.
	GO    // [name] Unconditional jump to tag.
	GO_IF // [name] Pop value; jump to tag if non-zero (true).

	// ── Procedures ─────────────────────────────────────────────────

	ROUTE          // [name] Declare start of a subroutine.
	FRAME          // [size] Allocate stack frame (must follow ROUTE for reentrant procs).
	LOCAL_B        // [name] Declare 8-bit frame-relative local (requires FRAME).
	LOCAL_W        // [name] Declare 16-bit frame-relative local (requires FRAME).
	RUN            // [name] Call a subroutine (return address on SP).
	DONE           // Return from subroutine (RET).
	DONE_INTERRUPT // Return from interrupt handler (RETI).
	DONE_NMI       // Return from NMI handler (RETN).

	// ── Tasks ──────────────────────────────────────────────────────

	JOB      // [name] Declare start of a cooperative task.
	PRIORITY // [n] One-shot: set priority (0-15) for the next JOB.
	BYE      // Yield control back to the scheduler.
	SLEEP    // Pop 16-bit tick count; sleep current task for that many ticks.
	STOP     // [name] Suspend the named task.
	START    // [name] Resume the named task.

	// ── Port I/O ───────────────────────────────────────────────────

	IN_B  // [port] Read byte from hardware port; push result.
	IN_W  // [port] Read word from hardware port; push result.
	OUT_B // [port] Pop byte value; write to hardware port.
	OUT_W // [port] Pop word value; write to hardware port.

	// ── Interrupts ─────────────────────────────────────────────────

	INT // [name] Install name as the maskable interrupt handler.
	NMI // [name] Install name as the non-maskable interrupt handler.
	HLT // Halt CPU until next interrupt.
	DII // Disable interrupts.
	ENI // Enable interrupts.

	// ── Random / Entropy ───────────────────────────────────────────

	SEED // Push a pseudo-random byte onto the data stack.

	// ── Bank Switching ─────────────────────────────────────────────

	BANK   // [number] Compile-time directive: place subsequent code/data in ROM bank.
	SWITCH // Runtime bank switch: pop bank number, perform mapper switch.

	// ── Data Emission ──────────────────────────────────────────────

	DATA_B    // [number] Emit a byte of ROM constant data.
	DATA_W    // [number] Emit a word of ROM constant data.
	DATA_STR  // [string] Emit a null-terminated string constant.

	// ── Pragma ─────────────────────────────────────────────────────

	PRAGMA // [number] Set runtime pragma flags (bitmask: bit 0 = BOUNDCHECK).

	// ── Inline Assembly ────────────────────────────────────────────

	INLINE // [string] Embed raw assembly text verbatim.

	// ── Battery RAM ────────────────────────────────────────────────

	SRAM_ON  // Enable battery-backed SRAM access (SMS port 0xFFFC bit 3).
	SRAM_OFF // Disable battery-backed SRAM access.
	SAVE     // [3 pops: length, dest addr, src addr] Block copy from src to dest.
	LOAD     // [3 pops: length, dest addr, src addr] Block copy from src to dest.
)

// OperandType describes the kind of operand an instruction carries.
type OperandType int

const (
	OpNone      OperandType = iota // no operand (e.g. ADD_B, DONE)
	OpNumber                       // unsigned 16-bit literal (e.g. PUSH_W 42, FRAME 6)
	OpName                         // identifier (e.g. GET_B x, TAG loop)
	OpString                       // string literal (e.g. DATA_STR "hello", INLINE "nop")
	OpCondition                    // comparison condition (e.g. IS_W LT)
)

// Condition enumerates the six unsigned comparison relations.
type Condition int

const (
	CondLT Condition = iota // <
	CondGT                  // >
	CondLE                  // <=
	CondGE                  // >=
	CondEQ                  // ==
	CondNE                  // !=
)

var condNames = map[Condition]string{
	CondLT: "LT",
	CondGT: "GT",
	CondLE: "LE",
	CondGE: "GE",
	CondEQ: "EQ",
	CondNE: "NE",
}

var condFromName = map[string]Condition{
	"LT": CondLT,
	"GT": CondGT,
	"LE": CondLE,
	"GE": CondGE,
	"EQ": CondEQ,
	"NE": CondNE,
}

// String returns the uppercase condition mnemonic.
func (c Condition) String() string {
	if s, ok := condNames[c]; ok {
		return s
	}
	return fmt.Sprintf("Cond(%d)", c)
}

// Operand is the (at most one) argument carried by an instruction.
type Operand struct {
	Type OperandType
	Num  uint16
	Name string
	Str  string
	Cond Condition
}

// Instr couples an opcode with an optional operand.
type Instr struct {
	Op      Instruction
	Operand Operand
}

// Program is a sequence of PIR instructions.
type Program struct {
	Instrs []Instr
}

// instructionNames maps each opcode to its text mnemonic.
var instructionNames = map[Instruction]string{
	NOP:            "NOP",
	PUSH_B:         "PUSH_B",
	PUSH_W:         "PUSH_W",
	ALLOC:          "ALLOC",
	VAR:            "VAR",
	AT:             "AT",
	GET_B:          "GET_B",
	GET_W:          "GET_W",
	PUT_B:          "PUT_B",
	PUT_W:          "PUT_W",
	PUSH_A:         "PUSH_A",
	PUSH_D:         "PUSH_D",
	READ_B:         "READ_B",
	READ_W:         "READ_W",
	WRITE_B:        "WRITE_B",
	WRITE_W:        "WRITE_W",
	ADD_B:          "ADD_B",
	ADD_W:          "ADD_W",
	SUB_B:          "SUB_B",
	SUB_W:          "SUB_W",
	MUL_B:          "MUL_B",
	MUL_W:          "MUL_W",
	DIV_B:          "DIV_B",
	DIV_W:          "DIV_W",
	MOD_B:          "MOD_B",
	MOD_W:          "MOD_W",
	SHL_B:          "SHL_B",
	SHL_W:          "SHL_W",
	SHR_B:          "SHR_B",
	SHR_W:          "SHR_W",
	AND_B:          "AND_B",
	AND_W:          "AND_W",
	OR_B:           "OR_B",
	OR_W:           "OR_W",
	XOR_B:          "XOR_B",
	XOR_W:          "XOR_W",
	NEG_B:          "NEG_B",
	NEG_W:          "NEG_W",
	NOT_B:          "NOT_B",
	NOT_W:          "NOT_W",
	CAST_W:         "CAST_W",
	CAST_B:         "CAST_B",
	DUP:            "DUP",
	DROP:           "DROP",
	SWAP:           "SWAP",
	IS_B:           "IS_B",
	IS_W:           "IS_W",
	TAG:            "TAG",
	GO:             "GO",
	GO_IF:          "GO_IF",
	ROUTE:          "ROUTE",
	FRAME:          "FRAME",
	LOCAL_B:        "LOCAL_B",
	LOCAL_W:        "LOCAL_W",
	RUN:            "RUN",
	DONE:           "DONE",
	DONE_INTERRUPT: "DONE_INTERRUPT",
	DONE_NMI:       "DONE_NMI",
	JOB:            "JOB",
	PRIORITY:       "PRIORITY",
	BYE:            "BYE",
	SLEEP:          "SLEEP",
	STOP:           "STOP",
	START:          "START",
	IN_B:           "IN_B",
	IN_W:           "IN_W",
	OUT_B:          "OUT_B",
	OUT_W:          "OUT_W",
	INT:            "INT",
	NMI:            "NMI",
	HLT:            "HLT",
	DII:            "DII",
	ENI:            "ENI",
	SEED:           "SEED",
	BANK:           "BANK",
	SWITCH:         "SWITCH",
	DATA_B:         "DATA_B",
	DATA_W:         "DATA_W",
	DATA_STR:       "DATA_STR",
	PRAGMA:         "PRAGMA",
	INLINE:         "INLINE",
	SAVE:           "SAVE",
	LOAD:           "LOAD",
	SRAM_ON:        "SRAM_ON",
	SRAM_OFF:       "SRAM_OFF",
}

// nameFromMnemonic is the reverse lookup of instructionNames.
var nameFromMnemonic map[string]Instruction

func init() {
	nameFromMnemonic = make(map[string]Instruction, len(instructionNames))
	for op, name := range instructionNames {
		nameFromMnemonic[name] = op
	}
}

// String returns the text form of the instruction, e.g. "PUSH_W 42" or "IS_W LT".
func (i Instr) String() string {
	opName := instructionNames[i.Op]
	if opName == "" {
		opName = fmt.Sprintf("INSTR(%d)", i.Op)
	}
	switch i.Operand.Type {
	case OpNone:
		return opName
	case OpNumber:
		return fmt.Sprintf("%s %d", opName, i.Operand.Num)
	case OpName:
		return fmt.Sprintf("%s %s", opName, i.Operand.Name)
	case OpString:
		return fmt.Sprintf("%s %q", opName, i.Operand.Str)
	case OpCondition:
		return fmt.Sprintf("%s %s", opName, i.Operand.Cond)
	default:
		return fmt.Sprintf("%s <?>", opName)
	}
}

// String returns the full program text, one instruction per line.
func (p *Program) String() string {
	var b strings.Builder
	for _, instr := range p.Instrs {
		b.WriteString(instr.String())
		b.WriteByte('\n')
	}
	return b.String()
}

// operands returns the expected OperandType for each opcode.
func expectedOperand(op Instruction) OperandType {
	switch op {
	case NOP, ADD_B, ADD_W, SUB_B, SUB_W,
		MUL_B, MUL_W, DIV_B, DIV_W, MOD_B, MOD_W,
		SHL_B, SHL_W, SHR_B, SHR_W,
		AND_B, AND_W, OR_B, OR_W, XOR_B, XOR_W,
		NEG_B, NEG_W, NOT_B, NOT_W,
		CAST_W, CAST_B,
		DUP, DROP, SWAP,
		READ_B, READ_W, WRITE_B, WRITE_W,
		BYE, SLEEP,
		HLT, DII, ENI, SEED, SWITCH,
		SAVE, LOAD, SRAM_ON, SRAM_OFF:
		return OpNone
	case PUSH_B, PUSH_W, AT, ALLOC, FRAME, PRIORITY,
		BANK, DATA_B, DATA_W, PRAGMA:
		return OpNumber
	case VAR, GET_B, GET_W, PUT_B, PUT_W,
		PUSH_A, PUSH_D, TAG, GO, GO_IF,
		ROUTE, RUN, LOCAL_B, LOCAL_W,
		JOB, STOP, START,
		INT, NMI:
		return OpName
	case DATA_STR, INLINE:
		return OpString
	case IS_B, IS_W:
		return OpCondition
	default:
		return OpNone
	}
}

// ParseError describes an error encountered during text parsing.
type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

// Parse converts PIR text format back into a Program.
// Lines starting with "//" are comments; empty lines are skipped.
// Each non-comment line is: MNEMONIC [operand].
func Parse(text string) (*Program, error) {
	lines := strings.Split(text, "\n")
	prog := &Program{}
	for lineIdx, line := range lines {
		lineNum := lineIdx + 1
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		parts := splitLine(line)
		if len(parts) == 0 {
			continue
		}
		mnemonic := parts[0]
		op, ok := nameFromMnemonic[mnemonic]
		if !ok {
			return nil, &ParseError{Line: lineNum, Msg: fmt.Sprintf("unknown instruction %q", mnemonic)}
		}
		exp := expectedOperand(op)
		var operand Operand
		switch exp {
		case OpNone:
			if len(parts) > 1 {
				return nil, &ParseError{Line: lineNum, Msg: fmt.Sprintf("%s takes no operand", mnemonic)}
			}
		case OpNumber:
			if len(parts) < 2 {
				return nil, &ParseError{Line: lineNum, Msg: fmt.Sprintf("%s needs a number operand", mnemonic)}
			}
			n, err := parseNumber(parts[1])
			if err != nil {
				return nil, &ParseError{Line: lineNum, Msg: err.Error()}
			}
			operand = Operand{Type: OpNumber, Num: n}
		case OpName:
			if len(parts) < 2 {
				return nil, &ParseError{Line: lineNum, Msg: fmt.Sprintf("%s needs a name operand", mnemonic)}
			}
			operand = Operand{Type: OpName, Name: parts[1]}
		case OpString:
			if len(parts) < 2 {
				return nil, &ParseError{Line: lineNum, Msg: fmt.Sprintf("%s needs a string operand", mnemonic)}
			}
			s, err := strconv.Unquote(parts[1])
			if err != nil {
				return nil, &ParseError{Line: lineNum, Msg: fmt.Sprintf("bad string: %v", err)}
			}
			operand = Operand{Type: OpString, Str: s}
		case OpCondition:
			if len(parts) < 2 {
				return nil, &ParseError{Line: lineNum, Msg: fmt.Sprintf("%s needs a condition operand", mnemonic)}
			}
			cond, ok := condFromName[parts[1]]
			if !ok {
				return nil, &ParseError{Line: lineNum, Msg: fmt.Sprintf("unknown condition %q", parts[1])}
			}
			operand = Operand{Type: OpCondition, Cond: cond}
		}
		prog.Instrs = append(prog.Instrs, Instr{Op: op, Operand: operand})
	}
	return prog, nil
}

// splitLine splits a text line on whitespace, respecting double-quoted strings.
func splitLine(line string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for _, ch := range line {
		switch {
		case ch == '"':
			inQuote = !inQuote
			cur.WriteRune(ch)
		case ch == ' ' || ch == '\t':
			if inQuote {
				cur.WriteRune(ch)
			} else if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// parseNumber parses a decimal or 0x-hex number.
func parseNumber(s string) (uint16, error) {
	if len(s) > 2 && s[:2] == "0x" {
		n, err := strconv.ParseUint(s[2:], 16, 16)
		if err != nil {
			return 0, fmt.Errorf("bad number %q", s)
		}
		return uint16(n), nil
	}
	n, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("bad number %q", s)
	}
	return uint16(n), nil
}
