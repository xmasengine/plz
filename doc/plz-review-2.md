# PLZ Compiler Suite — Design Review

## Priority 1 — Bugs (✓ All Done)

### 1.1 Scheduler ignores PRIORITY

**Fixed.** The scheduler uses `best_pri`/`best_idx` to pick the READY task with the lowest priority value. When priorities match, the first encountered in scan order wins (round-robin within priority level). SchedulerCode lines 2708-2755.

### 1.2 Scheduler wakes SUSPENDED tasks

**Fixed.** The sleep-decrement loop at line 2699 checks `cp 2` (SLEEPING) and only wakes tasks whose state is 2. SUSPENDED (1) and DEAD (3) tasks are skipped.

### 1.3 GOTO label not emitted

**Fixed.** `Program.Gen` calls `statement.Gen(g)` for all top-level statements, which emits the label before the instruction.

### 1.4 Task-local DECLARE emitted inline

**Fixed.** `Declare.Gen` emits `org 0xC000` only when `g.procName == ""` and there is no current procedure scope (`!g.InTask`).

## Priority 2 — Missing Semantic Checks (✓ Done)

### 2.1 RETURN at global scope

**Fixed.** `Return.Check` (line 733) calls `c.inProcedure()` and rejects RETURN at global scope. Test: `TestCheckReturn`.

### 2.2 GOTO target not validated

**Fixed.** `GoTo.Check` (line 749) validates that the target label exists in `c.Labels`.

### 2.3 Duplicate INTERRUPT installs

**Fixed.** `InterruptStmt.Check` (line 390) tracks `c.usedVectors` and rejects duplicate installs.

### 2.4 Array bounds not checked on LHS of LET

**Fixed.** `checkTarget` (line 631) calls `c.checkArrayBounds(r)` for LET assignment targets. Test: `TestCheckArrayBoundsWriteOutOfRange`.

## Priority 3 — Spec vs Implementation Discrepancies (✓ Done)

### 3.1 Priority scheduling not implemented

**Already implemented.** The scheduler (`SchedulerCode`) uses `best_pri`/`best_idx` to pick the READY task with the lowest priority value. Fixed in commit `67bfaf8`.

### 3.2 RETURN multi-value

**Already implemented.** The code generator emits `ld hl, <val1>` and `ld de, <val2>` for two-value return, and `Let.Gen` stores both into `Target` and `Target2`. Fixed in commit `67bfaf8`.

### 3.3 PRAGMA BOUNDCHECK undocumented

**Already documented.** `PRAGMA BOUNDCHECK` / `PRAGMA NOBOUNDCHECK` is listed in AGENTS.md and supported in parser, checker, and code generator. `NOBOUNDCHECK` toggle exists.

## Priority 4 — Spec Design Issues (all open)

See `doc/plz-todo.md` items 7.1–7.6.

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

## Priority 5 — Minor (open except 5.3)

### 5.1 TokenFloat defined but unused

`TokenFloat` in `token.go` matches Go scanner token kinds but the scanner never emits it.

### 5.2 DATA size calculation for mixed text/numeric

`saveSize` in `gen.go:2251` adds 1 for numeric values and `len(t.Value)` for text. Works but fragile with mixed data layouts.

### 5.3 No bounds check toggle ✓

`PRAGMA BOUNDCHECK` ~~is permanent once set. No `PRAGMA NOBOUNDCHECK`.~~
**Status:** ✓ Fixed. `PRAGMA NOBOUNDCHECK` is supported.
