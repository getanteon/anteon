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

package cli

import (
	"bytes"
	"io"
	"sync"
)

// heldWriter passes writes through, or keeps them while held and writes them out on release.
type heldWriter struct {
	mu   sync.Mutex
	w    io.Writer
	held bool
	buf  bytes.Buffer
}

func (h *heldWriter) Write(p []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.held {
		return h.buf.Write(p)
	}
	return h.w.Write(p)
}

func (h *heldWriter) hold() {
	h.mu.Lock()
	h.held = true
	h.mu.Unlock()
}

func (h *heldWriter) release() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.held = false
	_, _ = h.buf.WriteTo(h.w)
}
