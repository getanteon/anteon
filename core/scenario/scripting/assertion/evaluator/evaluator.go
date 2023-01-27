package evaluator

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/ast"
)

func Eval(node ast.Node, env *AssertEnv) (interface{}, error) {
	switch node := node.(type) {

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	// Expressions
	case *ast.IntegerLiteral:
		return node.GetVal(), nil
	case *ast.FloatLiteral:
		return node.GetVal(), nil
	case *ast.StringLiteral:
		return node.GetVal(), nil
	case *ast.ArrayLiteral:
		args, err := evalExpressions(node.Elems, env)
		if err != nil {
			return nil, err
		}
		return args, nil
	case *ast.Boolean:
		return node.GetVal(), nil
	case *ast.PrefixExpression:
		right, err := Eval(node.Right, env)
		if err != nil {
			return nil, err
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left, err := Eval(node.Left, env)
		if err != nil {
			return nil, err
		}

		right, err := Eval(node.Right, env)
		if err != nil {
			return nil, err
		}

		return evalInfixExpression(node.Operator, left, right)
	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.CallExpression:
		funcName := node.Function.(*ast.Identifier).Value
		if _, ok := assertionFuncMap[funcName]; ok {
			args, err := evalExpressions(node.Arguments, env)
			if err != nil {
				return nil, err
			}

			f := func() (result interface{}, err error) {
				defer func() {
					if r := recover(); r != nil {
						result = nil
						err = fmt.Errorf("%v", r) // TODO: meaningful error
					}
				}()

				// TODO: err check and propagation
				switch funcName {
				case NOT:
					boolArg, _ := strconv.ParseBool(fmt.Sprintf("%t", args[0])) // TODO err check
					return not(boolArg), nil
				case LESSTHAN:
					variable, _ := strconv.ParseInt(fmt.Sprintf("%d", args[0]), 10, 64) // TODO err check
					limit, _ := strconv.ParseInt(fmt.Sprintf("%d", args[1]), 10, 64)    // TODO err check
					return less_than(variable, limit), nil
				case EQUALS:
					return equals(args[0], args[1]), nil
				case IN:
					return in(args[0], args[1].([]interface{})), nil
				case JSONPATH:
					return jsonExtract(env.Body, args[0].(string)), nil
				case XMLPATH:
					return xmlExtract(env.Body, args[0].(string)), nil
				case REGEXP:
					return regexExtract(env.Body, args[0].(string), args[1].(int64)), nil
				case HAS:
					if args[0] != nil {
						return true, nil // if identifier evaluated, and exists
					}
					return false, nil
				case CONTAINS:
					return contains(args[0].(string), args[1].(string)), nil
				case RANGE:
					var x, low, high int64

					x, ok = args[0].(int64)
					if !ok {
						x, _ = strconv.ParseInt(args[0].(string), 0, 64)
					}

					low = args[1].(int64)
					high = args[2].(int64)

					return rangeF(x, low, high), nil
				}
				return nil, fmt.Errorf("func %s not defined", funcName)
			}
			return f()
		}
	}
	return nil, nil
}

func evalPrefixExpression(operator string, right interface{}) (interface{}, error) {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return nil, fmt.Errorf("unknown operator: %s%s", operator, right)
	}
}

func evalInfixExpression(
	operator string,
	left, right interface{},
) (interface{}, error) {
	// TODO: check type mismatch, add float

	var leftType, rightType string

	intLeft, ok := left.(int64)
	if ok {
		leftType = "int64"
	}

	intRight, ok := right.(int64)
	if ok {
		rightType = "int64"
	}

	if leftType == "int64" && rightType == "int64" {
		return evalIntegerInfixExpression(operator, intLeft, intRight)
	}

	if operator == "==" {
		return reflect.DeepEqual(left, right), nil
	}

	if operator == "!=" {
		return !reflect.DeepEqual(left, right), nil
	}

	return nil, fmt.Errorf("unknown operator: evalInfixExpression %s ", operator)

}

func evalBangOperatorExpression(right interface{}) (bool, error) {
	b, ok := right.(bool)
	if ok {
		return !b, nil
	}

	return false, fmt.Errorf("exp is not bool %s", right)
}

func evalMinusPrefixOperatorExpression(right interface{}) (int64, error) {
	i, ok := right.(int64)
	if !ok {
		return 0, fmt.Errorf("unknown operator: -%s", right)
	}

	return -i, nil
}

func evalIntegerInfixExpression(
	operator string,
	left, right int64,
) (interface{}, error) {

	switch operator {
	case "+":
		return left + right, nil
	case "-":
		return left - right, nil
	case "*":
		return left * right, nil
	case "/":
		return left / right, nil
	case "<":
		return left < right, nil
	case ">":
		return left > right, nil
	case "==":
		return left == right, nil
	case "!=":
		return left != right, nil
	default:
		return 0, fmt.Errorf("unknown operator: for integer infix expression %s",
			operator)
	}
}

func evalIdentifier(
	node *ast.Identifier,
	env *AssertEnv,
) (interface{}, error) {
	ident := node.Value
	if strings.EqualFold(ident, "status_code") {
		return env.StatusCode, nil
	}
	if strings.EqualFold(ident, "response_size") {
		return env.ResponseSize, nil
	}
	if strings.EqualFold(ident, "response_time") {
		return env.ResponseTime, nil
	}
	if strings.EqualFold(ident, "body") {
		return env.Body, nil
	}
	if strings.HasPrefix(ident, "variables.") {
		vr := strings.TrimPrefix(ident, "variables.")
		if v, ok := env.Variables[vr]; ok {
			return v, nil
		}
		return "", fmt.Errorf("variable not found %s", vr)
	}
	if strings.HasPrefix(ident, "headers.") {
		vr := strings.TrimPrefix(ident, "headers.")
		hv := env.Headers.Get(vr)
		if hv != "" {
			return hv, nil
		}
		return "", fmt.Errorf("header not found %s", vr)
	}

	return "", fmt.Errorf("identifier could not evaluated %s", ident)
}

func evalExpressions(
	exps []ast.Expression,
	env *AssertEnv,
) ([]interface{}, error) {
	var result []interface{}

	for _, e := range exps {
		evaluated, err := Eval(e, env)
		if err != nil {
			return nil, err
		}
		result = append(result, evaluated)
	}

	return result, nil
}
