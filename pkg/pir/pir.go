// Package pir defines the PLZ Intermediate Representation (PIR).
// PIR models an abstract stack machine (PAM) with a data stack, a return
// stack, named storage locations, and a one-shot AT directive.
// Each instruction has at most one operand.
package pir

// Instruction identifies a PIR opcode.
type Instruction int

const (
	// ── Data Movement ──────────────────────────────────────────────

	NOP    Instruction = iota // no operation
	PUSH_B                    // [number] Push 8-bit literal onto the data stack.
	PUSH_W                    // [number] Push 16-bit literal onto the data stack.
	VAR_B                     // [name] Define an 8-bit global variable.
	VAR_W                     // [name] Define a 16-bit global variable.
	AT                        // [number] One-shot: assign next VAR/DATA/ROUTE/JOB to this hardware address.
	GET_B                     // [name] Fetch 8-bit variable value; push to data stack.
	GET_W                     // [name] Fetch 16-bit variable value; push to data stack.
	PUT_B                     // [name] Pop 8-bit value from stack; write to variable.
	PUT_W                     // [name] Pop 16-bit value from stack; write to variable.

	// ── Pointers & Memory ──────────────────────────────────────────

	PUSH_A  // [name] Push the 16-bit hardware address of name onto the data stack.
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

	TAG  // [name] Declare a jump target label. Global scope, forward-referencable.
	GO   // [name] Unconditional jump to tag.
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

	JOB   // [name] Declare start of a cooperative task.
	BYE   // Yield control back to the scheduler.
	SLEEP // Pop 16-bit tick count; sleep current task for that many ticks.
	STOP  // [name] Suspend the named task.
	START // [name] Resume the named task.

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

	DATA_B   // [number] Emit a byte of ROM constant data.
	DATA_W   // [number] Emit a word of ROM constant data.
	DATA_STR // [string] Emit a null-terminated string constant.
	DATA_TILE // [string] Emit an 8x8 SMS tile from a backtick string.

	// ── Pragma ─────────────────────────────────────────────────────

	PRAGMA // [number] Set runtime pragma flags (bitmask: bit 0 = BOUNDCHECK).

	// ── Inline Assembly ────────────────────────────────────────────

	INLINE // [string] Embed raw assembly text verbatim.

	// ── Battery RAM ────────────────────────────────────────────────

	SAVE // Pop length, then destination address, then source address; copy to battery RAM.
	LOAD // Pop length, then destination address, then source address; copy from battery RAM.
)
