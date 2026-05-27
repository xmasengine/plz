package plz

type Node interface {
	Parse(*Parser) error
	Check(*Checker) error
	Gen(*Gen) error
	Nodes() []Node
}

type Program struct {
	Statements []Statement
}

type Statement struct {
	Label     *Label
	If        *If
	Constant  *Constant
	Data      *Data
	Declare   *Declare
	Group     *Group
	Let       *Let
	Procedure *Procedure
	Return    *Return
	Call      *Call
	GoTo      *GoTo
	Halt      *Halt
	Enable    *Enable
	Disable   *Disable
	Output    *Output
}

type Enable struct {
}

type Disable struct {
}

type Output struct {
	Port  int
	Value Expression
}

type Let struct {
	Reference
	Expression
}

type Procedure struct {
	Name       Label
	Type       Type
	Parameters []Identifier
	ParamTypes []Type
	Statements []Statement
	Interrupt  *Interrupt
	Reentrant  bool
	Returns    int
	Locals     []Declare
}

type Interrupt struct {
	Interrupt int
	NMI       bool
}

type Identifier string

type Predeclared int

const (
	PredeclaredNone Predeclared = iota
	PredeclaredByte
	PredeclaredWord
	PredeclaredLabel
	PredeclaredData
	PredeclaredConstant
)

type Type struct {
	Predeclared
	*Record
}

type Record struct {
	Fields []Field
}

type Field struct {
	Identifier
	Type
}

type Return struct {
	Expressions []Expression
}

type Call struct {
	Location int    // temporary until Reference parsing is implemented
	Name     string // temporary until Reference parsing is implemented
	Reference
	Arguments []Expression
}

type GoTo struct {
	Name     string
	Location int
}

type Halt struct{}

type Group struct {
	While      *While
	Case       *Case
	For        *For
	Statements []Statement
}

type While struct{ Expression }

type For struct {
	Reference
	Start Expression
	To    Expression
	By    *Expression
}

type Case struct{ Expression }

type If struct {
	Label     *Label
	Condition Expression // If expression condition
	Then      Statement  // Executed if TRUE
	Else      *Statement // Executed if false, optionally
}

type Label struct {
	Name     string
	Location int
}

type Data struct {
	Literal
}

type Literally struct {
	Identifier
	Literally string
}

type Declare struct {
	Identifier  Identifier
	Based       *Reference
	Type        Type
	Size        int
	Dimension   int
	Dims        []int  // per-dimension sizes from ARRAY declaration
	Initializer *Initializer
	ParamRef    bool   // true when this is a record/array parameter (passed by reference)
}

type Initializer struct {
	Literal
}

type Literal struct {
	Text      *string
	Number    *int
	Reference *Reference
}

type Constant struct {
	Name string
	Literal
}

type Expression struct {
	*Prefix
	*Infix
	*Suffix
	*Operand
}

type Prefix struct {
	Operator
	Operand Operand
}

type Infix struct {
	Operator
	Operands [2]Operand
}

type Suffix struct {
	Operator
	Operands []Operand
}

type Operand struct {
	*Call
	*Reference
	*Expression
	*Literal
}

// An operator can be up to 3 bytes long, shift the chars into the int.
type Operator TokenKind

const (
	OperatorNone Operator = 0
	OperatorADD  Operator = '+'
	OperatorSUB  Operator = '-'
	OperatorNEG  Operator = '-' + '@'<<8
	OperatorGT   Operator = '>'
	OperatorLT   Operator = '<'
	OperatorGTE  Operator = '>' + '='<<8
	OperatorLTE  Operator = '<' + '='<<8
	OperatorNEQ  Operator = '!' + '='<<8
	OperatorEQU  Operator = '=' + '='<<8
	OperatorMOD  Operator = '%'
	OperatorDIV  Operator = '/'
	OperatorNOT  Operator = '!'
	OperatorMUL  Operator = '*'
	OperatorAND  Operator = '&'
	OperatorOR   Operator = '|'
	OperatorXOR  Operator = '^'
	OperatorINDEX Operator = '[' + ']'<<8
	OperatorCALL  Operator = '(' + ')'<<8
	OperatorFIELD Operator = '.'
)

// Priority returns the piority of the operator, mostly for Pratt parsing.
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
	case OperatorINDEX:
		return 600
	case OperatorCALL:
		return 610
	case OperatorFIELD:
		return 590
	default:
		return 0
	}
}

type Reference struct {
	Identifier
	Address    bool
	Subscripts []Expression
	Fields     []Identifier
}

type LogicalExpression struct {
	*Reference
}
