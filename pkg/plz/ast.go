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
	Variable
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
	Location int    // temporary until Variable parsing is implemented
	Name     string // temporary until Variable parsing is implemented
	Variable
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
	Variable string
	Start    Expression
	To       Expression
	By       *Expression
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

type Declaration struct {
	Type    *TypeDeclarations
	Literal *LiteralDeclaration
	Data    *DataDeclaration
}

type LiteralDeclaration struct {
	Identifier
	Literally string
}

type VariableDeclaration struct {
	Identifier
	Based Variable
}

type TypeDeclarations struct {
	Declarations []TypeDeclaration
}

type TypeDeclaration struct {
	VariableDeclaration
	Type
	Dimension int
	*Initializer
}

type Initializer struct {
	Constant
}

type DataDeclaration struct {
	Identifier
	Constants []Constant
}

type Constant struct{}

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

type Variable struct {
	Identifier
	Reference bool
	Subscript *Expression
}

type LogicalExpression struct {
	*Variable
}
