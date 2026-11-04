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

package requester

import (
	"bytes"
	"io"
	"net/http"

	"go.ddosify.com/ddosify/core/llm"
)

// prepareLLM lets the step's llm config build or edit the body, after the variables are injected.
func (h *HttpRequester) prepareLLM(req *http.Request, envs, vars map[string]interface{}) (*llm.Prepared, error) {
	var body []byte
	if req.Body != nil {
		var err error
		if body, err = io.ReadAll(req.Body); err != nil {
			return nil, err
		}
	}
	p, err := h.llm.Prepare(body, envs)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(p.Body))
	req.ContentLength = int64(len(p.Body))
	for k, v := range p.Vars {
		vars[k] = v
	}
	return p, nil
}

func observerOf(p *llm.Prepared) eventObserver {
	if p == nil {
		return nil
	}
	return p.Observer
}
