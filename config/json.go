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
	"io/ioutil"
	"net/url"
	"strings"

	"ddosify.com/hammer/core/types"
	"ddosify.com/hammer/core/util"
)

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
	PayloadFile string                 `json:"payloadFile"`
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

type jsonReader struct {
	ReqCount int    `json:"reqCount"`
	LoadType string `json:"loadType"`
	Duration int    `json:"duration"`
	Steps    []step `json:"steps"`
	Output   string `json:"output"`
	Proxy    string `json:"proxy"`
}

func (j *jsonReader) UnmarshalJSON(data []byte) error {
	type jsonReaderAlias jsonReader
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

	*j = jsonReader(*defaultFields)
	return nil
}

func (c *jsonReader) init(jsonByte []byte) (err error) {
	err = json.Unmarshal(jsonByte, &c)
	if err != nil {
		return
	}
	return
}

func (c *jsonReader) CreateHammer() (h types.Hammer, err error) {
	// Scenario
	s := types.Scenario{}
	var si types.ScenarioItem
	for _, step := range c.Steps {
		si, err = stepToScenarioItem(step)
		if err != nil {
			return
		}

		s.Scenario = append(s.Scenario, si)
	}

	// Proxy
	var proxyURL *url.URL
	if c.Proxy != "" {
		proxyURL, err = url.Parse(c.Proxy)
		if err != nil {
			return
		}
	}
	p := types.Proxy{
		Strategy: types.ProxyTypeSingle,
		Addr:     proxyURL,
	}

	// Hammer
	h = types.Hammer{
		TotalReqCount:     c.ReqCount,
		LoadType:          strings.ToLower(c.LoadType),
		TestDuration:      c.Duration,
		Scenario:          s,
		Proxy:             p,
		ReportDestination: c.Output,
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
	url, err := util.StrToUrl(s.Protocol, s.Url)
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
