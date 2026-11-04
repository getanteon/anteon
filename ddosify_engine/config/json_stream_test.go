package config

import (
	"testing"

	"go.ddosify.com/ddosify/core/types"
)

func TestCreateHammerStream(t *testing.T) {
	t.Parallel()
	jsonReader, err := NewConfigReader(readConfigFile("config_testdata/config_stream.json"), ConfigTypeJson)
	if err != nil {
		t.Fatal(err)
	}
	h, err := jsonReader.CreateHammer()
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Validate(); err != nil {
		t.Fatal(err)
	}

	if h.LoadSpread != types.LoadSpreadEven || h.SLO == nil || h.SLO.FirstTokenMs != 10000 || h.SLO.TotalMs != 30000 {
		t.Errorf("unexpected hammer %#v %#v", h.LoadSpread, h.SLO)
	}
	if h.IterationCount != 94 || h.TestDuration != 720 {
		t.Errorf("manual_load should fill iteration_count and duration, got %d %d", h.IterationCount, h.TestDuration)
	}
	if st := h.TimeRunCountMap[0]; st.Name != "warm-up" || !st.Warmup || h.TimeRunCountMap[1].Warmup {
		t.Errorf("unexpected stages %#v", h.TimeRunCountMap)
	}

	steps := h.Scenario.Steps
	if steps[0].Stream != nil {
		t.Error("a step without stream config has no stream")
	}
	s2 := steps[1].Stream
	if s2 == nil || s2.Type != types.StreamTypeSSE || s2.Done != "[DONE]" || s2.MaxBodyBytes != types.DefaultStreamMaxBodyBytes ||
		s2.FirstToken.JsonPath != "type" || *s2.FirstToken.Equals != "text-delta" || *s2.Error.Equals != "error" {
		t.Errorf("defaults expected for step 2: %#v", s2)
	}
	s3 := steps[2].Stream
	if s3 == nil || s3.Type != types.StreamTypeSSE || s3.Done != "" || s3.MaxBodyBytes != 4096 ||
		s3.FirstToken.JsonPath != "choices.0.delta.content" || s3.FirstToken.Equals != nil || s3.Error.Event != "error" {
		t.Errorf("overrides expected for step 3: %#v", s3)
	}
}

func TestCreateHammerStreamInvalid(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"stream type":  `{"steps":[{"id":1,"url":"http://a.b","stream":{"type":"websocket"}}]}`,
		"regexp":       `{"steps":[{"id":1,"url":"http://a.b","stream":{"first_token":{"regexp":"("}}}]}`,
		"load spread":  `{"load_spread":"random","steps":[{"id":1,"url":"http://a.b"}]}`,
		"slo negative": `{"slo":{"total_ms":-1},"steps":[{"id":1,"url":"http://a.b"}]}`,
	}
	for name, c := range cases {
		jsonReader, err := NewConfigReader([]byte(c), ConfigTypeJson)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		h, err := jsonReader.CreateHammer()
		if err == nil {
			err = h.Validate()
		}
		if err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
}
