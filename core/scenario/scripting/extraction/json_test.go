package extraction

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestJsonExtract_String(t *testing.T) {
	payload := map[string]interface{}{
		"name": map[string]interface{}{
			"first": "Janet",
			"last":  "Prichard",
		},
		"age": 47,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "name.last")

	if val != "Prichard" {
		t.Errorf("Json Extract Error")
	}

	val, _ = je.extractFromString(string(byteSlice), "name.last")

	if val != "Prichard" {
		t.Errorf("Json Extract Error")
	}
}

func TestJsonExtract_Object(t *testing.T) {
	expected := map[string]interface{}{
		"first": "Janet",
		"last":  "Prichard",
	}
	payload := map[string]interface{}{
		"name": expected,
		"age":  47,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "name")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_Object failed, expected %#v, found %#v", expected, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "name")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_Object failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_Float(t *testing.T) {
	var expected float64 = 52.2
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	val2 := val.(float64) // json number -> float64
	if !reflect.DeepEqual(val2, expected) {
		t.Errorf("TestJsonExtract_Float failed, expected %#v, found %#v", expected, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "age")

	val22 := val.(float64) // json number -> float64
	if !reflect.DeepEqual(val22, expected) {
		t.Errorf("TestJsonExtract_Float failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_Int(t *testing.T) {
	var expected int = 52
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	val2 := val.(int64) // json number -> float64
	if !reflect.DeepEqual(int(val2), expected) {
		t.Errorf("TestJsonExtract_Int failed, expected %#v, found %#v", expected, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "age")

	val22 := val.(int64) // json number -> float64
	if !reflect.DeepEqual(int(val22), expected) {
		t.Errorf("TestJsonExtract_Int failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_Nil(t *testing.T) {
	payload := map[string]interface{}{
		"age": nil,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, nil) {
		t.Errorf("TestJsonExtract_Nil failed, expected %#v, found %#v", nil, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "age")

	if !reflect.DeepEqual(val, nil) {
		t.Errorf("TestJsonExtract_Nil failed, expected %#v, found %#v", nil, val)
	}
}

func TestJsonExtract_Bool(t *testing.T) {
	je := jsonExtractor{}
	expected := true
	expected1 := false

	payload := map[string]interface{}{
		"age":  expected,
		"age1": expected1,
	}

	byteSlice, _ := json.Marshal(payload)
	val, _ := je.extractFromByteSlice(byteSlice, "age")
	val1, _ := je.extractFromByteSlice(byteSlice, "age1")

	if !reflect.DeepEqual(val, expected) || !reflect.DeepEqual(val1, expected1) {
		t.Errorf("TestJsonExtract_Bool failed, expected %#v, found %#v", expected, val)
	}

	expected = false
	payload = map[string]interface{}{
		"age": expected,
	}
	byteSlice, _ = json.Marshal(payload)

	val, _ = je.extractFromString(string(byteSlice), "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_Bool failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_JsonArray(t *testing.T) {
	expected := []string{"a", "b"}
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_JsonArray failed, expected %#v, found %#v", expected, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_JsonArray failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_JsonIntArray(t *testing.T) {
	expected := []int{2, 4}
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	expectedFloat := []float64{2, 4}
	if !reflect.DeepEqual(val, expectedFloat) {
		t.Errorf("TestJsonExtract_JsonIntArray failed, expected %#v, found %#v", expected, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "age")

	if !reflect.DeepEqual(val, expectedFloat) {
		t.Errorf("TestJsonExtract_JsonIntArray failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_JsonFloatArray(t *testing.T) {
	expected := []float64{2.33, 4.55}
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_JsonFloatArray failed, expected %#v, found %#v", expected, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_JsonFloatArray failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_JsonBoolArray(t *testing.T) {
	expected := []bool{true, false}
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_JsonBoolArray failed, expected %#v, found %#v", expected, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_JsonBoolArray failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_ObjectArray(t *testing.T) {
	expected := []map[string]interface{}{
		{"x": "cc"},
	}
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_JsonBoolArray failed, expected %#v, found %#v", expected, val)
	}

	val, _ = je.extractFromString(string(byteSlice), "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtract_JsonBoolArray failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtract_JsonPathNotFound(t *testing.T) {
	payload := map[string]interface{}{
		"age": "24",
	}

	byteSlice, _ := json.Marshal(payload)
	je := jsonExtractor{}
	val, err := je.extractFromByteSlice(byteSlice, "age2")

	expected := "no match for this jsonPath"
	if !strings.EqualFold(err.Error(), expected) {
		t.Errorf("TestJsonExtract_JsonPathNotFound failed, expected %#v, found %#v", expected, err)
	}

	if !reflect.DeepEqual(val, "") {
		t.Errorf("TestJsonExtract_JsonPathNotFound failed, expected %#v, found %#v", expected, val)
	}

	val, err = je.extractFromString(string(byteSlice), "age2")

	if !strings.EqualFold(err.Error(), expected) {
		t.Errorf("TestJsonExtract_JsonPathNotFound failed, expected %#v, found %#v", expected, err)
	}

	if !reflect.DeepEqual(val, "") {
		t.Errorf("TestJsonExtract_JsonPathNotFound failed, expected %#v, found %#v", expected, val)
	}
}
