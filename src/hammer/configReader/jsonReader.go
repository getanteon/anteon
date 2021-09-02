package configReader

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"ddosify.com/hammer/core/types"
)

type step struct {
	Id          int16                  `json:"id"`
	Url         string                 `json:"url"`
	Protocol    string                 `json:"protocol"`
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers"`
	Payload     string                 `json:"payload"`
	PayloadFile string                 `json:"payloadFile"`
	Timeout     int                    `json:"timeout"`
	Others      map[string]interface{} `json:"others"`
}

type jsonReader struct {
	ReqCount int    `json:"reqCount"`
	LoadType string `json:"loadType"`
	Duration int    `json:"duration"`
	Steps    []step `json:"steps"`
	Output   string `json:"output"`
	Proxy    string `json:"proxy"`
}

func (c *jsonReader) init(path string) (err error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}

	err = json.Unmarshal(byteValue, &c)
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
		Strategy: "single",
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

	return types.ScenarioItem{
		ID:       s.Id,
		URL:      s.Url,
		Protocol: strings.ToUpper(s.Protocol),
		Method:   strings.ToUpper(s.Method),
		Headers:  s.Headers,
		Payload:  payload,
		Timeout:  s.Timeout,
		Custom:   s.Others,
	}, nil
}
