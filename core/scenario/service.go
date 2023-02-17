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

package scenario

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.ddosify.com/ddosify/core/scenario/requester"
	"go.ddosify.com/ddosify/core/scenario/scripting/injection"
	"go.ddosify.com/ddosify/core/types"
	"go.ddosify.com/ddosify/core/types/regex"
)

// ScenarioService encapsulates proxy/scenario/requester information and runs the scenario.
type ScenarioService struct {
	// Client map structure [proxy_addr][]scenarioItemRequester
	// Each proxy represents a client.
	// Each scenarioItem has a requester
	clients map[*url.URL][]scenarioItemRequester

	cPool *clientPool

	scenario types.Scenario
	ctx      context.Context

	clientMutex sync.Mutex
	debug       bool
	engineMode  string

	ei        *injection.EnvironmentInjector
	iterIndex int64
}

// NewScenarioService is the constructor of the ScenarioService.
func NewScenarioService() *ScenarioService {
	return &ScenarioService{}
}

type ScenarioOpts struct {
	Debug                  bool
	IterationCount         int
	MaxConcurrentIterCount int
	EngineMode             string
}

// Init initializes the ScenarioService.clients with the given types.Scenario and proxies.
// Passes the given ctx to the underlying requestor so we are able to control the life of each request.
func (s *ScenarioService) Init(ctx context.Context, scenario types.Scenario,
	proxies []*url.URL, opts ScenarioOpts) (err error) {
	s.scenario = scenario
	s.ctx = ctx
	s.debug = opts.Debug
	s.clients = make(map[*url.URL][]scenarioItemRequester, len(proxies))

	ei := &injection.EnvironmentInjector{}
	ei.Init()
	s.ei = ei

	for _, p := range proxies {
		err = s.createRequesters(p)
		if err != nil {
			return
		}
	}
	vi := &injection.EnvironmentInjector{}
	vi.Init()
	s.ei = vi
	s.engineMode = opts.EngineMode

	if s.engineInUserMode() {
		// create client pool
		var initialCount int
		if s.engineMode == types.EngineModeRepeatedUser {
			initialCount = opts.MaxConcurrentIterCount
		} else if s.engineMode == types.EngineModeDistinctUser {
			initialCount = opts.IterationCount
		}
		s.cPool, err = NewClientPool(initialCount, opts.IterationCount, func() *http.Client { return &http.Client{} })
	}
	// s.cPool will be nil otherwise

	return
}

// Do executes the scenario for the given proxy.
// Returns "types.Response" filled by the requester of the given Proxy, injects the given startTime to the response
// Returns error only if types.Response.Err.Type is types.ErrorProxy or types.ErrorIntented
func (s *ScenarioService) Do(proxy *url.URL, startTime time.Time) (
	response *types.ScenarioResult, err *types.RequestError) {
	response = &types.ScenarioResult{StepResults: []*types.ScenarioStepResult{}}
	response.StartTime = startTime
	response.ProxyAddr = proxy
	rand.Seed(time.Now().UnixNano())

	requesters, e := s.getOrCreateRequesters(proxy)
	if e != nil {
		return nil, &types.RequestError{Type: types.ErrorUnkown, Reason: e.Error()}
	}

	// start envs separately for each iteration
	envs := make(map[string]interface{}, len(s.scenario.Envs))
	for k, v := range s.scenario.Envs {
		envs[k] = v
	}
	// inject dynamic variables beforehand for each iteration
	injectDynamicVars(s.ei, envs)
	// pass a row from data for each iteration
	s.enrichEnvFromData(envs)
	atomic.AddInt64(&s.iterIndex, 1)

	var client *http.Client
	if s.engineInUserMode() {
		// get client from pool
		client = s.cPool.Get()
		defer s.cPool.Put(client)
	}

	for _, sr := range requesters {
		var res *types.ScenarioStepResult
		switch sr.requester.Type() {
		case "HTTP":
			httpRequester := sr.requester.(requester.HttpRequesterI)
			res = httpRequester.Send(client, envs)
		default:
			res = &types.ScenarioStepResult{Err: types.RequestError{Type: fmt.Sprintf("type not defined: %s", sr.requester.Type())}}
		}

		if res.Err.Type == types.ErrorProxy || res.Err.Type == types.ErrorIntented {
			err = &res.Err
			if res.Err.Type == types.ErrorIntented {
				// Stop the loop. ErrorProxy can be fixed in time. But ErrorIntented is a signal to stop all.
				return
			}
		}
		response.StepResults = append(response.StepResults, res)

		// Sleep before running the next step
		if sr.sleeper != nil && len(s.scenario.Steps) > 1 {
			sr.sleeper.sleep()
		}

		enrichEnvFromPrevStep(envs, res.ExtractedEnvs)
	}

	return
}

func enrichEnvFromPrevStep(m1 map[string]interface{}, m2 map[string]interface{}) {
	for k, v := range m2 {
		m1[k] = v
	}
}

func (s *ScenarioService) engineInUserMode() bool {
	if s.engineMode == types.EngineModeDistinctUser || s.engineMode == types.EngineModeRepeatedUser {
		return true
	}
	return false
}

func (s *ScenarioService) enrichEnvFromData(envs map[string]interface{}) {
	var row map[string]interface{}
	sb := strings.Builder{}
	for key, csvData := range s.scenario.Data {
		lenRows := len(csvData.Rows)
		if csvData.Random {
			row = csvData.Rows[rand.Intn(lenRows)]
		} else {
			row = csvData.Rows[s.iterIndex%int64(lenRows)]
		}

		for tag, v := range row {
			sb.WriteString("data.")
			sb.WriteString(key)
			sb.WriteString(".")
			sb.WriteString(tag)
			// data.info.name
			envs[sb.String()] = v
			sb.Reset()
		}
	}
}

func (s *ScenarioService) Done() {
	for _, v := range s.clients {
		for _, r := range v {
			r.requester.Done()
		}
	}

	if s.cPool != nil {
		s.cPool.Done()
	}
}

func (s *ScenarioService) getOrCreateRequesters(proxy *url.URL) (requesters []scenarioItemRequester, err error) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	requesters, ok := s.clients[proxy]
	if !ok {
		err = s.createRequesters(proxy)
		if err != nil {
			return
		}
	}
	return s.clients[proxy], err
}

func (s *ScenarioService) createRequesters(proxy *url.URL) (err error) {
	s.clients[proxy] = []scenarioItemRequester{}
	for _, si := range s.scenario.Steps {
		var r requester.Requester
		r, err = requester.NewRequester(si)
		if err != nil {
			return
		}
		s.clients[proxy] = append(
			s.clients[proxy],
			scenarioItemRequester{
				scenarioItemID: si.ID,
				sleeper:        newSleeper(si.Sleep),
				requester:      r,
			},
		)

		switch r.Type() {
		case "HTTP":
			httpRequester := r.(requester.HttpRequesterI)
			err = httpRequester.Init(s.ctx, si, proxy, s.debug, s.ei)
		default:
			err = fmt.Errorf("type not defined: %s", r.Type())
		}

		if err != nil {
			return
		}
	}
	return err
}

func injectDynamicVars(vi *injection.EnvironmentInjector, envs map[string]interface{}) {
	dynamicRgx := regexp.MustCompile(regex.DynamicVariableRegex)
	for k, v := range envs {
		vStr, isStr := v.(string)
		if !isStr {
			continue
		}
		if dynamicRgx.MatchString(vStr) {
			injected, err := vi.InjectDynamic(vStr)
			if err != nil {
				continue
			}
			envs[k] = injected
		}
	}
}

type scenarioItemRequester struct {
	scenarioItemID uint16
	sleeper        Sleeper
	requester      requester.Requester
}

// Sleeper is the interface for implementing different sleep strategies.
type Sleeper interface {
	sleep()
}

// RangeSleep is the implementation of the range sleep feature
type RangeSleep struct {
	min int
	max int
}

func (rs *RangeSleep) sleep() {
	rand.Seed(time.Now().UnixNano())
	dur := rand.Intn(rs.max-rs.min+1) + rs.min
	time.Sleep(time.Duration(dur) * time.Millisecond)
}

// DurationSleep is the implementation of the exact duration sleep feature
type DurationSleep struct {
	duration int
}

func (ds *DurationSleep) sleep() {
	time.Sleep(time.Duration(ds.duration) * time.Millisecond)
}

// newSleeper is the factor method for the Sleeper implementations.
func newSleeper(sleepStr string) Sleeper {
	if sleepStr == "" {
		return nil
	}

	var sl Sleeper

	// Sleep field already validated in types.scenario.validate(). No need to check parsing errors here.
	s := strings.Split(sleepStr, "-")
	if len(s) == 2 {
		min, _ := strconv.Atoi(s[0])
		max, _ := strconv.Atoi(s[1])
		if min > max {
			min, max = max, min
		}

		sl = &RangeSleep{
			min: min,
			max: max,
		}
	} else {
		dur, _ := strconv.Atoi(s[0])

		sl = &DurationSleep{
			duration: dur,
		}
	}

	return sl
}
