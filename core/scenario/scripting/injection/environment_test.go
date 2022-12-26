package injection

import (
	"encoding/json"
	"strings"
	"testing"

	"go.ddosify.com/ddosify/core/types/regex"
)

func TestInjectionRegexReplacer(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init(regex.EnvironmentVariableRegex)

	// injection to text target
	targetURL := "{{target}}/{{path}}/{{id}}"
	stringEnvs := map[string]interface{}{
		"target": "https://app.ddosify.com",
		"path":   "load/test-results",
		"id":     234,
	}
	expectedURL := "https://app.ddosify.com/load/test-results/234"

	// injection to flat json target
	jsonPaylaod := map[string]interface{}{
		"{{a}}":      5,
		"name":       "{{xyz}}",
		"numbers":    "{{listOfNumbers}}",
		"chars":      "{{object}}",
		"boolField":  "{{boolEnv}}",
		"intField":   "{{intEnv}}",
		"floatField": "{{floatEnv}}",
	}
	bJson, _ := json.Marshal(jsonPaylaod)
	targetJson := string(bJson)

	jsonEnvs := map[string]interface{}{
		"a":             "age",
		"xyz":           "kenan",
		"listOfNumbers": []int{23, 44, 11},
		"object":        map[string]interface{}{"abc": []string{"a,b,c"}},
		"boolEnv":       false,
		"intEnv":        52,
		"floatEnv":      52.24,
	}

	expectedJsonPayload := map[string]interface{}{
		"age":        5,
		"name":       "kenan",
		"numbers":    []int{23, 44, 11},
		"chars":      map[string]interface{}{"abc": []string{"a,b,c"}},
		"boolField":  false,
		"intField":   52,
		"floatField": 52.24,
	}
	expectedbJson, _ := json.Marshal(expectedJsonPayload)
	expectedTargetJson := string(expectedbJson)

	// injection to recusive json target
	jsonRecursivePaylaod := map[string]interface{}{
		"chars": "{{object}}",
		"nc":    map[string]interface{}{"max": "{{numVerstappen}}"},
	}
	brecursiveJson, _ := json.Marshal(jsonRecursivePaylaod)
	recursiveTargetJson := string(brecursiveJson)

	recursiveJsonEnvs := map[string]interface{}{
		"object":        map[string]interface{}{"abc": map[string]interface{}{"a": 1, "b": 1, "c": 1}},
		"numVerstappen": 33,
	}

	expectedRecursiveJsonPayload := map[string]interface{}{
		"chars": map[string]interface{}{"abc": map[string]interface{}{"a": 1, "b": 1, "c": 1}},
		"nc":    map[string]interface{}{"max": 33},
	}
	expectedRecursivebJson, _ := json.Marshal(expectedRecursiveJsonPayload)
	expectedRecursiveTargetJson := string(expectedRecursivebJson)

	// Sub Tests
	tests := []struct {
		name     string
		target   string
		expected string
		envs     map[string]interface{}
	}{
		{"String", targetURL, expectedURL, stringEnvs},
		{"JSONFlat", targetJson, expectedTargetJson, jsonEnvs},
		{"JSONRecursive", recursiveTargetJson, expectedRecursiveTargetJson, recursiveJsonEnvs},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			got, err := replacer.Inject(test.target, test.envs)

			if err != nil {
				t.Errorf("injection failed %v", err)
			}

			if !strings.EqualFold(got, test.expected) {
				t.Errorf("injection unsuccessful, expected : %s, got :%s", test.expected, got)
			}

		}
		t.Run(test.name, tf)
	}
}
