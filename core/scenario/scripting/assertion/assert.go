package assertion

import (
	"fmt"
	"strings"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/evaluator"
	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/lexer"
	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/parser"
)

type AssertionError struct { // UnWrappable
	failedAssertion string
	received        map[string]interface{}
	wrappedErr      error
}

func (ae AssertionError) Error() string {
	return fmt.Sprintf("input : %s, received: %v, wrappedErr: %v", ae.failedAssertion, ae.received, ae.wrappedErr)
}

func (ae AssertionError) Rule() string {
	return ae.failedAssertion
}

func (ae AssertionError) Received() map[string]interface{} {
	return ae.received
}

func (ae AssertionError) Unwrap() error {
	return ae.wrappedErr
}

func Assert(input string, env *evaluator.AssertEnv) (bool, error) {
	l := lexer.New(input)
	p := parser.New(l)

	node := p.ParseExpressionStatement()
	if len(p.Errors()) > 0 {
		return false, AssertionError{
			failedAssertion: input,
			received:        map[string]interface{}{},
			wrappedErr:      fmt.Errorf(strings.Join(p.Errors(), ",")),
		}
	}

	receivedMap := make(map[string]interface{})
	obj, err := evaluator.Eval(node, env, receivedMap)
	if err != nil {
		return false, AssertionError{
			failedAssertion: input,
			received:        receivedMap,
			wrappedErr:      err,
		}
	}

	b, ok := obj.(bool)
	if ok {
		if b == false {
			return false, AssertionError{
				failedAssertion: input,
				received:        receivedMap,
				wrappedErr:      fmt.Errorf("expression evaluated to false"),
			}
		}
		return b, nil
	}

	return false, AssertionError{
		failedAssertion: input,
		received:        receivedMap,
		wrappedErr:      fmt.Errorf("evaluated value is not bool : %v", obj),
	}
}
