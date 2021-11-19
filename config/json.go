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

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"go.ddosify.com/ddosify/core/proxy"
	"go.ddosify.com/ddosify/core/types"
	"go.ddosify.com/ddosify/core/util"
)

const ConfigTypeJson = "jsonReader"

func init() {
	AvailableConfigReader[ConfigTypeJson] = &JsonReader{}
}

type timeRunCount []struct {
	Duration int `json:"duration"`
	Count    int `json:"count"`
}

type auth struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type step struct {
	Id          int16                  `json:"id"`
	Url         string                 `json:"url"`
	Protocol    string                 `json:"protocol"`
	Auth        auth                   `json:"auth"`
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers"`
	Payload     string                 `json:"payload"`
	PayloadFile string                 `json:"payload_file"`
	Timeout     int                    `json:"timeout"`
	Others      map[string]interface{} `json:"others"`
}

func (s *step) UnmarshalJSON(data []byte) error {
	type stepAlias step
	defaultFields := &stepAlias{
		Protocol: types.DefaultProtocol,
		Method:   types.DefaultMethod,
		Timeout:  types.DefaultTimeout,
	}

	err := json.Unmarshal(data, defaultFields)
	if err != nil {
		return err
	}

	*s = step(*defaultFields)
	return nil
}

type JsonReader struct {
	ReqCount     int          `json:"request_count"`
	LoadType     string       `json:"load_type"`
	Duration     int          `json:"duration"`
	TimeRunCount timeRunCount `json:"manual_load"`
	Steps        []step       `json:"steps"`
	Output       string       `json:"output"`
	Proxy        string       `json:"proxy"`
}

func (j *JsonReader) UnmarshalJSON(data []byte) error {
	type jsonReaderAlias JsonReader
	defaultFields := &jsonReaderAlias{
		ReqCount: types.DefaultReqCount,
		LoadType: types.DefaultLoadType,
		Duration: types.DefaultDuration,
		Output:   types.DefaultOutputType,
	}

	err := json.Unmarshal(data, defaultFields)
	if err != nil {
		return err
	}

	*j = JsonReader(*defaultFields)
	return nil
}

func (j *JsonReader) Init(jsonByte []byte) (err error) {
	if !json.Valid(jsonByte) {
		err = fmt.Errorf("provided json is invalid")
		return
	}

	err = json.Unmarshal(jsonByte, &j)
	if err != nil {
		return
	}
	return
}

func (j *JsonReader) CreateHammer() (h types.Hammer, err error) {
	// Scenario
	s := types.Scenario{}
	var si types.ScenarioItem
	for _, step := range j.Steps {
		si, err = stepToScenarioItem(step)
		if err != nil {
			return
		}

		s.Scenario = append(s.Scenario, si)
	}

	// Proxy
	var proxyURL *url.URL
	if j.Proxy != "" {
		proxyURL, err = url.Parse(j.Proxy)
		if err != nil {
			return
		}
	}
	p := proxy.Proxy{
		Strategy: proxy.ProxyTypeSingle,
		Addr:     proxyURL,
	}

	// TimeRunCount
	if len(j.TimeRunCount) > 0 {
		j.ReqCount, j.Duration = 0, 0
		for _, t := range j.TimeRunCount {
			j.ReqCount += t.Count
			j.Duration += t.Duration
		}
	}

	// Hammer
	h = types.Hammer{
		TotalReqCount:     j.ReqCount,
		LoadType:          strings.ToLower(j.LoadType),
		TestDuration:      j.Duration,
		TimeRunCountMap:   types.TimeRunCount(j.TimeRunCount),
		Scenario:          s,
		Proxy:             p,
		ReportDestination: j.Output,
	}
	return
}

func stepToScenarioItem(s step) (types.ScenarioItem, error) {
	var payload string
	if s.PayloadFile != "" {
		buf, err := ioutil.ReadFile(s.PayloadFile)
		if err != nil {
			return types.ScenarioItem{}, err
		}

		payload = string(buf)
	} else {
		payload = s.Payload
	}

	// Set default Auth type if not set
	if s.Auth != (auth{}) && s.Auth.Type == "" {
		s.Auth.Type = types.AuthHttpBasic
	}

	// Protocol & URL
	url, err := util.StrToURL(s.Protocol, s.Url)
	if err != nil {
		return types.ScenarioItem{}, err
	}

	return types.ScenarioItem{
		ID:       s.Id,
		URL:      url.String(),
		Protocol: strings.ToUpper(url.Scheme),
		Auth:     types.Auth(s.Auth),
		Method:   strings.ToUpper(s.Method),
		Headers:  s.Headers,
		Payload:  payload,
		Timeout:  s.Timeout,
		Custom:   s.Others,
	}, nil
}
