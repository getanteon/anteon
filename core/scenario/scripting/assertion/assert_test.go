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
			input: "!(status_code == 200)",
			envs: &evaluator.AssertEnv{
				StatusCode: 200,
			},
			expected: false,
			received: map[string]interface{}{
				"status_code": int64(200),
			},
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
				Body: "{\n  \"employees\": [ {\"name\":\"Kursat\"}, {\"name\":\"Kenan\"}]\n}",
			},
			expected: false,
			received: map[string]interface{}{
				"json_path(employees.0.name)": "Kursat",
			},
		},
		{
			input: `equals(json_path("employees.1.name"),"Kursat")`,
			envs: &evaluator.AssertEnv{
				Body: "{\n  \"employees\": [{\"name\":\"Kenan\"}, {\"name\":\"Kursat\"}]\n}",
			},
			expected: true,
		},
		{
			input: `exists(headers.Content-Type)`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected: true,
		},
		{
			input: `exists(headers.Not-Exist-Header)`,
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
			received: map[string]interface{}{
				"body": "",
			},
		},
		{
			input: `regexp(body,"[a-z]+_[0-9]+",0) == "messi_10"`,
			envs: &evaluator.AssertEnv{
				Body: "messi_10alvarez_9",
			},
			expected: true,
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
			input: `equals(variables.arr,["Kenan","Faruk","Cakir"])`,
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"arr2": []interface{}{"Kenan", "Faruk", "Cakir"},
				},
			},
			expected:      false,
			expectedError: "NotFoundError",
		},
		{
			input: `variables.arr != ["Kenan","Faruk","Cakir"]`,
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"arr": []interface{}{"Cakir"},
				},
			},
			expected: true,
		},
		{
			input: `variables.arr !=["Kenan","Faruk","Cakir"])`,
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"arr": []interface{}{"Kenan", "Faruk", "Cakir"},
				},
			},
			expected: false,
			received: map[string]interface{}{
				"variables.arr": []interface{}{"Kenan", "Faruk", "Cakir"},
			},
		},
		{
			input: `equals(variables.xint,100)`, // int - int64 comparison
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"xint": 100,
				},
			},
			expected: true,
		},
		{
			input: `equals(100,variables.xint)`, // int - int64 comparison
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"xint": 100,
				},
			},
			expected: true,
		},
		{
			input:    `2*12/3+5-3 != 10`,
			expected: false,
		},
		{
			input: `equals(variables.xint,variables.yint)`, // int - int comparison
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"xint": 100,
					"yint": 100,
				},
			},
			expected: true,
		},
		{
			input:    `equals(100.5 + 200.5, 301)`, // float64 +
			envs:     &evaluator.AssertEnv{},
			expected: true,
		},
		{
			input:    `equals(100.5 - 200.5, -100)`, // float64 -
			envs:     &evaluator.AssertEnv{},
			expected: true,
		},
		{
			input:    `equals(4.0 * 10.5, 42)`, // float64 *
			envs:     &evaluator.AssertEnv{},
			expected: true,
		},
		{
			input:    `equals(60.0/5, 12)`, // float64 /
			envs:     &evaluator.AssertEnv{},
			expected: true,
		},
		{
			input:    `60.1 == 60.1`, // float64 ==
			envs:     &evaluator.AssertEnv{},
			expected: true,
		},
		{
			input:    `60.1 != 60.1`, // float64 !=
			envs:     &evaluator.AssertEnv{},
			expected: false,
		},
		{
			input:    `60.1 £ 60.1`, // illegal character
			envs:     &evaluator.AssertEnv{},
			expected: false,
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
			input:    `equals_on_file(variables.xyz,"./test_files/jsonMap.json")`,
			expected: true,
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"xyz": map[string]interface{}{
						"ask":                  130.75,
						"askSize":              float64(10),
						"averageAnalystRating": "2.0 - Buy",
					},
				},
			},
		},
		{
			input:    `equals_on_file(variables.xyz,"./test_files/jsonArray.json")`,
			expected: true,
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"xyz": []interface{}{"xyz", "abc"},
				},
			},
		},
		{
			input:    `equals_on_file(body,"./test_files/currencies.json")`,
			expected: true,
			envs: &evaluator.AssertEnv{
				Body: "[\n    \"AED\",\n    \"ARS\",\n    \"AUD\",\n    \"BGN\",\n    \"BHD\",\n    \"BRL\",\n    \"CAD\",\n    \"CHF\",\n    \"CNY\",\n    \"DKK\",\n    \"DZD\",\n    \"EUR\",\n    \"FKP\",\n    \"INR\",\n    \"JEP\",\n    \"JPY\",\n    \"KES\",\n    \"KWD\",\n    \"KZT\",\n    \"MXN\",\n    \"NZD\",\n    \"RUB\",\n    \"SEK\",\n    \"SGD\",\n    \"TRY\",\n    \"USD\"\n]",
			},
		},
		{
			input:    `equals_on_file(body,"./test_files/number.json")`,
			expected: true,
			envs: &evaluator.AssertEnv{
				Body: "5",
			},
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
		{
			input: "greater_than(status_code,201)",
			envs: &evaluator.AssertEnv{
				StatusCode: 400,
			},
			expected: true,
		},
		{
			input: `less_than(headers.content-length,500)`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected: true,
		},
		{
			input: "exists(headers.Content-Type2)",
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected: false,
		},
		{
			input: `in(headers.content-length,[222,445])`,
			envs: &evaluator.AssertEnv{
				Headers: testHeader,
			},
			expected: true,
		},
		{
			input: "equals(variables.x, -48.880005)",
			envs: &evaluator.AssertEnv{
				Variables: map[string]interface{}{
					"x": float64(-48.880005),
				},
			},
			expected: true,
		},
		{
			input: `equals(xml_path("//item/title"),"ABC")`,
			envs: &evaluator.AssertEnv{
				Body: `<?xml version="1.0" encoding="UTF-8" ?>
		<rss version="2.0">
		<channel>
		  <item>
			<title>ABC</title>
		  </item>
		</channel>
		</rss>`,
			},

			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			eval, err := Assert(tc.input, tc.envs)

			if tc.expected != eval {
				t.Errorf("assert expected %t", tc.expected)
				t.Log(err)
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
				} else if tc.expectedError == "OperatorError" {
					var opError evaluator.OperatorError
					if !errors.As(err, &opError) {
						t.Errorf("Should be evaluator.OperatorError, got %v", err)
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
