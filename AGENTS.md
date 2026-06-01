# AGENTS.md — PLZ Compiler Suite

## Project Overview

PLZ is a PL/M-inspired compiler suite for the Z80 CPU, written in Go. It
provides a complete toolchain including a PL/Z compiler, Z80 assembler, and Z80 emulator for testing.

## Repository Structure

```
cmd/plz/plz.go           — Main CLI binary (flags: -A assembler, -C compiler, -E emulator, -R run)
cmd/genins/genins.go     — Generates Z80 instruction table (OpInfo variant)
cmd/gensym/gensym.go     — Generates Z80 instruction table (OpInfo + Asm interface variant)
pkg/plz/                 — PL/Z compiler frontend
  token.go               — Lexer/scanner & token definitions
  ast.go                 — AST node definitions
  parser.go              — Pratt/recursive-descent parser
  check.go               — Semantic checker (symbol table, scope, type validation)
  gen.go                 — Code generator (PL/Z → Z80 assembly)
  compiler.go            — Top-level Compile() orchestration
pkg/z80asm/              — Z80 assembler backend (2-pass)
  assembler.go           — Main assembler engine
  tables.go              — Opcode encoding tables
  expressions.go         — Expression evaluator
  asm_parse.go           — Argument parsing
pkg/z80/emu/             — Z80 emulator wrapper (for testing)
pkg/z80/isa/             — Z80 instruction set definitions
test/plz_test.go         — Integration tests (compile→assemble→emulate→assert)
asm/                     — Raw assembly examples
example/                 — PL/Z language example programs
```

## Build & Test

```sh
go build ./cmd/plz          # Build the plz CLI
go test ./...               # Run all tests
```

## Key Architecture

- **Three interfaces** drive the compiler: `Parselet` (parse), `Checklet` (semantic check), `Genlet` (code gen). Every AST node may implement all three, the `Checklet` is optional.
- **Pipeline:** PL/Z source → Scanner (tokens) → Parser (AST) → Checker (validated AST) → Gen (Z80 assembly text) → Assembler (binary) → optionally Emulator
- **Parser:** Recursive-descent with Pratt parsing for expressions. Each AST node self-parses via `Parse(*Parser) error`.
- **Checker:** Two-pass — pass 1 collects declarations/signatures, pass 2 validates bodies.
- **Code Generator:** Translates AST to Z80 assembly text. Generates runtime helpers for mul/div/mod/comparison, procedure frames, and task scheduler.
- **Memory Model:** Code at 0x0000, stack at 0xDFF0, heap/data at 0xC000. Procedures use static frame allocation or stack-based (REENTRANT). HL = first return/param, DE = second param.
- **Task System:** Up to 16 cooperative tasks with static priority. TCB at `_plz_tcbs` (8 bytes each: SP, state, sleep counter, priority). Primitives: SLEEP, YIELD, SUSPEND, RESUME.

## Conventions

- All keywords in UPPERCASE (BYTE, WORD, PROCEDURE, LET, IF, WHILE, DO, etc.)
- Statements end at newline or `;` (semicolons optional)
- Labels: one named label per statement (`loop:`) as GOTO targets
- Comments: `//` line comments only
- Generated assembly uses lowercase mnemonics
- Runtime symbols use `_plz_` prefix
- Local labels use numeric IDs (`_while_1:`, `_if_2:`, `_end_3:`)
- Tests use `compileAndRun` helper: PL/Z source string → full pipeline → emulator output verification

## Important Instructions

- The assembler uses many singe letter identifiers as register names, including i and r.
- The assembler provides a `jmp` pseudo-instruction that emits `jr`+`nop` (3 bytes) when the relative offset is within ±127 bytes, and `jp` (3 bytes) otherwise. Use `jmp` instead of `jp` for control flow jumps (IF/WHILE/FOR/CASE) so the assembler picks `jr` for small bodies and `jp` when the range overflows.
- The assembler supports bank switching via three directives: `banksize <bytes>` (set ROM bank size, default 0x4000), `bankat <addr>` (set CPU window address, default 0x4000), and `bank <nr>` (switch to bank `nr`, setting PC to bankAt and target to bankAt + nr*banksize).

## Task System Details

### Architecture

At boot the scheduler initialises the TCBs, pushes each task's entry
address onto its dedicated 128-byte stack, then RETs into task 0.

Each TCB is 8 bytes:

| Offset | Size | Field       | Description                                  |
|--------|------|-------------|----------------------------------------------|
| 0      | 2    | SP          | Saved stack pointer when task not running    |
| 2      | 1    | State       | 0=READY, 1=SUSPENDED, 2=SLEEPING, 3=DEAD    |
| 3      | 1    | Sleep cnt   | Remaining ticks before wake-up               |
| 4      | 1    | Priority    | 0=highest, 15=lowest                         |
| 5      | 3    | (reserved)  |                                              |

The scheduler (`_plz_scheduler`) runs when a task calls SLEEP, YIELD,
HALT, or an interrupt re-enters it:

1. Save current task's SP into its TCB.
2. Decrement all sleeping tasks' sleep counters; when one reaches 0,
   set its state to READY.
3. Round-robin scan starting from the next slot for the first READY
   task with the highest priority.
4. If found, restore its SP and RET into it.
5. If none, HALT (CPU sleeps until an interrupt re-enters the scheduler).

### Timing

SLEEP advances the counter only when the scheduler runs, which happens
on three events:

| Event | Real-time? | Mechanism |
|-------|-----------|-----------|
| **Interrupt** (VBlank) | Yes — 60/50 Hz | `INTERRUPT PROCEDURE tick() CALL _plz_scheduler END` + `ENABLE` + VDP config |
| **Timer task** | No — CPU speed | A low-priority YIELD-spinning task: `TASK t PRIORITY 1 WHILE 1 DO YIELD END END` |
| **Busy-wait** | Approx | A `WHILE i < n DO i = i + 1 END` loop — no scheduler needed |

## Language Features

| Feature | Syntax |
|---------|--------|
| Declare | `DECLARE name type` |
| Assign | `LET ref = expr` |
| If | `IF expr THEN stmt [ELSE stmt]` |
| While | `WHILE expr [DO] stmts END` |
| For | `FOR var = start TO end [BY step] [DO] stmts END` |
| Do | `DO stmts END` |
| Case | `CASE expr [DO] { OF val stmt } [OF DEFAULT stmt] END` |
| Procedure | `PROCEDURE name(params) type [REENTRANT] [INTERRUPT\|NMI] stmts END` |
| Return | `RETURN [expr [, ...]]` |
| Call | `CALL name(args)` |
| Goto | `GOTO name` |
| At | `AT literal` (set data address) |
| At Bank | `AT BANK nr` (switch to ROM bank `nr` for subsequent code/data) |
| Bank | `BANK expr` (runtime bank switch via Sega mapper port 0xFFFD) |
| Interrupt/NMI install | `INTERRUPT name` or `NMI name` (install handler at vector) |
| Procedure modifier | `PROCEDURE name() INTERRUPT` or `PROCEDURE name() NMI` (define handler) |
| At suffix | `DECLARE name type AT literal` (declare at address) |
| Output | `OUTPUT port expr` |
| Input | `INPUT(port)` (as expression) |
| Data | `name: DATA vals` |
| Data Tile | `name: DATA TILE \`...\`` (8x8 SMS tile from backtick string, chars: `.`=pal0, `0-9`=pal0-9, `A-F`=pal10-15) |
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

## Operators (precedence low→high)

`==` `>` `<` `>=` `<=` `!=` | `+` `-` `<<` `>>` | `/` `%` `*` | `|` `^` `&` | unary `!` `-` | `.` (field) | `[]` (index) | `()` (call)

## Testing Approach

Integration tests in `test/plz_test.go` embed PL/Z source as Go strings, run the full pipeline, load into the koron-go/z80 emulator, and assert on output port bytes. Unit tests exist per package.

## Dependencies

- `github.com/koron-go/z80` — Z80 CPU emulator (MIT)
- `github.com/paulhankin/z80asm` — original Z80 assembler fork (MIT)
