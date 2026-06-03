# PL/Z Language Specification

## Introduction

This is a formal specification of the PL/Z programming language, a
PL/M-inspired systems language targeting the Z80 CPU. PL/Z programs are
compiled to Z80 assembly and assembled into ROM images for 8-bit systems.

PL/Z is designed for:
- Systems programming on resource-constrained Z80 platforms
- Sega Master System (SMS) ROM development
- Cooperative multitasking with a built-in task scheduler
- Direct hardware access via I/O ports and absolute addresses

The specification uses an extended Backus-Naur Form (EBNF) for grammar rules:

```
Production  = production_name "=" [ Expression ] "." .
Expression  = Alternative { "|" Alternative } .
Alternative = Term { Term } .
Term        = production_name | token [ "…" token ] | Group | Option | Repetition .
Group       = "(" Expression ")" .
Option      = "[" Expression "]" .
Repetition  = "{" Expression "}" .
```

Productions are written with lowercase names. Lexical tokens use uppercase
names or quoted literal strings.

---

## Notation

The syntax is specified using EBNF:

```
|   alternation
()  grouping
[]  optional (zero or one)
{}  repetition (zero or more)
```

Program source code must be UTF-8 text. Line breaks are significant in some
contexts (statements end at newline or semicolon).

---

## Source Code Representation

### Characters

PL/Z source consists of printable ASCII characters, newlines, tabs, spaces,
and carriage returns. All keywords and identifiers use ASCII letters and
digits.

### Comments

Comments serve as documentation and are ignored by the scanner.

```
Comment = "//" { character } newline .
```

There are no block comments.

### Tokens

Tokens form the vocabulary of the language. They are:

```
Token    = keyword | identifier | number | string | character | operator | terminator .
Terminator = ";" | newline .
```

Newlines are token separators. A semicolon may be used in place of a newline.
Multiple statements may appear on one line separated by semicolons.

---

## Lexical Elements

### Identifiers

Identifiers name variables, procedures, tasks, constants, types, data blocks,
and labels.

```
Identifier = letter { letter | digit } .
letter     = "A" … "Z" | "a" … "z" | "_" .
digit      = "0" … "9" .
```

Identifiers are case-sensitive. `count`, `Count`, and `COUNT` are distinct.

### Keywords

Keywords are reserved and may not be used as identifiers.

```
Keyword = "ARRAY" | "AT" | "BANK" | "BY" | "BYTE" | "CALL"
        | "CASE" | "CONSTANT" | "DATA" | "DECLARE" | "DEFAULT"
        | "DEFINE" | "DISABLE" | "DO" | "ELSE" | "ENABLE"
        | "END" | "FOR" | "GOTO" | "HALT" | "IF" | "INCLUDE"
        | "INPUT" | "INTERRUPT" | "LENGTH" | "LET" | "LOAD" | "NMI"
        | "OF" | "OUTPUT" | "PRAGMA" | "PRIORITY" | "PROCEDURE" | "RECORD"
        | "REENTRANT" | "RESUME" | "RETURN" | "SAVE" | "SLEEP"
        | "SUSPEND" | "TASK" | "THEN" | "TILE" | "TO" | "TYPE"
        | "WHILE" | "WORD" | "YIELD" .
```

### Integer Literals

```
NumberLit  = decimal .
decimal    = "0" … "9" { "0" … "9" } .
```

All numbers are decimal. The value must be in the range 0–65535. A number
used as a BYTE is implicitly truncated to 8 bits at assignment.

### Character Literals

```
CharLit = "'" ascii_char "'" .
```

A character literal is a single ASCII character enclosed in single quotes.
Its value is the ASCII code of the character (0–127).

### String Literals

```
StringLit = '"' { ascii_char } '"' .
```

A string literal denotes an anonymous data block containing the ASCII values
of the characters, terminated by a NUL byte (0). The string's address is used
where the literal appears.

```
"hello"    // denotes the 6 bytes: 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x00
```

### Operators and Delimiters

```
Operator = "==" | "!=" | "<" | ">" | "<=" | ">="
         | "+" | "-" | "*" | "/" | "%"
         | "<<" | ">>"
         | "&" | "|" | "^"
         | "!" | "=" | ","
         | "(" | ")" | "[" | "]" | "." .
```

The `=` token appears in both `LET` (assignment) and `CONSTANT` (definition).

---

## Constants

There are two categories of constants:

- **Literal constants**: integer literals, character literals
- **Named constants**: declared with `CONSTANT`

Named constants are compile-time bindings. They have no runtime storage.

```
ConstantDecl = "CONSTANT" Identifier [ "=" ] Expression .
```

The `=` is optional for readability. The expression must evaluate to a
constant at compile time.

```
CONSTANT max = 100
CONSTANT debug 1
CONSTANT greeting "hello"
```

Constants may be used anywhere an integer expression is expected, and in
`DECLARE` array sizes and `AT` addresses.

---

## Types

### Type System

PL/Z has a set of predeclared types and mechanisms for constructing composite
types.

### Predeclared Types

| Type      | Storage | Description                             |
|-----------|---------|-----------------------------------------|
| `BYTE`    | 1 byte  | Unsigned 8-bit integer                  |
| `WORD`    | 2 bytes | Unsigned 16-bit integer, little-endian  |
| `DATA`    | 2 bytes | Reference to a data block (address)     |
| `LABEL`   | 2 bytes | Code label address                      |
| `CONSTANT`| —       | Compile-time constant (no storage)      |

### Array Types

```
ArrayType = "ARRAY" [ "[" Expression "]" ] Type .
```

The expression specifies the element count. If omitted, the array is
unbounded (size unknown at compile time, used for parameters).

Array indexing is zero-based. Element size is determined by the element type:
1 byte for `BYTE`, 2 bytes for `WORD`, and the record size (rounded to next
power of two) for records.

```
DECLARE vec ARRAY [16] BYTE    // 16 bytes
DECLARE buf ARRAY WORD          // unbounded array of WORD
```

### Record Types

```
RecordType = "RECORD" FieldList "END" .
FieldList  = Field { "," Field } .
Field      = Identifier Type .
```

Records group related fields. Fields are laid out sequentially in memory at
offsets determined by the cumulative sizes of preceding fields. The total
record size is rounded up to the next power of two.

```
DECLARE point RECORD x BYTE, y BYTE END
// x at offset 0, y at offset 1, total size 2
```

Fields are accessed with the `.` operator: `point.x`.

### Type Aliases

```
TypeAliasDecl = "DEFINE" Identifier Type .
TypeRef       = "TYPE" Identifier .
```

`DEFINE` creates a type alias. `DECLARE` with `TYPE` uses the alias.

```
DEFINE counter BYTE
DECLARE ticks TYPE counter
```

### Type Properties

Each type has an associated **storage size**:

| Type          | Size                          |
|---------------|-------------------------------|
| `BYTE`        | 1                             |
| `WORD`        | 2                             |
| `DATA`        | 2                             |
| `LABEL`       | 2                             |
| `CONSTANT`    | 0                             |
| `ARRAY [n] T` | n × sizeof(T)                 |
| `RECORD … END`| nextPow2(sum of field sizes)  |

---

## Declarations

### Variable Declarations

```
VariableDecl = "DECLARE" Identifier [ BasedDecl ] Type [ AtClause ] [ Initializer ] .
BasedDecl    = "(" Expression ")" .
AtClause     = "AT" Expression .
Initializer  = "=" Expression .
```

`DECLARE` creates a named variable with a type. If `AT` is specified, the
variable is placed at the given memory address. `AT` is intended for
memory-mapped I/O registers and direct memory access — it provides no
memory safety guarantees and overlapping or invalid addresses are not
detected. Only the label is emitted (no data bytes), so code in the
target region is not overwritten. `AT` and an initializer are mutually
exclusive.

```
DECLARE count BYTE
DECLARE vdp_control WORD AT 0xBF
DECLARE name ARRAY [32] BYTE
DECLARE sum WORD = 0
```

Multiple declarations of the same name in the same scope is an error.

### Array Declarations

Arrays may be declared with or without the `ARRAY` keyword, using the
`DECLARE` syntax:

```
DECLARE vec ARRAY [10] BYTE
DECLARE vec3 ARRAY BYTE
```

### Parameter Declarations

Parameters appear in procedure signatures:

```
ParamList = Parameter { "," Parameter } .
Parameter = Identifier Type .
```

Parameters are BYTE or WORD values, or references to RECORD/DATA/ARRAY types.
Record and DATA parameters are passed by address.

---

## Expressions

### Operands

Operands are the atomic units of expressions:

```
Operand     = Literal | Reference | Call | "(" Expression ")"
            | "INPUT" "(" Expression ")"
            | "LENGTH" "(" Identifier ")" .
Literal     = NumberLit | CharLit | StringLit .
Reference   = Identifier { "[" Expression "]" } { "." Identifier } .
Call        = Identifier "(" [ ExpressionList ] ")" .
```

A `Reference` is an identifier optionally followed by array subscripts and
field selectors.

### Operators

```
Expression = ComparisonExpr .
ComparisonExpr = AdditiveExpr { ("==" | "!=" | "<" | ">" | "<=" | ">=") AdditiveExpr } .
AdditiveExpr   = MultiplicativeExpr { ("+" | "-" | "<<" | ">>") MultiplicativeExpr } .
MultiplicativeExpr = BitwiseExpr { ("*" | "/" | "%") BitwiseExpr } .
BitwiseExpr  = UnaryExpr { ("|" | "^" | "&") UnaryExpr } .
UnaryExpr    = PostfixExpr | "!" UnaryExpr | "-" UnaryExpr .
PostfixExpr  = Operand { "[" Expression "]" | "." Identifier | "(" [ ExpressionList ] ")" } .
```

Operator precedence, highest to lowest:

| Precedence | Operators                | Associativity |
|------------|--------------------------|---------------|
| 6 (highest)| `.` `[]` `()`            | left          |
| 5          | `!` `-` (unary)          | right         |
| 4          | `&` `\|` `^`             | left          |
| 3          | `*` `/` `%`              | left          |
| 2          | `+` `-` `<<` `>>`        | left          |
| 1 (lowest) | `==` `!=` `<` `>` `<=` `>=` | left       |

Comparison operators yield 1 (true) or 0 (false). Comparisons are unsigned.

### Type Widening

When a `BYTE` value appears in a `WORD` context, it is zero-extended to
16 bits. This occurs automatically in arithmetic, comparison, and assignment
to a WORD variable.

### Constant Expressions

A constant expression is an expression that the compiler can evaluate at
compile time. It may contain only literal constants, named constants, and
operators. Constant expressions are required in:

- Array sizes (`ARRAY [n]`)
- AT addresses
- CONSTANT definitions
- CASE branch values
- SLEEP durations

---

## Statements

### Statement List

```
StatementList = { Statement } .
Statement     = [ Label ":" ] Command .
Label         = Identifier .
```

Statements are separated by newlines or semicolons. A label is an identifier
followed by a colon before a command.

### Assignment

```
AssignmentStmt = "LET" Reference "=" Expression .
```

Evaluates the expression and stores the result in the referenced variable,
array element, or record field.

```
LET x = 42
LET arr[i] = x + 1
LET rec.field = 0
```

### IF

```
IfStmt = "IF" Expression [ "THEN" ] Command [ "ELSE" Command ] .
```

If the expression evaluates to non-zero, the first command executes.
Otherwise, if `ELSE` is present, the second command executes.

### WHILE

```
WhileStmt = "WHILE" Expression [ "DO" ] Command .
```

The expression is tested before each iteration. If non-zero, the command
executes and the expression is tested again. If zero, execution continues
after the loop.

### FOR

```
ForStmt = "FOR" Reference "=" Expression "TO" Expression [ "BY" Expression ] [ "DO" ] Command .
```

Assigns the start value to the reference, then iterates while the reference
value does not exceed the end value. After each iteration, the reference is
incremented by the step value (default 1). The reference must be a declared
variable.

### DO

```
DoStmt = "DO" StatementList "END" .
```

Groups a sequence of statements into a single compound statement. The
block executes exactly once and introduces a new scope for local
declarations.

### CASE

```
CaseStmt = "CASE" Expression [ "DO" ] { CaseBranch } [ DefaultBranch ] "END" .
CaseBranch = "OF" CaseValList Command .
CaseValList = CaseVal { "," CaseVal } .
CaseVal     = NumberLit | Identifier .
DefaultBranch = "OF" "DEFAULT" Command .
```

Evaluates the expression and transfers control to the branch whose value
matches. The `DEFAULT` branch matches any unmatched value. Multiple values
per branch are separated by commas.

### CALL

```
CallStmt = "CALL" [ ResultList "=" ] Identifier "(" [ ExpressionList ] ")" .
ResultList = Identifier { "," Identifier } .
```

Invokes a procedure. If the procedure returns values, they may be assigned
to variables.

```
CALL init()
CALL result = add(3, 4)
CALL lo, hi = divmod(a, b)
```

### RETURN

```
ReturnStmt = "RETURN" [ Expression { "," Expression } ] .
```

Returns control from a procedure to the caller. If the procedure declares
a return type, at least one expression must be present. For two return
values, two expressions are provided.

### GOTO

```
GotoStmt = "GOTO" Identifier .
```

Transfers control to the statement with the matching label. The label must
exist within the same procedure or at the top level.

### OUTPUT

```
OutputStmt = "OUTPUT" [ "WORD" ] Expression Expression .
```

Writes the second expression's low byte to the specified I/O port. If `WORD`
is specified, both the low and high bytes are written sequentially to the
same port (low byte first).

### INPUT

```
InputExpr = "INPUT" "(" Expression ")" .
```

Reads a byte from the specified I/O port. Only valid as an expression
operand.

### LENGTH

```
LengthExpr = "LENGTH" "(" Identifier ")" .
```

Returns the number of elements in an array or DATA block at compile time.
For a `DECLARE`d array, returns the declared element count. For a `DATA`
block, returns the number of values (or tiles for `DATA TILE`). For a
scalar variable, returns 1. The argument must be a plain identifier;
subscripts and field access are not supported.

```
DECLARE buf ARRAY [16] BYTE
LET n = LENGTH(buf)    // n = 16

myvec: DATA 10, 20, 30
LET n = LENGTH(myvec)  // n = 3
```

### ENABLE and DISABLE

```
EnableStmt  = "ENABLE" .
DisableStmt = "DISABLE" .
```

`ENABLE` enables maskable interrupts (Z80 `EI`). `DISABLE` disables them
(Z80 `DI`).

### HALT

```
HaltStmt = "HALT" .
```

Stops the CPU until the next interrupt. When used in a task, marks the
current task as DEAD and invokes the scheduler.

### PRAGMA

```
PragmaStmt = "PRAGMA" ident {ident} .
```

A compiler directive. Recognised pragmas:

| Pragma        | Effect                                              |
|---------------|-----------------------------------------------------|
| BOUNDCHECK    | Enable runtime array bounds checking                |

When `PRAGMA BOUNDCHECK` is active, each array access emits code that
compares the index against the declared size of the array and halts the
CPU if the index is out of range. Unrecognised pragmas produce a
compile-time error. Conforming compilers may ignore all pragmas.

### BANK

```
BankStmt = "BANK" Expression .
```

Writes the expression (0–255) to I/O port 0xFFFD, switching the active ROM
bank on the Sega mapper.

### SAVE and LOAD

```
SaveStmt = "SAVE" [ "AT" Expression ] Expression .
LoadStmt = "LOAD" [ "AT" Expression ] Reference .
```

`SAVE` copies data from the source variable to battery-backed RAM. If `AT`
is specified, the destination address is given by the expression; otherwise
the source variable must have been declared with an `AT` address, which is
used as the destination.

`LOAD` copies data from battery-backed RAM into the target variable. The
address rules are the same as `SAVE`.

The transfer size is the compile-time storage size of the source (SAVE) or
target (LOAD) variable.

### SLEEP

```
SleepStmt = "SLEEP" Expression .
```

Suspends the calling task for the specified number of scheduler ticks. The
expression must be a compile-time constant.

### YIELD

```
YieldStmt = "YIELD" .
```

Voluntarily yields control to the task scheduler.

### SUSPEND and RESUME

```
SuspendStmt = "SUSPEND" Identifier .
ResumeStmt  = "RESUME" Identifier .
```

`SUSPEND` suspends the named task. `RESUME` resumes a suspended task.

### AT Directive

```
AtStmt      = "AT" Expression [ "BANK" Expression ] .
```

Sets the memory address for subsequent code or data. When `BANK` is present,
it also switches the assembler's active ROM bank.

---

## Declarations and Definitions

### Procedures

```
ProcedureDecl = "PROCEDURE" Identifier [ "(" ParamList ")" ] [ Type ] [ ProcedureModifier ] StatementList "END" .
ProcedureModifier = "REENTRANT" | "INTERRUPT" | "NMI" .
```

Procedures encapsulate a sequence of statements. They may accept parameters
and return values. The return type is given after the parameter list; if
absent, the procedure returns no value.

Reentrant procedures use stack-based frame allocation instead of static
RAM. Interrupt procedures use `reti` (maskable) or `retn` (NMI) instead
of `ret`.

```
PROCEDURE add(a BYTE, b BYTE) WORD REENTRANT
  RETURN a + b
END
```

### Tasks

```
TaskDecl = "TASK" Identifier [ "PRIORITY" Expression ] StatementList "END" .
```

Tasks are cooperative threads with static priority. Each task receives a
128-byte stack. Up to 16 tasks may exist. Priority 0 is highest, 15 lowest.

```
TASK worker PRIORITY 3
  DO
    CALL work()
    YIELD
  END
END
```

### DATA

```
DataDecl = Identifier ":" "DATA" DataValueList .
DataValueList = Expression { "," Expression } | "TILE" TileString .
TileString = "`" { tile_char } "`" .
```

Defines a named data block in ROM with the given values. `DATA TILE` defines
an 8×8 pixel tile for the SMS, where each character in the backtick string
represents a palette index.

### CONSTANT

```
ConstantDecl = "CONSTANT" Identifier [ "=" ] Expression .
```

Defines a compile-time constant binding. The expression must be a constant
expression.

### DEFINE

```
DefineDecl = "DEFINE" Identifier Type .
```

Defines a type alias.

### INCLUDE

```
IncludeStmt = "INCLUDE" StringLit .
```

Replaces the statement with the contents of the named file, parsed
recursively. Paths are relative to the including file's directory.

### INTERRUPT and NMI Statements

```
InterruptStmt = "INTERRUPT" Identifier .
NmiStmt       = "NMI" Identifier .
```

Installs a jump instruction to the named procedure at the interrupt vector
address (0x0038 for maskable, 0x0066 for NMI).

---

## Program Structure

### Top-Level Statements

A PL/Z program is a sequence of top-level statements:

```
Program = StatementList .
```

There is no required entry point. Execution begins at the first statement.
The program may contain any mix of declarations, data definitions, procedure
definitions, task definitions, and executable statements.

### Scope Rules

Scopes are established by procedures and tasks:

- **Global scope**: top-level declarations, constants, and definitions
- **Procedure scope**: parameters and local declarations within a procedure
- **Task scope**: declarations within a task

A declaration in an inner scope may not shadow a declaration in an outer
scope. All declarations are processed in two passes: pass 1 collects all
signatures and declarations; pass 2 validates procedure and task bodies.

### Memory Layout

The compiler generates code and data sections:

| Section   | Start   | Description                         |
|-----------|---------|-------------------------------------|
| Boot code | 0x0000  | Initialization and scheduler setup  |
| Code      | 0x0000+ | Program and procedure code          |
| Vector    | 0x0038  | Interrupt vector (if used)          |
| Vector    | 0x0066  | NMI vector (if used)                |
| Data      | 0xC000+ | Variables, arrays, records          |
| TCBs      | 0xC000+ | Task control blocks (128 bytes)     |
| Task stacks| 0xC080+| Task stacks (128 bytes each)        |
| Stack     | 0xDFF0  | CPU stack (grows downward)          |

### Startup Sequence

1. The CPU resets at 0x0000.
2. The generated boot code initializes the stack pointer to 0xDFF0.
3. Data section is initialized (copying initial values from ROM to RAM).
4. Task system is initialized: TCBs filled, task entry addresses pushed
   onto each task's stack.
5. Scheduler performs a `RET` into task 0.
6. Task 0 begins executing.

---

## The Task System

### TCB Format

Each of the 16 tasks has a Task Control Block of 8 bytes:

| Offset | Size | Field       | Description                               |
|--------|------|-------------|-------------------------------------------|
| 0      | 2    | SP          | Saved stack pointer when not running      |
| 2      | 1    | State       | 0=READY, 1=SUSPENDED, 2=SLEEPING, 3=DEAD |
| 3      | 1    | Sleep cnt   | Remaining ticks before wake-up            |
| 4      | 1    | Priority    | 0 (highest) to 15 (lowest)                |
| 5      | 3    | Reserved    |                                           |

### Scheduler Algorithm

The scheduler (`_plz_scheduler`) is invoked by `YIELD`, `SLEEP`, `HALT`,
or an interrupt re-entry. It performs:

1. **Save context**: Store the current task's stack pointer into its TCB.
2. **Advance time**: For each SLEEPING task, decrement its sleep counter.
   If the counter reaches 0, set its state to READY.
3. **Select task**: Scan all tasks in round-robin order starting from the
   next slot. Select the first READY task with the highest priority.
4. **Dispatch**: Restore the selected task's stack pointer and `RET` into
   it. If no task is READY, execute `HALT`.

### Task States

| State     | Value | Meaning                                     |
|-----------|-------|---------------------------------------------|
| READY     | 0     | Runnable, will be scheduled                  |
| SUSPENDED | 1     | Suspended by another task via `SUSPEND`      |
| SLEEPING  | 2     | Sleeping via `SLEEP`; countdown active       |
| DEAD      | 3     | Terminated via `HALT`; not scheduled         |

### Timing

The scheduler's "tick" advances only when the scheduler itself runs.
Three mechanisms provide timing:

- **VBlank interrupt**: A hardware interrupt at 60 Hz (NTSC) or 50 Hz (PAL)
  re-enters the scheduler, providing real-time tick advancement.
- **Timer task**: A task that spins on `YIELD` at low priority, providing
  CPU-bound tick advancement.
- **Busy-wait**: A `WHILE` counting loop that consumes CPU cycles within a
  single task without invoking the scheduler.

---

## Runtime Support

### Comparison Operations

Comparison operators are implemented with runtime helpers:

| Operator | Helper      | Returns                           |
|----------|-------------|-----------------------------------|
| `==`     | `_plz_eq`   | 1 if HL = DE, else 0              |
| `!=`     | `_plz_ne`   | 1 if HL ≠ DE, else 0              |
| `<`      | `_plz_lt`   | 1 if HL < DE (unsigned), else 0   |
| `>`      | `_plz_gt`   | 1 if HL > DE (unsigned), else 0   |
| `<=`     | `_plz_lte`  | 1 if HL ≤ DE (unsigned), else 0   |
| `>=`     | `_plz_gte`  | 1 if HL ≥ DE (unsigned), else 0   |

### Arithmetic Operations

| Operator | Helper      | Operation              |
|----------|-------------|------------------------|
| `*`      | `_plz_mul`  | HL = HL × DE           |
| `/`      | `_plz_div`  | HL = HL ÷ DE           |
| `%`      | `_plz_mod`  | HL = HL % DE           |

### Calling Convention

Procedure parameters and return values are passed via Z80 registers and
optionally memory:

- **1st parameter**: HL
- **2nd parameter**: DE
- **3rd+ parameters**: static RAM labels (default) or stack (REENTRANT)
- **Return value**: HL
- **2nd return value**: DE
- **Record/DATA parameters**: passed by reference (address)

---

## Compilation

### Pipeline

A PL/Z source file is processed through these stages:

1. **Scanner**: Lexical analysis producing a token stream. Recognizes
   keywords, identifiers, literals, operators, and comments.
2. **Parser**: Recursive-descent parser with Pratt expression parsing.
   Produces an abstract syntax tree (AST).
3. **Semantic Checker**: Two-pass analysis. Pass 1 collects all declarations
   and procedure signatures. Pass 2 validates procedure bodies, resolves
   references, checks types.
4. **Code Generator**: Translates the validated AST into Z80 assembly text.
   Emits runtime helpers as needed, manages local labels for control flow.
5. **Assembler**: Two-pass assembler producing binary output. Supports `org`,
   `db`, `dw`, `ds`, `bank`, `banksize`, `bankat` directives.

### Output Formats

| Format | Description                          |
|--------|--------------------------------------|
| `bin`  | Flat binary ROM image                |
| `sms`  | Sega Master System ROM (with header) |

---

## Implementation Limits

| Quantity | Limit |
|----------|-------|
| Tasks    | 16    |
| Task priority | 0–15 (0 highest) |
| Word value    | 0–65535 |
| Byte value    | 0–255 |
| Task stack    | 128 bytes |
| Array elements | limited by available memory |
| Identifier length | no limit (practical: line length) |

---

## Grammar Summary

```
Program         = StatementList .

StatementList   = { Statement } .
Statement       = [ Label ":" ] Command .

Command         = VariableDecl | ConstantDecl | DefineDecl
                | DataDecl | ProcedureDecl | TaskDecl
                | AssignmentStmt | IfStmt | WhileStmt | ForStmt
                | DoStmt | CaseStmt | CallStmt | ReturnStmt
                | GotoStmt | OutputStmt | EnableStmt | DisableStmt
                | HaltStmt | BankStmt | SaveStmt | LoadStmt
                | SleepStmt | YieldStmt | SuspendStmt | ResumeStmt
                | AtStmt | InterruptStmt | NmiStmt
                | IncludeStmt | PragmaStmt .

VariableDecl    = "DECLARE" Identifier [ "(" Expression ")" ] Type
                  [ "AT" Expression ] [ "=" Expression ] .
ConstantDecl    = "CONSTANT" Identifier [ "=" ] Expression .
DefineDecl      = "DEFINE" Identifier Type .
DataDecl        = Identifier ":" "DATA" ( ExpressionList | "TILE" TileString ) .
ProcedureDecl   = "PROCEDURE" Identifier [ "(" ParamList ")" ] [ Type ]
                  [ "REENTRANT" | "INTERRUPT" | "NMI" ]
                  StatementList "END" .
TaskDecl        = "TASK" Identifier [ "PRIORITY" Expression ]
                  StatementList "END" .

AssignmentStmt  = "LET" Reference "=" Expression .
IfStmt          = "IF" Expression [ "THEN" ] Command [ "ELSE" Command ] .
WhileStmt       = "WHILE" Expression [ "DO" ] Command .
ForStmt         = "FOR" Reference "=" Expression "TO" Expression
                  [ "BY" Expression ] [ "DO" ] Command .
DoStmt          = "DO" StatementList "END" .
CaseStmt        = "CASE" Expression [ "DO" ]
                  { "OF" CaseValList Command }
                  [ "OF" "DEFAULT" Command ] "END" .
CallStmt        = "CALL" [ ResultList "=" ] Identifier "(" [ ExpressionList ] ")" .
ReturnStmt      = "RETURN" [ Expression { "," Expression } ] .
GotoStmt        = "GOTO" Identifier .
OutputStmt      = "OUTPUT" [ "WORD" ] Expression Expression .
EnableStmt      = "ENABLE" .
DisableStmt     = "DISABLE" .
HaltStmt        = "HALT" .
BankStmt        = "BANK" Expression .
SaveStmt        = "SAVE" [ "AT" Expression ] Expression .
LoadStmt        = "LOAD" [ "AT" Expression ] Reference .
SleepStmt       = "SLEEP" Expression .
YieldStmt       = "YIELD" .
SuspendStmt     = "SUSPEND" Identifier .
ResumeStmt      = "RESUME" Identifier .
AtStmt          = "AT" Expression [ "BANK" Expression ] .
InterruptStmt   = "INTERRUPT" Identifier .
NmiStmt         = "NMI" Identifier .
IncludeStmt     = "INCLUDE" StringLit .
PragmaStmt      = "PRAGMA" Identifier { Identifier } .

ParamList       = Parameter { "," Parameter } .
Parameter       = Identifier Type .
ResultList      = Identifier { "," Identifier } .
CaseValList     = CaseVal { "," CaseVal } .
CaseVal         = NumberLit | Identifier .

Type            = "BYTE" | "WORD" | "DATA" | "LABEL" | "CONSTANT"
                | ArrayType | RecordType
                | "TYPE" Identifier .
ArrayType       = "ARRAY" [ "[" Expression "]" ] Type .
RecordType      = "RECORD" FieldList "END" .
FieldList       = Field { "," Field } .
Field           = Identifier Type .

Expression      = ComparisonExpr .
ComparisonExpr  = AdditiveExpr { ("==" | "!=" | "<" | ">" | "<=" | ">=") AdditiveExpr } .
AdditiveExpr    = MultiplicativeExpr { ("+" | "-" | "<<" | ">>") MultiplicativeExpr } .
MultiplicativeExpr = BitwiseExpr { ("*" | "/" | "%") BitwiseExpr } .
BitwiseExpr     = UnaryExpr { ("|" | "^" | "&") UnaryExpr } .
UnaryExpr       = PostfixExpr | "!" UnaryExpr | "-" UnaryExpr .
PostfixExpr     = Operand { "[" Expression "]" | "." Identifier | "(" [ ExpressionList ] ")" } .
Operand         = NumberLit | CharLit | StringLit | Reference
                | "(" Expression ")" | "INPUT" "(" Expression ")"
                | "LENGTH" "(" Identifier ")" .
Reference       = Identifier { "[" Expression "]" } { "." Identifier } .
ExpressionList  = Expression { "," Expression } .
TileString      = "`" { tile_char } "`" .
```
