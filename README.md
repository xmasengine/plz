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
- **No semicolons required** — statements end at keywords or newlines; `;` is allowed
- **SWITCH statement** with proper CASE blocks and default handling
- **Up to 2 labels per statement** (one named, one numeric)

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

Up to 16 cooperative tasks with static priority. Task Control Blocks at
`_plz_tcbs` (8 bytes each: SP, state, sleep counter, priority).

States: READY (0), SUSPENDED (1), SLEEPING (2), DEAD (3)

Primitives: `SLEEP n`, `YIELD`, `SUSPEND name`, `RESUME name`

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
