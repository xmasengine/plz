# PLZ Compiler Suite — Design Review

## Priority 1 — Bugs

### 1.1 Scheduler ignores PRIORITY

**Location:** `pkg/plz/gen.go` lines 351-374, SchedulerCode lines 2428-2530

The scheduler does pure round-robin — starts from `current_task+1` and selects the first READY task. It never reads or compares the PRIORITY field at TCB offset +4.

**Impact:** Tasks with higher priority are not preferred.

### 1.2 Scheduler wakes SUSPENDED tasks

**Location:** SchedulerCode (`gen.go` lines 2448-2458)

The sleep-counter decrement loop sets state to READY when counter reaches 0, regardless of current state. A SUSPENDED task (state=1) with a stale sleep counter gets incorrectly woken.

**Fix:** Only wake tasks whose state is SLEEPING (2), not SUSPENDED (1) or DEAD (3).

### 1.3 GOTO label not emitted

**Location:** `pkg/plz/gen.go` line 404 (already fixed)

The default case in `Program.Gen` called `cmd.(interface{ Gen }).Gen(g)` directly, bypassing `Statement.Gen` which emits the label. GOTO targets at top level were silently dropped.

**Status:** Fixed in this session.

### 1.4 Task-local DECLARE emitted inline

**Location:** `pkg/plz/gen.go` lines 428-440 (already fixed)

`DECLARE` inside a task body called `Declare.Gen(g)` with `g.procName=""`, emitting `org 0xC000` inline — splitting the task code in half.

**Status:** Fixed in this session.

## Priority 2 — Missing Semantic Checks

### 2.1 RETURN at global scope

`RETURN` has no `Check` method. A `RETURN` at global scope passes semantic analysis and emits a `ret` in the middle of the main body. Should be rejected.

### 2.2 GOTO target not validated

`GOTO` has no `Check` method. A jump to a non-existent label compiles but fails at assembly time. The checker should validate that the target label exists.

### 2.3 Duplicate INTERRUPT installs

Installing the same interrupt vector twice (`INTERRUPT foo` followed by `INTERRUPT foo`) is not diagnosed.

### 2.4 Array bounds not checked on LHS of LET

Subscript bounds checking in `check.go` only applies to RHS references. `LET arr[out_of_bounds] = x` is not caught at compile time.

## Priority 3 — Spec vs Implementation Discrepancies

### 3.1 Priority scheduling not implemented

Spec says scheduler picks "the first READY task with the highest priority." Implementation is pure round-robin.

### 3.2 RETURN multi-value

Spec shows `RETURN [expr [, ...]]` but the code generator only handles single return values. Multi-value return is syntactically parsed but silently drops extra values.

### 3.3 PRAGMA BOUNDCHECK undocumented

`PRAGMA BOUNDCHECK` works in code but is not mentioned in AGENTS.md. Also there's no way to disable it (`PRAGMA NOBOUNDCHECK`).

## Priority 4 — Spec Design Issues

### 4.1 FOR loop only works for increasing ranges

`FOR i = start TO end [BY step]` always compares with unsigned `end < var` (carry flag). A negative step or decreasing range (`FOR i = 10 TO 0`) exits immediately.

### 4.2 No ELSEIF

Deeply nested IF/ELSE is verbose. No `ELSEIF` or `ELSIF` keyword.

### 4.3 CONSTANT without value is a no-op

`CONSTANT name` (no expression) parses but creates no entry. Subsequent references produce "undefined constant" errors.

### 4.4 OUTPUT port must be a literal integer

`OUTPUT port expr` requires `port` to be a `TokenInt` literal. No runtime-computed port addresses.

### 4.5 CASE only supports one value per OF branch

`OF 1, 2, 3 stmt` is not supported despite `CaseBranch.Values` being a slice.

### 4.6 DEFINE silently overwrites type aliases

`DEFINE foo BYTE` followed by `DEFINE foo WORD` silently overwrites without warning.

## Priority 5 — Minor

### 5.1 TokenFloat defined but unused

`TokenFloat` in `token.go` matches Go scanner token kinds but the scanner never emits it.

### 5.2 DATA size calculation for mixed text/numeric

`saveSize` in `gen.go:2251` adds 1 for numeric values and `len(t.Value)` for text. Works but fragile with mixed data layouts.

### 5.3 No bounds check toggle

`PRAGMA BOUNDCHECK` is permanent once set. No `PRAGMA NOBOUNDCHECK`.
