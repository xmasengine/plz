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
	Declaration     *Declaration
	Halt            *Halt
	LabelDefinition *LabelDefinition
}

type Assignment struct{}
type Procedure struct{}
type Return struct{}
type Call struct{ Name string }
type GoTo struct{ Label }
type Declaration struct{}
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

type Expression struct {
}
