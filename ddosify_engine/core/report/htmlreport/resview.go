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
	"fmt"
	"math"
	"strings"
)

// resRun is what the page's script draws of one run's resources. A comparison gives each run its
// own stages and length; a single run's page uses the page's.
type resRun struct {
	Label  string      `json:"label,omitempty"`
	Stages []stageData `json:"stages,omitempty"`
	End    float64     `json:"end"`
	Series []ResSeries `json:"series"`
}

type resRow struct {
	Run, Stage, Source           string
	First, Warmup, WarmupNote    bool
	CPUMean, CPUPeak, MemPeak    string
	GPUMean, GPUPeak, GPUMemPeak string
}

type resRateRow struct {
	Run, Stage           string
	Warmup, WarmupNote   bool
	Rate, Cores, PerCore string
	Sources              string
}

// resView is the Resources section's tables, of one run or of the runs compared.
type resView struct {
	Multi                             bool
	HasCPU, HasMem, HasGPU, HasGPUMem bool
	Rows                              []resRow
	Rates                             []resRateRow
	PerText                           string
}

type labeledRes struct {
	label string
	res   *Resources
}

func buildResView(runs []labeledRes) *resView {
	v := &resView{Multi: len(runs) > 1}
	var per []string
	for _, lr := range runs {
		if lr.res == nil {
			continue
		}
		per = lr.res.Per
		for _, st := range lr.res.Stages {
			note := st.Warmup && !strings.Contains(strings.ToLower(st.Name), "warm")
			if len(st.Sources) == 0 {
				v.Rows = append(v.Rows, resRow{Run: lr.label, Stage: st.Name, First: true, Warmup: st.Warmup, WarmupNote: note,
					Source: "no samples", CPUMean: "-", CPUPeak: "-", MemPeak: "-", GPUMean: "-", GPUPeak: "-", GPUMemPeak: "-"})
			}
			for i, s := range st.Sources {
				v.HasCPU = v.HasCPU || s.CPU.N > 0
				v.HasMem = v.HasMem || s.Mem.N > 0
				v.HasGPU = v.HasGPU || s.GPU.N > 0
				v.HasGPUMem = v.HasGPUMem || s.GPUMem.N > 0
				v.Rows = append(v.Rows, resRow{Run: lr.label, Stage: st.Name, Source: s.Source, First: i == 0,
					Warmup: st.Warmup, WarmupNote: note,
					CPUMean: statCell(s.CPU, true, fmtCores), CPUPeak: statCell(s.CPU, false, fmtCores),
					MemPeak: statCell(s.Mem, false, fmtBytes),
					GPUMean: statCell(s.GPU, true, fmtPct), GPUPeak: statCell(s.GPU, false, fmtPct),
					GPUMemPeak: statCell(s.GPUMem, false, fmtBytes)})
			}
			cores, n := CoresOf(st.Sources, lr.res.Per)
			row := resRateRow{Run: lr.label, Stage: st.Name, Warmup: st.Warmup, WarmupNote: note, Rate: "-", Cores: "-", PerCore: "-", Sources: "0"}
			if st.Seconds > 0 {
				row.Rate = fmtRate(float64(st.OK) / st.Seconds)
			}
			if n > 0 {
				row.Cores, row.Sources = fmtCores(cores), fmt.Sprint(n)
			}
			if pc := PerCore(st.OK, st.Seconds, cores); pc != nil && n > 0 {
				row.PerCore = fmtRate(*pc)
			}
			v.Rates = append(v.Rates, row)
		}
	}
	if !v.HasCPU {
		v.Rates = nil
	}
	v.PerText = "every CPU source"
	if len(per) > 0 {
		q := make([]string, len(per))
		for i, p := range per {
			q[i] = fmt.Sprintf("%q", p)
		}
		v.PerText = "the CPU sources whose name holds " + strings.Join(q, " or ")
	}
	return v
}

func statCell(s ResStat, mean bool, f func(float64) string) string {
	if s.N == 0 {
		return "-"
	}
	if mean {
		return f(s.Mean())
	}
	return f(s.Max)
}

func fmtCores(v float64) string {
	if v >= 100 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func fmtPct(v float64) string { return fmt.Sprintf("%.0f%%", v) }

func fmtRate(v float64) string {
	switch {
	case v >= 100:
		return fmtInt(int64(math.Round(v)))
	case v >= 10:
		return fmt.Sprintf("%.1f", v)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

func fmtBytes(v float64) string {
	switch {
	case v >= 1<<30:
		return fmt.Sprintf("%.2f GiB", v/(1<<30))
	case v >= 1<<20:
		return fmt.Sprintf("%.0f MiB", v/(1<<20))
	case v >= 1<<10:
		return fmt.Sprintf("%.0f KiB", v/(1<<10))
	default:
		return fmt.Sprintf("%.0f B", v)
	}
}

// resRunData is a run's resources for the script; stages is nil on a single run's page.
func resRunData(label string, r Report, withStages bool) *resRun {
	if r.Resources == nil {
		return nil
	}
	d := &resRun{Label: label, Series: r.Resources.Series, End: r.EndedAt.Sub(r.StartedAt).Seconds()}
	if d.Series == nil {
		d.Series = []ResSeries{}
	}
	if withStages {
		rel, _ := stageWindows(r.Spans, r.Stages)
		for i, s := range r.Stages {
			sd := stageData{Name: s.Name, Warmup: s.Warmup}
			if !math.IsNaN(rel[i][0]) {
				sd.Start, sd.End = pf(rel[i][0]), pf(rel[i][1])
			}
			d.Stages = append(d.Stages, sd)
		}
	}
	return d
}
