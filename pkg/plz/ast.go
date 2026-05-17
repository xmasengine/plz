package plz

type AST interface {
}

type Program struct {
	Statements []Statement
}

type Statement struct {
	Basic *BasicStatement
	If    *IfStatement
}

type BasicStatement struct {
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
type Variable struct{ Name string }

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

type IfStatement struct {
	Label     *Label
	Condition Expression     // IF expression confition
	Then      BasicStatement // Executed if TRUE
	Else      *Statement     // Executed if false, optionally
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
	Litterally string
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
}
