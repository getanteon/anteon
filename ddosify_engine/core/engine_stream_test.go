package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"go.ddosify.com/ddosify/core/assertion"
	"go.ddosify.com/ddosify/core/types"
)

func TestCreateEvenDistArr(t *testing.T) {
	cases := []struct {
		count int
		n     int
		want  []int
	}{
		{3, 10, []int{1, 0, 0, 1, 0, 0, 1, 0, 0, 0}},
		{10, 10, []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
		{25, 10, []int{3, 2, 3, 2, 3, 2, 3, 2, 3, 2}},
		{0, 4, []int{0, 0, 0, 0}},
	}
	for _, c := range cases {
		arr := make([]int, c.n)
		createEvenDistArr(c.count, arr)
		if !reflect.DeepEqual(arr, c.want) || arraySum(arr) != c.count {
			t.Errorf("%d over %d: expected %v, got %v", c.count, c.n, c.want, arr)
		}
	}

	// 6 a minute for 5 minutes: one every 10 seconds, not 30 in the first 30 seconds
	arr := make([]int, 300*10)
	createEvenDistArr(30, arr)
	for i, v := range arr {
		want := 0
		if i%100 == 0 {
			want = 1
		}
		if v != want {
			t.Fatalf("tick %d: expected %d, got %d", i, want, v)
		}
	}
}

func TestEvenSpreadReqCountArr(t *testing.T) {
	h := newDummyHammer()
	h.LoadSpread = types.LoadSpreadEven
	h.TestDuration = 3
	h.TimeRunCountMap = types.TimeRunCount{
		{Duration: 1, Count: 2, Name: "warm-up", Warmup: true},
		{Duration: 2, Count: 4, Name: "2/s"},
	}
	e := &engine{hammer: h}
	e.initReqCountArr()

	want := []int{1, 0, 0, 0, 0, 1, 0, 0, 0, 0,
		1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0}
	if !reflect.DeepEqual(e.reqCountArr, want) {
		t.Errorf("expected %v, got %v", want, e.reqCountArr)
	}
	if len(e.stages) != 2 || e.stageOf(9).Name != "warm-up" || !e.stageOf(9).Warmup || e.stageOf(10).Name != "2/s" || e.stageOf(30) != nil {
		t.Errorf("unexpected stages %#v", e.stages)
	}

	// unnamed stages are not tagged, so the report stays as it was
	h.TimeRunCountMap = types.TimeRunCount{{Duration: 1, Count: 2}, {Duration: 2, Count: 4}}
	e = &engine{hammer: h}
	e.initReqCountArr()
	if e.stages != nil {
		t.Errorf("stages without a name or warm-up should not be tagged: %#v", e.stages)
	}

	// linear with even spread
	h = newDummyHammer()
	h.LoadSpread = types.LoadSpreadEven
	h.TestDuration = 2
	h.IterationCount = 4
	e = &engine{hammer: h}
	e.initReqCountArr()
	want = []int{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0}
	if !reflect.DeepEqual(e.reqCountArr, want) {
		t.Errorf("linear even: expected %v, got %v", want, e.reqCountArr)
	}
}

type recordingReport struct {
	mu      sync.Mutex
	results []*types.ScenarioResult
	slo     *types.SLO
	done    chan bool
}

func (r *recordingReport) Init(debug bool, samplingRate int) error {
	r.done = make(chan bool, 1)
	return nil
}

func (r *recordingReport) SetSLO(slo *types.SLO) { r.slo = slo }

func (r *recordingReport) Start(input chan *types.ScenarioResult, _ <-chan assertion.TestAssertionResult) {
	for res := range input {
		r.mu.Lock()
		r.results = append(r.results, res)
		r.mu.Unlock()
	}
	r.done <- true
}

func (r *recordingReport) DoneChan() <-chan bool { return r.done }

func TestEngineTagsStagesAndStreams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"text-delta\",\"delta\":\"x\"}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()

	h := newDummyHammer()
	h.LoadSpread = types.LoadSpreadEven
	h.TestDuration = 2
	h.IterationCount = 4
	h.TimeRunCountMap = types.TimeRunCount{
		{Duration: 1, Count: 1, Name: "warm-up", Warmup: true},
		{Duration: 1, Count: 3, Name: "3/s"},
	}
	h.SLO = &types.SLO{FirstTokenMs: 1000}
	h.Scenario.Steps[0].URL = server.URL
	h.Scenario.Steps[0].Stream = types.DefaultStreamConf()

	es, err := InitEngineServices(h)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recordingReport{}
	rec.Init(false, 0)
	rec.SetSLO(h.SLO)
	es.ReportServ = rec

	e, err := NewEngine(context.Background(), h, es)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Init(); err != nil {
		t.Fatal(err)
	}
	e.Start()

	stages := map[string]int{}
	for _, res := range rec.results {
		st, ok := res.Others["loadStage"].(types.LoadStage)
		if !ok {
			t.Fatalf("result without a stage: %#v", res.Others)
		}
		stages[st.Name]++
		if sr := res.StepResults[0]; sr.Stream == nil || sr.Stream.End != types.StreamEndDone || !sr.Stream.HasFirstToken {
			t.Errorf("stream metrics missing: %#v", sr.Stream)
		}
	}
	if stages["warm-up"] != 1 || stages["3/s"] != 3 {
		t.Errorf("unexpected stage counts %v", stages)
	}
}
