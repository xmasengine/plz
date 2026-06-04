# PL/Z Language Design Review

This document analyzes the PL/Z language implementation against its
specification, identifying design problems, undefined behavior, and memory
safety issues. References point to the implementation in `pkg/plz/`.

> **Status:** Some items have been fixed since this review was written.
> See `doc/plz-todo.md` for the current consolidated TODO list, and
> `doc/plz-review-2.md` for a supplementary review. Items below are
> annotated with ✓ (fixed), ~ (partially fixed), or (open).

---

## Critical Bugs

### Initializer Values Silently Discarded ✓

`DECLARE x BYTE = 42` parses and checks without error, but the `42` is
silently thrown away during code generation. ~~The variable is zero-initialized
at runtime.~~ This is the most misleading bug in the compiler — the language
accepts initializer syntax, the semantic checker validates the expression, yet
the code generator never reads `Declare.Initializer`.

**Location:** `gen.go:1923-1966` (`Declare.Gen` never inspects `s.Initializer`)
**Impact:** Every `DECLARE ... = expr` is a no-op. Users believe their
variables have initial values when they do not.
**Status:** ✓ Fixed. `Declare.Gen` now emits store code for initializers.

### No Array Bounds Checking ~

Array subscripts are never validated ~~at compile time or~~ at runtime.
Accessing `arr[9999]` on a 10-element array silently reads or writes
arbitrary memory. ~~The checker validates that subscripts parse but never
compares them against the declared size.~~ The code generator emits address
arithmetic with no guard.

**Location:** `check.go:735-739`, `gen.go:1086-1143`
**Impact:** Out-of-bounds accesses corrupt memory silently. No diagnostic
is produced at any stage.
**Status:** ~ Partially fixed. Compile-time bounds checking for constant
subscripts is done. Runtime bounds checking is opt-in via `PRAGMA BOUNDCHECK`.

### CALL Argument Count Not Validated (open)

Calling a procedure with the wrong number of arguments causes a Go
index-out-of-bounds panic in the compiler itself. The checker passes all
argument expressions without comparing the count against the procedure's
parameter list. The code generator then indexes past the slice boundary.

**Location:** `check.go:628-635`, `gen.go:1181`
**Impact:** Compiler crash on invalid input. A malicious or erroneous source
file can crash the toolchain.

### Task Stack Overflow (open)

Each task gets a 128-byte stack. Neither the compiler nor the runtime
inserts a stack-limit check. A deep call chain, recursion, or many local
variables will silently write past the 128-byte boundary into the adjacent
TCB or another task's stack, corrupting scheduler state.

**Location:** `gen.go:462` (`ds 128` per task), scheduler code at `gen.go:2326-2430`
**Impact:** Silent memory corruption. Unpredictable program behavior.

### HALT in Task-Called Procedure Freezes CPU (open)

When a task calls a procedure that executes `HALT`, the entire CPU freezes
instead of marking that task as DEAD. This is because procedure bodies are
code-generated during `Program.Gen` at lines 418-422, which runs _before_
task bodies (line 432). At that point `g.InTask` is false, so all HALT
instructions inside procedures always emit a real Z80 `halt` instruction
rather than `jp _plz_task_done`.

**Location:** `gen.go:1988-1995` (`Halt.Gen`), `gen.go:418-422` (emit order)
**Impact:** Any procedure called from a task that calls HALT hangs the
entire system.

---

## Design Problems

### Priority Scheduler Is Dead Code ✓

The TCB layout reserves a priority byte (offset 4, 0=highest, 15=lowest),
the value is stored during task initialization, ~~but the scheduler code never
reads it. The scheduler uses pure round-robin starting from the current
task, picking the first READY task found regardless of priority.~~

**Location:** `gen.go:2367-2399` (scheduler scan), `gen.go:360-361` (priority stored)
**Impact:** Priority field is cosmetic. All tasks are treated equally.
**Status:** ✓ Fixed. Scheduler uses `best_pri`/`best_idx` to select the
READY task with lowest priority value, round-robin within priority level.

### Non-REENTRANT Recursion (open)

Procedures default to static RAM frame allocation. Parameters and locals are
stored in global labels like `_plz_procName_varName`. If such a procedure
calls itself (directly or indirectly through the call graph), the second
invocation overwrites the first invocation's parameters and locals. The
checker performs no call-graph analysis to detect this.

**Location:** `gen.go:1663-1679` (static frame allocation)
**Impact:** Recursive calls to non-REENTRANT procedures silently corrupt
state. Only `REENTRANT` procedures are safe for recursion, but nothing
enforces or warns about this.

### RETURN/GOTO Inside FOR Loop Corrupts Stack (open)

The FOR loop pushes the loop's step and end values onto the Z80 stack.
If a `RETURN` or `GOTO` inside the loop body exits early, the pushed values
remain on the stack, corrupting the caller's stack frame. The checker does
not prohibit early exits from FOR loops.

**Location:** `gen.go:1505-1559` (`Group.Gen` for FOR)
**Impact:** Early exit from a FOR loop crashes the program when the
procedure tries to return.

### Nested Procedures Emit Inline (open)

A `PROCEDURE` declared inside another procedure is parsed as a statement in
the outer body. During code generation, the inner procedure's `Gen` is called
inline, emitting its full prologue and epilogue (including `ret`) in the
middle of the outer procedure's code. Execution can fall through from the
outer procedure into the inner procedure's entry, causing spurious
invocations and stack corruption.

**Location:** `parser.go:286-294` (no special handling), `gen.go:382-412` (only top-level segregated)
**Impact:** Nested procedures are emitted as inline code reachable by
fall-through. They execute unexpectedly.

### Recursive INCLUDE (open)

The PL/Z parser recurses on `INCLUDE` with no cycle detection. If
`a.plz` includes `b.plz` which includes `a.plz`, the Go call stack
overflows. The assembler (`z80asm`) has recursive-include detection,
showing this was considered but not implemented in the compiler frontend.

**Location:** `parser.go:170-214` (`Program.Parse` INCLUDE handling)
**Impact:** Recursive includes crash the compiler with a stack overflow.

---

## Memory Safety Gaps (all open)

### No Definite-Assignment Analysis

Reading a variable before it has been written yields zero (from the
zeroed RAM). The checker does not perform any definite-assignment analysis
and emits no warning.

**Location:** `check.go:518-550` (`Let.Check`)
**Impact:** Logic bugs from uninitialized reads are not detected.

### TEXT Type Allocates One Byte

The built-in `TEXT` type alias is:

```
DEFINE TEXT RECORD length BYTE, text ARRAY BYTE END
```

The `text` field is an unbounded array (`Size=0`). `Type.Size()` for an
unbounded array returns 0, so `DECLARE t TEXT` allocates exactly 1 byte
(for `length`) + 0 bytes (for `text`) = 1 byte, rounded to 1. Storing any
string into a TEXT variable overflows its allocation.

**Location:** `parser.go:52-73` (builtin type aliases)
**Impact:** TEXT variables cannot hold any data. Every use overflows.

### AT Overlap Not Detected

`AT 0xC000` followed by a declaration, then `AT 0xC000` followed by another
declaration, creates two overlapping allocations at the same address. The
checker does not track used address ranges.

**Location:** `gen.go:1883-1895` (`At.Gen`), `check.go:287-297` (`At.Check`)
**Impact:** Data corruption from overlapping address assignments.

### No 17+ Task Limit

The scheduler loops exactly 16 times and the TCB array is hardcoded at 128
bytes (16 tasks × 8 bytes). If 17 or more tasks are defined, the init loop
writes TCB slots past the allocation boundary, corrupting adjacent memory.

**Location:** `gen.go:356-368` (init loop), `check.go:233-268` (no limit enforce)
**Impact:** Memory corruption with 17+ tasks.

### SAVE/LOAD Size Mismatch

The checker validates the structure of SAVE/LOAD operands but never compares
the byte count of source versus destination. Loading a large source into a
small destination overflows the target buffer.

**Location:** `gen.go:2043-2140` (`Save.Gen`, `Load.Gen`), `check.go:446-512`
**Impact:** Buffer overflow in SAVE/LOAD operations.

---

## Undefined Behavior (all open)

### Division by Zero

The runtime `_plz_divmod` routine does not check whether DE == 0 before
dividing. The compile-time constant evaluator catches division-by-zero in
constant expressions, but runtime division (`LET x = a / b` where `b`
happens to be zero) runs 16 iterations dividing by zero, producing garbage
without crashing.

**Location:** `gen.go:247-267` (`_plz_divmod`)
**Impact:** Division by zero at runtime produces undefined results.

### BYTE and WORD Overflow

All arithmetic is performed in 16-bit Z80 registers. When stored to a BYTE
variable, the high byte is silently truncated (e.g., `255 + 1` → 256,
stored as `0`). WORD arithmetic wraps modulo 65536 silently (`65535 + 1` →
0). No carry is checked or reported.

**Location:** `gen.go:834-838` (`Infix.Gen`), `gen.go:1290-1313` (`Let.Gen` store)
**Impact:** Overflow produces silently wrong results.

### Scheduler Not Re-entrant

The scheduler saves the current SP through a global memory word
`_plz_sch_sp`. If an interrupt handler re-enters the scheduler (e.g., the
handler calls YIELD), `_plz_sch_sp` is overwritten, corrupting the first
save.

**Location:** `gen.go:2341-2345` (scheduler saving SP)
**Impact:** Re-entrant scheduler calls corrupt SP save.

### GOTO Scope Not Validated

`GOTO label` emits `jp label` with no validation that the target label is
in the same function, or that jumping there is structurally legal. Jumping
into the middle of a loop body or past a variable declaration produces
undefined behavior.

**Location:** `gen.go:1872-1877` (`GoTo.Gen`)
**Impact:** Arbitrary jumps produce undefined runtime behavior.

### HALT With Interrupts Disabled

When the scheduler finds no READY task, it does:

```
halt
jp _plz_scheduler
```

If interrupts are disabled when the scheduler runs (e.g., from a
non-interrupt context), the HALT never wakes and the CPU freezes
permanently. The scheduler does not emit `ei` before `halt`.

**Location:** `gen.go:2397-2399`
**Impact:** CPU freezes if scheduler runs with interrupts disabled.

### SLEEP/YIELD at Top Level

If SLEEP or YIELD is used at top level when no tasks are defined, the
generated code references `_plz_current_task` and `_plz_tcbs` which are
never emitted. The assembler fails with an undefined symbol error.

**Location:** `gen.go:2282-2313` (`Sleep.Gen`, `Yield.Gen`)
**Impact:** Assembler error on SLEEP/YIELD outside task context.

---

## Type System Gaps (all open)

### No Type Compatibility Checking

The checker enforces almost no type compatibility rules:

- Assignment: `LET wordVar = byteVar` or `LET byteVar = wordVar` — always accepted
- Arguments: can pass BYTE where WORD expected and vice versa
- Return types: procedure declared `BYTE` can return a `WORD`
- Record types: two structurally identical record types are interchangeable
- Array element types: not validated against declared element type

### Implicit Narrowing

`LET b = w` where `b` is BYTE and `w` is WORD stores only `ld a, l` / `ld (b), a`, silently dropping the high byte. No warning is emitted.

**Location:** `gen.go:1290-1313` (`Let.Gen`)

### Negative Array Size

`DECLARE arr ARRAY [-5] BYTE` parses without error. The checker does not
validate that the size is non-negative. `Declare.StorageSize()` returns a
negative total, and the emit loop never executes, allocating zero bytes.

**Location:** `check.go:310-318` (`Declare.Check`), `ast.go:246-259`

---

## Additional Issues (all open)

### Constants Exceed Z80 Range

Constant expressions are evaluated in Go's `int` (64-bit on 64-bit
platforms). A constant value exceeding 65535 is emitted as `ld hl, 100000`
and the assembler truncates it to 16 bits. No warning is produced.

**Location:** `check.go:89-221` (`EvalConstExpr`)

### FOR Loop Step/End Stack Cleanup

The FOR loop pushes the step and end values onto the Z80 stack for the
comparison at each iteration. On normal exit (the loop completes), these
values remain on the stack and are never cleaned up. In nested FOR loops,
the stack depth grows by 2 per loop level, consuming stack space
permanently.

**Location:** `gen.go:1505-1559` (`Group.Gen` for FOR)

### `isSimpleOperand` Assumptions

The code generator's `isSimpleOperand` returns `true` for `Reference()`
unconditionally, without checking whether the reference has subscripts or
field accesses. A reference with subscripts calls a helper that may load
into HL and then clobber HL in subsequent code. The optimization path for
comparisons assumes the operand loads directly into HL without side
effects.

**Location:** `gen.go:770-801` (`isSimpleOperand`)

### Global Variable Name Collision

Global variables use their source-level names as assembly labels. A user
variable named `sp`, `hl`, `af`, or other Z80 register name would confuse
the assembler. The `_plz_` prefix convention protects runtime symbols but
not user names.

**Location:** `gen.go:116-125` (`localSym` outside procedures)

---

## Summary

| Severity | Issue | Status |
|----------|-------|--------|
| CRITICAL | Initializer values silently discarded | ✓ Fixed |
| CRITICAL | No array bounds checking | ~ Partial (compile-time only, runtime opt-in) |
| CRITICAL | CALL argument count not validated (Go panic) | open |
| CRITICAL | Task stack overflow (128 bytes, no guard) | open |
| CRITICAL | HALT in task-called procedure freezes CPU | open |
| HIGH | Priority scheduler is dead code (pure round-robin) | ✓ Fixed |
| HIGH | Non-REENTRANT recursion corrupts state | open |
| HIGH | RETURN/GOTO in FOR loop corrupts stack | open |
| HIGH | Nested procedures emit inline (fall-through bug) | open |
| HIGH | Recursive INCLUDE crashes compiler | open |
| HIGH | Scheduler not re-entrant (global `_plz_sch_sp`) | open |
| HIGH | TEXT type allocates 1 byte | open |
| HIGH | AT overlap not detected | open |
| HIGH | No 17+ task limit | open |
| MEDIUM | Division by zero at runtime (garbage result) | open |
| MEDIUM | BYTE/WORD overflow wraps silently | open |
| MEDIUM | GOTO scope not validated | open |
| MEDIUM | SLEEP/YIELD at top level crashes | open |
| MEDIUM | HALT with interrupts disabled freezes CPU | open |
| MEDIUM | SAVE/LOAD size mismatch | open |
| LOW | No type compatibility checking | open |
| LOW | Implicit WORD→BYTE narrowing | open |
| LOW | Negative array size accepted | open |
| LOW | Constants >65535 not warned | open |
| LOW | FOR loop stack values never popped | open |
| LOW | Global variable name collision | open |

See `doc/plz-todo.md` for the full consolidated TODO list.
