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
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"strings"
)

const (
	chartW, chartH                  = 800, 240
	padL, padR, padT, padB          = 64, 16, 12, 28
	plotW, plotH                    = chartW - padL - padR, chartH - padT - padB
	yTicks                          = 4
	xTicks                          = 5
	dashSolid, dashLong, dashDotted = "", "7 4", "2 3"
)

// series is one line. A NaN value is a gap.
type series struct {
	Name string
	Dash string
	vals []float64
}

type chart struct {
	ID     string
	Title  string
	Unit   string
	Note   string
	SVG    template.HTML
	Legend []series
	Data   string // JSON for the hover readout
}

type stageMark struct {
	T     float64
	Label string
}

// niceCeil rounds up to 1, 2, 2.5 or 5 times a power of ten.
func niceCeil(v float64) float64 {
	if v <= 0 || math.IsNaN(v) {
		return 1
	}
	p := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if m*p >= v {
			return m * p
		}
	}
	return 10 * p
}

func buildChart(id, title, unit, note string, ts []float64, ss []series, marks []stageMark, fixedMax float64) chart {
	c := chart{ID: id, Title: title, Unit: unit, Note: note, Legend: ss}
	tmax := 0.0
	if len(ts) > 0 {
		tmax = ts[len(ts)-1]
	}
	if tmax <= 0 {
		tmax = 1
	}
	vmax := fixedMax
	if vmax == 0 {
		for _, s := range ss {
			for _, v := range s.vals {
				if !math.IsNaN(v) {
					vmax = math.Max(vmax, v)
				}
			}
		}
		vmax = niceCeil(vmax)
	}
	x := func(t float64) float64 { return padL + t/tmax*plotW }
	y := func(v float64) float64 { return padT + plotH - v/vmax*plotH }

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %d %d" role="img" aria-labelledby="%s-t" class="chart">`, chartW, chartH, id)
	fmt.Fprintf(&b, `<title id="%s-t">%s</title>`, id, template.HTMLEscapeString(title))
	for i := 0; i <= yTicks; i++ {
		v := vmax * float64(i) / yTicks
		yy := y(v)
		fmt.Fprintf(&b, `<line class="grid" x1="%d" x2="%d" y1="%.1f" y2="%.1f"/>`, padL, chartW-padR, yy, yy)
		fmt.Fprintf(&b, `<text class="tick" x="%d" y="%.1f" text-anchor="end">%s</text>`, padL-6, yy+4, axisNum(v, vmax/yTicks))
	}
	for i := 0; i <= xTicks; i++ {
		t := tmax * float64(i) / xTicks
		anchor := "middle"
		if i == 0 {
			anchor = "start"
		} else if i == xTicks {
			anchor = "end"
		}
		fmt.Fprintf(&b, `<text class="tick" x="%.1f" y="%d" text-anchor="%s">%s</text>`, x(t), chartH-8, anchor, clock(t))
	}
	for _, m := range marks {
		xx := x(m.T)
		fmt.Fprintf(&b, `<line class="stage" x1="%.1f" x2="%.1f" y1="%d" y2="%d"/>`, xx, xx, padT, padT+plotH)
		fmt.Fprintf(&b, `<text class="stagelabel" x="%.1f" y="%d">%s</text>`, xx+3, padT+10, template.HTMLEscapeString(m.Label))
	}
	for i, s := range ss {
		var d strings.Builder
		pen := false
		for j, v := range s.vals {
			if j >= len(ts) || math.IsNaN(v) {
				pen = false
				continue
			}
			cmd := "L"
			if !pen {
				cmd = "M"
			}
			fmt.Fprintf(&d, "%s%.1f %.1f ", cmd, x(ts[j]), y(math.Min(v, vmax)))
			pen = true
		}
		dash := ""
		if s.Dash != "" {
			dash = fmt.Sprintf(` stroke-dasharray="%s"`, s.Dash)
		}
		fmt.Fprintf(&b, `<path class="line s%d"%s d="%s"/>`, i, dash, strings.TrimSpace(d.String()))
	}
	fmt.Fprintf(&b, `<line class="cursor" x1="0" x2="0" y1="%d" y2="%d" visibility="hidden"/>`, padT, padT+plotH)
	b.WriteString(`</svg>`)
	c.SVG = template.HTML(b.String())

	type js struct {
		N string    `json:"n"`
		V []float64 `json:"v"`
	}
	data := struct {
		T    []float64 `json:"t"`
		U    string    `json:"u"`
		S    []js      `json:"s"`
		Tmax float64   `json:"tmax"`
		L    int       `json:"l"`
		W    int       `json:"w"`
	}{T: round(ts), U: unit, Tmax: tmax, L: padL, W: plotW}
	for _, s := range ss {
		vals := make([]float64, len(s.vals))
		for i, v := range s.vals {
			if math.IsNaN(v) {
				v = -1
			}
			vals[i] = math.Round(v*10) / 10
		}
		data.S = append(data.S, js{N: s.Name, V: vals})
	}
	raw, _ := json.Marshal(data)
	c.Data = string(raw)
	return c
}

func round(vs []float64) []float64 {
	out := make([]float64, len(vs))
	for i, v := range vs {
		out[i] = math.Round(v*10) / 10
	}
	return out
}

// axisNum prints a tick with as many decimals as the tick step needs.
func axisNum(v, step float64) string {
	if v == 0 {
		return "0"
	}
	if step >= 1 && v == math.Trunc(v) {
		return fmtInt(int64(math.Round(v)))
	}
	dec := 0
	for s := step; dec < 4 && math.Abs(s-math.Round(s)) > 1e-9; s *= 10 {
		dec++
	}
	return fmt.Sprintf("%.*f", dec, v)
}

func clock(sec float64) string {
	t := int64(math.Round(sec))
	h, m, s := t/3600, (t/60)%60, t%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func fmtInt(v int64) string {
	s := fmt.Sprintf("%d", v)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func fmtMs(v float64) string {
	if v < 10 {
		return fmt.Sprintf("%.1f", v)
	}
	return fmtInt(int64(math.Round(v)))
}
