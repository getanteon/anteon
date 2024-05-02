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
			buff, err := replacer.InjectEnv(test.target, test.envs)

			if err != nil {
				t.Errorf("injection failed %v", err)
			}

			if !reflect.DeepEqual(buff, test.expected) {
				t.Errorf("injection unsuccessful, expected : %s, got :%s", test.expected, buff)
			}
		}
		t.Run(test.name, tf)
	}
}

func ExampleEnvironmentInjector() {
	replacer := EnvironmentInjector{}
	replacer.Init()

	res, err := replacer.InjectDynamic("{{_randomInt}}")
	if err == nil {
		fmt.Println(res)
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

	pieces := replacer.GenerateBodyPieces(payload, envs)
	r := DdosifyBodyReader{
		Body:   payload,
		Pieces: pieces,
	}

	res := make([]byte, 100)
	n, err := r.Read(res)

	if err != io.EOF {
		t.Errorf("injection unsuccessful, expected : %s, got :%s", expected.String(), res)
	}

	if !reflect.DeepEqual(string(res[0:n]), expected.String()) {
		t.Errorf("injection unsuccessful, expected : %s, got :%s", expected.String(), res)
	}

}

func TestConcatVariablesAndInjectAsTyped2(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()
	// injection to json payload
	payload := `{"a":["--{{number_int}}{{number_string}}--","23","{{number_int}}"]}`

	envs := map[string]interface{}{
		"number_int":    1,
		"number_string": "2",
	}

	expectedPayload := `{"a":["--12--","23",1]}`

	expected := &bytes.Buffer{}
	if err := json.Compact(expected, []byte(expectedPayload)); err != nil {
		panic(err)
	}

	pieces := replacer.GenerateBodyPieces(payload, envs)
	r := DdosifyBodyReader{
		Body:   payload,
		Pieces: pieces,
	}

	res := make([]byte, 100)
	n, err := r.Read(res)

	if err != io.EOF {
		t.Errorf("injection unsuccessful, expected : %s, got :%s", expected.String(), res)
	}

	if !reflect.DeepEqual(string(res[0:n]), expected.String()) {
		t.Errorf("injection unsuccessful, expected : %s, got :%s", expected.String(), res)
	}

}

func TestConcatVariablesAndInjectAsTypedDynamic(t *testing.T) {
	replacer := EnvironmentInjector{}
	replacer.Init()
	// injection to json payload
	payload := `{"a":["--{{_randomInt}}--{{number_string}}--","23","{{number_int}}"]}`

	envs := map[string]interface{}{
		"number_int":    1,
		"number_string": "2",
	}

	dynamicInjectFailPayload := `{"a":["--{{_randomInt}}--2--","23",1]}`

	notExpected := &bytes.Buffer{}
	if err := json.Compact(notExpected, []byte(dynamicInjectFailPayload)); err != nil {
		panic(err)
	}

	pieces := replacer.GenerateBodyPieces(payload, envs)
	r := DdosifyBodyReader{
		Body:   payload,
		Pieces: pieces,
	}

	res := make([]byte, 100)
	n, err := r.Read(res)

	if err != io.EOF {
		t.Error(err)
	}

	if reflect.DeepEqual(string(res[0:n]), notExpected.String()) {
		t.Errorf("injection unsuccessful, not expected : %s, got :%s", notExpected.String(), res)
	}

}

func TestInvalidDynamicVarInjection(t *testing.T) {
	text := "http://test.com/{{_invalidVar}}"

	replacer := EnvironmentInjector{}
	replacer.Init()

	_, err := replacer.InjectDynamic(text)
	if err == nil {
		t.Errorf("expected error not found")
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

func TestDdosifyBodyReaderSplittedPiece(t *testing.T) {
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

	firstPart := make([]byte, 2)
	n, err := customReader.Read(firstPart)

	// do not expect EOF here
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if n != 2 {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces)-5, n)
	}

	if string(firstPart) != "te" {
		t.Errorf("expected te, got %s", string(firstPart))
	}

	secondPart := make([]byte, GetContentLength(pieces)-2)
	n, err = customReader.Read(secondPart)

	// expect EOF here

	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}

	if n != GetContentLength(pieces)-2 {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces)-2, n)
	}

	if string(secondPart) != "st123xyz456" {
		t.Errorf("expected st123xyz456, got %s", string(secondPart))
	}
}

func TestDdosifyBodyReaderSplittedPiece2(t *testing.T) {
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

	firstPart := make([]byte, 2)
	n, err := customReader.Read(firstPart)

	// do not expect EOF here
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if n != 2 {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces)-5, n)
	}

	if string(firstPart) != "te" {
		t.Errorf("expected te, got %s", string(firstPart))
	}

	secondPart := make([]byte, GetContentLength(pieces))
	n, err = customReader.Read(secondPart)

	// expect EOF here
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}

	if n != GetContentLength(pieces)-2 {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces)-2, n)
	}

	if string(secondPart[0:n]) != "st123xyz456" {
		t.Errorf("expected st123xyz456, got %s", string(secondPart))
	}

	// try to read again, should be EOF
	emptyPart := make([]byte, GetContentLength(pieces))
	n, err = customReader.Read(emptyPart)

	if n != 0 {
		t.Errorf("expected to read %d bytes, read %d", 0, n)
	}

	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}

}

func TestDdosifyBodyReaderSplittedPiece3(t *testing.T) {
	body := "test{{env1}}xyz" // only for env vars for now

	ei := EnvironmentInjector{}
	ei.Init()

	envs := make(map[string]interface{})
	envs["env1"] = "123"

	pieces := ei.GenerateBodyPieces(body, envs)

	customReader := DdosifyBodyReader{
		Body:   body,
		Pieces: pieces,
	}

	firstPart := make([]byte, 2)
	n, err := customReader.Read(firstPart)

	// do not expect EOF here
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if n != 2 {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces)-5, n)
	}

	if string(firstPart) != "te" {
		t.Errorf("expected te, got %s", string(firstPart))
	}

	secondPart := make([]byte, 5)
	n, err = customReader.Read(secondPart)

	// fully read the second part, no EOF
	if err == io.EOF {
		t.Errorf("expected no EOF, got %v", err)
	}

	if n != 5 {
		t.Errorf("expected to read %d bytes, read %d", 5, n)
	}

	if string(secondPart[0:n]) != "st123" {
		t.Errorf("expected st123, got %s", string(secondPart))
	}

	// try to read again, should be EOF
	lastPart := make([]byte, GetContentLength(pieces))
	n, err = customReader.Read(lastPart)

	if n != 3 {
		t.Errorf("expected to read %d bytes, read %d", 0, n)
	}

	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}

}

func TestDdosifyBodyReaderSplittedPiece4(t *testing.T) {
	body := "test{{env1}}{{env2}}" // only for env vars for now

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

	firstPart := make([]byte, 2)
	n, err := customReader.Read(firstPart)

	// do not expect EOF here
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if n != 2 {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces)-5, n)
	}

	if string(firstPart) != "te" {
		t.Errorf("expected te, got %s", string(firstPart))
	}

	secondPart := make([]byte, 5)
	n, err = customReader.Read(secondPart)

	// fully read the second part, no EOF
	if err == io.EOF {
		t.Errorf("expected no EOF, got %v", err)
	}

	if n != 5 {
		t.Errorf("expected to read %d bytes, read %d", 5, n)
	}

	if string(secondPart[0:n]) != "st123" {
		t.Errorf("expected st123, got %s", string(secondPart))
	}

	// try to read again, should be EOF
	lastPart := make([]byte, GetContentLength(pieces))
	n, err = customReader.Read(lastPart)

	if n != 3 {
		t.Errorf("expected to read %d bytes, read %d", 0, n)
	}

	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}

}

func TestDdosifyBodyReaderSplittedPiece5(t *testing.T) {
	body := "test{{env1}}xyz" // only for env vars for now

	ei := EnvironmentInjector{}
	ei.Init()

	envs := make(map[string]interface{})
	envs["env1"] = "123"

	pieces := ei.GenerateBodyPieces(body, envs)

	customReader := DdosifyBodyReader{
		Body:   body,
		Pieces: pieces,
	}

	firstPart := make([]byte, 2)
	n, err := customReader.Read(firstPart)

	// do not expect EOF here
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if n != 2 {
		t.Errorf("expected to read %d bytes, read %d", GetContentLength(pieces)-5, n)
	}

	if string(firstPart) != "te" {
		t.Errorf("expected te, got %s", string(firstPart))
	}

	secondPart := make([]byte, 5)
	n, err = customReader.Read(secondPart)

	// fully read the second part, no EOF
	if err == io.EOF {
		t.Errorf("expected no EOF, got %v", err)
	}

	if n != 5 {
		t.Errorf("expected to read %d bytes, read %d", 5, n)
	}

	if string(secondPart[0:n]) != "st123" {
		t.Errorf("expected st123, got %s", string(secondPart))
	}

	// try to read again, should be EOF
	lastPart := make([]byte, 3)
	n, err = customReader.Read(lastPart)

	if n != 3 {
		t.Errorf("expected to read %d bytes, read %d", 0, n)
	}

	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
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

func TestGenerateBodyPiecesWithDynamicVars(t *testing.T) {
	body := "test{{env1}}xyz{{_randomInt}}"

	ei := EnvironmentInjector{}
	ei.Init()

	envs := make(map[string]interface{})
	envs["env1"] = "123"

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

	if pieces[3].start != 15 && pieces[3].end != 15+len(pieces[3].value) {
		t.Errorf("expected start 15 and end %d, got %d and %d", 15+len(pieces[3].value), pieces[3].start, pieces[3].end)
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

	// it will be random, so we can't test it
	// if pieces[3].value != "456" {
	// 	t.Errorf("expected piece 3 value to be 456")
	// }
}

func TestGenerateBodyPiecesSorted(t *testing.T) {
	body := "test{{_randomInt}}xyz{{env1}}{{env2}}{{_randomCity}}"

	ei := EnvironmentInjector{}
	ei.Init()

	envs := make(map[string]interface{})
	envs["env1"] = "123"
	envs["env2"] = "777"

	pieces := ei.GenerateBodyPieces(body, envs)

	if len(pieces) != 6 {
		t.Errorf("expected 6 pieces, got %d", len(pieces))
	}

	for i := 0; i < len(pieces)-1; i++ {
		if pieces[i].start > pieces[i+1].start {
			t.Errorf("expected pieces to be sorted by start")
		}
		if pieces[i].end > pieces[i+1].end {
			t.Errorf("expected pieces to be sorted by end")
		}

		if pieces[i].end != pieces[i+1].start {
			t.Errorf("expected pieces to be contiguous")
		}
	}
}
