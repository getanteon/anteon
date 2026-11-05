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

package live

// ring keeps the last cap values; memory stays fixed however long the test runs.
type ring struct {
	buf  []float64
	next int
	full bool
}

func newRing(capacity int) *ring {
	return &ring{buf: make([]float64, capacity)}
}

func (r *ring) push(v float64) {
	r.buf[r.next] = v
	r.next = (r.next + 1) % len(r.buf)
	if r.next == 0 {
		r.full = true
	}
}

func (r *ring) len() int {
	if r.full {
		return len(r.buf)
	}
	return r.next
}

// last returns up to n most recent values, oldest first.
func (r *ring) last(n int) []float64 {
	l := r.len()
	if n > l {
		n = l
	}
	out := make([]float64, n)
	start := r.next - n
	if start < 0 {
		start += len(r.buf)
	}
	for i := 0; i < n; i++ {
		out[i] = r.buf[(start+i)%len(r.buf)]
	}
	return out
}
