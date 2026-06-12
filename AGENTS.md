# AGENTS.md — PLZ Compiler Suite

## Project Overview

PLZ is a PL/M-inspired compiler suite for the Z80 CPU and 6502 CPU (NES),
written in Go. It provides a complete toolchain including a PL/Z compiler,
Z80 assembler, Z80/6502 backend code generators, and a Z80 emulator for
testing.

## Repository Structure

```
cmd/plz/plz.go             — Main CLI binary (flags: -A assembler, -C compiler, -E emulator, -R run, -arch, -legacy, -pir)
cmd/genins/genins.go       — Generates Z80 instruction table (OpInfo variant)
cmd/gensym/gensym.go       — Generates Z80 instruction table (OpInfo + Asm interface variant)
pkg/plz/                   — PL/Z compiler frontend
  token.go                 — Lexer/scanner & token definitions
  ast.go                   — AST node definitions
  parser.go                — Pratt/recursive-descent parser
  check.go                 — Semantic checker (symbol table, scope, type validation)
  gen.go                   — Legacy code generator (PL/Z → Z80 assembly)
  genpir.go                — PIR code generator (PL/Z → PIR instructions)
  compiler.go              — Top-level Compile() orchestration (selects backend)
  scope.go                 — Scope type with IsLoop field for BREAK/CONTINUE
pkg/pir/                   — PIR intermediate representation & backends
  pir.go                   — PIR core (~500 lines): instruction set, program, data types
  z80.go                   — PIR→Z80 translator (~1333 lines)
  gen6502.go               — PIR→6502 translator (~1462 lines) including NES
  optimize.go              — PIR-level peephole optimiser (~370 lines)
  z80_test.go              — 30+ tests for Z80 backend text output
  gen6502_test.go          — Tests for 6502 backend (shifts, mul/div/mod, tasks)
  optimize_test.go         — 27 optimiser unit tests
pkg/z80asm/                — Z80 assembler backend (2-pass)
  assembler.go             — Main assembler engine
  tables.go                — Opcode encoding tables
  expressions.go           — Expression evaluator
  asm_parse.go             — Argument parsing
pkg/z80/emu/               — Z80 emulator wrapper (for testing)
pkg/z80/isa/               — Z80 instruction set definitions
pkg/sms/                   — SMS VDP integration for pixel-level test assertions
test/                      — Integration tests
  helpers_test.go          — Shared test helpers (compileAndRun, testArchs, compilePIR, RunResult)
  control_flow_test.go     — BREAK, CASE, logical operator tests (testArchs cross-platform)
  cpu6502_test.go          — 6502 PIR backend unit tests (22 tests, all pass)
  nes_test.go              — NES ROM integration tests (4 tests)
  arch_check_test.go       — Arch-specific feature checks
  plz_test.go              — All legacy integration tests (compileAndRun, Z80-only)
asm/                       — Raw assembly examples
example/                   — PL/Z language example programs
include/                   — PL/Z standard library
  music.plz                — PSG sound driver (~200 lines: C3–C6, noise, sustain, loop)
  music_test.plz           — Music demo (compiles via full pipeline)
port/                      — Game porting artifacts
  lox.bas                  — CVBasic source for reference
  lox.plz                  — Partial PLZ port (PRINT_XY, constants, INCLUDE)
doc/                       — Documentation
  ir.md                    — Full PIR design spec with Z80/6502/NES backend docs
```

## Build & Test

```sh
go build ./cmd/plz          # Build the plz CLI
go test -count=1 ./...      # Run all tests (use -count=1 to avoid cache)
```

## CLI Flags

| Flag       | Description |
|-----------|-------------|
| `-C`      | Compile PL/Z source |
| `-A`      | Assemble Z80 source |
| `-E`      | Emulate Z80 binary |
| `-R`      | Run: compile + assemble + emulate |
| `-arch`   | Target architecture: `z80` (default), `6502`, `nes` |
| `-legacy` | Use legacy Gen backend instead of PIR backend |

## Key Architecture

- **Three interfaces** drive the compiler: `Parselet` (parse), `Checklet` (semantic check), `Genlet` (code gen). Every AST node may implement all three; `Checklet` is optional.
- **Pipeline (PIR):** PL/Z source → Scanner → Parser → Checker → GenPIR (PIR instructions) → Optimiser → Backend (Z80/6502/NES assembly) → Assembler → optionally Emulator
- **Pipeline (Legacy):** PL/Z source → Scanner → Parser → Checker → Gen (Z80 assembly) → Assembler → optionally Emulator
- **Parser:** Recursive-descent with Pratt parsing for expressions. Each AST node self-parses via `Parse(*Parser) error`.
- **Checker:** Two-pass — pass 1 collects declarations/signatures, pass 2 validates bodies. `TRUE`/`FALSE` predefined constants registered at init. `Scope.IsLoop` field set when entering WHILE/FOR body for BREAK/CONTINUE validation.
- **Memory Model (Z80):** Code at 0x0000, stack at 0xDFF0, heap/data at 0xC000. Procedures use static frame allocation or stack-based (REENTRANT). HL = first return/param, DE = second param.
- **Memory Model (6502):** Zero-page $00-$01 = data stack pointer, $02-$0B = scratch, $80-$FF = TCBs. Stack at $0100-$01FF (hardware stack, shared by all tasks). Code at $8000, data variables at $6000-$7FFF.
- **Task System (both):** Up to 16 cooperative tasks with static priority. TCBs: each 8 bytes (SP, state, sleep_cnt, priority, reserved). Primitives: SLEEP, YIELD, SUSPEND, RESUME, HALT.
- **PIR:** Stack-based intermediate representation. Each instruction has zero or one operand. `NEXT op TOS` data-stack order (Forth/JVM style). Full spec in `doc/ir.md`.

## PIR Optimiser (`pkg/pir/optimize.go`)

- **Constant folding** for all binary/unary ops (e.g., `PUSH_I 3; PUSH_I 4; ADD_B` → `PUSH_I 7`)
- **Strength reduction:** `MUL_B/W` / `DIV_B/W` by power-of-two → `SHL_B/W` / `SHR_B/W`; `MOD_B/W` by power-of-two → `AND_B/W`
- **Identity elimination:** `ADD 0`, `MUL 1`, `DIV 1`, `SHL 0`, `SHR 0`, `XOR same`, `SUB same`, etc.
- **Dead-instruction compaction:** NOP-mark-and-compact pass
- Called in `compiler.go` for all PIR-based paths (Z80, 6502, NES)

## Conventions

- All keywords in UPPERCASE (BYTE, WORD, PROCEDURE, LET, IF, WHILE, DO, etc.)
- Statements end at newline or `;` (semicolons optional)
- Labels: one named label per statement (`loop:`) as GOTO targets
- Comments: `//` line comments only
- Generated assembly uses lowercase mnemonics
- Runtime symbols use `_plz_` prefix
- Local labels use numeric IDs (`_while_1:`, `_if_2:`, `_end_3:`)
- Tests use `compileAndRun` helper: PL/Z source string → full pipeline → emulator output verification
- PIR instructions use PascalCase (`PushI`, `AddB`, `GoIf`)
- `pkg/pir` has zero dependencies on `pkg/plz`; dependency direction is `pkg/plz` → `pkg/pir`

## Important Instructions (Z80)

- The assembler uses many single-letter identifiers as register names, including `i` and `r`.
- The assembler provides a `jmp` pseudo-instruction that emits `jr`+`nop` (3 bytes) when the relative offset is within ±127 bytes, and `jp` (3 bytes) otherwise. Use `jmp` instead of `jp` for control flow jumps.
- Bank switching: `banksize <bytes>`, `bankat <addr>`, `bank <nr>`.
- Z80 data stack: DE = TOS cache, HL = data stack pointer (grows upward), SP = return stack.
- go6502 assembler quirks: accumulator `asl`/`lsr`/`rol`/`ror` omit `a` operand; no `bge`/`blt`; `#<`/`#>` work with labels and literals; `\t` indent required.

## Language Features

| Feature | Syntax |
|---------|--------|
| Cast | `BYTE(expr)` / `WORD(expr)` (explicit type cast, suppresses overflow errors) |
| Declare | `DECLARE name type` |
| Assign | `LET ref = expr` |
| If (no END) | `IF expr THEN stmt [ELSE stmt]` |
| While | `WHILE expr [DO] stmts END` |
| While (single) | `WHILE expr stmt` (no DO/END) |
| For | `FOR var = start TO end [BY step] [DO] stmts END` |
| For (single) | `FOR var = start TO end stmt` (no DO/END) |
| Do | `DO stmts END` |
| Case | `CASE expr [DO] { OF val stmt } [OF DEFAULT stmt] END` |
| Procedure | `PROCEDURE name(params) type [REENTRANT] [INTERRUPT\|NMI] stmts END` |
| Return | `RETURN [expr [, expr]]` |
| Call | `CALL name(args)` |
| Goto | `GOTO name` |
| Break | `BREAK` (exit innermost WHILE/FOR loop) |
| Continue | `CONTINUE` (skip to next WHILE/FOR iteration) |
| At | `AT literal` (set data address, one-shot) |
| At Bank | `AT BANK nr` (switch to ROM bank `nr`) |
| Bank | `BANK expr` (runtime bank switch via Sega mapper port 0xFFFD) |
| Interrupt install | `INTERRUPT name` or `NMI name` (install handler at vector) |
| Procedure modifier | `PROCEDURE name() INTERRUPT` or `PROCEDURE name() NMI` |
| At suffix | `DECLARE name type AT literal` (declare at address) |
| Output | `OUTPUT port expr` |
| Input | `INPUT(port)` (as expression) |
| Data | `DATA name vals` |
| Data Tile | `DATA name TILE \`...\`` (8x8 SMS tile from backtick string, `.`=pal0, `0-9`=pal0-9, `A-F`=pal10-15) |
| Constant | `CONSTANT name [=] val` |
| Define | `DEFINE name type` (type alias) |
| Task | `TASK name PRIORITY n stmts END` |
| Sleep | `SLEEP expr` |
| Yield | `YIELD` |
| Suspend | `SUSPEND name` |
| Resume | `RESUME name` |
| Halt | `HALT` |
| Enable | `ENABLE` |
| Disable | `DISABLE` |
| Save | `SAVE [AT expr] expr` (save to battery-backed RAM) |
| Load | `LOAD [AT expr] expr` (load from battery-backed RAM) |
| Include | `INCLUDE "filename"` |
| Pragma | `PRAGMA BOUNDCHECK` / `PRAGMA NOBOUNDCHECK` |
| TRUE | Predefined constant = 1 |
| FALSE | Predefined constant = 0 |

## Operators (precedence low→high)

`||` `OR` (50) | `&&` `AND` (80) | `==` `EQ` `>` `GT` `<` `LT` `>=` `GE` `<=` `LE` `!=` `NE` (100) | `+` `PLUS` `-` `MINUS` `<<` `SHL` `>>` `SHR` (120) | `/` `DIV` `%` `MOD` `*` `TIMES` (140) | `|` `BITOR` `^` `XOR` `&` `BITAND` (160) | unary `!` `NOT` `-` (180) | `.` field (200) | `[]` index (220) | `()` call (240)

Logical operators (`AND`, `OR`, `&&`, `||`) normalize to 0/1. Bitwise operators (`BITAND`, `BITOR`, `&`, `|`, `^`, `XOR`) do not. PIR backend uses `AND_B`/`AND_W` + `IS_W NE` normalization for logical ops.

## Task System Details (Z80)

### Architecture

Each TCB is 8 bytes:

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 0 | 2 | SP | Saved stack pointer when task not running |
| 2 | 1 | State | 0=READY, 1=SUSPENDED, 2=SLEEPING, 3=DEAD |
| 3 | 1 | Sleep cnt | Remaining ticks before wake-up |
| 4 | 1 | Priority | 0=highest, 15=lowest |
| 5 | 3 | (reserved) | |

The scheduler saves current SP, decrements sleep counters (waking only SLEEPING tasks), scans READY tasks for lowest priority value (round-robin within priority), restores target SP, and RETs. If none, HALT.

### Timing

SLEEP counters advance when the scheduler runs — triggered by interrupt (VBlank), a YIELD-spinning timer task, or a busy-wait loop.

## Task System Details (6502)

- Zero-page TCBs at $80-$FF (up to 16 tasks, 8 bytes each).
- All tasks share the hardware stack $0100-$01FF; each task gets `256 / TaskLimit` bytes.
- Scheduler: save SP, decrement sleep counters, scan READY tasks, restore SP, RTS.
- Task init emitted after all code (so JOB labels resolve) — init routine is JSR'd at boot.

## NES Target Details

- MMC5 mapper (assumed).
- 16KB PRG ROM, iNES header, CHR-RAM (no CHR-ROM).
- 8KB battery-backed SRAM at $6000–$7FFF, enabled via $5104.
- TILE handling in frontend (genpir.go) emits `DATA_B` instructions for tile bytes (no PIR-level TILE opcode).

## Testing Approach

Most integration tests in `test/` use `compileAndRun` (legacy Z80-only). Newer tests use `testArchs` for cross-platform (Z80/6502/NES) runs via the PIR pipeline. `testArchs` accepts an optional archs parameter to limit which architectures a test runs on. Unit tests exist per package (`pkg/pir`, `pkg/plz`, etc.). All tests must pass (`go test -count=1 ./...`).

## Known PIR Backend Limitations

- **BREAK/CONTINUE only on Z80 PIR**: 6502/NES backends don't implement BREAK/CONTINUE yet.
- **CASE only on Z80 PIR**: 6502/NES backends don't implement CASE yet.
- **Array indexing on variables uses GET_W (reads value as pointer) instead of PUSH_A (pushes address)**: `arr[i]` on a declared array variable computes `*(value_of_arr + i)` instead of `*(address_of_arr + i)`. This causes wrong results when the array's initial value is non-zero. Only works by coincidence when value is 0 (most arrays).
- **DATA refSize returns 1 instead of actual element count**: `SAVE my_data` copies only 1 byte regardless of DATA size.
- **Task system lacks PIR initialization**: `JOB`/`BYE`/`YIELD` emit labels and scheduler calls but no TCB setup or first-task startup.
- **6502 `.org` restriction**: `.org` can only appear before first instruction; `AT` for variable placement is effectively broken.
- **SAVE/LOAD only work on Z80 with legacy `compileAndRun`**: PIR backend's `_plz_save`/`_plz_load` were fixed for endianness and HL corruption but still exposed by frontend bugs (refSize, array indexing).

## Dependencies

- `github.com/koron-go/z80` — Z80 CPU emulator (MIT)
- `github.com/paulhankin/z80asm` — original Z80 assembler fork (MIT)

## Key Design Decisions

- **`IF` without `END` is intentional**: only block constructs with multiple statements (`DO`/`WHILE`/`FOR`/`CASE`) use `END`.
- **`ELSE IF` works without `ELIF` keyword**: ELSE body accepts a single statement, which can be another IF.
- **`TRUE`/`FALSE` are predefined constants** registered in checker global scope, not keywords.
- **BREAK/CONTINUE affect only the innermost loop** — validated via `Scope.IsLoop` field walked by `Checker.inLoop()`.
- **AT is one-shot**: only affects immediately following VAR/DATA/ROUTE/JOB; silently ignored otherwise.
- **Music belongs in the standard library**, not the language — `include/music.plz` implements PSG driver as PLZ procedures and DATA blocks.
- **PIR optimiser is a stack-simulator pass**: uses virtual stack tracking of constants, mark-NOP-and-compact for replacement.
