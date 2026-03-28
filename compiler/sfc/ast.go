package sfc

// Node represents a node in the template AST.
type Node interface {
	Pos() (int, int) // Start Line, Column
	End() (int, int) // End Line, Column
}

// NodeType identifies the type of an AST node.
type NodeType int

// NodeElement and following constants identify the type of an AST node.
const (
	NodeElement NodeType = iota
	NodeText
	NodeComment
	NodeIf
	NodeEach
	NodeSnippet
	NodeComponent
	NodeExpression
)

// BaseNode contains common fields for all nodes.
type BaseNode struct {
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
}

// Pos returns the starting line and column of the node.
func (n BaseNode) Pos() (int, int) { return n.StartLine, n.StartColumn }

// End returns the ending line and column of the node.
func (n BaseNode) End() (int, int) { return n.EndLine, n.EndColumn }

// ElementNode represents an HTML element.
type ElementNode struct {
	BaseNode
	TagName     string
	Attributes  []Attribute
	Children    []Node
	SelfClosing bool
}

// Attribute represents an HTML attribute.
type Attribute struct {
	Name         string
	Value        string
	IsExpression bool // If the value is a {expression}
}

// TextNode represents plain text.
type TextNode struct {
	BaseNode
	Content string
}

// CommentNode represents an HTML comment.
type CommentNode struct {
	BaseNode
	Content string
}

// ExpressionNode represents a {expression}.
type ExpressionNode struct {
	BaseNode
	Content string
}

// IfNode represents a {#if} block.
type IfNode struct {
	BaseNode
	Condition string
	Then      []Node
	ElseIfs   []ElseIfNode
	Else      []Node
}

// ElseIfNode represents an {:else if} block.
type ElseIfNode struct {
	BaseNode
	Condition string
	Then      []Node
}

// EachNode represents a {#each} block.
type EachNode struct {
	BaseNode
	Iteratee string
	As       string
	Children []Node
}

// SnippetNode represents a {#snippet} definition.
type SnippetNode struct {
	BaseNode
	Name     string
	Args     string
	Children []Node
}

// ComponentNode represents a component call (@Component).
type ComponentNode struct {
	BaseNode
	Name       string
	Attributes []Attribute
	Children   []Node
}
