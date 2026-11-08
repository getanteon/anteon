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

// Server resource samples: CPU, memory and GPU of the system under test, read from CSV files of
// timestamp,source,metric,value and placed on the run's timeline.

package htmlreport

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	elive "go.ddosify.com/ddosify/v2/core/live"
	"go.ddosify.com/ddosify/v2/core/report"
)

// The metrics the report knows. Any other name is kept and drawn as a series of its own.
const (
	MetricCPUCores = "cpu_cores"     // CPU in use, in cores
	MetricCPUPct   = "cpu_pct"       // percent of one core, as docker stats prints it; read as cores
	MetricMem      = "mem_bytes"     // memory in use
	MetricGPUUtil  = "gpu_util_pct"  // GPU utilisation, 0 to 100
	MetricGPUMem   = "gpu_mem_bytes" // GPU memory in use
)

const (
	// ResourceMargin is how far before the run's start and after its end samples are still drawn.
	ResourceMargin = 15 * time.Second
	// maxResourceSeries bounds the source and metric pairs of a run; samples of more are left out.
	maxResourceSeries = 256
	maxNameLen        = 128
)

// Sample is one reading of one metric of one source.
type Sample struct {
	At     float64 // Unix seconds
	Source string
	Metric string
	Value  float64
}

// ParseStats counts what a file held.
type ParseStats struct {
	Samples  int
	Bad      int // lines that could not be read
	FirstBad int // the first of them, 1 based
}

var errNoSamples = errors.New("no line could be read as timestamp,source,metric,value")

// ParseSamples reads CSV of timestamp,source,metric,value and calls fn with every sample. The
// header names the columns, in any order; without one the columns are in that order. Blank lines
// and lines starting with # are skipped; a line that can not be read is counted, not fatal. A
// file with no sample is an error.
func ParseSamples(r io.Reader, fn func(Sample)) (ParseStats, error) {
	var st ParseStats
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	cols := [4]int{0, 1, 2, 3}
	first := true
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if line == 1 {
			text = strings.TrimPrefix(text, "\xef\xbb\xbf")
		}
		if text == "" || text[0] == '#' {
			continue
		}
		f := splitCSV(text)
		if first {
			first = false
			if h, ok, err := header(f); err != nil {
				return st, fmt.Errorf("line %d: %w", line, err)
			} else if ok {
				cols = h
				continue
			}
		}
		s, ok := sample(f, cols)
		if !ok {
			st.Bad++
			if st.FirstBad == 0 {
				st.FirstBad = line
			}
			continue
		}
		st.Samples++
		fn(s)
	}
	if err := sc.Err(); err != nil {
		return st, err
	}
	if st.Samples == 0 {
		return st, errNoSamples
	}
	return st, nil
}

func splitCSV(line string) []string {
	f := strings.Split(line, ",")
	for i, v := range f {
		v = strings.TrimSpace(v)
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			v = strings.TrimSpace(v[1 : len(v)-1])
		}
		f[i] = v
	}
	return f
}

var columnNames = [4]string{"timestamp", "source", "metric", "value"}

// header reads a header line: ok when the line names a column, an error when it does not name all four.
func header(f []string) (cols [4]int, ok bool, err error) {
	at := map[string]int{}
	for i, v := range f {
		at[strings.ToLower(v)] = i
	}
	if _, named := at["timestamp"]; !named {
		return cols, false, nil
	}
	for i, n := range columnNames {
		j, has := at[n]
		if !has {
			return cols, false, fmt.Errorf("the header names no %q column: it is timestamp,source,metric,value", n)
		}
		cols[i] = j
	}
	return cols, true, nil
}

func sample(f []string, cols [4]int) (Sample, bool) {
	for _, c := range cols {
		if c >= len(f) {
			return Sample{}, false
		}
	}
	at, ok := parseStamp(f[cols[0]])
	src, metric := f[cols[1]], strings.ToLower(f[cols[2]])
	if !ok || src == "" || metric == "" || len(src) > maxNameLen || len(metric) > maxNameLen {
		return Sample{}, false
	}
	v, ok := parseQuantity(f[cols[3]])
	if !ok {
		return Sample{}, false
	}
	switch metric {
	case MetricCPUPct:
		metric, v = MetricCPUCores, v/100
	}
	if knownMetric(metric) && v < 0 {
		return Sample{}, false
	}
	return Sample{At: at, Source: src, Metric: metric, Value: v}, true
}

func knownMetric(m string) bool {
	switch m {
	case MetricCPUCores, MetricMem, MetricGPUUtil, MetricGPUMem:
		return true
	}
	return false
}

// parseStamp reads Unix seconds, fractional or not (milliseconds are told apart by their size), or
// RFC 3339; a time with no zone is UTC, and a space may stand for the T.
func parseStamp(s string) (float64, bool) {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return 0, false
		}
		if v > 1e11 {
			v /= 1000
		}
		return v, true
	}
	if len(s) > 10 && s[10] == ' ' {
		s = s[:10] + "T" + s[11:]
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		if t, err = time.Parse("2006-01-02T15:04:05.999999999", s); err != nil {
			return 0, false
		}
	}
	return float64(t.UnixNano()) / 1e9, true
}

// Units a value may carry, as docker stats, kubectl top and nvidia-smi print them. Longer first.
var units = []struct {
	suffix string
	k      float64
}{
	{"KiB", 1 << 10}, {"MiB", 1 << 20}, {"GiB", 1 << 30}, {"TiB", 1 << 40},
	{"kB", 1e3}, {"KB", 1e3}, {"MB", 1e6}, {"GB", 1e9}, {"TB", 1e12},
	{"Ki", 1 << 10}, {"Mi", 1 << 20}, {"Gi", 1 << 30}, {"Ti", 1 << 40},
	{"k", 1e3}, {"M", 1e6}, {"G", 1e9}, {"T", 1e12},
	{"m", 1e-3}, {"%", 1}, {"B", 1},
}

// parseQuantity reads a number with an optional unit: 3.9, 391.2%, 3912m, 2120Mi, 2.07GiB.
func parseQuantity(s string) (float64, bool) {
	k := 1.0
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			s, k = strings.TrimSpace(strings.TrimSuffix(s, u.suffix)), u.k
			break
		}
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v * k, true
}

// ResStat is one series over one stage: its samples, their mean and the highest.
type ResStat struct {
	N   int
	Sum float64
	Max float64
}

func (s *ResStat) add(v float64) {
	if s.N == 0 || v > s.Max {
		s.Max = v
	}
	s.N++
	s.Sum += v
}

// Mean is the samples' mean, NaN without one.
func (s ResStat) Mean() float64 {
	if s.N == 0 {
		return math.NaN()
	}
	return s.Sum / float64(s.N)
}

// ResSource is one source over one stage.
type ResSource struct {
	Source string
	CPU    ResStat // cores
	Mem    ResStat // bytes
	GPU    ResStat // percent
	GPUMem ResStat // bytes
}

// ResStage is one stage's resources and throughput.
type ResStage struct {
	Index   int // the stage's own index, -1 for a run without stages
	Name    string
	Warmup  bool
	Seconds float64 // how long it ran
	OK      int64   // its requests that succeeded
	Sources []ResSource
}

// ResSeries is one source's metric over time, at most about DefaultMaxPoints points: the mean of
// the samples in each slice of the run. T is seconds from the run's start; nil V breaks the line.
type ResSeries struct {
	Source string     `json:"source"`
	Metric string     `json:"metric"`
	Step   float64    `json:"step"` // the median time between points
	T      []float64  `json:"t"`
	V      []*float64 `json:"v"`
}

// Resources are a run's server resource samples, aligned to it.
type Resources struct {
	Series []ResSeries
	Stages []ResStage
	Per    []string // which CPU sources throughput per core divides by; empty is all of them
	Files  []ResFile
}

// ResFile is what one file gave.
type ResFile struct {
	Name     string
	Read     int // samples read
	Inside   int // of them, within the run and its margin
	Outside  int
	Overflow int // left out past maxResourceSeries series
	Bad      int
	FirstBad int
}

type bucket struct {
	sumT, sumV float64
	n          int
}

type seriesAcc struct {
	source, metric string
	buckets        []bucket
	stages         []ResStat
}

type stageSpan struct {
	from, to float64 // Unix seconds
	st       ResStage
}

// resAcc places samples on a run's timeline: into fixed slices of the run for the charts, and into
// each stage's statistics, so memory does not grow with the samples.
type resAcc struct {
	start, lo, hi, width float64
	n                    int
	stages               []stageSpan
	series               map[[2]string]*seriesAcc
}

func newResAcc(start, end float64, stages []stageSpan) *resAcc {
	m := ResourceMargin.Seconds()
	a := &resAcc{start: start, lo: start - m, hi: end + m, stages: stages, series: map[[2]string]*seriesAcc{}}
	a.n = int(math.Min(DefaultMaxPoints, math.Max(1, math.Ceil(a.hi-a.lo))))
	a.width = (a.hi - a.lo) / float64(a.n)
	return a
}

type placed int

const (
	inside placed = iota
	outside
	overflow
)

func (a *resAcc) add(s Sample) placed {
	if s.At < a.lo || s.At > a.hi {
		return outside
	}
	k := [2]string{s.Source, s.Metric}
	sa := a.series[k]
	if sa == nil {
		if len(a.series) >= maxResourceSeries {
			return overflow
		}
		sa = &seriesAcc{source: s.Source, metric: s.Metric, buckets: make([]bucket, a.n), stages: make([]ResStat, len(a.stages))}
		a.series[k] = sa
	}
	i := min(a.n-1, int((s.At-a.lo)/a.width))
	b := &sa.buckets[i]
	b.sumT += s.At
	b.sumV += s.Value
	b.n++
	for j, sp := range a.stages {
		if s.At >= sp.from && s.At < sp.to {
			sa.stages[j].add(s.Value)
			break
		}
	}
	return inside
}

var familyOrder = map[string]int{MetricCPUCores: 0, MetricMem: 1, MetricGPUUtil: 2, MetricGPUMem: 3}

func familyRank(m string) int {
	if r, ok := familyOrder[m]; ok {
		return r
	}
	return len(familyOrder)
}

func (a *resAcc) result() *Resources {
	accs := make([]*seriesAcc, 0, len(a.series))
	for _, s := range a.series {
		accs = append(accs, s)
	}
	sort.Slice(accs, func(i, j int) bool {
		x, y := accs[i], accs[j]
		if rx, ry := familyRank(x.metric), familyRank(y.metric); rx != ry {
			return rx < ry
		}
		if x.metric != y.metric {
			return x.metric < y.metric
		}
		return x.source < y.source
	})
	res := &Resources{}
	for _, s := range accs {
		res.Series = append(res.Series, s.points(a.start, a.width))
	}
	for j, sp := range a.stages {
		st := sp.st
		bySource := map[string]*ResSource{}
		var names []string
		for _, s := range accs {
			stat := s.stages[j]
			if stat.N == 0 {
				continue
			}
			src := bySource[s.source]
			if src == nil {
				src = &ResSource{Source: s.source}
				bySource[s.source] = src
				names = append(names, s.source)
			}
			switch s.metric {
			case MetricCPUCores:
				src.CPU = stat
			case MetricMem:
				src.Mem = stat
			case MetricGPUUtil:
				src.GPU = stat
			case MetricGPUMem:
				src.GPUMem = stat
			}
		}
		sort.Strings(names)
		for _, n := range names {
			st.Sources = append(st.Sources, *bySource[n])
		}
		res.Stages = append(res.Stages, st)
	}
	return res
}

// points is the series' kept slices, each at its samples' mean time, with a break where the
// samples stop for much longer than usual, such as a restarted pod.
func (s *seriesAcc) points(start, width float64) ResSeries {
	out := ResSeries{Source: s.source, Metric: s.metric}
	var ts, vs []float64
	for _, b := range s.buckets {
		if b.n > 0 {
			ts = append(ts, b.sumT/float64(b.n)-start)
			vs = append(vs, b.sumV/float64(b.n))
		}
	}
	gaps := make([]float64, 0, len(ts))
	for i := 1; i < len(ts); i++ {
		gaps = append(gaps, ts[i]-ts[i-1])
	}
	if len(gaps) > 0 {
		sorted := append([]float64(nil), gaps...)
		sort.Float64s(sorted)
		out.Step = sorted[len(sorted)/2]
	}
	limit := math.Max(5*out.Step, 3*width)
	for i := range ts {
		if i > 0 && gaps[i-1] > limit {
			out.T = append(out.T, (ts[i-1]+ts[i])/2)
			out.V = append(out.V, nil)
		}
		out.T = append(out.T, ts[i])
		out.V = append(out.V, pf(vs[i]))
	}
	return out
}

// PerCore is a stage's succeeded requests a second for each CPU core in use: okRequests over
// seconds, over cores. nil when the stage has no length or no CPU was measured.
func PerCore(okRequests int64, seconds, cores float64) *float64 {
	if !(seconds > 0) || !(cores > 0) || okRequests < 0 {
		return nil
	}
	return pf(float64(okRequests) / seconds / cores)
}

// CoresOf is the sum of the mean CPU cores of the sources whose name holds one of per, or of
// every source when per is empty, and how many sources it summed.
func CoresOf(sources []ResSource, per []string) (float64, int) {
	sum, n := 0.0, 0
	for _, s := range sources {
		if s.CPU.N == 0 || !matches(s.Source, per) {
			continue
		}
		sum += s.CPU.Mean()
		n++
	}
	return sum, n
}

func matches(source string, per []string) bool {
	if len(per) == 0 {
		return true
	}
	for _, p := range per {
		if strings.Contains(source, p) {
			return true
		}
	}
	return false
}

// spansOf is when each stage ran, in Unix seconds, from the run's own stage windows; a run without
// stages is one span over the whole run.
func spansOf(r Report, start, end float64) []stageSpan {
	rel, _ := stageWindows(r.Spans, r.Stages)
	var out []stageSpan
	for i, s := range r.Stages {
		if math.IsNaN(rel[i][0]) {
			continue
		}
		from, to := start+rel[i][0], start+rel[i][1]
		if to <= from {
			continue
		}
		out = append(out, stageSpan{from: from, to: to,
			st: ResStage{Index: s.Index, Name: s.Name, Warmup: s.Warmup, Seconds: to - from, OK: s.Outcomes[report.OutcomeOK]}})
	}
	if len(r.Stages) == 0 {
		out = append(out, stageSpan{from: start, to: end + 1e-9,
			st: ResStage{Index: -1, Name: "whole run", Seconds: end - start, OK: r.Outcomes[report.OutcomeOK]}})
	}
	return out
}

// ErrNoTimeline is returned when a log says too little about when the run happened to place samples.
var ErrNoTimeline = errors.New(`the log does not say when the run happened (no "progress" or "request" lines and no started_at on the report line), so resource samples can not be placed`)

// LoadResources reads a run's sample files and places them on its timeline. per picks the CPU
// sources of throughput per core. What a file held that could not be used is said on stderr.
func LoadResources(r Report, files []elive.ResourceLog, per []string, stderr io.Writer) (*Resources, error) {
	if r.StartedAt.IsZero() || !r.EndedAt.After(r.StartedAt) {
		return nil, ErrNoTimeline
	}
	start := float64(r.StartedAt.UnixNano()) / 1e9
	end := float64(r.EndedAt.UnixNano()) / 1e9
	acc := newResAcc(start, end, spansOf(r, start, end))
	var stats []ResFile
	for _, f := range files {
		fs := ResFile{Name: f.Name}
		ps, err := ParseSamples(f.In, func(s Sample) {
			switch acc.add(s) {
			case inside:
				fs.Inside++
			case outside:
				fs.Outside++
			case overflow:
				fs.Overflow++
			}
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		fs.Read, fs.Bad, fs.FirstBad = ps.Samples, ps.Bad, ps.FirstBad
		stats = append(stats, fs)
	}
	res := acc.result()
	res.Per, res.Files = per, stats
	if stderr != nil {
		for _, fs := range stats {
			if msg := fs.note(r.StartedAt, r.EndedAt); msg != "" {
				fmt.Fprintln(stderr, msg)
			}
		}
	}
	return res, nil
}

// note says, in one line, what of the file was not used; empty when all of it was.
func (f ResFile) note(start, end time.Time) string {
	var parts []string
	if f.Bad > 0 {
		parts = append(parts, fmt.Sprintf("%d lines could not be read as timestamp,source,metric,value, the first is line %d", f.Bad, f.FirstBad))
	}
	if f.Inside == 0 {
		parts = append(parts, fmt.Sprintf("none of its %d samples falls within the run, %s to %s UTC: check that the clocks agree",
			f.Read, start.UTC().Format("2006-01-02 15:04:05"), end.UTC().Format("15:04:05")))
	} else if f.Outside > 0 {
		parts = append(parts, fmt.Sprintf("%d samples outside the run and its %s margin were left out", f.Outside, ResourceMargin))
	}
	if f.Overflow > 0 {
		parts = append(parts, fmt.Sprintf("%d samples past the first %d sources and metrics were left out", f.Overflow, maxResourceSeries))
	}
	if len(parts) == 0 {
		return ""
	}
	return f.Name + ": " + strings.Join(parts, "; ")
}
