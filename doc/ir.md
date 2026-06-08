# Intermediate Representation for PLZ

The generator currently is a mess. I plan to fix it by switching from a direct
tree walking generator to one which uses an intermediate representation.
The intermediate representation or IR will model an abstract stack machine
with instructions that can have only zero or one operand.

This abstract machine is called the PAM for PLZ Abstract Machine.
The intermediate representation is named PIR.

The PAM has a return stack (for CALL/RETURN), a data stack (for expression
evaluation), and named variable locations.

This IR will then be translated to the target architecture (initially Z80).


# Data stack operation order

All binary operations pop **two** values from the data stack:

1. **TOP OF STACK (TOS)** — first pop, right operand
2. **NEXT** — second pop, left operand

The result `NEXT op TOS` is then pushed back.

Examples:

    PUSH_W 10          ; stack: [10]
    PUSH_W 3           ; stack: [10, 3]     (3 is TOS)
    ADD_W              ; pops 3 (TOS), pops 10 (NEXT); 10+3 → push 13

    PUSH_W 10
    PUSH_W 3
    SUB_W              ; 10 - 3 → 7

Unary operations (CAST, NOT via XOR -1, etc.) pop one value and push one result.


# Complete IR Instruction List

## Data Movement

PUSH_B [val]
PUSH_W [val]

Pushes 8-bit or 16-bit literal onto the data stack.

VAR_B [name]
VAR_W [name]

Defines an 8-bit or 16-bit variable. Variables have global scope.
It is allowed to refer to a variable defined later in the IR.

AT [address]

One-shot directive. When placed immediately before VAR, DATA_*, ROUTE,
or JOB, it assigns that declaration to the given hardware address.
If no declaration follows, the AT is silently ignored. Only the next
single declaration is affected.

GET_B [name]
GET_W [name]

Fetches a variable value and pushes it onto the data stack.

PUT_B [name]
PUT_W [name]

Pops a value from the data stack and writes it to the named variable.

## Pointers & Memory

PUSH_A [name]

Pushes the 16-bit hardware memory address of name onto the data stack.
Useful for array indexing: push base address, add index, then READ/WRITE.

READ_B
READ_W

Pops a 16-bit address; reads a value from that RAM address; pushes it.

WRITE_B
WRITE_W

Pops a value, then pops a 16-bit address; writes the value to RAM.

## Math & Logic (typed)

Every arithmetic and logic instruction comes in two widths: `_B` (byte)
and `_W` (word). The suffix determines both the operand size popped from
the stack and the result size pushed, with one exception noted below.

ADD_B / ADD_W

    NEXT + TOS

SUB_B / SUB_W

    NEXT - TOS

MUL_B / MUL_W

    NEXT * TOS

    For MUL_B: pops two byte operands, pushes a 16-bit result
    (since 255 * 255 = 65025). For MUL_W: pops two word operands,
    pushes a 16-bit result (truncated).

DIV_B / DIV_W

    NEXT / TOS  (unsigned)

MOD_B / MOD_W

    NEXT % TOS  (unsigned remainder)

SHL_B / SHL_W

    Pops shift count (TOS), then pops value (NEXT).
    Pushes `NEXT << count` (logical left shift).

SHR_B / SHR_W

    Pops shift count (TOS), then pops value (NEXT).
    Pushes `NEXT >> count` (logical right shift, zero-filled).

AND_B / AND_W
OR_B / OR_W
XOR_B / XOR_W

    NEXT & TOS  /  NEXT | TOS  /  NEXT ^ TOS

## Casting

CAST_W

    Pops a byte value, zero-extends it to 16 bits, pushes the result.

CAST_B

    Pops a word value, truncates to the low 8 bits (high byte discarded),
    pushes the result.

## Comparison

IS_B [cond]
IS_W [cond]

    Pops TOS (right operand), then pops NEXT (left operand).
    Compares `NEXT` against `TOS` using the given condition and
    pushes 1 (true) or 0 (false).

    Conditions: LT, GT, LE, GE, EQ, NE

    For IS_B both operands are byte-sized; for IS_W both are word-sized.

    Example — `IF a < b` where a and b are WORD:
        GET_W a          ; stack: [a]
        GET_W b          ; stack: [a, b]
        IS_W LT          ; pops b(TOS), a(NEXT); a < b → push 0/1
        GO_IF then_label ; jump if true

## Control Flow

TAG [name]

    Declares a label/tag that can be jumped to by GO or GO_IF.
    Tags have global scope; forward references are allowed.

GO [tag]

    Unconditional jump to tag.

GO_IF [tag]

    Pops a value from the data stack. If non-zero (true), jumps to tag.
    Falls through otherwise.

## Procedures

ROUTE [name]

    Declares the start of a subroutine. Code follows until the next
    ROUTE, JOB, or end of the IR program. Forward references allowed.

FRAME [size]

    Declares a stack frame of `size` bytes for the current ROUTE.
    Must appear immediately after ROUTE (before any other instruction
    in the subroutine body). The backend sets up a frame pointer
    (IX on Z80) and adjusts SP to allocate `size` bytes.

    A DONE/DONE_INTERRUPT/DONE_NMI in a FRAME-d procedure tears
    down the frame before returning.

    Non-REENTRANT procedures omit FRAME entirely and use global
    VAR storage instead.

LOCAL_B [name]
LOCAL_W [name]

    Declare an 8-bit or 16-bit local variable allocated on the stack
    frame. Valid only inside a ROUTE that uses FRAME. The variable
    is accessed via GET_B/PUT_B (same instructions as global VARs);
    the backend resolves the addressing mode from the declaration.

    The offset within the frame is assigned sequentially: first
    LOCAL_B uses offset 0, next LOCAL_W uses offset 1 (or 2 for
    word alignment), etc.

    For reentrant recursion to work, parameters are also stored
    in LOCAL slots — the prologue copies register-passed arguments
    to their LOCAL positions.

RUN [name]

    Calls a subroutine. The return address is pushed onto the hardware
    return stack (SP). Arguments must be set up by the caller before RUN;
    return values arrive on the data stack.

    For REENTRANT callees, the caller pushes arguments onto the stack
    (following the standard Z80 calling convention) before RUN. The
    callee's FRAME then references them as LOCALs at known offsets
    from the frame pointer.

DONE

    Returns from a subroutine (plain RET). If the procedure has a
    FRAME, the frame is torn down (SP restored, IX popped) before
    the RET.

DONE_INTERRUPT

    Returns from an interrupt handler (RETI).

DONE_NMI

    Returns from a non-maskable interrupt handler (RETN).

Example — Reentrant factorial (5!):

    ; Caller:
    PUSH_W 5        ; push argument on data stack
    RUN fact        ; call
    ; data stack now has result (120)

    ; Callee:
    ROUTE fact
    FRAME 6         ; 3 locals × 2 bytes each
    LOCAL_W n       ; n at ix+0
    LOCAL_W result  ; result at ix+2
    LOCAL_W i       ; i at ix+4

    PUT_W n         ; pop argument from data stack → LOCAL n

    GET_W n         ; if n <= 1: return 1
    PUSH_W 1
    IS_W LE
    GO_IF base

    PUSH_W 1        ; result = 1
    PUT_W result
    PUSH_W 2        ; i = 2
    PUT_W i

loop:
    GET_W i         ; if i > n: done
    GET_W n
    IS_W GT
    GO_IF done

    GET_W result    ; result = result * i
    GET_W i
    MUL_W
    PUT_W result

    GET_W i         ; i = i + 1
    PUSH_W 1
    ADD_W
    PUT_W i

    GO loop

base:
    PUSH_W 1
    DONE

done:
    GET_W result
    DONE

## Tasks

JOB [name]

    Declares the start of a cooperative task. Code follows until the
    next ROUTE, JOB, or end of the IR program. Forward references allowed.

BYE

    Yields control from the current task back to the scheduler.
    The task may be resumed later.

SLEEP

    Pops a 16-bit tick count from the data stack.
    Suspends the current task for that many system ticks.
    On SMS, one tick ≈ 16 ms (one VBlank at 60 Hz).

STOP [name]

    Suspends (stops) the named task. The task stays stopped until
    START is issued.

START [name]

    Resumes a previously stopped or newly declared task.

## Port I/O

IN_B [port]
IN_W [port]

    Reads a byte or word from the given hardware port and pushes it
    onto the data stack. Port is a literal 8-bit address.

OUT_B [port]
OUT_W [port]

    Pops a value from the data stack and writes it to the given
    hardware port.

## Interrupts

INT [name]

    Installs name as the maskable interrupt handler (IM 1).

NMI [name]

    Installs name as the non-maskable interrupt handler.

HLT

    Halts the CPU until the next interrupt arrives.

DII

    Disable interrupts.

ENI

    Enable interrupts.

## Random / Entropy

SEED

    Pushes a pseudo-random byte onto the data stack.
    On Z80 this typically reads the R register as an entropy source;
    the quality is platform-dependent.

## Bank Switching (Sega mapper)

BANK [number]

    Compile-time directive: subsequent code and data are placed in
    ROM bank `number`. This is resolved by the assembler/linker.

SWITCH

    Runtime bank switch. Pops a bank number from the data stack and
    performs a runtime mapper switch (e.g. writes to port 0xFFFD
    on Sega hardware).

## Data Emission (ROM constants)

DATA_B [number]

    Emits a single byte as ROM constant data at the current location.

DATA_W [number]

    Emits a 16-bit word as ROM constant data at the current location.

DATA_STR [string]

    Emits a null-terminated string as ROM constant data.

DATA_TILE [string]

    Emits an 8×8 SMS tile from a backtick string.
    Characters: `.` = palette index 0, `0-9` = palette 0-9,
    `A-F` or `a-f` = palette 10-15.

    Example:
        TAG arrow_tile
        DATA_TILE `..##..##`
        DATA_TILE `.##..##.`

    Named data blocks are created by preceding the DATA_* instructions
    with a TAG:

        TAG my_data
        DATA_B 10
        DATA_B 20
        DATA_B 30

## Pragma

PRAGMA [number]

    Sets runtime pragma flags. The number is interpreted as a bitmask:

        bit 0: BOUNDCHECK (enable array bounds checking)

## Inline Assembly

INLINE [string]

    Embeds a raw assembly string directly into the output.
    The string is passed through to the assembler verbatim.

## Battery RAM (Save/Load)

SAVE

    Pops a 16-bit length, then a 16-bit destination address, then a
    16-bit source address. Copies `length` bytes from source to
    destination. The destination is typically battery-backed SRAM
    at 0x8000 on SMS.

LOAD

    Pops a 16-bit length, then a 16-bit destination address, then a
    16-bit source address. Copies `length` bytes from source to
    destination. The source is typically battery-backed SRAM.


# Z80 Register Mapping

The following mapping is used for the Z80 backend:

┌────────────────────────────────────────────────────────────┐
│                    Z80 REGISTERS & FUNCTIONS                │
├───────────────────┬────────────────────────────────────────┤
│ REGISTER / PAIR   │ COMPILER ROLE                          │
├───────────────────┼────────────────────────────────────────┤
│     DE            │ Top of Data Stack (TOS Cache)          │
│       E           │   └─ Low byte (value for byte ops)     │
│       D           │   └─ Zero for byte ops; high byte      │
│                   │       for word ops                     │
├───────────────────┼────────────────────────────────────────┤
│     HL            │ Data Stack Pointer into SRAM           │
│                   │ Points to next free slot. Initialised  │
│                   │ past all static variables and tasks.    │
│                   │ Spill: (HL)=DE, HL++; Fill: --HL, DE=HL│
├───────────────────┼────────────────────────────────────────┤
│     SP            │ Return Stack (hardware CALL/RET stack) │
│                   │ Not used for data.                     │
├───────────────────┼────────────────────────────────────────┤
│     BC            │ General scratchpad / temp values       │
├───────────────────┼────────────────────────────────────────┤
│     AF            │ Accumulator (A) and flags (F)          │
│                   │ Used for 8-bit arithmetic and          │
│                   │ condition testing.                     │
├───────────────────┼────────────────────────────────────────┤
│     IX            │ Frame pointer for REENTRANT procedures │
│                   │ When not in a REENTRANT procedure,     │
│                   │ available as general scratchpad.       │
│                   │ In REENTRANT mode, locals are          │
│                   │ addressed as `ix + offset`.            │
├───────────────────┼────────────────────────────────────────┤
│     IY            │ Reserved. Generated code must NOT      │
│                   │ modify IY. On SMS, IY may be used by   │
│                   │ the BIOS or VDP interrupt handler.     │
└───────────────────┴────────────────────────────────────────┘

## Detailed Architecture Rules

### Top of Stack (TOS) Cache — DE

The absolute top item of the data stack must always reside in DE.
For an 8-bit value, `E` holds the value and `D` is cleared to 0.
For a 16-bit value, `DE` holds the full word.

This allows arithmetic operations to execute immediately on DE
for the TOS without a memory load.

### Data Stack Pointer — HL

HL points to the next available free position in SRAM data stack.
It is initialized past all static variable and task storage.

- **Spill** (when DE must be overwritten by a new TOS):
  `ld (hl), e` / `inc hl` / `ld (hl), d` / `inc hl`
- **Fill** (when TOS is consumed and next item must be loaded):
  `dec hl` / `ld d, (hl)` / `dec hl` / `ld e, (hl)`

### Return Stack — SP

The native Z80 SP is reserved exclusively for CALL/RET and task
switching. Data is never pushed/popped via SP by generated code.

### IX — REENTRANT Frame Pointer

In a REENTRANT procedure the prologue copies SP into IX, then
allocates local variable space by subtracting from SP. Locals are
accessed as `ix + offset`. This allows recursive calls since each
invocation gets its own frame.

Non-REENTRANT procedures use statically allocated storage and IX
is free as a scratch register.

### IY — Reserved

IY must not be modified by generated code. On SMS the BIOS and
VDP interrupt handler may rely on IY holding a system pointer.
