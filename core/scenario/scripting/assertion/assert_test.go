package assertion

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/evaluator"
)

func TestAssert(t *testing.T) {
	testHeader := http.Header{}
	testHeader.Add("Content-Type", "application/json")
	testHeader.Add("content-length", "222")

	tests := []struct {
		input         string
		envs          *evaluator.AssertEnv
		expected      bool
		received      map[string]interface{}
		expectedError string
	}{
		{
			input: "response_size < 300",
			envs: &evaluator.AssertEnv{
				ResponseSize: 200,
			},
			expected: true,
		},
		{
			input: "response_size < 300.5",
			envs: &evaluator.AssertEnv{
				ResponseSize: 200,
			},
			expected: true,
		},
		{
			input: "-response_size < 300.5",
			envs: &evaluator.AssertEnv{
				ResponseSize: 200,
			},
			expected: true,
		},
		{
			input: "in(status_code,[200,201])",
			envs: &evaluator.AssertEnv{
				StatusCode: 500,
			},
			expected: false,
			received: map[string]interface{}{
				"status_code": int64(500),
			},
		},
		{
			input: "in(status_code,[200,201])",
			envs: &evaluator.AssertEnv{
				StatusCode: 201,
			},
			expected: true,
		},
		{
			input: "equals(status_code,200)",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: true,
		},
		{
			input: "status_code == 200",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: true,
		},
		{
			input: "not(status_code == 500)",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: true,
		},
		{
			input: `equals(json_path("employees.0.name"),"Kenan")`,
			envs: &evaluator.AssertEnv{
				Body: "{\n  \"employees\": [{\"name\":\"Kenan\"}, {\"name\":\"Kursat\"}]\n}",
			},
			expected: true,
		},
		{
			input: `equals(json_path("employees.1.name"),"Kursat")`,
			envs: &evaluator.AssertEnv{
				Body: "{\n  \"employees\": [{\"name\":\"Kenan\"}, {\"name\":\"Kursat\"}]\n}",
			},
			expected: true,
		},
		{
			input: `has(headers.Content-Type)`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected: true,
		},
		{
			input: `has(headers.Not-Exist-Header)`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected:      false,
			expectedError: "NotFoundError",
		},
		{
			input: `contains(body,"xyz")`,
			envs: &evaluator.AssertEnv{
				Body: "xyza",
			},
			expected: true,
		},
		{
			input: `contains(body,"xyz")`,
			envs: &evaluator.AssertEnv{
				Body: "",
			},
			expected: false,
		},
		{
			input: `equals(variables.arr,["Kenan","Faruk","Cakir"])`,
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"arr": []interface{}{"Kenan", "Faruk", "Cakir"},
				},
			},
			expected: true,
		},
		{
			input: `range(headers.content-length,100,300)`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected: true,
		},
		{
			input: `range(headers.content-length,300,400)`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected: false,
		},
		{
			input: `range(headers.content-length,"300",400)`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected:      false,
			expectedError: "ArgumentError", // range params should be integer
		},
		{
			input: `range(headers.content-length,300,"400")`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected:      false,
			expectedError: "ArgumentError", // range params should be integer
		},
		{
			input:    `equals_on_file("abc","./test_files/a.txt")`,
			expected: true,
		},
		{
			input:    `equals_on_file("abcx","./test_files/a.txt")`,
			expected: false,
		},
		{
			input: "(status_code == 200) || (status_code == 201)",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: true,
		},
		{
			input: "(status_code == 200) && (status_code == 201)",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: false,
		},
		{
			input: "status_code > variables.envFloatVal", // int float comparison
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
				Variables: map[string]interface{}{
					"envFloatVal": 12.43,
				},
			},
			expected: true,
		},
		{
			input: "status_code && true",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected:      false,
			expectedError: "OperatorError", // int && bool, unsupported
		},
		{
			input: "status_code || true",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected:      false,
			expectedError: "OperatorError", // int || bool, unsupported
		},
		{
			input: "(status_code > 199) || false",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: true,
		},
		{
			input: "less_than(status_code,201)",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: true,
		},
		{
			input: `range(header.content-length,300,400)`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected:      false,
			expectedError: "NotFoundError", // should be headers....
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			// TODO add received checks
			eval, err := Assert(tc.input, tc.envs)

			if tc.expected != eval {
				t.Errorf("assert expected %t", tc.expected)
			}

			if err != nil && tc.expectedError != "" {
				if tc.expectedError == "NotFoundError" {
					var notFoundError evaluator.NotFoundError
					if !errors.As(err, &notFoundError) {
						t.Errorf("Should be evaluator.NotFoundError, got %v", err)
					}
				} else if tc.expectedError == "ArgumentError" {
					var argError evaluator.ArgumentError
					if !errors.As(err, &argError) {
						t.Errorf("Should be evaluator.ArgumentError, got %v", err)
					}
				}

			}

			if err != nil && tc.received != nil {
				assertErr := err.(AssertionError)
				if !reflect.DeepEqual(assertErr.Received(), tc.received) {
					t.Errorf("received expected %v, got %v", tc.received, assertErr.Received())
				}
			}

		})
	}

}
