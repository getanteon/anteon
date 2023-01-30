package assertion

import (
	"net/http"
	"testing"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/evaluator"
)

func TestAssert(t *testing.T) {
	testHeader := http.Header{}
	testHeader.Add("Content-Type", "application/json")
	testHeader.Add("content-length", "222")

	tests := []struct {
		input       string
		envs        *evaluator.AssertEnv
		expected    bool
		shouldError bool
	}{
		{
			input: "response_size < 300",
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
			shouldError: true, // ident not found
			expected:    false,
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
			expected:    false,
			shouldError: true, // range params should be integer
		},
		{
			input:       `equals_on_file("abc","./test_files/a.txt")`,
			expected:    true,
			shouldError: false,
		},
		{
			input:       `equals_on_file("abcx","./test_files/a.txt")`,
			expected:    false,
			shouldError: false,
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
			input: "status_code && true", // int && bool, unsupported
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected:    false,
			shouldError: true,
		},
		{
			input: "status_code || true", // int || bool, unsupported
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected:    false,
			shouldError: true,
		},
		{
			input: "(status_code > 199) || false",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			// TODO add received checks
			eval, _ := Assert(tc.input, tc.envs)

			if tc.expected != eval {
				t.Errorf("assert expected %t", tc.expected)
			}
		})
	}

}
