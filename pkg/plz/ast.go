package plz

type Node interface {
	Parse(*Parser) error
	Gen(*Gen) error
	Nodes() []Node
}

type Program struct {
	Statements []Statement
}

type Statement struct {
	Label        *Label
	If           *If
	Constant     *Constant
	Data         *Data
	Assignment   *Assignment
	Group        *Group
	Procedure    *Procedure
	Return       *Return
	Call         *Call
	GoTo         *GoTo
	Declarations *Declarations
	Halt         *Halt
	Enable       *Enable
	Disable      *Disable
	Output       *Output
}

type Enable struct {
}

type Disable struct {
}

type Output struct {
	Port  int
	Value byte
}

type Assignment struct {
	Reference
	Expression
}

type Procedure struct {
	Name       Label
	Type       Type
	Parameters []Identifier
	Statements []Statement
	Interrupt  *Interrupt
}

type Interrupt struct {
	Interrupt int
	NMI       bool
}

type Identifier string
type Type int

const (
	TypeNone Type = iota
	TypeByte
	TypeAddress
	TypeLabel
	TypeData
	TypeConstant
)

type Return struct {
	*Expression
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

type Declarations struct {
	Declarations []Declaration
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

type Declaration struct {
	Type    *TypeDeclarations
	Literal *LiteralDeclaration
}

type LiteralDeclaration struct {
	Identifier
	Literally string
}

type Variable struct {
	Identifier
	Based *Variable
}

type TypeDeclarations struct {
	Declarations []TypeDeclaration
}

type TypeDeclaration struct {
	Variable
	Type
	Dimension int
	*Initializer
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
	Literal
}

type Expression struct {
	*Operation
	*Call // inline function call
	*Variable
}

type Operation struct {
	Operands []Expression
	Operator
}

type Operand struct {
	Expression
}

type Operator int

const (
	OperatorNone Operator = 0
	OperatorADD
	OperatorSUB
	OperatorNEG
	OperatorGT
	OperatorLT
	OperatorGTE
	OperatorLTE
	OperatorNEQ
	OperatorEQU
	OperatorMOD
	OperatorDIV
	OperatorNOT
	OperatorMUL
	OperatorAND
	OperatorOR
	OperatorXOR
)

type Reference struct {
	Identifier
	Address   bool
	Subscript *Expression
}

type LogicalExpression struct {
	*Variable
}
