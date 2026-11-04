package requester

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/hooks"
	"go.ddosify.com/ddosify/v2/core/types"
)

type wordCounter struct{ words int }

func (w *wordCounter) Event(at time.Duration, event string, data []byte) {
	if strings.Contains(string(data), "text-delta") {
		w.words++
	}
}

func (w *wordCounter) End(r *types.StreamResult) (map[string]float64, map[string]interface{}) {
	return map[string]float64{"words": float64(w.words)}, map[string]interface{}{"ANSWER_WORDS": w.words}
}

func TestHooksSeeTheStreamAndTheRequest(t *testing.T) {
	hooks.RegisterStreamObserver("test-words", func(step *types.ScenarioStep) hooks.StreamObserver {
		if step.Name != "hooked" {
			return nil
		}
		return &wordCounter{}
	})
	hooks.RegisterRequestHook("test-header", func(step *types.ScenarioStep, req *http.Request, envs map[string]interface{}) error {
		if step.Name == "hooked" {
			req.Header.Set("X-Hooked", "yes")
		}
		return nil
	})

	var gotHeader string
	srv := sseServer(t, []sseStep{{0, "data: {\"type\":\"text-delta\"}\n\ndata: {\"type\":\"text-delta\"}\n\ndata: [DONE]\n\n"}})
	srv.Config.Handler = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotHeader = r.Header.Get("X-Hooked")
			next.ServeHTTP(w, r)
		})
	}(srv.Config.Handler)
	defer srv.Close()

	res := sendStep(t, types.ScenarioStep{ID: 1, Name: "hooked", Method: http.MethodPost, URL: srv.URL, Timeout: 5,
		Stream: types.DefaultStreamConf()})
	if res.Stream == nil || res.Stream.Extra["words"] != 2 || res.ExtractedEnvs["ANSWER_WORDS"] != 2 {
		t.Errorf("observer metrics and vars missing: %#v %#v", res.Stream, res.ExtractedEnvs)
	}
	if gotHeader != "yes" {
		t.Errorf("request hook did not run, header %q", gotHeader)
	}

	other := sendStep(t, types.ScenarioStep{ID: 2, Name: "plain", Method: http.MethodPost, URL: srv.URL, Timeout: 5,
		Stream: types.DefaultStreamConf()})
	if other.Stream.Extra != nil || gotHeader != "" {
		t.Error("hooks that skip a step leave it alone")
	}
}
