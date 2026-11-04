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

// Package hooks is where optional features plug into the engine without the engine importing them.
// A step hook prepares each request of a step and sees its answer, with one state per request:
// a tokenizer counting a stream, a prompt builder, prefix cache busting, multi-turn sessions.
// There are also {{_name}} variables, config presets, and top level config keys that build a
// scenario (the OpenAI preset). A package registers itself in init(), and the binary imports it
// for that side effect (see ddosify_engine/plugins.go). Register at init time; lookups are safe
// from any goroutine.
package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"go.ddosify.com/ddosify/v2/core/types"
)

var (
	mu        sync.RWMutex
	stepHooks = map[string]StepHookFactory{}
	variables = map[string]func() interface{}{}
	presets   = map[string]Preset{}
	scenarios = map[string]ScenarioBuilder{}
)

// StepHookFactory builds a step's hook once per requester, or returns nil, nil to skip the step.
type StepHookFactory func(step *types.ScenarioStep) (StepHook, error)

// StepHook serves one step. It must be safe for concurrent use: requests of a step run in parallel.
type StepHook interface {
	// Prepare is called for every request, after the variables are injected and before it is sent.
	// It may change r.Body and r.HTTP's headers and add r.Vars, and returns the request's
	// Exchange, which then sees the answer, or nil.
	Prepare(r *Request) (Exchange, error)
}

// Request is one request a StepHook prepares.
type Request struct {
	Step *types.ScenarioStep
	HTTP *http.Request

	// The body after injection; set it to change what is sent
	Body []byte

	// The iteration's variables; read them, do not change them
	Envs map[string]interface{}

	// Variables for the later steps, set whatever the answer is
	Vars map[string]interface{}
}

// Exchange is one request's state, from Prepare to the end of its answer.
type Exchange interface {
	// Event is called for every dispatched SSE event, the done marker excluded.
	// at is the time since the request was sent.
	Event(at time.Duration, event, data string)

	// End is called once per request. r is the stream's result, which End may extend (r.LLM),
	// or nil when the answer was not read as a stream: an error status or a failed request.
	// Metrics are aggregated by the reports as distributions under their name; Vars go to the
	// iteration's variables for the later steps.
	End(r *types.StreamResult) (metrics map[string]float64, vars map[string]interface{})
}

// Preset changes a decoded JSON config before it is read.
type Preset func(config map[string]interface{}) error

// ScenarioBuilder turns a top level config key's value into the scenario's steps.
type ScenarioBuilder func(raw json.RawMessage) ([]types.ScenarioStep, error)

func RegisterStepHook(name string, f StepHookFactory) {
	mu.Lock()
	defer mu.Unlock()
	if f == nil {
		delete(stepHooks, name)
		return
	}
	stepHooks[name] = f
}

// RegisterVariable makes {{_name}} available in URLs, headers, payloads and env values.
func RegisterVariable(name string, fn func() interface{}) {
	mu.Lock()
	defer mu.Unlock()
	variables[name] = fn
}

func RegisterPreset(name string, p Preset) {
	mu.Lock()
	defer mu.Unlock()
	presets[name] = p
}

// RegisterScenario makes a top level config key, such as "llm", build the steps.
func RegisterScenario(key string, b ScenarioBuilder) {
	mu.Lock()
	defer mu.Unlock()
	scenarios[key] = b
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// StepHooksFor builds the hooks that want this step, in name order.
func StepHooksFor(step *types.ScenarioStep) ([]StepHook, error) {
	mu.RLock()
	defer mu.RUnlock()
	var out []StepHook
	for _, name := range sortedKeys(stepHooks) {
		h, err := stepHooks[name](step)
		if err != nil {
			return nil, fmt.Errorf("step %d, %s: %w", step.ID, name, err)
		}
		if h != nil {
			out = append(out, h)
		}
	}
	return out, nil
}

// Scenario returns the builder of a top level config key.
func Scenario(key string) (ScenarioBuilder, bool) {
	mu.RLock()
	defer mu.RUnlock()
	b, ok := scenarios[key]
	return b, ok
}

func ScenarioKeys() []string {
	mu.RLock()
	defer mu.RUnlock()
	return sortedKeys(scenarios)
}

func Variable(name string) (func() interface{}, bool) {
	mu.RLock()
	defer mu.RUnlock()
	fn, ok := variables[name]
	return fn, ok
}

func PresetNames() []string {
	mu.RLock()
	defer mu.RUnlock()
	return sortedKeys(presets)
}

// ApplyPresets applies the presets the config lists under "presets", then the given ones,
// and returns the config without the "presets" key.
func ApplyPresets(config []byte, extra []string) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(config, &m); err != nil {
		return nil, err
	}
	var names []string
	switch v := m["presets"].(type) {
	case nil:
	case string:
		names = append(names, v)
	case []interface{}:
		for _, n := range v {
			s, ok := n.(string)
			if !ok {
				return nil, fmt.Errorf("presets must be names, got %v", n)
			}
			names = append(names, s)
		}
	default:
		return nil, fmt.Errorf("presets must be a name or a list of names")
	}
	names = append(names, extra...)
	if len(names) == 0 {
		return config, nil
	}
	delete(m, "presets")
	for _, name := range names {
		mu.RLock()
		p, ok := presets[name]
		mu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("unknown preset %q, available: %v", name, PresetNames())
		}
		if err := p(m); err != nil {
			return nil, fmt.Errorf("preset %s: %w", name, err)
		}
	}
	return json.Marshal(m)
}
