# Intermediate Representation for PLZ

The generator currently is a mess. I plan to fix it by switching from a direct
tree walking generator to one which uses an intermediate representation.
The intermediate representation or IR will model an abstract stack machine
with instructions that can have only zero or one operand.

This abstract machine is called the PLZAM for PLZ Abstract Machine.
The intermediate representation is named PLZIR.

The PLZAM has a return stack, a data stack, and named variable locations.

This IR will then be translated to the z80 architecture.


# Complete IR Instruction List

The following list contains all instructions required for PLZIR.
Most instructions use the data stack, which is referred to simply as "stack"
Instructions that use the "return stack" mention it.

## Data Movement

PUSH_B [val]
PUSH_W [val]

Pushes 8-bit or 16-bit literal onto the stack.

VAR_B [name]
VAR_W [name]
Defines a 8 bits or 16 bits variable. These have global scope.
It is allowed to refer to a variable defined later in the IR.

GET_B [name]
GET_W [name]

Fetches variable value; pushes to stack.

PUT_B [name]
PUT_W [name]

Pops value from stack; saves to named memory slot.

## Pointers & Memory

PUSH_A [name]

Pushes 16-bit raw hardware memory address to stack for name.

READ_B
READ_W

Pops 16-bit address; reads value from RAM; pushes to stack.
If []

WRITE_B
WRITE_W

Pops value, pops address; writes value to RAM address.

## Math & Logic

All math is performed checking the operand size as described below.

ADD
SUB
MUL
DIV

Pops right-hand side, pops left-hand side; executes math; pushes result.

SHL
SHR

Pops shift count, pops value; shifts data; pushes result.

AND
OR
XOR
Performs bitwise operations on stack values.

## Casting
CAST_W
CAST_B

Widens byte to word (zero-pad) / Truncates word to byte.

## Control Flow

IS [cond]
Compares two popped values (LT,GT,LE,GE,EQ,NE); pushes 0 or 1.

TAG [name]
GO  [tag]
GO_IF [tag]
GO_ELSE [tag]

Text anchors and branching mechanics for loops and if blocks.
It is allowed to jump to a tag defined later in the IR.


# Procedures & Tasks

SUB [name]
Declares a subroutine, code follows.
It is allowed to call a sub defined later in the IR.


RUN [name]
Executes a subroutine call using the "return stack" or CPU hardware (jump) stack.

DONE <NMI|INTERRUPT>
Returns from a subroutine, Can flag that it returns from an interrupt handler
or NMI.


JOB [name]
Declares a task, code follows.
It is allowed to refer to a task defined later in the IR.

BYE
Yields the current job.

SLEEP
Pops a 16 bits duration from the stack and makes the current task sleep for that
many centiseconds. So, 100 is one second.

STOP [name]
Suspends/stops the named job.

START [name]
Restarts/starts the named job.

# Low level input, output, and interrupt handling

IN_B [port]
Reads a byte from the port and stores it in the stack.

IN_W [port]
Reads a word from the port and stores it in the stack.

OUT_B [port]
Pops a byte from stack and writes it to the port.

OUT_W [port]
Pops a word from the stack and writes it to the port.

INT [name]
Sets name as the interrupt hanldler

NMI [name]
Sets name as the NMI  hanldler

RND
Generates a random byte, for example on the Z80 by reading the r register,
and pushes it to the stack.

HLT
Halts the processor until an interrupt arrives.

DII
Disable interrupts

ENI
Enable interrupts


# Z80 Hardware Register Mapping

To achieve maximum performance on the Z80 using this IR, enforce a universal
register allocation discipline where registers are strictly categorized into
Stack Caching, Stack Control, and Temporary Scratchpads.

┌────────────────────────────────────────────────────────┐
│               Z80 REGISTERS & FUNCTIONS                │
├───────────────────┬────────────────────────────────────┤
│ REGISTER / PAIR   │ COMPILER ROLE / FUNCTION           │
├───────────────────┼────────────────────────────────────┤
│     DE            │ Top of Data Stack (TOS Cache)      │
│       E           │   └─ Lower 8 bits (Used for Bytes) │
│       D           │   └─ Upper 8 bits (Cleared if Byte)│
├───────────────────┼────────────────────────────────────┤
│     HL            │ Data Stack Pointer (in SRAM)       │
├───────────────────┼────────────────────────────────────┤
│     SP            │ Return Stack Pointer (Hardware)    │
├───────────────────┼────────────────────────────────────┤
│     BC            │ General Temporary Scratchpad       │
├───────────────────┼────────────────────────────────────┤
│     AF            │ Accumulator & Flag Status          │
└───────────────────┴────────────────────────────────────┘

# Detailed Architecture Rules

Top of Stack (TOS) Cache (DE):

The absolute top item of your data stack must always reside in the DE register
pair. If the current operand is an 8-bit byte, it lives in E, and D is kept
strictly cleared to 0.

This allows arithmetic operations to execute immediately without checking
variable width.

Data Stack Pointer (HL):
HL points to the next available position in your Sega Master System Static RAM
data stack. It is initialized to point after all variable and task definitions.
When DE needs to be overwritten by a new push instruction, the current DE value
is stored into the RAM address pointed to by HL, and HL increments.

When an operation consumes data, the underlying values are popped back into
registers from (HL), and HL decrements.

Return Stack Pointer (SP):
The native Z80 hardware stack pointer (SP) is reserved exclusively for control
flow and tasks. It handles return addresses generated by CALL and RETURN
instructions. Recursions is normally not allowed, except for REENTRANT functions.

Scratchpads (BC, A):
BC is your main assembly workhorse register. It is used to load immediate
constants, hold temporary variables during multi-step math operations, or serve
as a memory index.

A (Accumulator) is used heavily for raw 8-bit math calculations, bitwise masks
(AND/OR), and condition flag testing.


