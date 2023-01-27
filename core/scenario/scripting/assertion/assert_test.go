package assertion

import (
	"net/http"
	"testing"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/evaluator"
)

func TestAssert(t *testing.T) {
	testHeader := http.Header{}
	testHeader.Add("Content-Type", "application/json")

	tests := []struct {
		input    string
		envs     *evaluator.AssertEnv
		expected bool
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
			expected: false,
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
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if tc.expected != Assert(tc.input, tc.envs) {
				t.Errorf("assert expected %t", tc.expected)
			}
		})
	}

}
