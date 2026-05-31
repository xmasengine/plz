# plz

PLZ is a PL/M-inspired compiler suite for the Z80 CPU, written in Go.
PL/M is a high-level-syntax, low-level-effect language developed in the
1970–1980 era by Kendall at Intel.

While C is often called a low-level language, it abstracts the machine
and has no direct support for interrupts, chip-level I/O, ROM location,
or memory mapping. In C we rely on undefined behavior or assembly.

In PLZ there is no undefined behavior. Statements like `INPUT`, `OUTPUT`,
and `INTERRUPT` allow true low-level programming without assembly. The
syntax uses easy-to-read full English keywords, resembling BASIC or PL/1
without being as verbose as COBOL or FORTRAN.

## Differences from PL/M

Although PLZ is inspired by PL/M, it differs in several ways:

- **LET required** for assignments to simplify parsing
- **CONSTANT, DATA** are head keywords instead of the convoluted `DECLARE` syntax
- **One declaration per statement** (BYTE, WORD) to simplify the parser
- **Types BYTE and WORD** instead of BYTE and ADDRESS
- **No semicolons required**, statements end at keywords, `;` is allowed
- **SWITCH statement** with proper OF blocks and DEFAULT handling
- **Only one named label per statement** AT is used for locating code precisely.

## Planned Features

- **TASK**: Lightweight cooperative task system (up to 16 statically allocated named
  tasks with SLEEP, SUSPEND, RESUME, YIELD)
- **BANK**: Banked ROM handling
- **SAVE**: Battery-backed RAM handling
- **MUSIC/SCREEN/TILE/SPRITE**: Game-oriented library support via the TASK system

## EBNF Grammar

```ebnf
program      = { statement } .

statement    = [ label ] command [ ";" ] .

command      = let | if | while | for | do | case
             | procedure | call | return | goto | output
             | declare | constant | data | define
             | task | suspend | resume | sleep | yield
             | enable | disable | halt | at .

label        = identifier , ":" .

(* -- Statements -- *)

let          = "LET" reference "=" expression .

if           = "IF" expression "THEN" statement [ "ELSE" statement ] .

while        = "WHILE" expression [ "DO" ] { statement } "END" .

for          = "FOR" reference "=" expression "TO" expression
               [ "BY" expression ] [ "DO" ] { statement } "END" .

do           = "DO" { statement } "END" .

case         = "CASE" expression [ "DO" ]
               { "OF" ( numeric | "DEFAULT" ) statement } "END" .

procedure    = [ "INTERRUPT" | "NMI" ] "PROCEDURE" identifier
               [ "(" parameter { "," parameter } ")" ]
               [ type ] [ "REENTRANT" ]
               { statement } "END" .

parameter    = identifier type .

call         = "CALL" identifier [ "(" [ expression { "," expression } ] ")" ] .

return       = "RETURN" [ expression { "," expression } ] .

goto         = "GOTO" identifier .

at           = "AT" literal .

output       = "OUTPUT" numeric expression .

declare      = "DECLARE" identifier [ "ARRAY" [ "OF" ] [ "[" numeric "]" ] ] type
               [ "=" literal | "AT" literal ] .

constant     = "CONSTANT" identifier [ "=" ] literal .

data         = "DATA" literal { "," literal } .

define       = "DEFINE" identifier type .

task         = "TASK" identifier [ "PRIORITY" numeric ]
               { statement } "END" .

suspend      = "SUSPEND" identifier .

resume       = "RESUME" identifier .

sleep        = "SLEEP" expression .

yield        = "YIELD" .

enable       = "ENABLE" .

disable      = "DISABLE" .

halt         = "HALT" .

(* -- Types -- *)

type         = "BYTE" | "WORD" | "DATA"
             | "RECORD" field { "," field } "END"
             | "ARRAY" [ "[" numeric "]" ] type
             | "TYPE" identifier .

field        = identifier type .

(* -- References (l-values) -- *)

reference    = identifier { "[" expression "]" | "." identifier } .

(* -- Expressions (Pratt parser, precedence low → high) -- *)

expression   = comparison { ("==" | ">" | "<" | ">=" | "<=" | "!=") comparison }
             | sum { ("+" | "-" | "<<" | ">>") sum }
             | product { ("/" | "%" | "*") product }
             | bitwise { ("|" | "^" | "&") bitwise }
             | unary ( "!" | "-" ) unary
             | primary { "." identifier | "[" expression "]" | "(" [ expression { "," expression } ] ")" } .

primary      = numeric | string | char | identifier | "(" expression ")"
             | "INPUT" "(" expression ")" .

(* -- Literals -- *)

literal      = numeric | string | identifier .
```

## CLI Usage

```
plz [flags] [source files]

Flags:
  -A        Assembler mode (assemble .asm → binary)
  -C        Compiler mode (default: compile .plz → assemble → binary)
  -E        Emulator mode (run binary in Z80 emulator)
  -R        Run mode (compile + emulate in one step)
  -h        Display help
  -o file   Output file name
  -f fmt    Output format: bin (default) or sms
  -i file   Input file for emulation
  -p port   Output port for emulation
  -q port   Input port for emulation
  -d dir    Include directories (colon-separated)
  -t dur    Emulation timeout (e.g., 5s)
```

## Pipeline

```
PL/Z source → Scanner (tokens) → Parser (AST)
  → Checker (validated AST) → Gen (Z80 assembly text)
  → Assembler (binary .bin / .sms ROM)
  → Emulator (execute & verify)
```

## Memory Model

- Code at `0x0000` (boot vector, interrupt handlers, main entry)
- Stack at `0xDFF0` (near top of 64 KB)
- Heap/data at `0xC000`
- Procedures use static frame allocation (default) or stack-based (`REENTRANT`)
- Register convention: HL = first return value/parameter, DE = second parameter

## Task System

Up to 16 cooperative tasks with static priority, declared with
`TASK name PRIORITY n stmts END` where lower numbers = higher priority.

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
HALT, or the main loop calls it from an interrupt:

1. Save current task's SP into its TCB.
2. Decrement all sleeping tasks' sleep counters; when one reaches 0,
   set its state to READY.
3. Round-robin scan starting from the next slot for the first READY
   task with the highest priority.
4. If found, restore its SP and RET into it.
5. If none, HALT (CPU sleeps until an interrupt re-enters the scheduler).

### States

- **READY (0)** — Task is eligible to run.
- **SUSPENDED (1)** — Task is paused by another task via `SUSPEND name`.
  Only `RESUME name` can transition it back to READY.
- **SLEEPING (2)** — Task voluntarily slept via `SLEEP expr`. The
  scheduler decrements the sleep counter each time it runs; SLEEP N
  requires N scheduler invocations to wake up.
- **DEAD (3)** — Task has returned from its body or hit `HALT`.
  Dead task slots are skipped by the scheduler.

### Timing

SLEEP advances the counter only when the scheduler runs, which happens
on three events:

| Event | Real-time? | Mechanism |
|-------|-----------|-----------|
| **Interrupt** (VBlank) | Yes — 60/50 Hz | `INTERRUPT PROCEDURE tick() CALL _plz_scheduler END` + `ENABLE` + VDP VBlank config |
| **Timer task** | No — CPU speed | A low-priority YIELD-spinning task: `TASK t PRIORITY 1 WHILE 1 DO YIELD END END` |
| **Busy-wait** | Approx | A `WHILE i < n DO i = i + 1 END` loop — no scheduler needed, works in basic emulator |

Without at least one of these mechanisms, SLEEP with a single task
will HALT permanently (the sleeping task never wakes).

### Example: interrupt-driven SLEEP

```plz
// VBlank handler (placed at 0x0038 by the INTERRUPT keyword)
INTERRUPT PROCEDURE tick()
  CALL _plz_scheduler
END

// In the main body:
ENABLE                             // EI
OUTPUT 0xBF 0xE0                  // VDP reg 1: display + VBlank enable
OUTPUT 0xBF 0x81                  // select reg 1

// Now SLEEP times are real-time:
SLEEP 60                          // ~1 second on NTSC (60 Hz)
```

### Generated symbols

| Symbol | Purpose |
|--------|---------|
| `_plz_tcbs` | TCB array (128 bytes = 16 tasks × 8) |
| `_plz_current_task` | Currently running task index (0-15) |
| `_plz_scheduler` | Scheduler entry point |
| `_plz_task_done` | Task exit handler (marks task DEAD, re-enters scheduler) |
| `_plz_task_N_stack` | 128-byte dedicated stack for task N |
| `_plz_task_N` | Entry label for task N's body |

## Credits

PLZ uses [koron-go/z80](https://github.com/koron-go/z80) as the Z80 emulator
for testing, under the MIT License.

The Z80 assembler is based on [paulhankin/z80asm](https://github.com/paulhankin/z80asm),
forked and modified under the MIT License.

## License

MIT License

Copyright (c) 2026 xmasengine

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
