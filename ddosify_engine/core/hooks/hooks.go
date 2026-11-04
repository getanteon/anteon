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

// Package hooks is where optional features plug into the engine without the engine importing them:
// stream observers that see every SSE event (a tokenizer), request hooks (prefix cache busting),
// {{_name}} variables, and config presets (an OpenAI compatible endpoint). A package registers
// itself in init(), and the binary imports it for that side effect (see ddosify_engine/plugins.go).
// Register is meant for init time; lookups are safe from any goroutine.
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
	mu              sync.RWMutex
	streamObservers = map[string]StreamObserverFactory{}
	requestHooks    = map[string]RequestHook{}
	variables       = map[string]func() interface{}{}
	presets         = map[string]Preset{}
)

// StreamObserver sees one streamed response, on the goroutine that reads it.
type StreamObserver interface {
	// Event is called for every dispatched event, the done marker excluded.
	// at is the time since the request was sent. data must not be kept after the call.
	Event(at time.Duration, event string, data []byte)

	// End is called once, after the stream ended. Metrics are aggregated by the reports as
	// distributions under their name, such as "output_tokens" or "tpot_ms". Vars are added to
	// the iteration's variables, so a later step can use them, such as the answer for the next turn.
	End(r *types.StreamResult) (metrics map[string]float64, vars map[string]interface{})
}

// StreamObserverFactory returns an observer for one response of step, or nil to skip the step.
type StreamObserverFactory func(step *types.ScenarioStep) StreamObserver

// RequestHook may change a request before it is sent. envs holds the iteration's variables.
type RequestHook func(step *types.ScenarioStep, req *http.Request, envs map[string]interface{}) error

// Preset changes a decoded JSON config before it is read.
type Preset func(config map[string]interface{}) error

func RegisterStreamObserver(name string, f StreamObserverFactory) {
	mu.Lock()
	defer mu.Unlock()
	streamObservers[name] = f
}

func RegisterRequestHook(name string, h RequestHook) {
	mu.Lock()
	defer mu.Unlock()
	requestHooks[name] = h
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

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// StreamObserversFor returns the observers that want this step, in name order.
func StreamObserversFor(step *types.ScenarioStep) []StreamObserver {
	mu.RLock()
	defer mu.RUnlock()
	var out []StreamObserver
	for _, name := range sortedKeys(streamObservers) {
		if o := streamObservers[name](step); o != nil {
			out = append(out, o)
		}
	}
	return out
}

// RunRequestHooks runs every request hook, in name order.
func RunRequestHooks(step *types.ScenarioStep, req *http.Request, envs map[string]interface{}) error {
	mu.RLock()
	defer mu.RUnlock()
	for _, name := range sortedKeys(requestHooks) {
		if err := requestHooks[name](step, req, envs); err != nil {
			return fmt.Errorf("request hook %s: %w", name, err)
		}
	}
	return nil
}

func HasRequestHooks() bool {
	mu.RLock()
	defer mu.RUnlock()
	return len(requestHooks) > 0
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
