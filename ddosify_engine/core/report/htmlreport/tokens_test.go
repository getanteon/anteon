package htmlreport

import (
	"bytes"
	"strings"
	"testing"

	"go.ddosify.com/ddosify/v2/core/llm"
)

func TestTokenSection(t *testing.T) {
	d := &llm.Dist{Count: 3, P50: 20, P90: 25, P95: 27, P99: 30, Max: 31, Mean: 21}
	r := Report{LLM: &llm.Summary{Tokens: &llm.OutputStats{Unit: "tokens", Requests: 3, Total: 900, Output: d, TimePer: d, OutputPerSec: d, SystemPerSec: 450}, ITL: d},
		Schedule: &llm.ScheduleSummary{Iterations: 3, Lag: d}, PrefixCache: "busted"}
	var b bytes.Buffer
	if err := Write(&b, r); err != nil {
		t.Fatal(err)
	}
	page := b.String()
	for _, want := range []string{`id="tokens"`, "TPOT, ms per output token", "900 tokens over 3 answers", "Prefix cache: busted.", "Scheduling lag, ms"} {
		if !strings.Contains(page, want) {
			t.Errorf("token section misses %q", want)
		}
	}

	r.LLM = &llm.Summary{Chunks: &llm.OutputStats{Unit: "chunks", Requests: 3, Total: 90, Output: d, OutputPerSec: d}, ITL: d}
	b.Reset()
	if err := Write(&b, r); err != nil {
		t.Fatal(err)
	}
	page = b.String()
	page = page[:strings.Index(page, `<script type="application/json"`)] // the chart code names TPOT for runs that have it
	if strings.Contains(page, "TPOT") || strings.Contains(page, "Output per answer (tokens)") || !strings.Contains(page, "90 chunks over 3 answers") {
		t.Errorf("chunks must not read as tokens")
	}
}
