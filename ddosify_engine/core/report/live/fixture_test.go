package live

import (
	"time"
)

var testPlans = []StagePlan{
	{Name: "warm-up", Warmup: true, Duration: 2 * time.Minute, Count: 6},
	{Name: "6/min", Duration: 5 * time.Minute, Count: 30},
	{Name: "12/min", Duration: 5 * time.Minute, Count: 60},
	{Name: "24/min", Duration: 5 * time.Minute, Count: 120},
	{Name: "48/min", Duration: 5 * time.Minute, Count: 240},
	{Name: "96/min", Duration: 5 * time.Minute, Count: 480},
}

var testThresholds = []Threshold{{Name: "first token", Ms: 10000}, {Name: "total", Ms: 30000}}

// synthetic is the i-th snapshot of a run in its fourth stage, deterministic.
func synthetic(i int) Snapshot {
	f := float64(i)
	return Snapshot{
		At:       time.Date(2026, 11, 6, 13, 0, 0, 0, time.UTC).Add(time.Duration(i) * 250 * time.Millisecond),
		Elapsed:  17*time.Minute + 36*time.Second + time.Duration(i)*250*time.Millisecond,
		Duration: 27 * time.Minute,
		Stage: StageRef{
			Index: 3, Count: 6, Name: "24/min",
			Elapsed:  2*time.Minute + 36*time.Second + time.Duration(i)*250*time.Millisecond,
			Duration: 5 * time.Minute,
		},
		Stages:      testPlans,
		TargetRPS:   0.4,
		AchievedRPS: 0.30 + float64(i%10)*0.01,
		InFlight:    12,
		Iterations:  1234 + int64(i),
		TTFT:        Latency{Count: 1100, P50: 1204, P90: 2310, P95: 2000 + f*10, P99: 4980},
		ITL:         Latency{Count: 1100, P50: 41, P90: 88, P95: 120, P99: 210},
		E2E:         Latency{Count: 1100, P50: 9870, P90: 14200, P95: 17000, P99: 21300},
		Goodput:     Goodput{Good: 1120, Total: 1199},
		Outcomes: Outcomes{
			OK: 1120, RateLimited: 31, Unavailable: 4, ServerError: 2, Timeout: 3, StreamCut: 1,
		},
	}
}

func feed(m *Model, n int) {
	for i := 0; i < n; i++ {
		m.Update(snapshotMsg(synthetic(i)))
	}
}
