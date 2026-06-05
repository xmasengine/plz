# PL/Z — The PL/Z Programming Language

This document describes the PL/Z programming language, a PL/M-inspired systems
programming language for the Z80 CPU. PL/Z is designed for writing ROM-based
software for 8-bit systems, particularly the Sega Master System.

PL/Z programs are compiled to Z80 assembly, assembled to binary, and may be
run on a Z80 emulator or real hardware.

---

## 1. A Tutorial Introduction

### 1.1 Getting Started

The classic first program in PL/Z:

```
OUTPUT 0 42
```

This writes the value 42 to I/O port 0. Every PL/Z program is a sequence of
statements executed in order. The language has no required `main` wrapper;
execution begins at the first statement and continues until the program halts.

A program that loops forever:

```
DO
  OUTPUT 0 42
END
```

### 1.2 Variables and Arithmetic

Variables must be declared before use:

```
DECLARE i BYTE
DECLARE sum WORD

LET i = 10
LET sum = 0

DO
  LET sum = sum + i
  LET i = i - 1
END
WHILE i > 0

OUTPUT 0 sum
```

Two basic types exist: `BYTE` (8-bit unsigned, 0–255) and `WORD` (16-bit
unsigned, 0–65535). Expressions mix them freely; BYTE values are widened to
WORD automatically.

### 1.3 Control Flow

PL/Z provides `IF`, `WHILE`, `FOR`, `DO`, and `CASE`:

```
IF x = 0 THEN
  OUTPUT 0 1
ELSE
  OUTPUT 0 2
END

WHILE i < 10 DO
  OUTPUT 0 i
  LET i = i + 1
END

FOR i = 1 TO 5 BY 2 DO
  OUTPUT 0 i
END
```

Statements end at newline or semicolon; semicolons are optional.
`DO` ... `END` groups multiple statements. `THEN` and `DO` are optional
in `IF` and `WHILE` respectively when followed by a single statement.

### 1.4 Procedures

A procedure computes a value or performs an action:

```
PROCEDURE add(a BYTE, b BYTE) WORD
  LET a = a + b
  RETURN a
END

CALL result = add(3, 4)
OUTPUT 0 result
```

Procedures without return values omit the return type. Parameters and return
values pass through the Z80 registers: HL for the first value, DE for the
second.

### 1.5 The Task System

PL/Z supports cooperative multitasking with up to 16 tasks:

```
TASK worker PRIORITY 3
  DO
    OUTPUT 0 1
    YIELD
  END
END

TASK main PRIORITY 0
  DO
    OUTPUT 0 2
    YIELD
  END
END
```

The scheduler runs when a task calls `YIELD`, `SLEEP`, or `HALT`. Tasks may
`SUSPEND` and `RESUME` each other.

---

## 2. Types, Operators, and Expressions

### 2.1 Type System

PL/Z has a small set of types:

| Type    | Size   | Range        | Description              |
|---------|--------|--------------|--------------------------|
| `BYTE`  | 1 byte | 0–255        | Unsigned 8-bit integer   |
| `WORD`  | 2 bytes| 0–65535      | Unsigned 16-bit integer  |
| `DATA`  | 2 bytes| (reference)  | Reference to data block  |
| `LABEL` | 2 bytes| (address)    | Code label reference     |
| `CONSTANT` | —  | (compile)    | Compile-time constant    |

**Array** types are declared with `ARRAY`:

```
DECLARE vec ARRAY [10] BYTE
DECLARE buf ARRAY OF WORD
```

Unbounded arrays (no size) are valid for parameters and DATA references.

**Record** types group fields:

```
DECLARE point RECORD
  x BYTE
  y BYTE
END

LET point.x = 10
LET point.y = 20
```

The built-in `TEXT` alias provides:

```
DEFINE TEXT RECORD length BYTE, text ARRAY BYTE END
```

**Type aliases** use `DEFINE`:

```
DEFINE counter BYTE
DECLARE ticks TYPE counter
```

### 2.2 Declarations

```
DECLARE name type               // simple variable
DECLARE name ARRAY [n] type     // array
DECLARE name type = expr        // with initializer
DECLARE name type AT addr       // at absolute address (memory-mapped I/O, DMA)
DECLARE name TYPE alias         // via type alias
```

`AT` places a variable at a fixed memory address for memory-mapped I/O or
direct memory access. It provides no memory safety guarantees — overlapping
or invalid addresses are not detected. Only the label is emitted (no data
bytes), so code in the target region is not overwritten. `AT` and an
initializer (`= expr`) are mutually exclusive.

### 2.3 Constants

```
CONSTANT max = 255
CONSTANT debug ON
```

Constants are resolved at compile time and have no runtime storage.

### 2.4 Operators

Operators are listed from lowest to highest precedence:

| Precedence | Operators       | Description       |
|------------|-----------------|-------------------|
| 1          | `==` `!=` `<` `>` `<=` `>=` | Comparison |
| 2          | `+` `-` `<<` `>>`           | Additive, shift |
| 3          | `*` `/` `%`                 | Multiplicative |
| 4          | `\|` `^` `&`                | Bitwise OR, XOR, AND |
| 5          | `!` `-` (unary)             | Logical NOT, negate |
| 6          | `.` `[]` `()`               | Field, index, call |

All binary operators are left-associative. Comparisons yield 1 (true) or
0 (false). There is no boolean type; zero is false, non-zero is true.

### 2.5 Expressions

```
42                      // numeric literal
'H'                     // character literal
"hello"                 // string literal (address of data)
x                       // variable reference
x + 1                   // arithmetic
arr[i]                  // array indexing
rec.field               // record field access
f(a, b)                 // procedure call
INPUT(port)             // port input in expression
LENGTH(x)               // compile-time element count
```

### 2.6 Type Conversions

BYTE values are implicitly widened to WORD when used in WORD context.
There are no implicit narrowing conversions. Explicit conversion is
done through assignment to a BYTE variable, which truncates.

---

## 3. Control Flow

### 3.1 Statements and Blocks

A statement is a command optionally preceded by a label. Multiple statements
are grouped with `DO` ... `END`:

```
DO
  stmt1
  stmt2
END
```

Semicolons between statements are optional; newlines serve as separators.

### 3.2 IF

```
IF expr THEN
  statement
ELSE
  statement
END
```

The `THEN` keyword is optional if the statement follows on the next line.
The `ELSE` clause is optional.

### 3.3 WHILE

```
WHILE expr DO
  statements
END
```

Tests the expression before each iteration; zero is false, non-zero is true.

### 3.4 FOR

```
FOR var = start TO end BY step DO
  statements
END
```

The variable is assigned `start`, then incremented by `step` (default 1)
each iteration until it exceeds `end`. The loop variable must be declared
before the `FOR`.

### 3.5 DO ... END (grouping)

```
DO
  statements
END
```

Groups multiple statements into a single compound statement. The block
executes exactly once, introducing a new scope for local declarations.

### 3.6 CASE

```
CASE expr DO
  OF 1  OUTPUT 0 'A'
  OF 2  OUTPUT 0 'B'
  OF 3, 4, 5  OUTPUT 0 'C'
  OF DEFAULT OUTPUT 0 '?'
END
```

Multiple values per branch are separated by commas. The `DEFAULT` branch
matches any value not covered by prior branches.

### 3.7 GOTO

```
loop:
  OUTPUT 0 1
GOTO loop
```

Labels are identifiers followed by a colon. A `GOTO` transfers control to
the labeled statement.

### 3.8 RETURN

```
RETURN              // no value
RETURN expr         // single value
RETURN a, b         // two values
```

Terminates procedure execution and returns control to the caller.

### 3.9 HALT

```
HALT
```

Stops the CPU until the next interrupt. In a task context, marks the task as
DEAD and reschedules.

### 3.10 PRAGMA

```
PRAGMA name [name ...]
```

Compiler directive. Pragmas may be ignored by conforming compilers; PL/Z
recognises:

| Pragma        | Effect                                              |
|---------------|-----------------------------------------------------|
| BOUNDCHECK    | Enable runtime array bounds checking                |

When `PRAGMA BOUNDCHECK` is active, every array access emits code that
compares the index against the declared array size and halts the CPU
(`HALT`) if the index is out of range. The check covers both reads
(`LET x = arr[i]`) and writes (`LET arr[i] = x`). Non-constant indices
that cannot be verified at compile time are checked at runtime.

---

### 3.11 LENGTH

```
LET n = LENGTH(identifier)
```

Returns the number of elements in an array, DATA block, or scalar variable.
For a `DECLARE`d array, returns the element count from the `ARRAY [N]`
declaration. For a `DATA` block, returns the number of values (or tiles
for `DATA TILE`). For a scalar variable, returns 1.

```
DECLARE buf ARRAY [16] BYTE
LET n = LENGTH(buf)    // n = 16

myvec: DATA 10, 20, 30
LET n = LENGTH(myvec)  // n = 3
```

The argument must be a plain identifier — subscripts and field access are
not supported. The result is a compile-time constant usable anywhere an
expression is accepted.

---

## 4. Procedures and Tasks

### 4.1 Procedure Declarations

```
PROCEDURE name (param TYPE, ...) result_type MODIFIERS
  statements
END
```

Modifiers:

| Modifier      | Effect                                       |
|---------------|----------------------------------------------|
| `REENTRANT`   | Use stack frames instead of static storage   |
| `INTERRUPT`   | Handler for maskable interrupts (uses `reti`)|
| `NMI`         | Handler for non-maskable interrupts (uses `retn`)|

Procedures with no return value omit the result type:

```
PROCEDURE init()
  OUTPUT 0 1
END
```

Parameters are passed in registers (HL, DE) and then memory. Record and DATA
parameters are passed by reference (address on stack or in HL).

### 4.2 CALL

```
CALL name(args)
CALL result = name(args)
CALL r1, r2 = name(args)
```

CALL invokes a procedure. If the procedure returns a value, it may be captured.

### 4.3 INTERRUPT and NMI

Interrupt handlers are declared with the `INTERRUPT` or `NMI` modifier:

```
PROCEDURE tick() INTERRUPT
  CALL _plz_scheduler
END

INTERRUPT tick        // install at vector 0x0038
ENABLE                // enable maskable interrupts
```

The `INTERRUPT` statement installs a jump to the handler at the vector address
(0x0038 for maskable, 0x0066 for NMI). `ENABLE` issues the Z80 `EI`
instruction.

### 4.4 Task Declarations

```
TASK name PRIORITY n
  statements
END
```

At boot, all tasks are initialized and the first task (task 0) begins
execution. Priority 0 is highest, 15 is lowest. The scheduler uses
round-robin among the highest-priority ready tasks.

### 4.5 Task Primitives

| Statement          | Effect                                        |
|--------------------|-----------------------------------------------|
| `YIELD`            | Voluntarily yield to the scheduler            |
| `SLEEP expr`       | Sleep for expr ticks, then become ready       |
| `SUSPEND name`     | Suspend the named task                        |
| `RESUME name`      | Resume the named task                         |
| `HALT`             | Mark current task DEAD and reschedule         |

---

## 5. The Task System

### 5.1 Scheduler Architecture

The cooperative scheduler (`_plz_scheduler`) is invoked by `YIELD`, `SLEEP`,
`HALT`, or an interrupt handler. It performs these steps:

1. Save the current task's stack pointer into its TCB.
2. Decrement sleep counters; wake tasks that reach 0.
3. Scan for the highest-priority READY task.
4. Restore its stack pointer and jump to it.
5. If no task is ready, HALT the CPU.

### 5.2 Task Control Block

Each of the 16 tasks has an 8-byte TCB:

| Offset | Size | Field       | Description                    |
|--------|------|-------------|--------------------------------|
| 0      | 2    | SP          | Saved stack pointer            |
| 2      | 1    | State       | 0=READY, 1=SUSP, 2=SLEEP, 3=DEAD |
| 3      | 1    | Sleep count | Ticks remaining                |
| 4      | 1    | Priority    | 0 (highest) to 15 (lowest)     |
| 5      | 3    | Reserved    |                                |

Each task has a dedicated 128-byte stack allocated in RAM.

### 5.3 Timing

The scheduler advances time only when it runs. Three timing mechanisms:

- **VBlank interrupt** (60/50 Hz): `INTERRUPT PROCEDURE tick() CALL _plz_scheduler END`
- **Timer task**: A low-priority task that spins on `YIELD`
- **Busy-wait**: A `WHILE` counting loop (no scheduler involvement)

---

## 6. Input, Output, and Memory

### 6.1 Port I/O

```
OUTPUT port expr         // write byte to port
OUTPUT WORD port expr    // write low byte, then high byte to port
LET x = INPUT(port)      // read byte from port
```

`OUTPUT WORD` is used for 16-bit writes to VDP control ports on the SMS.

### 6.2 Memory Model

| Region  | Address | Description       |
|---------|---------|-------------------|
| Code    | 0x0000  | Program storage   |
| Stack   | 0xDFF0  | Grows downward    |
| Heap    | 0xC000  | Data/BSS section  |

The `AT` directive places variables at a fixed memory address, intended for
memory-mapped I/O registers and direct memory access. No memory safety
guarantees are provided — overlapping or invalid addresses are not detected.

```
DECLARE vdp_control WORD AT 0xBF
```

### 6.3 DATA

Named data blocks:

```
hello: DATA 'H', 'e', 'l', 'l', 'o', 0
tiles: DATA TILE `
  .......X
  ......XX
  .....XXX
  ....XXXX
  ...XXXXX
  ..XXXXXX
  .XXXXXXX
  XXXXXXXX
`
```

`DATA TILE` defines 8x8 pixel tiles for the SMS, where `.` is palette index
0 and `A`–`F` and `0`–`9` specify palette indices 10–15 and 0–9 respectively.

### 6.4 SAVE and LOAD

Battery-backed RAM persistence:

```
DECLARE save_data BYTE AT 0x8000

SAVE save_data          // save to declared AT address
SAVE AT 0x8000 data     // save to arbitrary address
LOAD save_data          // load from declared AT address
LOAD AT 0x8000 data     // load from arbitrary address
```

The transfer size is determined at compile time from the variable's size.

### 6.5 Bank Switching

```
BANK 1                  // switch to ROM bank 1 (port 0xFFFD)
AT BANK 2               // assembler directive: place subsequent code in bank 2
AT 0x8000               // set next data address
```

`BANK` is a runtime switch. `AT BANK` is a compile-time directive for
the assembler's bank switching system.

---

## 7. Program Structure

### 7.1 File Organization

A PL/Z source file is a sequence of statements. There is no required `main`
function. Execution starts at the first statement.

```
// comments are C++ style

INCLUDE "libplz.plz"     // file inclusion

// declarations
DECLARE count BYTE
CONSTANT limit = 100

// code
LET count = 0
DO
  OUTPUT 0 count
  LET count = count + 1
END
WHILE count < limit

HALT
```

### 7.2 INCLUDE

```
INCLUDE "filename"
```

Includes another PL/Z source file at the current position. Paths are resolved
relative to the including file's directory. Includes may be nested.

### 7.3 DEFINE and TYPE

```
DEFINE byte_ptr WORD     // type alias
DEFINE vector ARRAY [10] BYTE

DECLARE buf TYPE vector  // use alias
```

### 7.4 CONSTANT

```
CONSTANT screen_width = 256
CONSTANT debug = 1
```

Constants are substituted at compile time. The value must be a constant
expression.

### 7.5 Labels

Labels mark targets for `GOTO`:

```
start:
  OUTPUT 0 1
GOTO start
```

Branch labels in generated assembly use numeric identifiers
(`_if_1`, `_while_2`, `_for_3`, etc.) and are managed by the compiler.

---

## 8. The Compilation Pipeline

### 8.1 Pipeline Stages

```
PL/Z source
  → Scanner (lexical analysis, tokenization)
  → Parser (recursive descent + Pratt expression parsing)
  → Checker (2-pass semantic analysis)
  → Code Generator (→ Z80 assembly text)
  → Assembler (2-pass, → binary)
  → Emulator (optional, for testing)
```

### 8.2 Runtime Library

The compiler generates runtime helpers for arithmetic and comparison:

| Helper          | Operation                        |
|-----------------|----------------------------------|
| `_plz_mul`      | HL = HL × DE (unsigned)          |
| `_plz_div`      | HL = HL ÷ DE (unsigned); DE=0 → returns 1  |
| `_plz_mod`      | HL = HL % DE (unsigned); DE=0 → returns 0  |
| `_plz_eq`–`_plz_lte` | HL = HL Compare DE          |
| `_plz_scheduler` | Cooperative task scheduler      |

### 8.3 Calling Convention

- First parameter/return: HL
- Second parameter/return: DE
- Additional parameters: static RAM or stack (REENTRANT)
- Record/DATA parameters: by reference (address)

---

## A. Syntax Summary

```
program     = { statement }
statement   = [label ":"] command
command     = declaration
            | assignment
            | if | while | for | do | case | call | return
            | goto | output | input | halt | enable | disable
            | yield | sleep | suspend | resume
            | bank | save | load | data | constant | define
            | include | at | pragma | interrupt_stmt | nmi_stmt
            | procedure | task

declaration = "DECLARE" ident type ["AT" expr] ["=" expr]
assignment  = "LET" ref "=" expr
if          = "IF" expr ["THEN"] command ["ELSE" command]
while       = "WHILE" expr ["DO"] command
for         = "FOR" ref "=" expr "TO" expr ["BY" expr] ["DO"] command
do          = "DO" { statement } "END"
case        = "CASE" expr ["DO"] { "OF" val {"," val} command } ["OF" "DEFAULT" command] "END"
procedure   = "PROCEDURE" ident ["(" params ")"] [type] ["REENTRANT"|"INTERRUPT"|"NMI"] { statement } "END"
task        = "TASK" ident ["PRIORITY" expr] { statement } "END"
call        = "CALL" [ident {"=" ident} "="] ident "(" [expr {"," expr}] ")"
return      = "RETURN" [expr {"," expr}]
goto        = "GOTO" ident
output      = "OUTPUT" ["WORD"] expr expr
data        = ident ":" "DATA" { expr } | ident ":" "DATA" "TILE" "`" ... "`"
save        = "SAVE" ["AT" expr] expr
load        = "LOAD" ["AT" expr] ref
constant    = "CONSTANT" ident ["="] expr
define      = "DEFINE" ident type
include     = "INCLUDE" string
at          = "AT" expr ["BANK" expr]
pragma      = "PRAGMA" ident {ident}
type        = "BYTE" | "WORD" | "DATA" | "LABEL" | "CONSTANT"
            | "ARRAY" ["[" expr "]"] type
            | "RECORD" field {"," field} "END"
            | "TYPE" ident
```

## B. Reserved Words

```
ARRAY     AT        BANK      BY        BYTE      CALL
CASE      CONSTANT  DATA      DECLARE   DEFAULT   DEFINE
DISABLE   DO        ELSE      ENABLE    END       FOR
GOTO      HALT      IF        INCLUDE   INPUT     INTERRUPT  LENGTH
LET       LOAD      NMI       OF        OUTPUT    PRAGMA
PRIORITY  PROCEDURE RECORD    REENTRANT RESUME    RETURN
SAVE      SLEEP     SUSPEND   TASK      THEN      TILE
TO        TYPE      WHILE     WORD      YIELD
```

## C. Operation Precedence

```
1   ==  !=  <  >  <=  >=        comparison
2   +   -   <<  >>              additive, shift
3   *   /   %                   multiplicative
4   |   ^   &                   bitwise
5   !   - (unary)               logical NOT, negation
6   .   []   ()                 field, index, call
```
