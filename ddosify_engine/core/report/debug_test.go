package report

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	h := http.Header{}
	h.Add("Content-Type", "application/json")

	type Temp struct {
		X float64 `json:"x"`
	}

	body := Temp{
		X: 52.2,
	}

	bBody, _ := json.Marshal(body)

	_, bodyDecoded, err := decode(h, bBody)

	if err != nil {
		t.Errorf("%v", err)
	}

	expected := reflect.ValueOf(map[string]interface{}{"x": 52.2})
	ei := expected.Interface()
	if !reflect.DeepEqual(ei, bodyDecoded) {
		t.Errorf("TestDecode, expected:%s got:%s", ei, bodyDecoded)
	}

}
