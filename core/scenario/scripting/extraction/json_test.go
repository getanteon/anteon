package extraction

import (
	"encoding/json"
	"testing"
)

func TestJsonExtractFromString(t *testing.T) {
	json := `{"name":{"first":"Janet","last":"Prichard"},"age":47}`
	je := JsonExtractor{}

	val, _ := je.ExtractFromString(json, "name.last")

	if val != "Prichard" {
		t.Errorf("Json Extract Error")
	}
}

func TestJsonExtractFromByteSlice(t *testing.T) {
	payload := map[string]interface{}{
		"name": map[string]interface{}{
			"first": "Janet",
			"last":  "Prichard",
		},
		"age": 47,
	}

	byteSlice, _ := json.Marshal(payload)
	je := JsonExtractor{}
	val, _ := je.ExtractFromByteSlice(byteSlice, "name.last")

	if val != "Prichard" {
		t.Errorf("Json Extract Error")
	}
}
