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

package htmlreport

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"go.ddosify.com/ddosify/v2/core/report"
	"go.ddosify.com/ddosify/v2/core/report/live"
)

type logLatency struct {
	Count int64   `json:"count"`
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
}

func (l logLatency) live() live.Latency {
	return live.Latency{Count: l.Count, P50: l.P50, P90: l.P90, P95: l.P95, P99: l.P99}
}

// logLine holds the fields of a "progress" or a "request" line that the report reads.
type logLine struct {
	Type string `json:"type"`
	Time string `json:"time"`

	// progress
	ElapsedS    float64          `json:"elapsed_s"`
	StageRaw    json.RawMessage  `json:"stage"`
	TargetRPS   float64          `json:"target_rps"`
	AchievedRPS float64          `json:"achieved_rps"`
	InFlight    int64            `json:"in_flight"`
	Iterations  int64            `json:"iterations"`
	TTFT        logLatency       `json:"ttft_ms"`
	ITL         logLatency       `json:"itl_ms"`
	ITLKind     string           `json:"itl_kind"`
	E2E         logLatency       `json:"e2e_ms"`
	Outcomes    map[string]int64 `json:"outcomes"`
	Goodput     struct {
		Good  int64 `json:"good"`
		Total int64 `json:"total"`
	} `json:"goodput"`

	// request; "stage" is the stage's name there
	StageIndex *int     `json:"stage_index"`
	Warmup     bool     `json:"warmup"`
	StepID     uint16   `json:"step_id"`
	StepName   string   `json:"step_name"`
	Intended   string   `json:"intended_start"`
	Sent       string   `json:"sent"`
	Outcome    string   `json:"outcome"`
	FirstToken *float64 `json:"first_token_ms"`
	TotalMs    *float64 `json:"total_ms"`
}

type progressStage struct {
	Index  int    `json:"index"`
	Count  int    `json:"count"`
	Name   string `json:"name"`
	Warmup bool   `json:"warmup"`
}

// parseTime reads the log's RFC 3339 times; a time with no zone is taken as UTC.
func parseTime(s string) (int64, bool) {
	if s == "" {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		if t, err = time.Parse("2006-01-02T15:04:05.999999999", s); err != nil {
			return 0, false
		}
	}
	return t.UnixMilli(), true
}

// ErrNoReport is returned for a log with no "report" line, such as a run that was killed.
var ErrNoReport = errors.New(`no "report" line: run ddosify with --json and keep the whole output`)

// maxLine bounds one line; the report line of a test with many steps is the longest.
const maxLine = 64 << 20

// ReadLog builds the report of a run from its --json output: the "report" line, and the
// "progress" and "request" lines when they are there. Other lines are skipped.
func ReadLog(r io.Reader) (Report, error) {
	rec := NewRecorder(0)
	reqs := NewRequests()
	reqs.first, reqs.last = math.Inf(1), math.Inf(-1)
	var result *report.Result
	br := bufio.NewReaderSize(r, 1<<20)
	stageIndex := map[string]int{} // from the progress lines, for a request line with no stage_index
	for {
		line, err := br.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			buf := append([]byte(nil), line...)
			for errors.Is(err, bufio.ErrBufferFull) && len(buf) < maxLine {
				line, err = br.ReadSlice('\n')
				buf = append(buf, line...)
			}
			if errors.Is(err, bufio.ErrBufferFull) {
				return Report{}, fmt.Errorf("a line is longer than %d MB", maxLine>>20)
			}
			line = buf
		}
		if t := bytes.TrimSpace(line); len(t) > 0 && t[0] == '{' {
			if e := readLine(t, rec, reqs, &result, stageIndex); e != nil {
				return Report{}, e
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return Report{}, err
		}
	}
	if result == nil {
		return Report{}, ErrNoReport
	}
	cfg := Config{}
	if g := result.Goodput; g != nil {
		cfg.SLO = live.ThresholdsFromSLO(&g.SLO)
	}
	for _, s := range result.Stages {
		cfg.Stages = append(cfg.Stages, StageConfig{Name: s.Name, Warmup: s.Warmup, Count: int(s.Iterations), Concurrency: s.Concurrency})
	}
	rep := FromResult(result, cfg, rec)
	for _, s := range rep.Steps {
		rep.Config.Steps = append(rep.Config.Steps, StepConfig{ID: s.ID, Name: s.Name, Stream: s.Stream != nil})
	}
	if reqs.Len() > 0 {
		rep.Requests = reqs
	}
	rep.StartedAt, rep.EndedAt = runSpan(rep.Series, result.StartedAt, reqs.first, reqs.last)
	return rep, nil
}

// runSpan is when the run began and ended: the progress lines' first time less its elapsed
// seconds, else the report line's started_at, else the first request's intended start; and the
// last progress line or the last request's end, whichever is later.
func runSpan(pts []Point, startedAt string, firstReq, lastReq float64) (start, end time.Time) {
	var startMs, endMs float64
	for _, p := range pts {
		if p.At > 0 {
			startMs = float64(p.At) - p.T*1000
			break
		}
	}
	if startMs == 0 {
		if ms, ok := parseTime(startedAt); ok {
			startMs = float64(ms)
		}
	}
	if startMs == 0 && !math.IsInf(firstReq, 0) {
		startMs = firstReq
	}
	if startMs == 0 {
		return time.Time{}, time.Time{}
	}
	if n := len(pts); n > 0 {
		endMs = startMs + pts[n-1].T*1000
	}
	if !math.IsInf(lastReq, 0) {
		endMs = max(endMs, lastReq)
	}
	return time.UnixMilli(int64(startMs)), time.UnixMilli(int64(endMs))
}

func readLine(b []byte, rec *Recorder, reqs *Requests, result **report.Result, stageIndex map[string]int) error {
	var l logLine
	if err := json.Unmarshal(b, &l); err != nil {
		return nil // not one of ours
	}
	switch l.Type {
	case "report":
		var r report.Result
		if err := json.Unmarshal(b, &r); err != nil {
			return fmt.Errorf("the report line: %w", err)
		}
		*result = &r
	case "progress":
		s := live.Snapshot{
			Elapsed:     time.Duration(l.ElapsedS * float64(time.Second)),
			TargetRPS:   l.TargetRPS,
			AchievedRPS: l.AchievedRPS,
			InFlight:    l.InFlight,
			Iterations:  l.Iterations,
			TTFT:        l.TTFT.live(),
			ITL:         l.ITL.live(),
			E2E:         l.E2E.live(),
			Goodput:     live.Goodput{Good: l.Goodput.Good, Total: l.Goodput.Total},
			Outcomes:    live.OutcomesFromMap(l.Outcomes),
		}
		s.ITLIsEventGap = l.ITLKind == "event_gap"
		var st progressStage
		if len(l.StageRaw) > 0 && json.Unmarshal(l.StageRaw, &st) == nil && st.Count > 0 {
			s.Stage = live.StageRef{Index: st.Index, Count: st.Count, Name: st.Name, Warmup: st.Warmup}
			stageIndex[st.Name] = st.Index
		}
		if at, ok := parseTime(l.Time); ok {
			s.At = time.UnixMilli(at)
		}
		rec.Add(s)
	case "request":
		done, ok1 := parseTime(l.Time)
		sent, ok2 := parseTime(l.Sent)
		if !ok1 || !ok2 {
			return nil // never sent
		}
		intended, ok := parseTime(l.Intended)
		if !ok {
			intended = sent
		}
		var stageName string
		_ = json.Unmarshal(l.StageRaw, &stageName)
		r := Record{Stage: -1, Warmup: l.Warmup, StepID: l.StepID, StepName: l.StepName,
			Intended: float64(intended), Sent: float64(sent), Done: float64(done),
			Outcome: l.Outcome, FT: math.NaN()}
		if l.StageIndex != nil {
			r.Stage = *l.StageIndex
		} else if i, ok := stageIndex[stageName]; ok && stageName != "" {
			r.Stage = i
		}
		if l.Outcome == "" {
			r.Outcome = report.OutcomeOK
		}
		if l.FirstToken != nil {
			r.FT, r.HasFT = *l.FirstToken, true
		}
		if l.TotalMs != nil {
			r.Total = *l.TotalMs
		}
		reqs.Add(r)
		reqs.first, reqs.last = min(reqs.first, r.Intended), max(reqs.last, r.Done)
	}
	return nil
}
