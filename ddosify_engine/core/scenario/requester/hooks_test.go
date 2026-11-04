package requester

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/hooks"
	"go.ddosify.com/ddosify/v2/core/types"
)

type wordCounter struct{ words int }

func (w *wordCounter) Event(at time.Duration, event, data string) {
	if strings.Contains(data, "text-delta") {
		w.words++
	}
}

func (w *wordCounter) End(r *types.StreamResult) (map[string]float64, map[string]interface{}) {
	if r == nil {
		return nil, map[string]interface{}{"ANSWER_WORDS": -1}
	}
	return map[string]float64{"words": float64(w.words)}, map[string]interface{}{"ANSWER_WORDS": w.words}
}

type testHook struct{}

func (testHook) Prepare(r *hooks.Request) (hooks.Exchange, error) {
	r.HTTP.Header.Set("X-Hooked", "yes")
	r.Body = append([]byte("prefix:"), r.Body...)
	r.Vars["PREPARED"] = true
	return &wordCounter{}, nil
}

func TestStepHookSeesTheRequestAndTheStream(t *testing.T) {
	hooks.RegisterStepHook("test-words", func(step *types.ScenarioStep) (hooks.StepHook, error) {
		if step.Name != "hooked" {
			return nil, nil
		}
		return testHook{}, nil
	})
	defer hooks.RegisterStepHook("test-words", nil)

	var gotHeader, gotBody string
	srv := sseServer(t, []sseStep{{0, "data: {\"type\":\"text-delta\"}\n\ndata: {\"type\":\"text-delta\"}\n\ndata: [DONE]\n\n"}})
	srv.Config.Handler = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotHeader = r.Header.Get("X-Hooked")
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			next.ServeHTTP(w, r)
		})
	}(srv.Config.Handler)
	defer srv.Close()

	res := sendStep(t, types.ScenarioStep{ID: 1, Name: "hooked", Method: http.MethodPost, URL: srv.URL, Timeout: 5,
		Payload: "q", Stream: types.DefaultStreamConf()})
	if res.Stream == nil || res.Stream.Extra["words"] != 2 || res.ExtractedEnvs["ANSWER_WORDS"] != 2 || res.ExtractedEnvs["PREPARED"] != true {
		t.Errorf("exchange metrics and vars missing: %#v %#v", res.Stream, res.ExtractedEnvs)
	}
	if gotHeader != "yes" || gotBody != "prefix:q" || res.Stream.SentAt.IsZero() {
		t.Errorf("the hook did not prepare the request: %q %q", gotHeader, gotBody)
	}

	other := sendStep(t, types.ScenarioStep{ID: 2, Name: "plain", Method: http.MethodPost, URL: srv.URL, Timeout: 5,
		Stream: types.DefaultStreamConf()})
	if other.Stream.Extra != nil || gotHeader != "" {
		t.Error("a hook that skips a step leaves it alone")
	}

	// an answer that is not a stream still ends the exchange
	refused := sendStep(t, types.ScenarioStep{ID: 3, Name: "hooked", Method: http.MethodPost, URL: srv.URL + "/x", Timeout: 5,
		Stream: types.DefaultStreamConf()})
	_ = refused
	dead := sendStep(t, types.ScenarioStep{ID: 4, Name: "hooked", Method: http.MethodPost, URL: "http://127.0.0.1:1", Timeout: 1})
	if dead.ExtractedEnvs["ANSWER_WORDS"] != -1 {
		t.Errorf("a failed request ends its exchange with nil: %#v", dead.ExtractedEnvs)
	}
}
