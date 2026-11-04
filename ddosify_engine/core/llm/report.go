/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package llm

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

// Samples kept per metric for its percentiles; a uniform reservoir sample beyond this.
const maxSamples = 100000

type sampler struct {
	n             int64
	sum, min, max float64
	samples       []float64
	rnd           *rand.Rand
}

func (s *sampler) add(v float64) {
	if s.n == 0 || v < s.min {
		s.min = v
	}
	if s.n == 0 || v > s.max {
		s.max = v
	}
	s.n++
	s.sum += v
	if len(s.samples) < maxSamples {
		s.samples = append(s.samples, v)
		return
	}
	if s.rnd == nil {
		s.rnd = rand.New(rand.NewSource(1))
	}
	if j := s.rnd.Int63n(s.n); j < maxSamples {
		s.samples[j] = v
	}
}

// Dist is a distribution: milliseconds for times, counts for tokens.
type Dist struct {
	Count int64   `json:"count"`
	Min   float64 `json:"min"`
	Mean  float64 `json:"mean"`
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Max   float64 `json:"max"`
}

func round(v float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(v*p) / p
}

func nearestRank(sorted []float64, p float64) float64 {
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func (s *sampler) dist(digits int) *Dist {
	if s.n == 0 {
		return nil
	}
	sorted := append([]float64(nil), s.samples...)
	sort.Float64s(sorted)
	return &Dist{
		Count: s.n,
		Min:   round(s.min, digits),
		Mean:  round(s.sum/float64(s.n), digits),
		P50:   round(nearestRank(sorted, 0.50), digits),
		P90:   round(nearestRank(sorted, 0.90), digits),
		P95:   round(nearestRank(sorted, 0.95), digits),
		P99:   round(nearestRank(sorted, 0.99), digits),
		Max:   round(s.max, digits),
	}
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// OutputStats are the requests measured in one unit, tokens or chunks.
type OutputStats struct {
	Unit     string           `json:"unit"`
	Requests int64            `json:"requests"`
	Sources  map[string]int64 `json:"sources"`
	Total    int64            `json:"total"`

	// per request
	Output       *Dist `json:"per_request"`
	TimePer      *Dist `json:"time_per_output_ms,omitempty"`
	OutputPerSec *Dist `json:"per_sec_per_request,omitempty"`

	// all requests: Total over the time from the first send to the last answer's end
	SystemPerSec  float64 `json:"per_sec"`
	WindowSeconds float64 `json:"window_seconds"`
}

type outputAgg struct {
	unit                   string
	requests, total        int64
	sources                map[string]int64
	output, tpot, perSec   sampler
	firstSent, lastEndedAt time.Time
}

func (a *outputAgg) add(r *types.LLMResult) {
	if a.sources == nil {
		a.sources = map[string]int64{}
	}
	a.requests++
	a.sources[r.OutputSource]++
	a.total += int64(r.Output)
	a.output.add(float64(r.Output))
	if r.HasTPOT {
		a.tpot.add(ms(r.TPOT))
	}
	if r.HasContent && r.LastContent > 0 {
		a.perSec.add(r.OutputPerSec)
	}
	if !r.SentAt.IsZero() {
		if a.firstSent.IsZero() || r.SentAt.Before(a.firstSent) {
			a.firstSent = r.SentAt
		}
		if end := r.SentAt.Add(r.LastContent); end.After(a.lastEndedAt) {
			a.lastEndedAt = end
		}
	}
}

func (a *outputAgg) stats() *OutputStats {
	if a.requests == 0 {
		return nil
	}
	s := &OutputStats{
		Unit:         a.unit,
		Requests:     a.requests,
		Sources:      a.sources,
		Total:        a.total,
		Output:       a.output.dist(1),
		TimePer:      a.tpot.dist(2),
		OutputPerSec: a.perSec.dist(1),
	}
	if w := a.lastEndedAt.Sub(a.firstSent).Seconds(); w > 0 {
		s.WindowSeconds = round(w, 3)
		s.SystemPerSec = round(float64(a.total)/w, 1)
	}
	return s
}

// Summary is the token side of a step, a stage or the whole test.
type Summary struct {
	// Requests whose output was counted in tokens: from the server's usage or a tokenizer
	Tokens *OutputStats `json:"tokens,omitempty"`

	// Requests whose output could only be counted in content events
	Chunks *OutputStats `json:"chunks,omitempty"`

	// Time to the first event with output text, reasoning included
	FirstContent *Dist `json:"first_content_ms,omitempty"`

	// Gaps between consecutive content events, all requests pooled, as vLLM reports ITL
	ITL *Dist `json:"itl_ms,omitempty"`

	InputTokens  *Dist            `json:"input_tokens,omitempty"`
	InputSources map[string]int64 `json:"input_sources,omitempty"`

	// Requests that ended with no output text
	NoContent int64 `json:"no_content"`

	CacheBusted  int64 `json:"cache_busted,omitempty"`
	PrefixTokens int   `json:"shared_prefix_tokens,omitempty"`
}

// Agg aggregates LLM results. Not safe for concurrent use, like the report that holds it.
type Agg struct {
	tokens, chunks outputAgg
	firstContent   sampler
	itl            sampler
	input          sampler
	inputSources   map[string]int64
	noContent      int64
	cacheBusted    int64
	prefixTokens   int
	n              int64
}

// Add takes one request; nil is ignored.
func (a *Agg) Add(r *types.LLMResult) {
	if r == nil {
		return
	}
	a.n++
	if r.Tokens() {
		a.tokens.unit = "tokens"
		a.tokens.add(r)
	} else {
		a.chunks.unit = "chunks"
		a.chunks.add(r)
	}
	if r.HasContent {
		a.firstContent.add(ms(r.FirstContent))
	} else {
		a.noContent++
	}
	for _, g := range r.ITL {
		a.itl.add(ms(g))
	}
	if r.InputSource != "" {
		if a.inputSources == nil {
			a.inputSources = map[string]int64{}
		}
		a.inputSources[r.InputSource]++
		a.input.add(float64(r.Input))
	}
	if r.CacheBusted {
		a.cacheBusted++
	}
	if r.PrefixTokens > a.prefixTokens {
		a.prefixTokens = r.PrefixTokens
	}
}

// Summary is nil when no result was added.
func (a *Agg) Summary() *Summary {
	if a == nil || a.n == 0 {
		return nil
	}
	return &Summary{
		Tokens:       a.tokens.stats(),
		Chunks:       a.chunks.stats(),
		FirstContent: a.firstContent.dist(1),
		ITL:          a.itl.dist(2),
		InputTokens:  a.input.dist(1),
		InputSources: a.inputSources,
		NoContent:    a.noContent,
		CacheBusted:  a.cacheBusted,
		PrefixTokens: a.prefixTokens,
	}
}

func row(w io.Writer, indent, name string, d *Dist, format string) {
	if d == nil {
		return
	}
	f := "%s%s\t" + format + "\t" + format + "\t" + format + "\t" + format + "\t" + format + "\t" + format + "\t%d\n"
	fmt.Fprintf(w, f, indent, name, d.P50, d.P90, d.P95, d.P99, d.Max, d.Mean, d.Count)
}

func writeOutput(w io.Writer, indent string, s *OutputStats) {
	if s == nil {
		return
	}
	if s.Unit == "chunks" {
		row(w, indent, "Output Chunks", s.Output, "%.0f")
		row(w, indent, "Time/Chunk (ms)", s.TimePer, "%.2f")
		row(w, indent, "Chunks/s/Request", s.OutputPerSec, "%.1f")
		return
	}
	row(w, indent, "Output Tokens", s.Output, "%.0f")
	row(w, indent, "TPOT (ms)", s.TimePer, "%.2f")
	row(w, indent, "Tokens/s/Request", s.OutputPerSec, "%.1f")
}

// WriteText prints a step's or stage's summary as rows of a tab separated table.
func WriteText(w io.Writer, s *Summary) {
	if s == nil {
		return
	}
	fmt.Fprintf(w, "\nTokens (warm-up excluded):%s\n", sourcesText(s))
	fmt.Fprintln(w, "  \tp50\tp90\tp95\tp99\tmax\tmean\tcount")
	row(w, "  ", "First Content (ms)", s.FirstContent, "%.1f")
	writeOutput(w, "  ", s.Tokens)
	writeOutput(w, "  ", s.Chunks)
	row(w, "  ", "ITL (ms)", s.ITL, "%.2f")
	row(w, "  ", "Input Tokens", s.InputTokens, "%.0f")
	if s.NoContent > 0 {
		fmt.Fprintf(w, "  Answers with no output text: %d\n", s.NoContent)
	}
	if s.CacheBusted > 0 || s.PrefixTokens > 0 {
		fmt.Fprintf(w, "  Prefix cache: %d requests cache-busted, shared prefix %d tokens\n", s.CacheBusted, s.PrefixTokens)
	}
}

func sourcesText(s *Summary) string {
	out := ""
	if s.Tokens != nil {
		out += fmt.Sprintf(" %d in tokens (usage %d, tokenizer %d)", s.Tokens.Requests,
			s.Tokens.Sources[types.TokenSourceUsage], s.Tokens.Sources[types.TokenSourceTokenizer])
	}
	if s.Chunks != nil {
		if out != "" {
			out += ","
		}
		out += fmt.Sprintf(" %d in chunks, no usage and no tokenizer: not tokens", s.Chunks.Requests)
	}
	return out
}

// WriteSystem prints the whole test's throughput and the scheduling lag.
func WriteSystem(w io.Writer, s *Summary, sched *ScheduleSummary) {
	if s != nil {
		if t := s.Tokens; t != nil {
			fmt.Fprintf(w, "\nOutput throughput (warm-up excluded):\t%.1f tokens/s, %d tokens over %.1fs\n", t.SystemPerSec, t.Total, t.WindowSeconds)
		}
		if c := s.Chunks; c != nil {
			fmt.Fprintf(w, "Output throughput in chunks:\t%.1f chunks/s, %d chunks over %.1fs (not tokens)\n", c.SystemPerSec, c.Total, c.WindowSeconds)
		}
	}
	WriteSchedule(w, sched)
}
