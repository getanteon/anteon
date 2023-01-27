package assertion

import (
	"fmt"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/evaluator"
	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/lexer"
	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/parser"
)

func Assert(input string, env *evaluator.AssertEnv) (bool, error) {
	// TODO: optimize
	l := lexer.New(input)
	p := parser.New(l)

	node := p.ParseExpressionStatement()
	if len(p.Errors()) > 0 {
		return false, fmt.Errorf("%v", p.Errors())
	}

	obj, err := evaluator.Eval(node, env)
	if err != nil {
		return false, err
	}

	b, ok := obj.(bool)
	if ok {
		return b, nil
	}
	return false, fmt.Errorf("evaluated value is not bool %s", obj)
}
