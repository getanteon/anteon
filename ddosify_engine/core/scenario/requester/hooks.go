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

	"go.ddosify.com/ddosify/v2/core/hooks"
)

// prepareHooks hands the injected request to the step's hooks; each may rewrite the body.
func (h *HttpRequester) prepareHooks(req *http.Request, envs, vars map[string]interface{}) ([]hooks.Exchange, error) {
	var body []byte
	if req.Body != nil {
		var err error
		if body, err = io.ReadAll(req.Body); err != nil {
			return nil, err
		}
	}
	r := &hooks.Request{Step: &h.packet, HTTP: req, Body: body, Envs: envs, Vars: map[string]interface{}{}}
	var out []hooks.Exchange
	for _, hk := range h.stepHooks {
		ex, err := hk.Prepare(r)
		if err != nil {
			endExchanges(out, nil)
			return nil, err
		}
		if ex != nil {
			out = append(out, ex)
		}
	}
	req.Body = io.NopCloser(bytes.NewReader(r.Body))
	req.ContentLength = int64(len(r.Body))
	for k, v := range r.Vars {
		vars[k] = v
	}
	return out, nil
}

// endExchanges ends the exchanges of an answer that was not read as a stream.
func endExchanges(exs []hooks.Exchange, vars map[string]interface{}) {
	for _, ex := range exs {
		_, v := ex.End(nil)
		for k, val := range v {
			if vars != nil {
				vars[k] = val
			}
		}
	}
}
