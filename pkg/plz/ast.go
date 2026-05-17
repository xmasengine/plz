package plz

type AST interface {
}

type Program struct {
	Statements []Statement
}

type Statement struct {
	*Basic
	*If
}

type Basic struct {
	Assignment      *Assignment
	Group           *Group
	Procedure       *Procedure
	Return          *Return
	Call            *Call
	GoTo            *GoTo
	Declarations    *Declarations
	Halt            *Halt
	LabelDefinition *LabelDefinition
}

type Assignment struct{}
type Procedure struct {
	Name       Label
	Type       Type
	Parameters []Identifier
	Statements []Statement
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

type Return struct{}
type Call struct{ Variable }
type GoTo struct{ Label }
type Declarations struct {
	Declarations []Declaration
}
type Halt struct{}
type LabelDefinition struct{ Label }

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
	Then      Basic      // Executed if TRUE
	Else      *Statement // Executed if false, optionally
}

type Label struct {
	Name     string
	Location int
}

type Declaration struct {
	Type    *TypeDeclaration
	Literal *LiteralDeclaration
	Data    *DataDeclaration
}

type LiteralDeclaration struct {
	Identifier
	Literally string
}

type TypeDeclaration struct {
	*Initializer
}

type Initializer struct {
}

type DataDeclaration struct {
	Identifier
	Constants []Constant
}

type Constant struct{}

type Expression struct {
	*Operation
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
