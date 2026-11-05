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
	"encoding/json"
	"fmt"
	"time"

	"go.ddosify.com/ddosify/v2/core/hooks"
	"go.ddosify.com/ddosify/v2/core/types"
)

// The llm package plugs into the engine through core/hooks only.
func init() {
	hooks.RegisterStepHook("llm", newStepHook)
	hooks.RegisterScenario("llm", buildScenario)
	hooks.RegisterPreset(PresetOpenAI, func(c map[string]interface{}) error {
		m, ok := c["llm"].(map[string]interface{})
		if !ok {
			if c["llm"] != nil {
				return fmt.Errorf("llm must be an object")
			}
			m = map[string]interface{}{}
			c["llm"] = m
		}
		m["preset"] = PresetOpenAI
		return nil
	})
}

type stepHook struct{ s *Step }

func newStepHook(step *types.ScenarioStep) (hooks.StepHook, error) {
	if step.LLM == nil {
		return nil, nil
	}
	s, err := NewStep(step.LLM)
	if err != nil {
		return nil, err
	}
	return stepHook{s}, nil
}

// Prepare builds the chat body or edits the prompt, and returns the request's observer.
func (h stepHook) Prepare(r *hooks.Request) (hooks.Exchange, error) {
	p, err := h.s.Prepare(r.Body, r.Envs)
	if err != nil {
		return nil, err
	}
	r.Body = p.Body
	for k, v := range p.Vars {
		r.Vars[k] = v
	}
	return exchange{p.Observer}, nil
}

type exchange struct{ o *Observer }

func (e exchange) Event(at time.Duration, event, data string) { e.o.Event(data, at) }

func (e exchange) End(r *types.StreamResult) (map[string]float64, map[string]interface{}) {
	if r != nil {
		r.LLM = e.o.Finish(r.SentAt)
	}
	return nil, e.o.Vars()
}

func buildScenario(raw json.RawMessage) ([]types.ScenarioStep, error) {
	var p Preset
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return p.Steps()
}
