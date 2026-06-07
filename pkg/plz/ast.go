package plz

// Program represents the top-level AST node containing all statements
// in a PL/Z source file. It implements Parselet, Checklet, and Genlet
// to drive the compiler pipeline.
type Program struct {
	Statements    []Statement
	IncludedFiles map[string]bool     // visited files, used to detect recursive INCLUDE
	Templates     map[string]Template // Templates defined in this file
}

// Parselet is the interface for AST nodes that can be parsed.
type Parselet interface {
	Parse(*Parser) error
}

// Checklet is the interface for AST nodes that can be semantically checked.
type Checklet interface {
	Check(*Checker) error
}

// Genlet is the interface for AST nodes that can generate Z80 assembly code.
type Genlet interface {
	Gen(*Gen) error
}

// Node is the composite interface that all AST nodes implement.
// It combines Parselet, Checklet, and Genlet so that every node can
// be parsed, checked, and compiled.
type Node interface {
	Parselet
	Checklet
	Genlet
}

// Commander is a marker interface that discriminates between the
// various statement command types. Each concrete command type
// implements Command by returning itself.
type Commander interface {
	Command() Commander
}

func (c If) Command() Commander            { return c }
func (c Constant) Command() Commander      { return c }
func (c Data) Command() Commander          { return c }
func (c Declare) Command() Commander       { return c }
func (c Define) Command() Commander        { return c }
func (c Group) Command() Commander         { return c }
func (c Let) Command() Commander           { return c }
func (c InterruptStmt) Command() Commander { return c }
func (c Procedure) Command() Commander     { return c }
func (c Return) Command() Commander        { return c }
func (c Call) Command() Commander          { return c }
func (c GoTo) Command() Commander          { return c }
func (c Halt) Command() Commander          { return c }
func (c Enable) Command() Commander        { return c }
func (c Disable) Command() Commander       { return c }
func (c Output) Command() Commander        { return c }
func (c Task) Command() Commander          { return c }
func (c Suspend) Command() Commander       { return c }
func (c Resume) Command() Commander        { return c }
func (c Sleep) Command() Commander         { return c }
func (c Yield) Command() Commander         { return c }
func (c At) Command() Commander            { return c }
func (c BankStmt) Command() Commander      { return c }
func (c Save) Command() Commander          { return c }
func (c Load) Command() Commander          { return c }
func (c Pragma) Command() Commander        { return c }

// Statement is a labeled or unlabeled command within a PL/Z program.
// Each statement may carry an optional named label and exactly one
// Commander which determines the statement kind.
type Statement struct {
	Label   *Label    // optional label
	Command Commander // a statement is one of the commands
}

// Enable represents an ENABLE statement that re-enables maskable interrupts.
type Enable struct {
}

// Disable represents a DISABLE statement that disables maskable interrupts.
type Disable struct {
}

// Output represents an OUTPUT statement that writes a value to an I/O port.
type Output struct {
	Port   int
	Value  Expression
	IsWord bool // if true, writes low byte then high byte (WORD output)
}

// Let represents an assignment statement (LET) that stores one or two values
// into variables, array elements, or record fields. The second target captures
// the DE return value from a multi-value CALL.
type Let struct {
	Reference            // first target (HL return value)
	Target2   *Reference // optional second target (DE return value)
	Expression
}

// Procedure represents a PROCEDURE declaration. It contains the procedure
// name, return type, parameters, local variable declarations, and body
// statements. Procedures may be marked as INTERRUPT, NMI, or REENTRANT.
type Procedure struct {
	Name       Label
	Type       Type
	Parameters []Identifier
	ParamTypes []Type
	Statements []Statement
	Interrupt  *Interrupt
	Reentrant  bool
}

// Interrupt specifies whether a procedure is an interrupt handler and,
// if so, which interrupt vector it services if the platform has multiple
// vectors.
type Interrupt struct {
	Interrupt int
	NMI       bool
}

// InterruptStmt installs an interrupt handler at the default vector address.
//
//	INTERRUPT name  — installs handler at 0x0038 (maskable interrupt)
//	NMI name        — installs handler at 0x0066 (non-maskable interrupt)
//
// The statement emits a JP to the named procedure at the vector address.
type InterruptStmt struct {
	NMI    bool
	Target Identifier
}

// Identifier is a string type used for names of variables, procedures,
// labels, fields, and type aliases throughout the AST.
type Identifier string

// Predeclared is a predeclared type like BYTE, WORD, DATA, LABEL, CONSTANT, ...
type Predeclared int

const (
	PredeclaredNone     Predeclared = iota
	PredeclaredByte                 // 1-byte unsigned integer type
	PredeclaredWord                 // 2-byte unsigned integer type
	PredeclaredData                 // data storage type
)

// Typer is a marker interface for the variants of Type.
// It exists solely to enable a closed set of type discriminants
// (*PredeclaredType, *Record, *Array) via the typer() method.
type Typer interface {
	typer() Typer
}

// Type represents the type of a variable, parameter, field, or expression.
// The actual type variant is stored in the Typ field as a Typer.
type Type struct {
	Typ Typer
}

// PredeclaredType is a Typer variant representing a predeclared type
// such as BYTE or WORD.
type PredeclaredType struct {
	Kind Predeclared
}

// Array represents an array type used in RECORD fields or TYPE aliases.
type Array struct {
	Size     int // number of elements (0 = unbounded)
	ElemType Type
}

// Record represents a RECORD type composed of a sequence of named fields.
type Record struct {
	Fields []Field
}

// Field is a single named field within a Record type, consisting of
// an identifier and its associated Type.
type Field struct {
	Identifier
	Type
}

// FindField looks up a field by name, returning its index and a pointer
// to the field, or (-1, nil) if not found.
func (r *Record) FindField(name Identifier) (int, *Field) {
	for i := range r.Fields {
		if r.Fields[i].Identifier == name {
			return i, &r.Fields[i]
		}
	}
	return -1, nil
}

// Predeclared returns the Predeclared kind of t, or PredeclaredNone
// if t is not a predeclared type.
func (t Type) Predeclared() Predeclared {
	if p, ok := t.Typ.(*PredeclaredType); ok {
		return p.Kind
	}
	return PredeclaredNone
}

// Record returns the underlying *Record if t is a record type, or nil otherwise.
func (t Type) Record() *Record { r, _ := t.Typ.(*Record); return r }

// Array returns the underlying *Array if t is an array type, or nil otherwise.
func (t Type) Array() *Array { a, _ := t.Typ.(*Array); return a }

func (p *PredeclaredType) typer() Typer { return p }
func (r *Record) typer() Typer          { return r }
func (a *Array) typer() Typer           { return a }

// TotalSize returns the raw sum of field sizes for the record, before
// any power-of-two rounding that Size would apply. This is used both
// for offset calculations and as input to the rounding logic.
func (r *Record) TotalSize() int {
	s := 0
	for _, f := range r.Fields {
		s += f.Type.Size()
	}
	return s
}

// FieldOffset returns the byte offset of the i-th field within the record,
// computed as the cumulative Size of all preceding fields.
func (r *Record) FieldOffset(i int) int {
	off := 0
	for j := 0; j < i; j++ {
		off += r.Fields[j].Type.Size()
	}
	return off
}

// Size returns the storage size of the type in bytes. For arrays it
// multiplies the element size by the element count. For records it
// rounds TotalSize up to the next power of two. BYTE is 1 byte;
// everything else (WORD, LABEL, etc.) is 2 bytes.
func (t Type) Size() int {
	if a := t.Array(); a != nil {
		elemSize := a.ElemType.Size()
		if a.Size > 0 {
			return a.Size * elemSize
		}
		return 0
	}
	if r := t.Record(); r != nil {
		return nextPow2(r.TotalSize())
	}
	if t.Predeclared() == PredeclaredByte {
		return 1
	}
	return 2
}

// StorageSize returns the byte size needed for the declaration.
// Each element in a multi-element DECLARE is sized according to its
// type, and the total is the element count times the element size.
// Records are rounded up to the next power of two.
func (d Declare) StorageSize() int {
	if arr := d.Type.Array(); arr != nil {
		elemSize := arr.ElemType.Size()
		total := arr.Size * elemSize
		if total == 0 {
			total = elemSize
		}
		return total
	}
	elemSize := 1
	if r := d.Type.Record(); r != nil {
		elemSize = r.TotalSize()
		elemSize = nextPow2(elemSize)
	} else if d.Type.Predeclared() == PredeclaredWord {
		elemSize = 2
	}
	return elemSize
}

// nextPow2 rounds n up to the next power of two using bit-manipulation.
// If n is zero or negative it returns 0. This is used to align record
// storage sizes to powers of two.
func nextPow2(n int) int {
	if n <= 0 {
		return 0
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

// Return represents a RETURN statement that exits a procedure and
// optionally provides one or more result values.
type Return struct {
	Expressions []Expression
}

// Call represents a CALL expression or statement that invokes a
// procedure with the given arguments. The procedure name is stored
// in the embedded Reference.Identifier.
type Call struct {
	Reference
	Arguments []Expression
}

// GoTo represents a GOTO statement that jumps to a named or numbered label.
type GoTo struct {
	Name string
}

// Halt represents a HALT statement that stops program execution,
// or on some platforms like the SMS until an interrupt occurs
// if the interrupts were enabled.
type Halt struct{}

// BankStmt represents a BANK statement that switches the active ROM
// bank at runtime. On SMS this writes the bank number to the Sega
// mapper port (0xFFFD) to switch the slot-1 (0x4000-0x7FFF) bank.
type BankStmt struct {
	Number Expression
}

// Save represents a SAVE statement that copies data to battery-backed RAM.
// The optional AT location specifies the destination address in SRAM.
// The source expression must be a reference with a size that is known
// at compile time.
type Save struct {
	Location *Expression // optional AT address (nil = use declared AT)
	Source   Expression  // reference to data to save
}

// Load represents a LOAD statement that copies data from battery-backed RAM
// into a variable. The optional AT location specifies the source address in SRAM.
// The target expression must be a reference whose size is known at compile time.
type Load struct {
	Location *Expression // optional AT address in SRAM (nil = use declared AT)
	Target   Expression  // reference to variable to load into
}

// Pragma represents a compiler pragma directive. Pragmas may be ignored
// by conforming compilers. PL/Z implements PRAGMA BOUNDCHECK to enable
// runtime array bounds checking, and NOBOUNDCHECK to disable it, although
// the latter is the default.
type Pragma struct {
	Idents []Identifier // pragma identifiers, e.g. ["BOUNDCHECK"]
}

// Group represents a compound statement constructed from WHILE, CASE,
// FOR, or a bare DO-block. Exactly one of While, Case, or For may be
// set; when all are nil the group is a simple DO...END block.
// For and While block iterate, Case and Do blocks do not.
type Group struct {
	While      *While
	Case       *Case
	For        *For
	Statements []Statement
}

// While represents the condition expression of a WHILE loop.
type While struct{ Expression }

// For represents a FOR loop with an iteration variable, start and
// end expressions, and an optional step expression.
type For struct {
	Reference
	Start Expression
	To    Expression
	By    *Expression
}

// CaseVal is a single value in a CASE branch. It holds either a resolved
// integer literal or the name of a constant to be resolved at codegen time.
type CaseVal struct {
	Value int    // resolved integer value (0 if Name is set)
	Name  string // constant name (empty for numeric literals)
}

// CaseBranch is a single branch within a CASE group, consisting of one
// or more case values and the statement to execute when matched.
type CaseBranch struct {
	Values    []CaseVal
	Statement Statement
}

// Case represents a CASE group with a selector expression, a list of
// branches, and an optional default branch (DEFAULT). When no branch
// matches and no default is provided, execution falls through to the
// statement following the CASE...END block.
type Case struct {
	Expression
	Branches []CaseBranch
	Default  *Statement
}

// If represents an IF-THEN-ELSE conditional statement.
type If struct {
	Label     *Label
	Condition Expression // If expression condition
	Then      Statement  // Executed if TRUE
	Else      *Statement // Executed if false, optionally
}

// Label represents an optional named label attached to a statement.
// Labels are used as targets for GOTO statements.
// Since PLZ is a low level language it is allowed to jump out of any inner
// scope to any outer scope.
type Label struct {
	Name string
}

// Define represents a TYPE alias definition that assigns a name to
// a record, array, or predeclared type.
type Define struct {
	Name Identifier
	Type Type
}

// Input represents a port input expression. The syntax is INPUT(port),
// which reads a byte from the given Z80 I/O port.
type Input struct {
	Port Expression
}

// Operander marks Input as an expression operand.
func (i Input) operand() Operander { return i }

// Length represents a LENGTH(identifier) expression. It evaluates to
// the declared element count of an array, or 1 for a simple variable,
// as a compile-time constant.
type Length struct {
	Identifier Identifier
}

// Operander marks Length as an expression operand.
func (l Length) operand() Operander { return l }

// Data represents a DATA directive that embeds literal bytes or
// words directly into the output. This is useful for program data
// like image and sound.
type Data struct {
	Name   string
	Values []Expression
	Tile   *Tile    // Optional graphical data
	Text   *TextLit // Optional TEXT string literal
}

// Tile represents a DATA directive that embeds a SMS 8x8 Tile in a
// more convenient format, for example from a string or a file.
type Tile struct {
	Tiles []*SMSTile
}

// At represents an AT directive that sets the absolute memory address
// for subsequent data declarations, or switches the active ROM bank
// the program section will be compiled to when HasBank is true.
type At struct {
	Address    Expression
	HasBank    bool
	BankNumber int
}

// Declare represents a DECLARE statement that introduces a variable,
// array, record instance, or parameter. Based variables store their
// address in an indirect pointer. If At is set, the variable is placed
// at the given absolute address and no initializer is allowed.
// ParamRef marks record/array parameters that are passed by reference.
type Declare struct {
	Identifier  Identifier
	Based       *Reference
	Type        Type
	Size        int
	Initializer *Initializer
	At          *Expression // Absolute address (no initializer allowed)
	ParamRef    bool        // True when this is a record/array parameter (passed by reference)
	ConstantValue *Literal  // Set for CONSTANT declarations
	DataValue     *Data     // Set for DATA declarations
}

// Initializer wraps an Expression to provide an initial value for a
// DECLARE statement.
type Initializer struct {
	Expr Expression
}

// Literaler is a marker interface for the three variants of Literal.
// It uses an unexported literaler method to keep the set of
// implementations closed to NumberLit, TextLit, and ReferenceLit.
type Literaler interface {
	literaler() Literaler
}

// Literal represents a compile-time constant value, which may be a
// number, a text string, or a reference to another constant.
type Literal struct {
	Lit Literaler
}

// NumberLit is a Literaler variant holding an integer numeric value.
type NumberLit struct {
	Value int
}

// TextLit is a Literaler variant holding a string of text.
type TextLit struct {
	Value string
}

// ReferenceLit is a Literaler variant holding a reference to a named
// constant, used for constant aliasing.
type ReferenceLit struct {
	Value *Reference
}

func (n *NumberLit) literaler() Literaler    { return n }
func (t *TextLit) literaler() Literaler      { return t }
func (r *ReferenceLit) literaler() Literaler { return r }

// Number returns the underlying *NumberLit if l is a numeric literal,
// or nil otherwise.
func (l Literal) Number() *NumberLit { n, _ := l.Lit.(*NumberLit); return n }

// Text returns the underlying *TextLit if l is a text literal,
// or nil otherwise.
func (l Literal) Text() *TextLit { t, _ := l.Lit.(*TextLit); return t }

// Reference returns the underlying *ReferenceLit if l is a reference
// literal, or nil otherwise.
func (l Literal) Reference() *ReferenceLit { r, _ := l.Lit.(*ReferenceLit); return r }

// Constant represents a CONSTANT declaration that binds a name to a
// compile-time expression value.
type Constant struct {
	Name string
	Expr Expression
}

// Expresser is a marker interface for the four variants of Expression.
// It uses an unexported expr method to keep the set of implementations
// closed to Operand, Prefix, Infix, and Suffix.
type Expresser interface {
	expr() Expresser
}

// Expression represents a computed value in the PL/Z language. The
// actual expression kind is stored in the Expr field as an Expresser,
// which may be an Operand (leaf), Prefix (unary), Infix (binary), or
// Suffix (function-call-like postfix).
type Expression struct {
	Expr Expresser
}

// Operand extracts the underlying *Operand from e, or returns nil
// if e is not an operand expression.
func (e Expression) Operand() *Operand { o, _ := e.Expr.(*Operand); return o }

// Prefix extracts the underlying *Prefix from e, or returns nil
// if e is not a prefix expression.
func (e Expression) Prefix() *Prefix { p, _ := e.Expr.(*Prefix); return p }

// Infix extracts the underlying *Infix from e, or returns nil
// if e is not an infix expression.
func (e Expression) Infix() *Infix { i, _ := e.Expr.(*Infix); return i }

// Suffix extracts the underlying *Suffix from e, or returns nil
// if e is not a suffix expression.
func (e Expression) Suffix() *Suffix { s, _ := e.Expr.(*Suffix); return s }

// Ref walks through Expression to find the innermost *Reference (if any).
// It returns nil if the expression does not ultimately refer to a
// named reference such as a variable, field, or array element.
func (e Expression) Ref() *Reference {
	if e.Expr == nil {
		return nil
	}
	op, ok := e.Expr.(*Operand)
	if !ok {
		return nil
	}
	return op.Ref()
}

func (p *Prefix) expr() Expresser  { return p }
func (i *Infix) expr() Expresser   { return i }
func (s *Suffix) expr() Expresser  { return s }
func (o *Operand) expr() Expresser { return o }

// Prefix represents a unary operator expression such as negation (-x)
// or logical NOT (!x). The Operator field holds the operator and the
// Operand field holds the sub-expression being operated on.
type Prefix struct {
	Operator
	Operand Operand
}

// Infix represents a binary operator expression such as addition (x + y)
// or comparison (x == y). The Operator field holds the operator and
// Operands holds the left and right sub-expressions.
type Infix struct {
	Operator
	Operands [2]Operand
}

// Suffix represents a postfix operator expression such as an array
// index a[i] or a procedure call f(). The Operator field identifies
// the suffix kind (INDEX or CALL) and Operands holds the index or
// argument sub-expressions.
type Suffix struct {
	Operator
	Operands []Operand
}

// Operander is a marker interface for the four variants of Operand.
// It uses an unexported operand method to keep the set of implementations
// closed to Call, Reference, Expression, and Literal.
type Operander interface {
	operand() Operander
}

// Operand represents a leaf or parenthesized sub-expression in the
// expression tree. An Operand may be a Call, a Reference, a nested
// Expression (for parenthesized sub-expressions), or a Literal.
type Operand struct {
	Op Operander
}

// Call extracts the underlying *Call from o, or returns nil if o
// is not a call operand.
func (o Operand) Call() *Call { c, _ := o.Op.(*Call); return c }

// Reference extracts the underlying *Reference from o, or returns nil
// if o is not a reference operand.
func (o Operand) Reference() *Reference { r, _ := o.Op.(*Reference); return r }

// Expr extracts the underlying *Expression from o, or returns nil
// if o is not a nested expression operand.
func (o Operand) Expr() *Expression { e, _ := o.Op.(*Expression); return e }

// Literal extracts the underlying *Literal from o, or returns nil
// if o is not a literal operand.
func (o Operand) Literal() *Literal { l, _ := o.Op.(*Literal); return l }

// Input extracts the underlying *Input from o, or returns nil if o
// is not a port input expression.
func (o Operand) Input() *Input { i, _ := o.Op.(*Input); return i }

// Length extracts the underlying *Length from o, or returns nil if o
// is not a length expression.
func (o Operand) Length() *Length { l, _ := o.Op.(*Length); return l }

// Ref walks through Operand to find the innermost *Reference (if any).
// It unwraps nested Expression operands recursively and returns nil
// if no Reference is found.
func (o Operand) Ref() *Reference {
	if o.Op == nil {
		return nil
	}
	switch v := o.Op.(type) {
	case *Reference:
		return v
	case *Expression:
		return v.Ref()
	}
	return nil
}

func (c *Call) operand() Operander       { return c }
func (r *Reference) operand() Operander  { return r }
func (e *Expression) operand() Operander { return e }
func (l *Literal) operand() Operander    { return l }

// Operator is a numeric encoding of a PL/Z operator.
// An operator can be up to 3 bytes long — the characters that form
// the operator token are shifted into the lower bytes of the integer.
// For example, '<' + '<'<<8 encodes the << operator.
type Operator TokenKind

const (
	OperatorNone       Operator = 0
	OperatorADD        Operator = '+'          // +
	OperatorSUB        Operator = '-'          // - (binary subtraction)
	OperatorNEG        Operator = '-' + '@'<<8 // - (unary negation)
	OperatorGT         Operator = '>'          // >
	OperatorLT         Operator = '<'          // <
	OperatorGTE        Operator = '>' + '='<<8 // >=
	OperatorLTE        Operator = '<' + '='<<8 // <=
	OperatorNEQ        Operator = '!' + '='<<8 // !=
	OperatorEQU        Operator = '=' + '='<<8 // ==
	OperatorMOD        Operator = '%'          // %
	OperatorDIV        Operator = '/'          // /
	OperatorNOT        Operator = '!'          // ! (unary logical NOT)
	OperatorMUL        Operator = '*'          // *
	OperatorAND        Operator = '&'          // &
	OperatorOR         Operator = '|'          // |
	OperatorXOR        Operator = '^'          // ^
	OperatorINDEX      Operator = '[' + ']'<<8 // [] (array indexing)
	OperatorCALL       Operator = '(' + ')'<<8 // () (procedure call)
	OperatorFIELD      Operator = '.'          // . (record field access)
	OperatorShiftLeft  Operator = '<' + '<'<<8 // <<
	OperatorShiftRight Operator = '>' + '>'<<8 // >>
)

// Priority returns the precedence of the operator for use in Pratt
// parsing. Higher values bind tighter. The precedence tiers (from
// lowest to highest) are: comparison (100-150), additive shift
// (200-230), multiplicative (310-330), bitwise (410-430), unary
// (500-510), field access (590), indexing (600), and call (610).
func (o Operator) Priority() int {
	switch o {
	case OperatorEQU:
		return 100
	case OperatorGT:
		return 110
	case OperatorLT:
		return 120
	case OperatorGTE:
		return 130
	case OperatorLTE:
		return 140
	case OperatorNEQ:
		return 150

	case OperatorSUB:
		return 200
	case OperatorADD:
		return 210
	case OperatorShiftLeft:
		return 220
	case OperatorShiftRight:
		return 230

	case OperatorDIV:
		return 310
	case OperatorMOD:
		return 320
	case OperatorMUL:
		return 330

	case OperatorOR:
		return 410
	case OperatorXOR:
		return 420
	case OperatorAND:
		return 430

	case OperatorNOT:
		return 500
	case OperatorNEG:
		return 510

	case OperatorFIELD:
		return 590
	case OperatorINDEX:
		return 600
	case OperatorCALL:
		return 610
	default:
		return 0
	}
}

// Reference represents a named reference to a variable, procedure,
// array element, or record field. It optionally carries subscripts
// for array indexing and field identifiers for record field access.
type Reference struct {
	Identifier
	Subscripts []Expression
	Fields     []Identifier
}

// Task represents a TASK declaration that defines a cooperative
// task with a static priority level. Tasks are scheduled by the
// runtime and can be suspended, resumed, and slept.
type Task struct {
	Name     Label
	Priority int
	Body     []Statement
}

// Suspend represents a SUSPEND statement that suspends execution of
// the named task. Suspended tasks remain suspended until resumed.
type Suspend struct {
	Name Identifier
}

// Resume represents a RESUME statement that resumes execution of
// a previously suspended task.
type Resume struct {
	Name Identifier
}

// Sleep represents a SLEEP statement that suspends the current task
// for the specified number of cycles.
type Sleep struct {
	Duration Expression
}

// Yield represents a YIELD statement that voluntarily yields control
// to the task scheduler, allowing another ready task to run.
type Yield struct {
}

// paramByteSize returns the storage size (in bytes) for the i-th parameter of proc.
// Records are passed by reference so they occupy 2 bytes (a pointer) in the frame.
func (p Procedure) paramByteSize(i int) int {
	if i < len(p.ParamTypes) {
		if p.ParamTypes[i].Predeclared() == PredeclaredByte {
			return 1
		}
		// Records and arrays are passed by reference → 2-byte pointer.
		if p.ParamTypes[i].Record() != nil {
			return 2
		}
	}
	return 2
}
