package injection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
			buff, err := replacer.InjectEnvIntoBuffer(test.target, test.envs, nil)

			if err != nil {
				t.Errorf("injection failed %v", err)
			}

			if !reflect.DeepEqual(buff.String(), test.expected) {
				t.Errorf("injection unsuccessful, expected : %s, got :%s", test.expected, buff.String())
			}
		}
		t.Run(test.name, tf)
	}
}

func ExampleEnvironmentInjector() {
	replacer := EnvironmentInjector{}
	replacer.Init()

	buff, err := replacer.InjectDynamicIntoBuffer("{{_randomInt}}", nil)
	if err == nil {
		fmt.Println(buff.String())
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

	buff, err := replacer.InjectEnvIntoBuffer(payload, envs, nil)

	if err != nil {
		t.Errorf("injection failed %v", err)
	}

	if !reflect.DeepEqual(buff.String(), expected.String()) {
		t.Errorf("injection unsuccessful, expected : %s, got :%s", expected.String(), buff.String())
	}

}

func TestOSEnvInjection(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()

	actualEnvVal := os.Getenv("PATH")

	envs := map[string]interface{}{
		"key1": "val1",
	}

	val, err := replacer.getEnv(envs, "$PATH")
	if err != nil {
		t.Errorf("%v", err)
	}

	found := false

	if reflect.DeepEqual(actualEnvVal, val) {
		found = true
	}

	if !found {
		t.Errorf("expected os env val not found")
	}

}

func TestGenerateBodyPieces(t *testing.T) {
	body := "test{{env1}}xyz{{env2}}" // only for env vars for now

	ei := EnvironmentInjector{}
	ei.Init()

	envs := make(map[string]interface{})
	envs["env1"] = "123"
	envs["env2"] = "456"

	pieces := ei.GenerateBodyPieces(body, envs)

	if len(pieces) != 4 {
		t.Errorf("expected 4 pieces, got %d", len(pieces))
	}

	if pieces[0].start != 0 && pieces[0].end != 4 {
		t.Errorf("expected start 0 and end 4, got %d and %d", pieces[0].start, pieces[0].end)
	}

	if pieces[1].start != 4 && pieces[1].end != 12 {
		t.Errorf("expected start 4 and end 12, got %d and %d", pieces[1].start, pieces[1].end)
	}

	if pieces[2].start != 12 && pieces[2].end != 15 {
		t.Errorf("expected start 12 and end 15, got %d and %d", pieces[2].start, pieces[2].end)
	}

	if pieces[3].start != 15 && pieces[3].end != 23 {
		t.Errorf("expected start 15 and end 23, got %d and %d", pieces[3].start, pieces[3].end)
	}

	if !pieces[1].injectable {
		t.Errorf("expected piece 1 to be injectable")
	}
	if !pieces[3].injectable {
		t.Errorf("expected piece 3 to be injectable")
	}

	if pieces[0].injectable {
		t.Errorf("expected piece 0 to not be injectable")
	}
	if pieces[2].injectable {
		t.Errorf("expected piece 2 to not be injectable")
	}

	if pieces[1].value != "123" {
		t.Errorf("expected piece 1 value to be 123")
	}
	if pieces[3].value != "456" {
		t.Errorf("expected piece 3 value to be 456")
	}

	// test content length
	// 4 + {8} + 3 + {8} = 23
	// 4 + {3} + 3 + {3} = 13
	if GetContentLength(pieces) != 13 {
		t.Errorf("expected content length to be 13")
	}
}

func TestDdosifyBodyReader(t *testing.T) {
	body := "test{{env1}}xyz{{env2}}" // only for env vars for now

	ei := EnvironmentInjector{}
	ei.Init()

	envs := make(map[string]interface{})
	envs["env1"] = "123"
	envs["env2"] = "456"

	pieces := ei.GenerateBodyPieces(body, envs)

	customReader := DdosifyBodyReader{
		Body:   body,
		Pieces: pieces,
	}

	byteArray := make([]byte, GetContentLength(pieces))
	n, err := customReader.Read(byteArray)

	// expect EOF

	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}

	if n != GetContentLength(pieces) {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces), n)
	}

	if string(byteArray) != "test123xyz456" {
		t.Errorf("expected test123xyz456, got %s", string(byteArray))
	}
}

func TestDdosifyBodyReaderSplitted(t *testing.T) {
	body := "test{{env1}}xyz{{env2}}" // only for env vars for now

	ei := EnvironmentInjector{}
	ei.Init()

	envs := make(map[string]interface{})
	envs["env1"] = "123"
	envs["env2"] = "456"

	pieces := ei.GenerateBodyPieces(body, envs)

	customReader := DdosifyBodyReader{
		Body:   body,
		Pieces: pieces,
	}

	firstPart := make([]byte, GetContentLength(pieces)-5)
	n, err := customReader.Read(firstPart)

	// do not expect EOF here
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if n != GetContentLength(pieces)-5 {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces)-5, n)
	}

	if string(firstPart) != "test123x" {
		t.Errorf("expected test123x, got %s", string(firstPart))
	}

	secondPart := make([]byte, 5)
	n, err = customReader.Read(secondPart)

	// expect EOF here

	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}

	if n != 5 {
		t.Errorf("expected to read %d bytes, read %d", 5, n)
	}

	if string(secondPart) != "yz456" {
		t.Errorf("expected yz456, got %s", string(secondPart))
	}
}
