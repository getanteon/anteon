package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/types"
)

type liveRecorder struct {
	mu      sync.Mutex
	plan    live.Plan
	updates int
	final   *live.Final
}

func (r *liveRecorder) Start(p live.Plan)      { r.mu.Lock(); r.plan = p; r.mu.Unlock() }
func (r *liveRecorder) Update(s live.Snapshot) { r.mu.Lock(); r.updates++; r.mu.Unlock() }
func (r *liveRecorder) Finish(f live.Final)    { r.mu.Lock(); r.final = &f; r.mu.Unlock() }

func TestEnginePublishesToObservers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"text-delta\"}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()

	h := newDummyHammer()
	h.TestDuration = 1
	h.IterationCount = 5
	h.Scenario.Steps[0].URL = server.URL
	h.Scenario.Steps[0].Stream = types.DefaultStreamConf()

	es, err := InitEngineServices(h)
	if err != nil {
		t.Fatal(err)
	}
	rec := &liveRecorder{}
	es.Observers = []live.Observer{rec}
	e, err := NewEngine(context.Background(), h, es)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Init(); err != nil {
		t.Fatal(err)
	}
	e.Start()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.plan.Iterations != 5 || len(rec.plan.Steps) != 1 || !rec.plan.Steps[0].Streamed {
		t.Errorf("unexpected plan %+v", rec.plan)
	}
	if rec.updates < 2 || rec.final == nil {
		t.Fatalf("updates %d final %v", rec.updates, rec.final)
	}
	f := rec.final
	if f.Status != "done" || !f.Passed || f.Result == nil || f.Last.Completed != 5 || f.Last.Total.TTFT.Count != 5 {
		t.Errorf("unexpected final %+v", f.Last)
	}
}
