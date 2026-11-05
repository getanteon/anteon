package live

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTokensInTheView(t *testing.T) {
	m := newTestModel(100, 40, true)
	s := synthetic(0)
	s.Tokens = &Tokens{Unit: "tokens", Output: Latency{Count: 10, P50: 412}, TPOT: Latency{Count: 10, P50: 21, P90: 30, P99: 44},
		ITL: Latency{Count: 4000, P50: 20, P90: 28, P99: 40}, PerSec: 1234.5}
	s.SchedulingLag = Latency{Count: 10, P50: 0.4, P99: 3}
	s.PrefixCache = "busted"
	m.Update(snapshotMsg(s))
	view := ansi.Strip(m.Render())
	for _, want := range []string{"TPOT", "sched lag", "tokens/s", "412 tokens", "prefix cache busted"} {
		if !strings.Contains(view, want) {
			t.Errorf("view misses %q:\n%s", want, view)
		}
	}
	checkFits(t, view, 100, 40)

	s.Tokens = &Tokens{Unit: "chunks", Output: Latency{Count: 10, P50: 90}, ITL: Latency{Count: 900, P50: 30}, PerSec: 300}
	s.ITLIsEventGap = true
	m.Update(snapshotMsg(s))
	view = ansi.Strip(m.Render())
	if strings.Contains(view, "TPOT") || strings.Contains(view, "tokens/s") || strings.Contains(view, "90 tokens") {
		t.Errorf("chunks must never read as tokens:\n%s", view)
	}
	if !strings.Contains(view, "chunks/s") || !strings.Contains(view, "give a tokenizer") {
		t.Errorf("chunks are labelled as chunks:\n%s", view)
	}
}

func TestTokensInNDJSON(t *testing.T) {
	s := synthetic(0)
	s.Tokens = &Tokens{Unit: "tokens", TPOT: Latency{Count: 2, P50: 20}, PerSec: 10}
	s.SchedulingLag = Latency{Count: 1, P99: 2}
	s.PrefixCache = "shared 512 tokens"
	var b bytes.Buffer
	if err := WriteNDJSON(&b, s); err != nil {
		t.Fatal(err)
	}
	var line struct {
		Tokens      struct{ Unit string }
		SchedLag    *struct{ P99 float64 } `json:"scheduling_lag_ms"`
		PrefixCache string                 `json:"prefix_cache"`
	}
	if err := json.Unmarshal(b.Bytes(), &line); err != nil || line.Tokens.Unit != "tokens" || line.SchedLag.P99 != 2 || line.PrefixCache != "shared 512 tokens" {
		t.Errorf("ndjson token fields: %v %s", err, b.String())
	}
}
