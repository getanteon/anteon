package assertion

import (
	"fmt"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/evaluator"
	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/lexer"
	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/parser"
)

func Assert(input string, env *evaluator.AssertEnv) bool {
	// TODO: optimize
	l := lexer.New(input)
	p := parser.New(l)

	node := p.ParseExpressionStatement()
	if len(p.Errors()) > 0 {
		fmt.Println(p.Errors())
		return false
	}

	obj, err := evaluator.Eval(node, env)
	if err != nil {
		fmt.Println(err)
		return false
	}

	b, ok := obj.(bool)
	if ok {
		return b
	}
	fmt.Printf("evaluated value is not bool %s", obj)
	return false
}
