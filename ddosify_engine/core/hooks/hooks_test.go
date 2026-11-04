package hooks

import (
	"encoding/json"
	"testing"
)

func TestApplyPresets(t *testing.T) {
	RegisterPreset("test-a", func(c map[string]interface{}) error { c["a"] = true; return nil })
	RegisterPreset("test-b", func(c map[string]interface{}) error { c["b"] = c["a"]; return nil })

	out, err := ApplyPresets([]byte(`{"presets":"test-a","x":1}`), []string{"test-b"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["a"] != true || m["b"] != true || m["x"] != float64(1) || m["presets"] != nil {
		t.Errorf("presets applied in order and the key removed, got %v", m)
	}

	same := []byte(`{"x":1}`)
	if out, err := ApplyPresets(same, nil); err != nil || string(out) != string(same) {
		t.Errorf("no presets leaves the config as it is: %s %v", out, err)
	}
	if _, err := ApplyPresets([]byte(`{"presets":["nope"]}`), nil); err == nil {
		t.Error("an unknown preset is an error")
	}
	if _, err := ApplyPresets([]byte(`{"presets":3}`), nil); err == nil {
		t.Error("presets must be names")
	}
}

func TestVariables(t *testing.T) {
	RegisterVariable("testNonce", func() interface{} { return "n1" })
	fn, ok := Variable("testNonce")
	if !ok || fn() != "n1" {
		t.Error("registered variable not found")
	}
	if _, ok := Variable("missing"); ok {
		t.Error("unregistered variable found")
	}
}
