package ast

import (
	"bytes"
	"strings"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/token"
)

// The base Node interface
type Node interface {
	TokenLiteral() string
	String() string
}

// All statement nodes implement this
type Statement interface {
	Node
	statementNode()
}

// All expression nodes implement this
type Expression interface {
	Node
	expressionNode()
}

type ExpressionStatement struct {
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// Expressions
type Identifier struct {
	Token token.Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

type Boolean struct {
	Token token.Token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }
func (il *Boolean) GetVal() interface{} { return il.Value }

type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }
func (il *IntegerLiteral) GetVal() interface{}  { return il.Value }

type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (il *FloatLiteral) expressionNode()      {}
func (il *FloatLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *FloatLiteral) String() string       { return il.Token.Literal }
func (il *FloatLiteral) GetVal() interface{}  { return il.Value }

type NullLiteral struct {
	Token token.Token
	Value interface{}
}

func (il *NullLiteral) expressionNode()      {}
func (il *NullLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *NullLiteral) String() string       { return il.Token.Literal }
func (il *NullLiteral) GetVal() interface{}  { return il.Value }

type StringLiteral struct {
	Token token.Token
	Value string
}

func (il *StringLiteral) expressionNode()      {}
func (il *StringLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *StringLiteral) String() string       { return il.Token.Literal }
func (il *StringLiteral) GetVal() interface{}  { return il.Value }

type ArrayLiteral struct {
	Token token.Token
	Elems []Expression
}

func (il *ArrayLiteral) expressionNode()      {}
func (il *ArrayLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *ArrayLiteral) String() string {
	x := []string{}
	for _, e := range il.Elems {
		x = append(x, e.String())
	}

	return "[" + strings.Join(x, ",") + "]"
}

type ObjectLiteral struct {
	Token token.Token
	Elems map[string]Expression
}

func (il *ObjectLiteral) expressionNode()      {}
func (il *ObjectLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *ObjectLiteral) String() string {
	x := []string{}
	for k, e := range il.Elems {
		x = append(x, k+":"+e.String())
	}

	return "{" + strings.Join(x, ",") + "}"
}

type PrefixExpression struct {
	Token    token.Token // The prefix token, e.g. !
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

type InfixExpression struct {
	Token    token.Token // The operator token, e.g. +
	Left     Expression
	Operator string
	Right    Expression
}

func (oe *InfixExpression) expressionNode()      {}
func (oe *InfixExpression) TokenLiteral() string { return oe.Token.Literal }
func (oe *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(oe.Left.String())
	out.WriteString(" " + oe.Operator + " ")
	out.WriteString(oe.Right.String())
	out.WriteString(")")

	return out.String()
}

type CallExpression struct {
	Token     token.Token // The '(' token
	Function  Expression  // Identifier or FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }

func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ","))
	out.WriteString(")")

	return out.String()
}
