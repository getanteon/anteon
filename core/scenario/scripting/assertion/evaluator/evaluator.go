package evaluator

import (
	"fmt"
	"reflect"
	"strconv"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/ast"
)

func Eval(node ast.Node, env map[string]interface{}) (interface{}, error) {
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

			// TODO: err check and propagation
			switch funcName {
			case "not":
				boolArg, _ := strconv.ParseBool(fmt.Sprintf("%t", args[0])) // TODO err check
				return not(boolArg), nil
			case "less_than":
				variable, _ := strconv.ParseInt(fmt.Sprintf("%d", args[0]), 10, 64) // TODO err check
				limit, _ := strconv.ParseInt(fmt.Sprintf("%d", args[1]), 10, 64)    // TODO err check
				return less_than(variable, limit), nil
			case "equals":
				return equals(args[0], args[1]), nil
			case "in":
				return in(args[0], args[1].([]interface{})), nil
			case "json_path":
				return jsonExtract(env["body"].(string), args[0].(string)), nil
			}

		} else {
			return nil, fmt.Errorf("func %s not defined", funcName)
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
	env map[string]interface{},
) (interface{}, error) {
	env["variable"] = 20 // // TODO add keywords, native values, add response body, headers, responsesize
	val, ok := env[node.Value]
	if !ok {
		return nil, fmt.Errorf("identifier not found: " + node.Value)
	}

	return val, nil
}

func evalExpressions(
	exps []ast.Expression,
	env map[string]interface{},
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
