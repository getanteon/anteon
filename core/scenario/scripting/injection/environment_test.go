package injection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestInjectionRegexReplacer(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()
	// injection to text target
	targetURL := "{{target}}/{{path}}/{{id}}/{{boolField}}/{{floatField}}/{{uuidField}}"
	uuid := uuid.New()
	stringEnvs := map[string]interface{}{
		"target":     "https://app.ddosify.com",
		"path":       "load/test-results",
		"id":         234,
		"boolField":  true,
		"floatField": 22.3,
		"uuidField":  uuid,
	}
	expectedURL := "https://app.ddosify.com/load/test-results/234/true/22.3/" + uuid.String()

	// injection to flat json target
	targetJson := `{
		"{{a}}": 5,
		"name": "{{xyz}}",
		"numbers": "{{listOfNumbers}}",
		"chars": "{{object}}",
		"boolField": "{{boolEnv}}",
		"intField": "{{intEnv}}",
		"floatField": "{{floatEnv}}"
	}`

	jsonEnvs := map[string]interface{}{
		"a":             "age",
		"xyz":           "kenan",
		"listOfNumbers": []float64{23, 44, 11},
		"object":        map[string]interface{}{"abc": []string{"a", "b", "c"}},
		"boolEnv":       false,
		"intEnv":        52,
		"floatEnv":      52.24,
	}

	expectedJsonPayload := `{
		"age": 5,
		"name": "kenan",
		"numbers": [23,44,11],
		"chars": {"abc":["a","b","c"]},
		"boolField": false,
		"intField": 52,
		"floatField": 52.24
	}`

	// injection to recusive json target
	jsonRecursivePaylaod := `{
		"chars": "{{object}}",
		"nc": {"max": "{{numVerstappen}}"}
	}`

	recursiveJsonEnvs := map[string]interface{}{
		"object":        map[string]interface{}{"abc": map[string]interface{}{"a": 1, "b": 1, "c": 1}},
		"numVerstappen": 33,
	}

	expectedRecursiveJsonPayload := `{
		"chars": {"abc":{"a":1,"b":1,"c":1}},
		"nc": {"max": 33}
	}`

	// Sub Tests
	tests := []struct {
		name     string
		target   string
		expected interface{}
		envs     map[string]interface{}
	}{
		{"String", targetURL, expectedURL, stringEnvs},
		{"JSONFlat", targetJson, expectedJsonPayload, jsonEnvs},
		{"JSONRecursive", jsonRecursivePaylaod, expectedRecursiveJsonPayload, recursiveJsonEnvs},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			got, err := replacer.InjectEnv(test.target, test.envs)

			if err != nil {
				t.Errorf("injection failed %v", err)
			}

			if !reflect.DeepEqual(got, test.expected) {
				t.Errorf("injection unsuccessful, expected : %s, got :%s", test.expected, got)
			}
		}
		t.Run(test.name, tf)
	}
}

func ExampleEnvironmentInjector() {
	replacer := EnvironmentInjector{}
	replacer.Init()

	randInt, err := replacer.InjectDynamic("{{_randomInt}}")
	if err == nil {
		fmt.Println(randInt)
	}
}

func TestRandomInjectionStringSlice(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()

	vals := []string{
		"Kenan", "Kursat", "Fatih",
	}

	envs := map[string]interface{}{
		"vals": vals,
	}

	val, err := replacer.getEnv(envs, "rand(vals)")
	if err != nil {
		t.Errorf("%v", err)
	}

	found := false

	for _, n := range vals {
		if reflect.DeepEqual(val, n) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("rand method did not return one of the expecteds")
	}
}

func TestRandomInjectionBoolSlice(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()

	vals := []bool{
		true, false, true,
	}

	envs := map[string]interface{}{
		"vals": vals,
	}

	val, err := replacer.getEnv(envs, "rand(vals)")
	if err != nil {
		t.Errorf("%v", err)
	}

	found := false

	for _, n := range vals {
		if reflect.DeepEqual(val, n) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("rand method did not return one of the expecteds")
	}

}

func TestRandomInjectionIntSlice(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()

	vals := []int{
		3, 55, 42,
	}

	envs := map[string]interface{}{
		"vals": vals,
	}

	val, err := replacer.getEnv(envs, "rand(vals)")
	if err != nil {
		t.Errorf("%v", err)
	}

	found := false

	for _, n := range vals {
		if reflect.DeepEqual(val, n) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("rand method did not return one of the expecteds")
	}

}

func TestRandomInjectionFloat64Slice(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()

	vals := []float64{
		3.3, 55.23, 42.1,
	}

	envs := map[string]interface{}{
		"vals": vals,
	}

	val, err := replacer.getEnv(envs, "rand(vals)")
	if err != nil {
		t.Errorf("%v", err)
	}

	found := false

	for _, n := range vals {
		if reflect.DeepEqual(val, n) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("rand method did not return one of the expecteds")
	}

}

func TestRandomInjectionInterfaceSlice(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()

	vals := []interface{}{
		map[string]int{"s": 33},
		[]string{"v", "c"},
	}

	envs := map[string]interface{}{
		"vals": vals,
	}

	val, err := replacer.getEnv(envs, "rand(vals)")
	if err != nil {
		t.Errorf("%v", err)
	}

	found := false

	for _, n := range vals {
		if reflect.DeepEqual(val, n) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("rand method did not return one of the expecteds")
	}

}

func TestConcatVariablesAndInjectAsTyped(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()
	// injection to json payload
	payload := `{"a":["--{{number_int}}--{{number_string}}--","23","{{number_int}}"]}`

	envs := map[string]interface{}{
		"number_int":    1,
		"number_string": "2",
	}

	expectedPayload := `{"a":["--1--2--","23",1]}`

	expected := &bytes.Buffer{}
	if err := json.Compact(expected, []byte(expectedPayload)); err != nil {
		panic(err)
	}

	got, err := replacer.InjectEnv(payload, envs)

	if err != nil {
		t.Errorf("injection failed %v", err)
	}

	if !reflect.DeepEqual(got, expected.String()) {
		t.Errorf("injection unsuccessful, expected : %s, got :%s", expected.String(), got)
	}

}
