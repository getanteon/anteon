package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

func poissonEngine(seed int64, stages types.TimeRunCount) *engine {
	h := newDummyHammer()
	h.LoadSpread = types.LoadSpreadPoisson
	h.Seed = seed
	h.TimeRunCountMap = stages
	h.TestDuration, h.IterationCount = 0, 0
	for _, s := range stages {
		h.TestDuration += s.Duration
		h.IterationCount += s.Count
	}
	e := &engine{hammer: h}
	e.initReqCountArr()
	return e
}

var poissonStages = types.TimeRunCount{
	{Duration: 60, Count: 30, Name: "warm-up", Warmup: true},
	{Duration: 300, Count: 300, Name: "1/s"},
	{Duration: 7, Count: 1000, Name: "burst"},
	{Duration: 30, Count: 0, Name: "idle"},
}

func TestPoissonArrivalsPerStage(t *testing.T) {
	e := poissonEngine(1, poissonStages)
	tick := time.Duration(tickerInterval) * time.Millisecond
	tickPerSecond := int(time.Second / tick)

	if len(e.arrivals) != e.hammer.IterationCount || arraySum(e.reqCountArr) != e.hammer.IterationCount {
		t.Fatalf("expected %d arrivals, got %d (ticks hold %d)", e.hammer.IterationCount, len(e.arrivals), arraySum(e.reqCountArr))
	}
	if !slices.IsSorted(e.arrivals) {
		t.Fatal("arrivals are not sorted")
	}

	start, i := 0, 0
	for _, s := range poissonStages {
		end := start + s.Duration*tickPerSecond
		if got := arraySum(e.reqCountArr[start:end]); got != s.Count {
			t.Errorf("stage %s: expected exactly %d iterations, got %d", s.Name, s.Count, got)
		}
		lo, hi := time.Duration(start)*tick, time.Duration(end)*tick
		for _, a := range e.arrivals[i : i+s.Count] {
			if a < lo || a >= hi {
				t.Fatalf("stage %s: arrival %v outside [%v, %v)", s.Name, a, lo, hi)
			}
		}
		i += s.Count
		start = end
	}

	// each arrival is counted in the tick it falls in
	perTick := make([]int, len(e.reqCountArr))
	for _, a := range e.arrivals {
		perTick[int(a/tick)]++
	}
	if !reflect.DeepEqual(perTick, e.reqCountArr) {
		t.Error("the ticks' counts do not match the arrivals")
	}

	// not even: the gaps vary, as exponential gaps do
	gaps := map[time.Duration]bool{}
	for j := 1; j < 31; j++ {
		gaps[e.arrivals[j]-e.arrivals[j-1]] = true
	}
	if len(gaps) < 20 {
		t.Errorf("arrivals look regular, %d distinct gaps of 30", len(gaps))
	}
}

func TestPoissonArrivalsSeed(t *testing.T) {
	a := poissonEngine(1, poissonStages)
	b := poissonEngine(1, poissonStages)
	if !reflect.DeepEqual(a.arrivals, b.arrivals) || !reflect.DeepEqual(a.reqCountArr, b.reqCountArr) {
		t.Error("the same seed should give the same arrivals")
	}
	c := poissonEngine(2, poissonStages)
	if reflect.DeepEqual(a.arrivals, c.arrivals) {
		t.Error("another seed should give other arrivals")
	}

	// stages are drawn independently: two equal stages do not repeat each other
	same := types.TimeRunCount{{Duration: 10, Count: 20}, {Duration: 10, Count: 20}}
	d := poissonEngine(1, same)
	for i := 0; i < 20; i++ {
		if d.arrivals[20+i]-10*time.Second != d.arrivals[i] {
			return
		}
	}
	t.Error("two equal stages got the same offsets")
}

func TestPoissonOffsets(t *testing.T) {
	d := 3 * time.Second
	offs := poissonOffsets(500, d, stageRand(7, 0))
	if len(offs) != 500 || !sort.SliceIsSorted(offs, func(i, j int) bool { return offs[i] < offs[j] }) {
		t.Fatalf("expected 500 sorted offsets, got %d", len(offs))
	}
	for _, o := range offs {
		if o < 0 || o >= d {
			t.Fatalf("offset %v outside [0, %v)", o, d)
		}
	}
	if len(poissonOffsets(0, d, stageRand(7, 0))) != 0 {
		t.Error("a stage of 0 has no arrivals")
	}
}

func TestPoissonLinearLoad(t *testing.T) {
	h := newDummyHammer()
	h.LoadSpread = types.LoadSpreadPoisson
	h.Seed = 1
	h.TestDuration = 5
	h.IterationCount = 17
	e := &engine{hammer: h}
	e.initReqCountArr()
	if len(e.arrivals) != 17 || arraySum(e.reqCountArr) != 17 || e.arrivals[16] >= 5*time.Second {
		t.Errorf("linear poisson: %d arrivals, %d in ticks, last %v", len(e.arrivals), arraySum(e.reqCountArr), e.arrivals[16])
	}
}

func TestPoissonValidation(t *testing.T) {
	h := newDummyHammer()
	h.LoadSpread = "uniform"
	err := h.Validate()
	if err == nil || err.Error() != "unsupported load_spread: uniform, supported: front, even, poisson" {
		t.Errorf("unexpected error %v", err)
	}
	h.LoadSpread = types.LoadSpreadPoisson
	if err := h.Validate(); err != nil {
		t.Errorf("poisson should be valid: %v", err)
	}
	h.LoadType = types.LoadTypeWaved
	if err := h.Validate(); err == nil {
		t.Error("poisson with a waved load and no manual_load should be refused")
	}
}

// The intended start of an iteration is its drawn time, so the scheduling lag is measured from it.
func TestPoissonIntendedStart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	h := newDummyHammer()
	h.LoadSpread = types.LoadSpreadPoisson
	h.Seed = 3
	h.TimeRunCountMap = types.TimeRunCount{{Duration: 1, Count: 15, Name: "a"}, {Duration: 1, Count: 5, Name: "b"}}
	h.TestDuration, h.IterationCount = 2, 20
	h.Scenario.Steps[0].URL = server.URL

	es, err := InitEngineServices(h)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recordingReport{}
	rec.Init(false, 0)
	es.ReportServ = rec
	e, err := NewEngine(context.Background(), h, es)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Init(); err != nil {
		t.Fatal(err)
	}
	e.Start()

	if len(rec.results) != 20 {
		t.Fatalf("expected 20 iterations, got %d", len(rec.results))
	}
	zero := e.startedAt.Add(tickerInterval * time.Millisecond)
	want := map[time.Time]bool{}
	for _, a := range e.arrivals {
		want[zero.Add(a)] = true
	}
	stages := map[string]int{}
	for _, res := range rec.results {
		intended, ok := res.Others[types.OthersIntendedStart].(time.Time)
		if !ok || !want[intended] {
			t.Fatalf("intended start %v is not a drawn arrival", intended)
		}
		if lag := res.StepResults[0].RequestTime.Sub(intended); lag < 0 || lag > time.Second {
			t.Errorf("sent %v after its arrival", lag)
		}
		stages[res.Others["loadStage"].(types.LoadStage).Name]++
	}
	if stages["a"] != 15 || stages["b"] != 5 {
		t.Errorf("unexpected stage counts %v", stages)
	}
}
