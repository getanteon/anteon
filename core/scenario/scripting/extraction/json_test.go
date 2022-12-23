package extraction

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestJsonExtractFromString(t *testing.T) {
	json := `{"name":{"first":"Janet","last":"Prichard"},"age":47}`
	je := JsonExtractor{}

	val, _ := je.extractFromString(json, "name.last")

	if val != "Prichard" {
		t.Errorf("Json Extract Error")
	}
}

func TestJsonExtractFromByteSlice_String(t *testing.T) {
	payload := map[string]interface{}{
		"name": map[string]interface{}{
			"first": "Janet",
			"last":  "Prichard",
		},
		"age": 47,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "name.last")

	if val != "Prichard" {
		t.Errorf("Json Extract Error")
	}
}

func TestJsonExtractFromByteSlice_Object(t *testing.T) {
	expected := map[string]interface{}{
		"first": "Janet",
		"last":  "Prichard",
	}
	payload := map[string]interface{}{
		"name": expected,
		"age":  47,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "name")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtractFromByteSlice_Object failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtractFromByteSlice_Float(t *testing.T) {
	var expected float64 = 52
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtractFromByteSlice_Object failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtractFromByteSlice_Int(t *testing.T) {
	var expected int = 52
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	val2 := val.(float64) // json number -> float64
	if !reflect.DeepEqual(int(val2), expected) {
		t.Errorf("TestJsonExtractFromByteSlice_Object failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtractFromByteSlice_Nil(t *testing.T) {
	payload := map[string]interface{}{
		"age": nil,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, nil) {
		t.Errorf("TestJsonExtractFromByteSlice_Nil failed, expected %#v, found %#v", nil, val)
	}
}

func TestJsonExtractFromByteSlice_Bool(t *testing.T) {
	expected := true
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtractFromByteSlice_Bool failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtractFromByteSlice_JsonArray(t *testing.T) {
	expected := []string{"a", "b"}
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtractFromByteSlice_JsonArray failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtractFromByteSlice_JsonIntArray(t *testing.T) {
	expected := []int{2, 4}
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	expectedFloat := []float64{2, 4}
	if !reflect.DeepEqual(val, expectedFloat) {
		t.Errorf("TestJsonExtractFromByteSlice_JsonArray failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtractFromByteSlice_JsonFloatArray(t *testing.T) {
	expected := []float64{2.33, 4.55}
	payload := map[string]interface{}{
		"age": expected,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.extractFromByteSlice(byteSlice, "age")

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("TestJsonExtractFromByteSlice_JsonArray failed, expected %#v, found %#v", expected, val)
	}
}

func TestJsonExtractFromByteSlice_JsonPathNotFound(t *testing.T) {
	payload := map[string]interface{}{
		"age": "24",
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, err := je.extractFromByteSlice(byteSlice, "age2")

	expected := "json path not found"
	if !strings.EqualFold(err.Error(), expected) {
		t.Errorf("TestJsonExtractFromByteSlice_NotFoundJsonPath failed, expected %#v, found %#v", expected, err)
	}

	if !reflect.DeepEqual(val, "") {
		t.Errorf("TestJsonExtractFromByteSlice_NotFoundJsonPath failed, expected %#v, found %#v", expected, val)
	}
}
