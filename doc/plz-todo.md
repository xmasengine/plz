# PLZ Compiler Suite — TODO

This file consolidates all open issues from `doc/plz-review.md` and
`doc/plz-review-2.md`. Each item is numbered as `N.M` where `N` is the
source priority level and `M` is the item number within that level.

Items already fixed are **not** listed here.

---

## Critical Bugs

### 1.1 Task stack overflow ✓

**File:** `pkg/plz/gen.go:441-443`

Each task gets a 128-byte stack (`ds 128`). ~~No stack-limit check is inserted.
A deep call chain or many locals silently writes past the boundary into the
adjacent TCB or another task's stack, corrupting scheduler state.~~
**Status:** ✓ Fixed. Stack canary (0xDE, 0xAD) written at bottom of each task
stack during init; checked in the scheduler on task yield. If overwritten
(stack overflowed), the task is marked DEAD.

### 1.2 HALT in task-called procedure freezes CPU ✓

**File:** `pkg/plz/gen.go:2312-2317`

~~`Halt.Gen` checks `g.InTask` to decide between `jp _plz_task_done` and a real
`halt`. But procedure bodies are generated before task bodies (order in
`Program.Gen`), so `g.InTask` is always false inside procedures — even when
the procedure is called from a task. The CPU freezes instead of marking the
task DEAD.~~
**Status:** ✓ Fixed. HALT is rejected inside task bodies at check time (use
YIELD instead). HALT remains valid at global scope and in normal procedures.

---

## Design Problems

### 2.1 Non-REENTRANT recursion ✓

**File:** `pkg/plz/gen.go` (static frame allocation), `pkg/plz/check.go`

~~Procedures default to static RAM labels for parameters/locals. A recursive
call overwrites the first invocation's frame. No call-graph analysis exists
to detect this or warn about it.~~
**Status:** ✓ Fixed. Call graph is built from all procedures after pass 1.
DFS cycle detection runs for each non-REENTRANT procedure. Direct self-calls
are also caught in `Call.Check` via scope chain lookup.

### 2.2 RETURN/GOTO inside FOR loop corrupts stack ✓

**File:** `pkg/plz/check.go` (Group.Check, Return.Check, GoTo.Check)

~~`FOR` pushes loop step and end values onto the Z80 stack. A `RETURN` or
`GOTO` inside the loop exits without popping them. The checker does not
prohibit early exits from FOR loops.~~
**Status:** ✓ Fixed. `Group.Check` pushes a loop scope (`IsLoop=true`) for FOR
body statements. `Return.Check` and `GoTo.Check` reject when `c.inLoop()` is
true.

### 2.3 Nested procedures emit inline ✓

**File:** `pkg/plz/parser.go:286-294`, `pkg/plz/gen.go` (Program.Gen)

~~A `PROCEDURE` inside another procedure is parsed as a statement in the outer
body. During code generation it is emitted inline, so execution can fall
through from the outer procedure into the inner procedure's entry.~~
**Status:** ✓ Fixed. `Procedure.Check` rejects when `c.inProcedure()` is true,
preventing nested procedure definitions.

### 2.4 Recursive INCLUDE ✓

**File:** `pkg/plz/parser.go:170-214`, `pkg/plz/compiler.go`, `pkg/plz/ast.go`

~~`INCLUDE` recurses with no cycle detection. `a.plz → b.plz → a.plz` crashes
the compiler with a Go stack overflow.~~
**Status:** ✓ Fixed. `Program.IncludedFiles` tracks visited absolute paths.
Recursive includes are detected and reported as errors.

---

## Memory Safety Gaps

### 3.1 No definite-assignment analysis

**File:** `pkg/plz/check.go:518-550`

Reading a variable before it has been written yields zero. No warning is
produced.

### 3.2 TEXT type allocates 1 byte

**File:** `pkg/plz/parser.go:52-73`

`DEFINE TEXT RECORD length BYTE, text ARRAY BYTE END` — the `text` field is
an unbounded array (Size=0). `DECLARE t TEXT` allocates exactly 1 byte
(length) + 0 bytes (text) = 1 byte. Storing any string overflows.

### 3.3 AT overlap not detected ✓

**File:** `pkg/plz/gen.go:1883-1895` (At.Gen), `pkg/plz/check.go:287-297` (~~At.Check~~)

~~`AT 0xC000` followed by a declaration, then `AT 0xC000` followed by another
declaration, creates overlapping allocations. No address-range tracking.~~
**Status:** ✓ Fixed. Checker tracks address ranges of AT-placed declarations
and standalone AT directives via `usedRanges` map. Overlaps are rejected at
check time.

### 3.4 No 17+ task limit ✓

**File:** `pkg/plz/gen.go` (init loop), `pkg/plz/check.go:233-268` (~~no enforcement~~)

~~TCB array is 128 bytes (16 tasks × 8 bytes). The init loops 16 times and
the scheduler scans 16 slots. 17+ tasks overflow the TCB allocation.~~
**Status:** ✓ Fixed. `Program.Check` rejects task definitions when
`len(c.TaskDefs) >= 16`.

### 3.5 SAVE/LOAD size mismatch ✓

**File:** `pkg/plz/gen.go:2043-2140` (Save.Gen, Load.Gen), `pkg/plz/check.go:446-512`

~~No comparison of source vs destination byte count. Loading a large source
into a small destination overflows the target.~~
**Status:** ✓ Fixed. `Save.Check` and `Load.Check` reject references with
`StorageSize() == 0` (e.g. TEXT variables with zero-length array field).

---

## Undefined Behavior

### 4.1 Division by zero ✓

**File:** `pkg/plz/gen.go:247-267` (`_plz_divmod`)

~~The runtime `_plz_divmod` does not check DE == 0 before dividing. Runtime
division by zero runs 16 iterations dividing by zero, producing garbage.~~
**Status:** ✓ Fixed. `_plz_divmod` checks DE == 0 at entry; if zero, returns
quotient=1, remainder=0. Documented in `doc/plz.md`.

### 4.2 BYTE and WORD overflow ✓

**File:** `pkg/plz/check.go:1322-1385`, `pkg/plz/gen.go:852-862`

All arithmetic is 16-bit. BYTE storage truncates silently. WORD wraps modulo
65536. No carry is checked or reported.

**Status:** ✓ Fixed. `BYTE(expr)` and `WORD(expr)` cast expressions added as
prefix operators. Overflow errors are raised for literal values that exceed
the target type's range (BYTE: >255, WORD: >65535) and for WORD→BYTE
assignments without an explicit `BYTE()` cast. Casts suppress the overflow
error, providing an explicit narrowing mechanism.

### 4.3 Scheduler not re-entrant

**File:** `pkg/plz/gen.go:2341-2345` (scheduler SP save)

The scheduler saves SP through global `_plz_sch_sp`. If an interrupt handler
re-enters the scheduler (e.g., calls YIELD), `_plz_sch_sp` is overwritten.

### 4.4 GOTO scope not validated ✓

**File:** `pkg/plz/check.go:375-420` (collectLabels, GoTo.Check)

`GOTO label` emits `jp label`. The checker validates that the label exists
but not that the jump is structurally legal (e.g., jumping into a loop body
or past a variable declaration).

**Status:** ✓ Fixed. Label collection tracks loop nesting depth. GOTO is
rejected when it targets a label inside a loop body (which would bypass
loop setup and potentially cause stack/frame issues).

### 4.5 HALT with interrupts disabled freezes CPU ✓

**File:** `pkg/plz/gen.go:2397-2399` (scheduler)

~~When the scheduler finds no READY task, it does `halt; jp _plz_scheduler`.
If interrupts are disabled at that point the HALT never wakes. No `ei` is
emitted before `halt`.~~
**Status:** ✓ Fixed. Added `ei` before `halt` in the idle loop.

### 4.6 SLEEP/YIELD at top level ✓

**File:** `pkg/plz/gen.go:2282-2313` (Sleep.Gen, Yield.Gen)

~~SLEEP/YIELD at top level when no tasks are defined references
`_plz_current_task` and `_plz_tcbs` which are never emitted. Assembler error.~~
**Status:** ✓ Fixed. `Sleep.Check` and `Yield.Check` reject when no tasks exist.

---

## Type System Gaps

### 5.1 No type compatibility checking

**File:** `pkg/plz/check.go` throughout

Assignment, arguments, return types, and record interchangeability are never
validated. BYTE vs WORD is always accepted.

### 5.2 Implicit narrowing ✓

**File:** `pkg/plz/check.go:1322-1385` (checkLetOverflow)

`LET b = w` where b is BYTE and w is WORD drops the high byte silently.
No warning.

**Status:** ✓ Fixed. This is now an error unless an explicit `BYTE()` cast is
used. `BYTE(expr)` narrows the value and signals intent. Literal overflow
(>255 for BYTE, >65535 for WORD) is also an error.

### 5.3 Negative array size ✓

**File:** `pkg/plz/check.go:310-318`, `pkg/plz/ast.go:246-259`

~~`DECLARE arr ARRAY [-5] BYTE` parses without error. `StorageSize()` returns
a negative total.~~
**Status:** ✓ Fixed. Parser rejects negative array sizes at both parse sites.

---

## Additional Issues

### 6.1 Constants exceed Z80 range

**File:** `pkg/plz/check.go:89-221` (EvalConstExpr)

Constants are evaluated in Go `int` (64-bit). A value > 65535 emits
`ld hl, 100000` and the assembler silently truncates.

### 6.2 FOR loop step/end stack values never popped ✓

**File:** `pkg/plz/gen.go` (FOR loop, Group.Gen)

~~`FOR` pushes step and end onto the Z80 stack. On normal loop exit these
remain on the stack. Stack depth grows by 2 per nested FOR level.~~
**Status:** ✓ Fixed. FOR loop uses RAM temp variables (`_for_step_N`,
`_for_end_N`) instead of the Z80 stack. No stack leak.

### 6.3 isSimpleOperand assumptions

**File:** `pkg/plz/gen.go:770-801` (isSimpleOperand)

`isSimpleOperand` returns true for any `Reference()` without checking for
subscripts or field accesses. The optimization path for comparisons may
clobber HL.

### 6.4 Global variable name collision ✓

**File:** `pkg/plz/gen.go:116-125` (localSym)

~~Global variables use source-level names as assembly labels. A user variable
named `sp`, `hl`, `af`, etc. would confuse the assembler.~~
**Status:** ✓ Fixed. Checker rejects global declarations using Z80 register
names (`af`, `bc`, `de`, `hl`, `sp`, `pc`, `ix`, `iy`, `a`–`r`).

---

## Spec Design Issues

### 7.1 FOR loop only works for increasing ranges

**File:** `pkg/plz/gen.go` (FOR loop)

`FOR i = start TO end` compares with unsigned `end < var` (carry flag). A
negative step or decreasing range exits immediately.

### 7.2 No ELSEIF

**File:** language syntax

No `ELSEIF` or `ELSIF` keyword. Deeply nested IF/ELSE is verbose.

### 7.3 CONSTANT without value is a no-op ✓

**File:** `pkg/plz/check.go:415-436`

~~`CONSTANT name` (no expression) parses and checks without error but creates
no constant entry. Subsequent references produce "undefined constant" errors
at a confusing distance from the declaration.~~
**Status:** ✓ Fixed. `Constant.Check` rejects `CONSTANT name` when no value
expression is provided.

### 7.4 OUTPUT port must be a literal integer

**File:** `pkg/plz/parser.go:752-773`

`OUTPUT port expr` requires `port` to be a `TokenInt` literal. No runtime-
computed port addresses.

### 7.5 CASE only supports one value per OF branch

**File:** `pkg/plz/ast.go` (CaseBranch), parser

`OF 1, 2, 3 stmt` is not supported despite `CaseBranch.Values` being a
slice.

### 7.6 DEFINE silently overwrites type aliases ✓

**File:** `pkg/plz/check.go` (no duplicate check)

~~`DEFINE foo BYTE` followed by `DEFINE foo WORD` silently overwrites without
warning.~~
**Status:** ✓ Fixed. `Define.Parse` rejects duplicate type alias names.

---

## Minor

### 8.1 TokenFloat defined but unused

**File:** `pkg/plz/token.go`

`TokenFloat` is defined but the scanner never emits it directly. It is kept
as a placeholder because removing it would shift the iota values and break
alignment with Go's `text/scanner` token constants which are matched by
value.

### 8.2 DATA saveSize fragile for mixed text/numeric

**File:** `pkg/plz/gen.go:2534-2550` (saveSize)

`saveSize` adds 1 for numeric values and `len(t.Value)` for TEXT. Works but
fragile with mixed data layouts in DATA blocks.

**Status:** ✓ Fixed. Added handling for tile data (`len(tile.Bytes())`) and
restructured to handle all DATA variants (tile, text, values) explicitly.
