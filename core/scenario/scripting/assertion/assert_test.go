package assertion

import (
	"testing"
)

func TestAssert(t *testing.T) {
	tests := []struct {
		input    string
		envs     map[string]interface{}
		expected bool
	}{
		{
			input: "response_size < 300",
			envs: map[string]interface{}{
				"response_size": int64(245),
			},
			expected: true,
		},
		{
			input: "in(status_code,[200,201])",
			envs: map[string]interface{}{
				"status_code": int64(500),
			},
			expected: false,
		},
		{
			input: "in(status_code,[200,201])",
			envs: map[string]interface{}{
				"status_code": int64(200),
			},
			expected: true,
		},
		{
			input: "equals(status_code,200)",
			envs: map[string]interface{}{
				"status_code": int64(200),
			},
			expected: true,
		},
		{
			input: "status_code == 200",
			envs: map[string]interface{}{
				"status_code": int64(200),
			},
			expected: true,
		},
		{
			input: "not(status_code == 500)",
			envs: map[string]interface{}{
				"status_code": int64(401),
			},
			expected: true,
		},
		{
			input: `equals(json_path("employees.0.name"),"Kenan")`,
			envs: map[string]interface{}{
				"body": "{\n  \"employees\": [{\"name\":\"Kenan\"}, {\"name\":\"Kursat\"}]\n}",
			},
			expected: true,
		},
		{
			input: `equals(json_path("employees.1.name"),"Kursat")`,
			envs: map[string]interface{}{
				"body": "{\n  \"employees\": [{\"name\":\"Kenan\"}, {\"name\":\"Kursat\"}]\n}",
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
