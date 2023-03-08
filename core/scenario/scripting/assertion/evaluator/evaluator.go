package evaluator

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/ast"
)

func Eval(node ast.Node, env *AssertEnv, receivedMap map[string]interface{}) (interface{}, error) {
	switch node := node.(type) {

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env, receivedMap)

	// Expressions
	case *ast.IntegerLiteral:
		return node.GetVal(), nil
	case *ast.FloatLiteral:
		return node.GetVal(), nil
	case *ast.StringLiteral:
		return node.GetVal(), nil
	case *ast.NullLiteral:
		return node.GetVal(), nil
	case *ast.ArrayLiteral:
		args, err := evalExpressions(node.Elems, env, receivedMap)
		if err != nil {
			return nil, err
		}
		return args, nil
	case *ast.Boolean:
		return node.GetVal(), nil
	case *ast.PrefixExpression:
		right, err := Eval(node.Right, env, receivedMap)
		if err != nil {
			return nil, err
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left, err := Eval(node.Left, env, receivedMap)
		if err != nil {
			return nil, err
		}

		right, err := Eval(node.Right, env, receivedMap)
		if err != nil {
			return nil, err
		}

		return evalInfixExpression(node.Operator, left, right)
	case *ast.Identifier:
		return evalIdentifier(node, env, receivedMap)

	case *ast.CallExpression:
		funcName := node.Function.(*ast.Identifier).Value
		if _, ok := assertionFuncMap[funcName]; ok {
			args, err := evalExpressions(node.Arguments, env, receivedMap)
			if err != nil {
				return false, err
			}

			f := func() (result interface{}, err error) {
				defer func() {
					if r := recover(); r != nil {
						result = nil
						err = fmt.Errorf("probably error during type conversion , %v", r)
					}
				}()

				switch funcName {
				case NOT:
					p, ok := args[0].(bool)
					if !ok {
						return false, ArgumentError{
							msg:        "arg of not func must be a bool",
							wrappedErr: nil,
						}
					}
					return not(p), nil
				case LESSTHAN:
					variable, ok := args[0].(int64)
					if !ok {
						variable, _ = strconv.ParseInt(args[0].(string), 0, 64)
					}
					limit, ok := args[1].(int64)
					if !ok {
						return false, ArgumentError{
							msg:        "limit of less_than should be integer",
							wrappedErr: nil,
						}
					}
					return less_than(variable, limit), nil
				case GREATERTHAN:
					variable, ok := args[0].(int64)
					if !ok {
						variable, _ = strconv.ParseInt(args[0].(string), 0, 64)
					}
					limit, ok := args[1].(int64)
					if !ok {
						return false, ArgumentError{
							msg:        "limit of greater_than should be integer",
							wrappedErr: nil,
						}
					}
					return greater_than(variable, limit), nil
				case EQUALS:
					return equals(args[0], args[1])
				case EQUALSONFILE:
					filepath, ok := args[1].(string)
					if !ok {
						return false, ArgumentError{
							msg:        "filepath must be a string",
							wrappedErr: nil,
						}
					}
					return equalsOnFile(args[0], filepath)
				case IN:
					elems, ok := args[1].([]interface{})
					if !ok {
						return false, ArgumentError{
							msg:        "second arg of in func must be an array",
							wrappedErr: nil,
						}
					}
					return in(args[0], elems)
				case JSONPATH:
					jsonpath, ok := args[0].(string)
					if !ok {
						return false, ArgumentError{
							msg:        "jsonpath must be a string",
							wrappedErr: nil,
						}
					}
					return jsonExtract(env.Body, jsonpath)
				case XMLPATH:
					xpath, ok := args[0].(string)
					if !ok {
						return false, ArgumentError{
							msg:        "xpath must be a string",
							wrappedErr: nil,
						}
					}
					return xmlExtract(env.Body, xpath)
				case REGEXP:
					regexp, ok := args[1].(string)
					if !ok {
						return false, ArgumentError{
							msg:        "regexp must be a string",
							wrappedErr: nil,
						}
					}
					matchNo, ok := args[2].(int64)
					if !ok {
						return false, ArgumentError{
							msg:        "matchNo must be an int64",
							wrappedErr: nil,
						}
					}
					return regexExtract(env.Body, regexp, matchNo)
				case HAS:
					if args[0] != nil {
						return true, nil // if identifier evaluated, and exists
					}
					return false, nil
				case CONTAINS:
					p1, ok := args[0].(string)
					if !ok {
						return false, ArgumentError{
							msg:        "args of contains func must be string",
							wrappedErr: nil,
						}
					}
					p2, ok := args[1].(string)
					if !ok {
						return false, ArgumentError{
							msg:        "args of contains func must be string",
							wrappedErr: nil,
						}
					}
					return contains(p1, p2), nil
				case AVG:
					arr, ok := args[0].([]int64)
					if !ok {
						return false, ArgumentError{
							msg:        "argument of avg func must be an int64 array",
							wrappedErr: nil,
						}
					}
					return avg(arr)
				case MIN:
					arr, ok := args[0].([]int64)
					if !ok {
						return false, ArgumentError{
							msg:        "argument of min func must be an int64 array",
							wrappedErr: nil,
						}
					}
					return min(arr)
				case MAX:
					arr, ok := args[0].([]int64)
					if !ok {
						return false, ArgumentError{
							msg:        "argument of max func must be an int64 array",
							wrappedErr: nil,
						}
					}
					return max(arr)
				// TODO only one func percentile(arr, num) ?
				case P99:
					arr, ok := args[0].([]int64)
					if !ok {
						return false, ArgumentError{
							msg:        "argument of percentile funcs must be an int64 array",
							wrappedErr: nil,
						}
					}
					return percentile(arr, 99)
				case P95:
					arr, ok := args[0].([]int64)
					if !ok {
						return false, ArgumentError{
							msg:        "argument of percentile funcs must be an int64 array",
							wrappedErr: nil,
						}
					}
					return percentile(arr, 95)
				case P90:
					arr, ok := args[0].([]int64)
					if !ok {
						return false, ArgumentError{
							msg:        "argument of percentile funcs must be an int64 array",
							wrappedErr: nil,
						}
					}
					return percentile(arr, 90)
				case P80:
					arr, ok := args[0].([]int64)
					if !ok {
						return false, ArgumentError{
							msg:        "argument of percentile funcs must be an int64 array",
							wrappedErr: nil,
						}
					}
					return percentile(arr, 80)
				case RANGE:
					var x, low, high int64

					x, ok = args[0].(int64)
					if !ok {
						x, _ = strconv.ParseInt(args[0].(string), 0, 64)
					}

					low, ok = args[1].(int64)
					if !ok {
						return false, ArgumentError{
							msg:        "arguments of range should be integer",
							wrappedErr: nil,
						}
					}
					high, ok = args[2].(int64)
					if !ok {
						return false, ArgumentError{
							msg:        "arguments of range should be integer",
							wrappedErr: nil,
						}
					}

					return rangeF(x, low, high), nil
				}
				return nil, NotFoundError{
					source:     fmt.Sprintf("func %s not defined", funcName),
					wrappedErr: nil,
				}

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
		return nil, OperatorError{
			msg:        fmt.Sprintf("unknown operator: %s%s", operator, right),
			wrappedErr: nil,
		}
	}
}

func evalInfixExpression(
	operator string,
	left, right interface{},
) (interface{}, error) {
	leftType := reflect.ValueOf(left).Kind()
	rightType := reflect.ValueOf(right).Kind()

	// int - int
	if leftType == reflect.Int64 && rightType == reflect.Int64 {
		return evalIntegerInfixExpression(operator, left.(int64), right.(int64))
	}
	if leftType == reflect.Int64 && rightType == reflect.Int {
		return evalIntegerInfixExpression(operator, left.(int64), int64(right.(int)))
	}
	if leftType == reflect.Int && rightType == reflect.Int64 {
		return evalIntegerInfixExpression(operator, int64(left.(int)), right.(int64))
	}
	if leftType == reflect.Int && rightType == reflect.Int {
		return evalIntegerInfixExpression(operator, int64(left.(int)), int64(left.(int)))
	}

	// int - float, convert int64 to float64, data loss for big int64 numbers
	if leftType == reflect.Int64 && rightType == reflect.Float64 {
		return evalFloatInfixExpression(operator, float64(left.(int64)), right.(float64))
	}
	if leftType == reflect.Float64 && rightType == reflect.Int64 {
		return evalFloatInfixExpression(operator, left.(float64), float64(right.(int64)))
	}

	// float - float
	if leftType == reflect.Float64 && rightType == reflect.Float64 {
		return evalFloatInfixExpression(operator, left.(float64), right.(float64))
	}

	// string - int
	if leftType == reflect.String && rightType == reflect.Int64 {
		leftInt, _ := strconv.ParseInt(left.(string), 0, 64)
		return evalIntegerInfixExpression(operator, leftInt, right.(int64))
	}
	if leftType == reflect.Int64 && rightType == reflect.String {
		rightInt, _ := strconv.ParseInt(right.(string), 0, 64)
		return evalIntegerInfixExpression(operator, left.(int64), rightInt)
	}

	// other types
	if operator == "==" {
		return reflect.DeepEqual(left, right), nil
	}

	if operator == "!=" {
		return !reflect.DeepEqual(left, right), nil
	}

	if operator == "&&" {
		if leftType == reflect.Bool && rightType == reflect.Bool {
			return left.(bool) && right.(bool), nil
		}
		return nil, OperatorError{
			msg:        fmt.Sprintf("operator && unsupported for types: %s and %s", leftType, rightType),
			wrappedErr: nil,
		}
	}

	if operator == "||" {
		if leftType == reflect.Bool && rightType == reflect.Bool {
			return left.(bool) || right.(bool), nil
		}
		return nil, OperatorError{
			msg:        fmt.Sprintf("operator || unsupported for types: %s and %s", leftType, rightType),
			wrappedErr: nil,
		}
	}

	return nil, OperatorError{
		msg:        fmt.Sprintf("unknown operator: evalInfixExpression %s", operator),
		wrappedErr: nil,
	}
}

func evalBangOperatorExpression(right interface{}) (bool, error) {
	b, ok := right.(bool)
	if ok {
		return !b, nil
	}

	return false, OperatorError{
		msg:        fmt.Sprintf("identifier before ! operator must be bool, %s", right),
		wrappedErr: nil,
	}
}

func evalMinusPrefixOperatorExpression(right interface{}) (interface{}, error) {
	i, ok := right.(int64)
	if ok {
		return -i, nil
	}

	var j float64
	j, ok = right.(float64)
	if ok {
		return -j, nil
	}

	if !ok {
		return 0, OperatorError{
			msg:        fmt.Sprintf("- operator not applicable for %v", right),
			wrappedErr: nil,
		}
	}

	return -i, nil
}

func evalFloatInfixExpression(operator string,
	left, right float64,
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
		return 0, OperatorError{
			msg:        fmt.Sprintf("unknown operator %s for floats", operator),
			wrappedErr: nil,
		}
	}
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
		return 0, OperatorError{
			msg:        fmt.Sprintf("unknown operator %s for integers", operator),
			wrappedErr: nil,
		}
	}
}

func evalIdentifier(
	node *ast.Identifier,
	env *AssertEnv,
	receivedMap map[string]interface{},
) (interface{}, error) {
	ident := node.Value
	if strings.EqualFold(ident, "status_code") {
		receivedMap[ident] = env.StatusCode
		return env.StatusCode, nil
	}
	if strings.EqualFold(ident, "response_size") {
		receivedMap[ident] = env.ResponseSize
		return env.ResponseSize, nil
	}
	if strings.EqualFold(ident, "response_time") {
		receivedMap[ident] = env.ResponseTime
		return env.ResponseTime, nil
	}
	if strings.EqualFold(ident, "body") {
		receivedMap[ident] = env.Body
		return env.Body, nil
	}

	// test-wide identifiers
	if strings.EqualFold(ident, "fail_count") {
		receivedMap[ident] = env.FailCount
		return env.FailCount, nil
	}
	if strings.EqualFold(ident, "fail_count_perc") {
		receivedMap[ident] = env.FailCountPerc
		return env.FailCountPerc, nil
	}
	if strings.EqualFold(ident, "iteration_duration") {
		receivedMap[ident] = env.TotalTime
		return env.TotalTime, nil
	}

	if strings.HasPrefix(ident, "variables.") {
		vr := strings.TrimPrefix(ident, "variables.")
		if v, ok := env.Variables[vr]; ok {
			receivedMap[ident] = v
			return v, nil
		}
		return "", NotFoundError{
			source:     fmt.Sprintf("variable not found %s", vr),
			wrappedErr: nil,
		}
	}
	if strings.HasPrefix(ident, "headers.") {
		vr := strings.TrimPrefix(ident, "headers.")
		hv := env.Headers.Get(vr)
		if hv != "" {
			receivedMap[ident] = hv
			return hv, nil
		}
		return "", NotFoundError{ //
			source:     fmt.Sprintf("header not found %s", vr),
			wrappedErr: nil,
		}
	}

	return "", NotFoundError{ //
		source:     fmt.Sprintf("%s not defined", ident),
		wrappedErr: nil,
	}
}

func evalExpressions(
	exps []ast.Expression,
	env *AssertEnv,
	receivedMap map[string]interface{},
) ([]interface{}, error) {
	var result []interface{}

	for _, e := range exps {
		evaluated, err := Eval(e, env, receivedMap)
		if err != nil {
			return nil, err
		}
		switch e.(type) {
		case *ast.Identifier:
			receivedMap[e.String()] = evaluated
		case *ast.CallExpression:
			receivedMap[e.String()] = evaluated
		}

		result = append(result, evaluated)
	}

	return result, nil
}

type NotFoundError struct { // UnWrappable
	source     string
	wrappedErr error
}

func (nf NotFoundError) Error() string {
	return fmt.Sprintf("%s", nf.source)
}

func (nf NotFoundError) Unwrap() error {
	return nf.wrappedErr
}

type ArgumentError struct { // UnWrappable
	msg        string
	wrappedErr error
}

func (nf ArgumentError) Error() string {
	return fmt.Sprintf("%s", nf.msg)
}

func (nf ArgumentError) Unwrap() error {
	return nf.wrappedErr
}

type OperatorError struct { // UnWrappable
	msg        string
	wrappedErr error
}

func (nf OperatorError) Error() string {
	return fmt.Sprintf("%s", nf.msg)
}

func (nf OperatorError) Unwrap() error {
	return nf.wrappedErr
}
